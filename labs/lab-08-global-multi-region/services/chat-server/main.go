package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
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

type eventEnvelope struct {
	EventID      string `json:"event_id"`
	Type         string `json:"type"`
	UserID       string `json:"user_id"`
	RoomID       string `json:"room_id"`
	Content      string `json:"content,omitempty"`
	NodeID       string `json:"node_id"`
	OriginRegion string `json:"origin_region"`
	SourceRegion string `json:"source_region"`
	Timestamp    int64  `json:"timestamp"`
}

type clientInfo struct {
	Conn   *websocket.Conn
	UserID string
	RoomID string
}

type statusResponse struct {
	NodeID            string              `json:"node_id"`
	Region            string              `json:"region"`
	ActiveConnections int                 `json:"active_connections"`
	Rooms             map[string][]string `json:"rooms"`
}

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	nodeID   = envOrDefault("NODE_ID", "global-01")
	region   = envOrDefault("REGION", "us")
	port     = envOrDefault("PORT", "8080")
	redisURL = envOrDefault("REDIS_URL", "redis://redis-us:6379")
	channel  = envOrDefault("EVENTS_CHANNEL", "chat:global-events")

	redisClient *redis.Client

	clients      = make(map[*websocket.Conn]*clientInfo)
	clientsMutex sync.Mutex

	seenEvents = make(map[string]int64)
	seenMutex  sync.Mutex

	activeConnections int64

	publishedEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_global_published_events_total",
		Help: "Events published by this region node",
	})
	receivedEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_global_received_events_total",
		Help: "Events received from Redis stream",
	})
	duplicateEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_global_duplicate_events_total",
		Help: "Duplicate events dropped by ID",
	})
)

func main() {
	redisClient = connectRedis(redisURL)
	go subscribeEvents(context.Background())
	go pruneSeenEvents()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	fmt.Printf("Lab 08 node %s (%s) listening on :%s\n", nodeID, region, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	// Initial registration to "global" room so they can see activity immediately
	addClient(conn, "guest", "world-room")
	defer removeClient(conn)

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var evt eventEnvelope
		if err := json.Unmarshal(payload, &evt); err != nil {
			continue
		}

		if evt.UserID == "" {
			evt.UserID = "guest"
		}
		if evt.RoomID == "" {
			evt.RoomID = "world-room"
		}
		if evt.Type == "" {
			evt.Type = "message"
		}
		evt.Timestamp = time.Now().UnixMilli()
		evt.NodeID = nodeID
		evt.SourceRegion = region
		if evt.OriginRegion == "" {
			evt.OriginRegion = region
		}
		if evt.EventID == "" {
			evt.EventID = fmt.Sprintf("%s-%s-%d", region, evt.UserID, evt.Timestamp)
		}

		// Update registration if they changed room/user in the UI
		updateClient(conn, evt.UserID, evt.RoomID)

		publishEvent(evt)
	}
}

func addClient(conn *websocket.Conn, userID, roomID string) {
	clientsMutex.Lock()
	clients[conn] = &clientInfo{Conn: conn, UserID: userID, RoomID: roomID}
	clientsMutex.Unlock()
	atomic.AddInt64(&activeConnections, 1)
	metrics.ActiveConnections.Inc()
}

func updateClient(conn *websocket.Conn, userID, roomID string) {
	clientsMutex.Lock()
	if info, ok := clients[conn]; ok {
		info.UserID = userID
		info.RoomID = roomID
	}
	clientsMutex.Unlock()
}

func removeClient(conn *websocket.Conn) {
	clientsMutex.Lock()
	delete(clients, conn)
	clientsMutex.Unlock()
	atomic.AddInt64(&activeConnections, -1)
	metrics.ActiveConnections.Dec()
}

func publishEvent(evt eventEnvelope) {
	payload, err := json.Marshal(evt)
	if err != nil {
		metrics.DroppedMessagesTotal.Inc()
		return
	}
	
	// CRITICAL FIX: The publish call was missing!
	if err := redisClient.Publish(context.Background(), channel, payload).Err(); err != nil {
		metrics.DroppedMessagesTotal.Inc()
		return
	}

	markSeen(evt.EventID)
	publishedEventsTotal.Inc()
}

func subscribeEvents(ctx context.Context) {
	for {
		if err := subscribeOnce(ctx); err != nil {
			log.Println("subscribe error:", err)
			time.Sleep(2 * time.Second)
		}
	}
}

func subscribeOnce(ctx context.Context) error {
	pubsub := redisClient.Subscribe(ctx, channel)
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	for msg := range pubsub.Channel() {
		var evt eventEnvelope
		if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
			metrics.DroppedMessagesTotal.Inc()
			continue
		}

		if isDuplicate(evt.EventID) {
			duplicateEventsTotal.Inc()
			continue
		}
		markSeen(evt.EventID)
		receivedEventsTotal.Inc()
		broadcastEvent(evt)
	}
	return nil
}

func broadcastEvent(evt eventEnvelope) {
	start := time.Now()
	payload, err := json.Marshal(evt)
	if err != nil {
		metrics.DroppedMessagesTotal.Inc()
		return
	}

	clientsMutex.Lock()
	for conn, info := range clients {
		if info.RoomID != evt.RoomID {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			conn.Close()
			delete(clients, conn)
			metrics.DroppedMessagesTotal.Inc()
		}
	}
	clientsMutex.Unlock()

	metrics.MessagesTotal.Inc()
	metrics.MessageLatency.Observe(float64(time.Since(start).Milliseconds()))
}

func isDuplicate(eventID string) bool {
	if eventID == "" {
		return false
	}
	seenMutex.Lock()
	_, ok := seenEvents[eventID]
	seenMutex.Unlock()
	return ok
}

func markSeen(eventID string) {
	if eventID == "" {
		return
	}
	seenMutex.Lock()
	seenEvents[eventID] = time.Now().UnixMilli()
	seenMutex.Unlock()
}

func pruneSeenEvents() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-2 * time.Minute).UnixMilli()
		seenMutex.Lock()
		for id, ts := range seenEvents {
			if ts < cutoff {
				delete(seenEvents, id)
			}
		}
		seenMutex.Unlock()
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	clientsMutex.Lock()
	rooms := map[string][]string{}
	for _, info := range clients {
		rooms[info.RoomID] = append(rooms[info.RoomID], info.UserID)
	}
	for roomID := range rooms {
		sort.Strings(rooms[roomID])
	}
	clientsMutex.Unlock()

	json.NewEncoder(w).Encode(statusResponse{
		NodeID:            nodeID,
		Region:            region,
		ActiveConnections: int(atomic.LoadInt64(&activeConnections)),
		Rooms:             rooms,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
}

func connectRedis(redisAddr string) *redis.Client {
	for i := 0; i < 15; i++ {
		opts, err := redis.ParseURL(redisAddr)
		if err == nil {
			client := redis.NewClient(opts)
			if err := client.Ping(context.Background()).Err(); err == nil {
				return client
			}
		}
		time.Sleep(2 * time.Second)
	}
	log.Fatal("redis not available")
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
