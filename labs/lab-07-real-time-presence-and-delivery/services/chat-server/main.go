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

type clientInfo struct {
	Conn   *websocket.Conn
	UserID string
	RoomID string
}

type roomState struct {
	OnlineUsers map[string]bool `json:"online_users"`
	TypingUsers map[string]int64 `json:"typing_users"`
}

type eventEnvelope struct {
	Type      string `json:"type"`
	EventID   string `json:"event_id"`
	MessageID string `json:"message_id,omitempty"`
	UserID    string `json:"user_id"`
	RoomID    string `json:"room_id"`
	Content   string `json:"content,omitempty"`
	NodeID    string `json:"node_id"`
	Timestamp int64  `json:"timestamp"`
	Status    string `json:"status,omitempty"`
	Recipients int   `json:"recipients,omitempty"`
}

type statusResponse struct {
	NodeID            string                `json:"node_id"`
	ActiveConnections int                   `json:"active_connections"`
	Rooms             map[string]roomStatus `json:"rooms"`
}

type roomStatus struct {
	Online int      `json:"online"`
	Users  []string `json:"users"`
	Typing []string `json:"typing"`
}

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	nodeID       = envOrDefault("NODE_ID", "presence-01")
	port         = envOrDefault("PORT", "8080")
	redisURL     = envOrDefault("REDIS_URL", "redis://redis:6379")
	eventsChannel = envOrDefault("EVENTS_CHANNEL", "chat:presence-events")
	redisClient  *redis.Client

	clients      = make(map[*websocket.Conn]*clientInfo)
	clientsMutex sync.Mutex

	rooms      = make(map[string]*roomState)
	roomsMutex sync.Mutex

	activeConnections int64

	presenceEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_presence_events_total",
		Help: "Total presence online/offline events",
	})
	typingEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_typing_events_total",
		Help: "Total typing events",
	})
	readReceiptEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_read_receipt_events_total",
		Help: "Total read receipt events",
	})
	deliveryEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_delivery_events_total",
		Help: "Total delivery status events",
	})
)

func main() {
	redisClient = connectRedis(redisURL)
	go subscribeEvents(context.Background())
	go pruneTypingState()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	fmt.Printf("Lab 07 presence node %s listening on :%s\n", nodeID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	var registered bool
	var userID, roomID string

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			if registered {
				disconnectClient(conn, userID, roomID)
			}
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
			evt.RoomID = "general"
		}
		evt.Timestamp = time.Now().UnixMilli()
		evt.NodeID = nodeID
		if evt.EventID == "" {
			evt.EventID = fmt.Sprintf("%d-%s", evt.Timestamp, evt.UserID)
		}
		if evt.Type == "" {
			evt.Type = "message" // FIX: Backward compatibility for legacy bench messages
		}

		if !registered {
			registerClient(conn, evt.UserID, evt.RoomID)
			registered = true
			userID = evt.UserID
			roomID = evt.RoomID
			publishEvent(eventEnvelope{
				Type:      "presence",
				EventID:   fmt.Sprintf("presence-online-%d-%s", time.Now().UnixMilli(), evt.UserID),
				UserID:    evt.UserID,
				RoomID:    evt.RoomID,
				NodeID:    nodeID,
				Timestamp: time.Now().UnixMilli(),
				Status:    "online",
			})
		}

		handleIncomingEvent(evt)
	}
}

func handleIncomingEvent(evt eventEnvelope) {
	switch evt.Type {
	case "message":
		if evt.MessageID == "" {
			evt.MessageID = fmt.Sprintf("msg-%d-%s", evt.Timestamp, evt.UserID)
		}
		publishEvent(evt)
		recipients := roomRecipients(evt.RoomID)
		publishEvent(eventEnvelope{
			Type:       "delivery",
			EventID:    fmt.Sprintf("delivery-%d-%s", time.Now().UnixMilli(), evt.UserID),
			MessageID:  evt.MessageID,
			UserID:     evt.UserID,
			RoomID:     evt.RoomID,
			NodeID:     nodeID,
			Timestamp:  time.Now().UnixMilli(),
			Status:     "delivered",
			Recipients: recipients,
		})
	case "typing":
		recordTyping(evt.RoomID, evt.UserID)
		typingEventsTotal.Inc()
		publishEvent(evt)
	case "read":
		readReceiptEventsTotal.Inc()
		publishEvent(evt)
	case "presence":
		publishEvent(evt)
	}
}

func registerClient(conn *websocket.Conn, userID, roomID string) {
	clientsMutex.Lock()
	clients[conn] = &clientInfo{Conn: conn, UserID: userID, RoomID: roomID}
	clientsMutex.Unlock()

	atomic.AddInt64(&activeConnections, 1)
	metrics.ActiveConnections.Inc()
	addPresence(roomID, userID, true)
}

func disconnectClient(conn *websocket.Conn, userID, roomID string) {
	clientsMutex.Lock()
	delete(clients, conn)
	clientsMutex.Unlock()

	atomic.AddInt64(&activeConnections, -1)
	metrics.ActiveConnections.Dec()
	addPresence(roomID, userID, false)

	publishEvent(eventEnvelope{
		Type:      "presence",
		EventID:   fmt.Sprintf("presence-offline-%d-%s", time.Now().UnixMilli(), userID),
		UserID:    userID,
		RoomID:    roomID,
		NodeID:    nodeID,
		Timestamp: time.Now().UnixMilli(),
		Status:    "offline",
	})
}

func addPresence(roomID, userID string, online bool) {
	roomsMutex.Lock()
	defer roomsMutex.Unlock()

	state := ensureRoomState(roomID)
	if online {
		state.OnlineUsers[userID] = true
		presenceEventsTotal.Inc()
		return
	}
	delete(state.OnlineUsers, userID)
	delete(state.TypingUsers, userID)
	presenceEventsTotal.Inc()
}

func recordTyping(roomID, userID string) {
	roomsMutex.Lock()
	defer roomsMutex.Unlock()
	state := ensureRoomState(roomID)
	state.TypingUsers[userID] = time.Now().UnixMilli()
}

func pruneTypingState() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UnixMilli()
		roomsMutex.Lock()
		for _, state := range rooms {
			for user, ts := range state.TypingUsers {
				if now-ts > 3500 {
					delete(state.TypingUsers, user)
				}
			}
		}
		roomsMutex.Unlock()
	}
}

func ensureRoomState(roomID string) *roomState {
	state, ok := rooms[roomID]
	if !ok {
		state = &roomState{OnlineUsers: map[string]bool{}, TypingUsers: map[string]int64{}}
		rooms[roomID] = state
	}
	return state
}

func roomRecipients(roomID string) int {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	count := 0
	for _, client := range clients {
		if client.RoomID == roomID {
			count++
		}
	}
	return count
}

func publishEvent(evt eventEnvelope) {
	payload, err := json.Marshal(evt)
	if err != nil {
		metrics.DroppedMessagesTotal.Inc()
		return
	}
	if err := redisClient.Publish(context.Background(), eventsChannel, payload).Err(); err != nil {
		metrics.DroppedMessagesTotal.Inc()
	}
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
	pubsub := redisClient.Subscribe(ctx, eventsChannel)
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
		applyEventState(evt)
		broadcastEvent(evt)
	}
	return nil
}

func applyEventState(evt eventEnvelope) {
	switch evt.Type {
	case "presence":
		if evt.Status == "online" {
			addPresence(evt.RoomID, evt.UserID, true)
		} else if evt.Status == "offline" {
			addPresence(evt.RoomID, evt.UserID, false)
		}
	case "typing":
		recordTyping(evt.RoomID, evt.UserID)
	case "delivery":
		deliveryEventsTotal.Inc()
	case "message":
		metrics.MessagesTotal.Inc()
	}
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
		if evt.RoomID != "" && info.RoomID != evt.RoomID {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			conn.Close()
			delete(clients, conn)
			metrics.DroppedMessagesTotal.Inc()
		}
	}
	clientsMutex.Unlock()
	metrics.MessageLatency.Observe(float64(time.Since(start).Milliseconds()))
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	roomsMutex.Lock()
	defer roomsMutex.Unlock()

	result := map[string]roomStatus{}
	for roomID, state := range rooms {
		users := make([]string, 0, len(state.OnlineUsers))
		for user := range state.OnlineUsers {
			users = append(users, user)
		}
		sort.Strings(users)

		typing := make([]string, 0, len(state.TypingUsers))
		for user := range state.TypingUsers {
			typing = append(typing, user)
		}
		sort.Strings(typing)

		result[roomID] = roomStatus{Online: len(users), Users: users, Typing: typing}
	}

	json.NewEncoder(w).Encode(statusResponse{
		NodeID:            nodeID,
		ActiveConnections: int(atomic.LoadInt64(&activeConnections)),
		Rooms:             result,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
}

func connectRedis(redisAddr string) *redis.Client {
	for i := 0; i < 15; i++ {
		client, err := redisClientFromURL(redisAddr)
		if err == nil {
			err = client.Ping(context.Background()).Err()
		}
		if err == nil {
			return client
		}
		time.Sleep(2 * time.Second)
	}
	log.Fatal("redis not available")
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
