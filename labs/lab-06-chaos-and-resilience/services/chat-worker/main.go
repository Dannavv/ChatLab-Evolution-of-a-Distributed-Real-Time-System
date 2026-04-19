package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/antigravity/chat-lab/shared/backend/metrics"
	"github.com/antigravity/chat-lab/shared/backend/protocol"
	"github.com/antigravity/chat-lab/shared/backend/telemetry"
)

var (
	nodeID        = envOrDefault("NODE_ID", "chaos-worker-01")
	port          = envOrDefault("PORT", "8081")
	redisURL      = envOrDefault("REDIS_URL", "redis://redis:6379")
	dbURL         = envOrDefault("DB_URL", "postgres://user:pass@db:5432/chat?sslmode=disable")
	queueKey      = envOrDefault("QUEUE_KEY", "chat:ingest")
	processingKey = envOrDefault("PROCESSING_KEY", "chat:processing")
	eventsChannel = envOrDefault("EVENTS_CHANNEL", "chat:events")
	processedSet  = envOrDefault("PROCESSED_SET", "chat:processed")
	deadLetterKey = envOrDefault("DEAD_LETTER_KEY", "chat:dead")
	archivePath   = envOrDefault("ARCHIVE_DIR", "/archive")
	maxRetries    = envIntDefault("MAX_RETRIES", 3)

	dbConn      *sql.DB
	redisClient *redis.Client

	chaosMu       sync.RWMutex
	chaosFailRate float64
	chaosDelayMS  int

	workerProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_worker_processed_total",
		Help: "Total messages processed by worker",
	})
	workerFailuresTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_worker_failures_total",
		Help: "Total failed worker processing attempts",
	})
	workerRetriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_worker_retries_total",
		Help: "Total worker retries",
	})
	workerDeadLetterTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_worker_dead_letter_total",
		Help: "Total messages sent to dead letter queue",
	})
	workerIdempotentSkipsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_worker_idempotent_skips_total",
		Help: "Total duplicate messages skipped by idempotency check",
	})
	workerChaosInjectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_worker_chaos_injected_total",
		Help: "Total worker failures injected by chaos",
	})
)

type queuedMessage struct {
	ID      string           `json:"id"`
	Attempt int              `json:"attempt"`
	Message protocol.Message `json:"message"`
}

type chaosRequest struct {
	FailRate *float64 `json:"fail_rate"`
	DelayMS  *int     `json:"delay_ms"`
}

type statusResponse struct {
	NodeID      string  `json:"node_id"`
	QueueKey    string  `json:"queue_key"`
	Processing  string  `json:"processing_key"`
	DeadLetter  string  `json:"dead_letter_key"`
	MaxRetries  int     `json:"max_retries"`
	ChaosRate   float64 `json:"chaos_fail_rate"`
	ChaosDelay  int     `json:"chaos_delay_ms"`
}

func main() {
	rand.Seed(time.Now().UnixNano())
	redisClient = connectRedis(redisURL)
	dbConn = connectDB(dbURL)

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/chaos", handleChaos)
	http.HandleFunc("/status", handleStatus)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	go processQueue(context.Background())

	fmt.Printf("Lab 06 worker %s listening on :%s\n", nodeID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func processQueue(ctx context.Context) {
	for {
		payload, err := redisClient.BRPopLPush(ctx, queueKey, processingKey, 0).Result()
		if err != nil {
			workerFailuresTotal.Inc()
			time.Sleep(300 * time.Millisecond)
			continue
		}

		var item queuedMessage
		if err := json.Unmarshal([]byte(payload), &item); err != nil {
			workerFailuresTotal.Inc()
			metrics.DroppedMessagesTotal.Inc()
			ackProcessing(ctx, payload)
			continue
		}

		alreadyProcessed, err := redisClient.SIsMember(ctx, processedSet, item.ID).Result()
		if err == nil && alreadyProcessed {
			workerIdempotentSkipsTotal.Inc()
			ackProcessing(ctx, payload)
			continue
		}

		if err := applyChaos(); err != nil {
			workerFailuresTotal.Inc()
			metrics.DroppedMessagesTotal.Inc()
			handleRetry(ctx, item, payload)
			continue
		}

		// Process item (DB + Archive + PubSub)
		isNew, err := processItem(ctx, item)
		if err != nil {
			workerFailuresTotal.Inc()
			metrics.DroppedMessagesTotal.Inc()
			handleRetry(ctx, item, payload)
			continue
		}

		if !isNew {
			workerIdempotentSkipsTotal.Inc()
		} else {
			workerProcessedTotal.Inc()
			if err := redisClient.SAdd(ctx, processedSet, item.ID).Err(); err != nil {
				workerFailuresTotal.Inc()
			}
		}

		ackProcessing(ctx, payload)
	}
}

func processItem(ctx context.Context, item queuedMessage) (bool, error) {
	isNew, err := saveToDB(item)
	if err != nil {
		return false, err
	}
	if !isNew {
		return false, nil
	}
	if err := appendArchive(item); err != nil {
		return false, err
	}
	return true, publishEvent(ctx, item.Message)
}

func handleRetry(ctx context.Context, item queuedMessage, originalPayload string) {
	item.Attempt++

	if item.Attempt > maxRetries {
		payload, _ := json.Marshal(item)
		if err := redisClient.LPush(ctx, deadLetterKey, payload).Err(); err != nil {
			workerFailuresTotal.Inc()
		}
		workerDeadLetterTotal.Inc()
		ackProcessing(ctx, originalPayload)
		return
	}

	payload, err := json.Marshal(item)
	if err != nil {
		workerFailuresTotal.Inc()
		return
	}

	if err := redisClient.LPush(ctx, queueKey, payload).Err(); err != nil {
		workerFailuresTotal.Inc()
		return
	}

	workerRetriesTotal.Inc()
	ackProcessing(ctx, originalPayload) // Only ack AFTER successful push back
}

func ackProcessing(ctx context.Context, payload string) {
	if err := redisClient.LRem(ctx, processingKey, 1, payload).Err(); err != nil {
		workerFailuresTotal.Inc()
	}
}

func saveToDB(item queuedMessage) (bool, error) {
	res, err := dbConn.Exec(
		"INSERT INTO messages (message_id, user_id, room_id, content, ingress_node, worker_node, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW()) ON CONFLICT (message_id) DO NOTHING",
		item.ID,
		item.Message.UserID,
		item.Message.RoomID,
		item.Message.Content,
		item.Message.NodeID,
		nodeID,
	)
	if err != nil {
		return false, err
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}

func appendArchive(item queuedMessage) error {
	if err := os.MkdirAll(archivePath, 0o755); err != nil {
		return err
	}

	fileName := filepath.Join(archivePath, time.Now().UTC().Format("2006-01-02")+".jsonl")
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	if _, err := writer.Write(append(data, '\n')); err != nil {
		return err
	}
	return writer.Flush()
}

func publishEvent(ctx context.Context, msg protocol.Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return redisClient.Publish(ctx, eventsChannel, payload).Err()
}

func applyChaos() error {
	chaosMu.RLock()
	rate := chaosFailRate
	delay := chaosDelayMS
	chaosMu.RUnlock()

	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	if rate > 0 && rand.Float64() < rate {
		workerChaosInjectedTotal.Inc()
		return fmt.Errorf("chaos worker failure injected")
	}
	return nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
}

func handleChaos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req chaosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	chaosMu.Lock()
	if req.FailRate != nil {
		if *req.FailRate < 0 || *req.FailRate > 1 {
			chaosMu.Unlock()
			http.Error(w, "fail_rate must be between 0 and 1", http.StatusBadRequest)
			return
		}
		chaosFailRate = *req.FailRate
	}
	if req.DelayMS != nil {
		if *req.DelayMS < 0 {
			chaosMu.Unlock()
			http.Error(w, "delay_ms must be >= 0", http.StatusBadRequest)
			return
		}
		chaosDelayMS = *req.DelayMS
	}
	chaosMu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	chaosMu.RLock()
	rate := chaosFailRate
	delay := chaosDelayMS
	chaosMu.RUnlock()

	json.NewEncoder(w).Encode(statusResponse{
		NodeID:     nodeID,
		QueueKey:   queueKey,
		Processing: processingKey,
		DeadLetter: deadLetterKey,
		MaxRetries: maxRetries,
		ChaosRate:  rate,
		ChaosDelay: delay,
	})
}

func connectRedis(redisAddr string) *redis.Client {
	var client *redis.Client
	var err error

	for attempt := 0; attempt < 15; attempt++ {
		client, err = redisClientFromURL(redisAddr)
		if err == nil {
			err = client.Ping(context.Background()).Err()
		}
		if err == nil {
			return client
		}
		time.Sleep(2 * time.Second)
	}

	log.Fatal(err)
	return nil
}

func redisClientFromURL(redisAddr string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisAddr)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opts), nil
}

func connectDB(dbConn string) *sql.DB {
	var database *sql.DB
	var err error

	for attempt := 0; attempt < 15; attempt++ {
		database, err = sql.Open("postgres", dbConn)
		if err == nil {
			err = database.Ping()
		}
		if err == nil {
			return database
		}
		time.Sleep(2 * time.Second)
	}

	log.Fatal(err)
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envIntDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
