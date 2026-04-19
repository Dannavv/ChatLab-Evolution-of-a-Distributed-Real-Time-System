package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/antigravity/chat-lab/shared/backend/metrics"
	"github.com/antigravity/chat-lab/shared/backend/protocol"
	"github.com/antigravity/chat-lab/shared/backend/telemetry"
)

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	clients      = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex

	activeConnections int64
	nodeID            = envOrDefault("NODE_ID", "chaos-api-01")
	port              = envOrDefault("PORT", "8080")
	redisURL          = envOrDefault("REDIS_URL", "redis://redis:6379")
	queueKey          = envOrDefault("QUEUE_KEY", "chat:ingest")
	eventsChannel     = envOrDefault("EVENTS_CHANNEL", "chat:events")
	redisClient       *redis.Client

	maxEnqueueAttempts   = envIntDefault("ENQUEUE_ATTEMPTS", 3)
	enqueueRetryBackoff  = time.Duration(envIntDefault("ENQUEUE_RETRY_BACKOFF_MS", 120)) * time.Millisecond
	breakerFailThreshold = envIntDefault("BREAKER_FAIL_THRESHOLD", 3)
	breakerCooldown      = time.Duration(envIntDefault("BREAKER_COOLDOWN_MS", 10000)) * time.Millisecond

	consecutiveFailures int64
	openedUntilMillis   int64

	chaosMu       sync.RWMutex
	chaosDropRate float64
	chaosDelayMs  int

	messagesEnqueued = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_messages_enqueued_total",
		Help: "Total number of messages accepted by API and enqueued",
	})
	queueDepthGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_ingress_queue_depth",
		Help: "Current length of Redis ingestion queue",
	})
	enqueueRetriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_enqueue_retries_total",
		Help: "Total number of enqueue retry attempts",
	})
	breakerOpenGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_api_circuit_breaker_open",
		Help: "Whether API enqueue circuit breaker is open (1 open, 0 closed)",
	})
	chaosInjectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_api_chaos_injected_total",
		Help: "Total number of API requests affected by chaos injection",
	})
)

type queuedMessage struct {
	ID      string           `json:"id"`
	Attempt int              `json:"attempt"`
	Message protocol.Message `json:"message"`
}

type statusResponse struct {
	NodeID             string  `json:"node_id"`
	ActiveConnections  int     `json:"active_connections"`
	QueueDepth         int64   `json:"queue_depth"`
	QueueKey           string  `json:"queue_key"`
	EventsChannel      string  `json:"events_channel"`
	BreakerOpen        bool    `json:"breaker_open"`
	BreakerThreshold   int     `json:"breaker_threshold"`
	BreakerCooldownMS  int64   `json:"breaker_cooldown_ms"`
	ConsecutiveFailure int64   `json:"consecutive_failures"`
	ChaosDropRate      float64 `json:"chaos_drop_rate"`
	ChaosDelayMS       int     `json:"chaos_delay_ms"`
}

type chaosRequest struct {
	DropRate *float64 `json:"drop_rate"`
	DelayMS  *int     `json:"delay_ms"`
}

func main() {
	rand.Seed(time.Now().UnixNano())
	redisClient = connectRedis(redisURL)
	go subscribeEvents(context.Background())
	go trackQueueDepth(context.Background())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/send", handleSend)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/chaos", handleChaos)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	fmt.Printf("Lab 06 API node %s listening on :%s\n", nodeID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	clientsMutex.Lock()
	clients[conn] = true
	atomic.AddInt64(&activeConnections, 1)
	metrics.ActiveConnections.Inc()
	clientsMutex.Unlock()

	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		atomic.AddInt64(&activeConnections, -1)
		metrics.ActiveConnections.Dec()
		clientsMutex.Unlock()
	}()

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg protocol.Message
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}

		prepareMessage(&msg)
		if err := enqueueWithResilience(r.Context(), msg); err != nil {
			metrics.DroppedMessagesTotal.Inc()
		}
	}
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	prepareMessage(&msg)
	if err := enqueueWithResilience(r.Context(), msg); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		metrics.DroppedMessagesTotal.Inc()
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	depth, err := redisClient.LLen(r.Context(), queueKey).Result()
	if err != nil {
		depth = -1
	}

	chaosMu.RLock()
	dropRate := chaosDropRate
	delayMS := chaosDelayMs
	chaosMu.RUnlock()

	json.NewEncoder(w).Encode(statusResponse{
		NodeID:             nodeID,
		ActiveConnections:  int(atomic.LoadInt64(&activeConnections)),
		QueueDepth:         depth,
		QueueKey:           queueKey,
		EventsChannel:      eventsChannel,
		BreakerOpen:        isBreakerOpen(),
		BreakerThreshold:   breakerFailThreshold,
		BreakerCooldownMS:  breakerCooldown.Milliseconds(),
		ConsecutiveFailure: atomic.LoadInt64(&consecutiveFailures),
		ChaosDropRate:      dropRate,
		ChaosDelayMS:       delayMS,
	})
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
	if req.DropRate != nil {
		if *req.DropRate < 0 || *req.DropRate > 1 {
			chaosMu.Unlock()
			http.Error(w, "drop_rate must be between 0 and 1", http.StatusBadRequest)
			return
		}
		chaosDropRate = *req.DropRate
	}
	if req.DelayMS != nil {
		if *req.DelayMS < 0 {
			chaosMu.Unlock()
			http.Error(w, "delay_ms must be >= 0", http.StatusBadRequest)
			return
		}
		chaosDelayMs = *req.DelayMS
	}
	chaosMu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
}

func prepareMessage(msg *protocol.Message) {
	if msg.UserID == "" {
		msg.UserID = "guest"
	}
	if msg.RoomID == "" {
		msg.RoomID = "general"
	}
	msg.Timestamp = time.Now().UnixMilli()
	msg.NodeID = nodeID
}

func enqueueWithResilience(ctx context.Context, msg protocol.Message) error {
	if isBreakerOpen() {
		return errors.New("enqueue circuit breaker open")
	}

	envelope := queuedMessage{
		ID:      fmt.Sprintf("%d-%d", msg.Timestamp, rand.Int63()),
		Attempt: 0,
		Message: msg,
	}

	for attempt := 1; attempt <= maxEnqueueAttempts; attempt++ {
		if err := applyChaos(); err != nil {
			// Don't call onEnqueueFailure here, wait for all retries to fail
			if attempt == maxEnqueueAttempts {
				onEnqueueFailure()
				return err
			}
			enqueueRetriesTotal.Inc()
			time.Sleep(enqueueRetryBackoff)
			continue
		}

		data, err := json.Marshal(envelope)
		if err != nil {
			return err
		}

		err = redisClient.LPush(ctx, queueKey, data).Err()
		if err == nil {
			messagesEnqueued.Inc()
			// Only reset consecutive failures if the entire operation succeeded
			onEnqueueSuccess()
			return nil
		}

		if attempt == maxEnqueueAttempts {
			onEnqueueFailure()
			return err
		}

		enqueueRetriesTotal.Inc()
		time.Sleep(enqueueRetryBackoff)
	}

	return errors.New("enqueue failed after retries")
}

func applyChaos() error {
	chaosMu.RLock()
	dropRate := chaosDropRate
	delay := chaosDelayMs
	chaosMu.RUnlock()

	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	if dropRate > 0 && rand.Float64() < dropRate {
		chaosInjectedTotal.Inc()
		return errors.New("chaos dropped enqueue operation")
	}
	return nil
}

func onEnqueueFailure() {
	newFailures := atomic.AddInt64(&consecutiveFailures, 1)
	if newFailures >= int64(breakerFailThreshold) {
		until := time.Now().Add(breakerCooldown).UnixMilli()
		atomic.StoreInt64(&openedUntilMillis, until)
		breakerOpenGauge.Set(1)
	}
}

func onEnqueueSuccess() {
	atomic.StoreInt64(&consecutiveFailures, 0)
	if !isBreakerOpen() {
		breakerOpenGauge.Set(0)
	}
}

func isBreakerOpen() bool {
	until := atomic.LoadInt64(&openedUntilMillis)
	if until == 0 {
		return false
	}
	if time.Now().UnixMilli() >= until {
		atomic.StoreInt64(&openedUntilMillis, 0)
		breakerOpenGauge.Set(0)
		return false
	}
	return true
}

func subscribeEvents(ctx context.Context) {
	for {
		if err := subscribeOnce(ctx); err != nil {
			log.Println("event subscribe error:", err)
			time.Sleep(2 * time.Second)
		}
	}
}

func subscribeOnce(ctx context.Context) error {
	pubsub := redisClient.Subscribe(ctx, eventsChannel)
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	for redisMsg := range pubsub.Channel() {
		var msg protocol.Message
		if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
			metrics.DroppedMessagesTotal.Inc()
			continue
		}
		broadcast(msg)
	}

	return nil
}

func trackQueueDepth(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			depth, err := redisClient.LLen(ctx, queueKey).Result()
			if err != nil {
				continue
			}
			queueDepthGauge.Set(float64(depth))
		}
	}
}

func broadcast(msg protocol.Message) {
	start := time.Now()

	clientsMutex.Lock()
	msg.Connections = len(clients)
	data, err := json.Marshal(msg)
	if err != nil {
		clientsMutex.Unlock()
		metrics.DroppedMessagesTotal.Inc()
		return
	}

	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			client.Close()
			delete(clients, client) // FIX: Purge dead connection
			metrics.DroppedMessagesTotal.Inc()
		}
	}
	clientsMutex.Unlock()

	metrics.MessagesTotal.Inc()
	metrics.MessageLatency.Observe(float64(time.Since(start).Milliseconds()))
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
