package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/antigravity/chat-lab/shared/backend/metrics"
	"github.com/antigravity/chat-lab/shared/backend/protocol"
	"github.com/antigravity/chat-lab/shared/backend/telemetry"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	clients      = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex

	gatewayNodeID  = envOrDefault("NODE_ID", "mesh-gateway-01")
	port          = envOrDefault("PORT", "8080")
	redisURL      = envOrDefault("REDIS_URL", "redis://redis:6379")
	messageSvcURL = envOrDefault("MESSAGE_SERVICE_URL", "http://message-service:8081")
	historySvcURL = envOrDefault("HISTORY_SERVICE_URL", "http://history-service:8082")
	eventsChannel = envOrDefault("EVENTS_CHANNEL", "chat:events")
	redisClient   *redis.Client

	activeConnections int64
	traceSeq          uint64

	rateLimitPerSec = envFloatDefault("RATE_LIMIT_PER_SEC", 2)
	rateBurst       = envFloatDefault("RATE_LIMIT_BURST", 4)
	
	// Circuit Breaker State
	breakerMu          sync.RWMutex
	breakerState       = "closed" // closed, open, half-open
	consecutiveErrors  int
	lastErrorTime      time.Time
	breakerThreshold   = envIntDefault("BREAKER_THRESHOLD", 5)
	breakerRecoveryMS  = envIntDefault("BREAKER_RECOVERY_MS", 5000)
	halfOpenSuccesses  int

	gatewayRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_gateway_requests_total",
		Help: "Total gateway message requests received",
	})
	gatewayRateLimitedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_gateway_rate_limited_total",
		Help: "Total gateway requests rejected by rate limiting",
	})
	gatewayForwardErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_gateway_forward_errors_total",
		Help: "Total errors forwarding messages to the message service",
	})
	gatewayForwardLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_gateway_forward_latency_ms",
		Help:    "Latency spent forwarding chat requests from gateway to message service",
		Buckets: prometheus.LinearBuckets(0, 10, 20),
	})
)

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

type statusResponse struct {
	NodeID            string  `json:"node_id"`
	ActiveConnections int     `json:"active_connections"`
	RateLimitPerSec   float64 `json:"rate_limit_per_sec"`
	RateBurst         float64 `json:"rate_burst"`
	ConnectedRooms    int     `json:"connected_rooms"`
	EventsChannel     string  `json:"events_channel"`
	MessageServiceURL string  `json:"message_service_url"`
	HistoryServiceURL string  `json:"history_service_url"`
}

type historyResponse struct {
	Room     string            `json:"room"`
	Count    int               `json:"count"`
	Messages []protocol.Message `json:"messages"`
}

func main() {
	if collectorURL := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); collectorURL != "" {
		shutdown := telemetry.InitTracer("gateway", collectorURL)
		defer shutdown()
	}

	redisClient = connectRedis(redisURL)
	go subscribeEvents(context.Background())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/send", handleSend)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/history", handleHistory)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	
	server := &http.Server{Addr: ":" + port}
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Printf("\nShutting down gateway %s...\n", gatewayNodeID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	fmt.Printf("Lab 11/10 Hardened Gateway %s listening on :%s\n", gatewayNodeID, port)
	log.Fatal(server.ListenAndServe())
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
		if !allowRequest(msg.UserID) {
			gatewayRateLimitedTotal.Inc()
			continue
		}
		if err := forwardMessage(r.Context(), &msg); err != nil {
			log.Println("forward error:", err)
		}
	}
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !allowRequest(msg.UserID) {
		gatewayRateLimitedTotal.Inc()
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	if err := forwardMessage(r.Context(), &msg); err != nil {
		gatewayForwardErrorsTotal.Inc()
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	room := r.URL.Query().Get("room")
	if room == "" {
		room = "general"
	}
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "20"
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, fmt.Sprintf("%s/history?room=%s&limit=%s", historySvcURL, room, limit), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("X-Trace-Id", ulid.Make().String())
	otel.GetTextMapPropagator().Inject(r.Context(), propagation.HeaderCarrier(req.Header))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(statusResponse{
		NodeID:            gatewayNodeID,
		ActiveConnections: int(atomic.LoadInt64(&activeConnections)),
		RateLimitPerSec:   rateLimitPerSec,
		RateBurst:         rateBurst,
		ConnectedRooms:    0, // Redis-backed, local count no longer applies
		EventsChannel:     eventsChannel,
		MessageServiceURL: messageSvcURL,
		HistoryServiceURL: historySvcURL,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: gatewayNodeID})
}

func forwardMessage(ctx context.Context, msg *protocol.Message) error {
	if !allowBreaker() {
		return fmt.Errorf("circuit breaker is open")
	}

	tr := otel.Tracer("gateway")
	ctx, span := tr.Start(ctx, "forwardMessage")
	defer span.End()

	msg.Timestamp = time.Now().UnixMilli()
	msg.NodeID = gatewayNodeID
	msg.SourceService = gatewayNodeID
	
	// Generate stable ULID
	msg.MessageID = ulid.Make().String()
	msg.TraceID = msg.MessageID // Reuse for simplicity in this lab

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, messageSvcURL+"/messages", bytesReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", msg.TraceID)
	req.Header.Set("X-Parent-Service", gatewayNodeID)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	start := time.Now()
	// Set a reasonable timeout for the request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		onBreakerError()
		gatewayForwardErrorsTotal.Inc()
		return err
	}
	defer resp.Body.Close()
	gatewayForwardLatency.Observe(float64(time.Since(start).Milliseconds()))

	if resp.StatusCode >= 500 {
		onBreakerError()
		gatewayForwardErrorsTotal.Inc()
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("message service status %d: %s", resp.StatusCode, string(body))
	}

	onBreakerSuccess()
	gatewayRequestsTotal.Inc()
	return nil
}

func subscribeEvents(ctx context.Context) {
	for {
		if err := subscribeOnce(ctx); err != nil {
			log.Println("gateway subscribe error:", err)
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

func broadcast(msg protocol.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		metrics.DroppedMessagesTotal.Inc()
		return
	}

	clientsMutex.Lock()
	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			client.Close()
			metrics.DroppedMessagesTotal.Inc()
		}
	}
	clientsMutex.Unlock()

	metrics.MessagesTotal.Inc()
	latency := float64(time.Now().UnixMilli() - msg.Timestamp)
	metrics.MessageLatency.Observe(latency)
}

func allowRequest(userID string) bool {
	if userID == "" {
		userID = "guest"
	}

	// Redis Lua script for Token Bucket rate limiting
	// Keys: [rate_key]
	// Args: [max_tokens, fill_rate, now_ts]
	const script = `
		local key = KEYS[1]
		local max_tokens = tonumber(ARGV[1])
		local fill_rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])

		local bucket = redis.call('HMGET', key, 'tokens', 'last_fill')
		local tokens = tonumber(bucket[1]) or max_tokens
		local last_fill = tonumber(bucket[2]) or now

		local delta = math.max(0, now - last_fill)
		tokens = math.min(max_tokens, tokens + delta * fill_rate)

		if tokens < 1 then
			return 0
		else
			tokens = tokens - 1
			redis.call('HMSET', key, 'tokens', tokens, 'last_fill', now)
			redis.call('EXPIRE', key, 60)
			return 1
		end
	`

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	key := fmt.Sprintf("ratelimit:%s", userID)
	now := float64(time.Now().UnixNano()) / 1e9

	res, err := redisClient.Eval(ctx, script, []string{key}, rateBurst, rateLimitPerSec, now).Int()
	if err != nil {
		// Fallback to allowing if Redis is down (Best effort)
		return true
	}

	return res == 1
}

func allowBreaker() bool {
	breakerMu.Lock()
	defer breakerMu.Unlock()

	if breakerState == "closed" {
		return true
	}

	if breakerState == "open" {
		if time.Since(lastErrorTime) > time.Duration(breakerRecoveryMS)*time.Millisecond {
			breakerState = "half-open"
			halfOpenSuccesses = 0
			return true
		}
		return false
	}

	// half-open: allow limited traffic
	return true
}

func onBreakerError() {
	breakerMu.Lock()
	defer breakerMu.Unlock()

	if breakerState == "closed" {
		consecutiveErrors++
		if consecutiveErrors >= breakerThreshold {
			breakerState = "open"
			lastErrorTime = time.Now()
		}
	} else if breakerState == "half-open" {
		breakerState = "open"
		lastErrorTime = time.Now()
	}
}

func onBreakerSuccess() {
	breakerMu.Lock()
	defer breakerMu.Unlock()

	if breakerState == "half-open" {
		halfOpenSuccesses++
		if halfOpenSuccesses >= 3 {
			breakerState = "closed"
			consecutiveErrors = 0
		}
	} else if breakerState == "closed" {
		consecutiveErrors = 0
	}
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

func envFloatDefault(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
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

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func bytesReader(payload []byte) *bytes.Buffer {
	return bytes.NewBuffer(payload)
}
