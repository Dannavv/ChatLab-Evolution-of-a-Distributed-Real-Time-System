package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	nodeID        = envOrDefault("NODE_ID", "cloud-worker-01")
	port          = envOrDefault("PORT", "8081")
	redisURL      = envOrDefault("REDIS_URL", "redis://redis:6379")
	dbURL         = envOrDefault("DB_URL", "postgres://user:pass@db:5432/chat?sslmode=disable")
	queueKey      = envOrDefault("QUEUE_KEY", "chat:ingest")
	eventsChannel = envOrDefault("EVENTS_CHANNEL", "chat:events")
	archivePath   = envOrDefault("ARCHIVE_DIR", "/archive")

	dbConn      *sql.DB
	redisClient *redis.Client

	workerProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_worker_processed_total",
		Help: "Total messages processed by worker",
	})
	workerFailuresTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_worker_failures_total",
		Help: "Total failed worker processing attempts",
	})
)

func main() {
	redisClient = connectRedis(redisURL)
	dbConn = connectDB(dbURL)

	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	go processQueue(context.Background())

	fmt.Printf("Lab 05 worker %s listening on :%s\n", nodeID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func processQueue(ctx context.Context) {
	for {
		result, err := redisClient.BRPop(ctx, 0, queueKey).Result()
		if err != nil {
			workerFailuresTotal.Inc()
			time.Sleep(250 * time.Millisecond)
			continue
		}
		if len(result) < 2 {
			continue
		}

		var msg protocol.Message
		if err := json.Unmarshal([]byte(result[1]), &msg); err != nil {
			workerFailuresTotal.Inc()
			metrics.DroppedMessagesTotal.Inc()
			continue
		}

		msg.NodeID = nodeID
		if err := saveToDB(msg); err != nil {
			workerFailuresTotal.Inc()
			metrics.DroppedMessagesTotal.Inc()
			continue
		}

		if err := appendArchive(msg); err != nil {
			workerFailuresTotal.Inc()
		}

		if err := publishEvent(ctx, msg); err != nil {
			workerFailuresTotal.Inc()
			metrics.DroppedMessagesTotal.Inc()
			continue
		}

		workerProcessedTotal.Inc()
	}
}

func saveToDB(msg protocol.Message) error {
	_, err := dbConn.Exec(
		"INSERT INTO messages (user_id, room_id, content, ingress_node, worker_node, created_at) VALUES ($1, $2, $3, $4, $5, NOW())",
		msg.UserID,
		msg.RoomID,
		msg.Content,
		msg.NodeID,
		nodeID,
	)
	return err
}

func appendArchive(msg protocol.Message) error {
	if err := os.MkdirAll(archivePath, 0o755); err != nil {
		return err
	}

	fileName := filepath.Join(archivePath, time.Now().UTC().Format("2006-01-02")+".jsonl")
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(msg)
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

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
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
