package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	nodeID            = envOrDefault("NODE_ID", "cloud-api-01")
	port              = envOrDefault("PORT", "8080")
	redisURL          = envOrDefault("REDIS_URL", "redis://redis:6379")
	queueKey          = envOrDefault("QUEUE_KEY", "chat:ingest")
	eventsChannel     = envOrDefault("EVENTS_CHANNEL", "chat:events")
	redisClient       *redis.Client

	messagesEnqueued = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_messages_enqueued_total",
		Help: "Total number of messages accepted by API and enqueued",
	})

	queueDepthGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_ingress_queue_depth",
		Help: "Current length of Redis ingestion queue",
	})
)

type statusResponse struct {
	NodeID            string `json:"node_id"`
	ActiveConnections int    `json:"active_connections"`
	QueueDepth        int64  `json:"queue_depth"`
	QueueKey          string `json:"queue_key"`
	EventsChannel     string `json:"events_channel"`
}

func main() {
	redisClient = connectRedis(redisURL)
	go subscribeEvents(context.Background())
	go trackQueueDepth(context.Background())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/send", handleSend)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)

	fmt.Printf("Lab 05 API node %s listening on :%s\n", nodeID, port)
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
		if err := enqueueMessage(r.Context(), msg); err != nil {
			log.Println("enqueue error:", err)
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
	if err := enqueueMessage(r.Context(), msg); err != nil {
		http.Error(w, "failed to enqueue", http.StatusServiceUnavailable)
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
	json.NewEncoder(w).Encode(statusResponse{
		NodeID:            nodeID,
		ActiveConnections: int(atomic.LoadInt64(&activeConnections)),
		QueueDepth:        depth,
		QueueKey:          queueKey,
		EventsChannel:     eventsChannel,
	})
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

func enqueueMessage(ctx context.Context, msg protocol.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := redisClient.LPush(ctx, queueKey, data).Err(); err != nil {
		return err
	}
	messagesEnqueued.Inc()
	return nil
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
