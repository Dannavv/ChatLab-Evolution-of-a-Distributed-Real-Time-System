package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/antigravity/chat-lab/shared/backend/metrics"
	"github.com/antigravity/chat-lab/shared/backend/protocol"
	"github.com/antigravity/chat-lab/shared/backend/telemetry"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	clients      = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex

	nodeID       = envOrDefault("NODE_ID", "redis-01")
	port         = envOrDefault("PORT", "8080")
	redisChannel = envOrDefault("REDIS_CHANNEL", "chat_messages")
	db           *sql.DB
	redisClient  *redis.Client
)

func main() {
	db = connectDB(envOrDefault("DB_URL", envOrDefault("DATABASE_URL", "postgres://user:pass@db:5432/chat?sslmode=disable")))
	redisClient = connectRedis(envOrDefault("REDIS_URL", "redis://redis:6379"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/send", handleSendMessage)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	go subscribeToRedis(context.Background())

	fmt.Printf("Chat Server %s with Redis Pub/Sub starting on :%s\n", nodeID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
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
	metrics.ActiveConnections.Inc()
	clientsMutex.Unlock()
	publishSystem("Client connected to the Redis mesh.")

	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		metrics.ActiveConnections.Dec()
		clientsMutex.Unlock()
		publishSystem("Client disconnected from the Redis mesh.")
	}()

	for {
		_, msgData, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg protocol.Message
		if err := json.Unmarshal(msgData, &msg); err != nil {
			continue
		}

		msg.Timestamp = time.Now().UnixMilli()
		msg.NodeID = nodeID

		saveToDB(msg)
		publishMessage(msg)
	}
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg.Timestamp = time.Now().UnixMilli()
	msg.NodeID = nodeID

	saveToDB(msg)
	publishMessage(msg)
	w.WriteHeader(http.StatusAccepted)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
}

func connectDB(dbConn string) *sql.DB {
	var database *sql.DB
	var err error

	for i := 0; i < 10; i++ {
		database, err = sql.Open("postgres", dbConn)
		if err == nil {
			err = database.Ping()
		}
		if err == nil {
			return database
		}
		fmt.Printf("Attempt %d: DB not ready, waiting...\n", i+1)
		time.Sleep(2 * time.Second)
	}

	log.Fatal(err)
	return nil
}

func connectRedis(redisURL string) *redis.Client {
	var client *redis.Client
	var err error

	for i := 0; i < 10; i++ {
		client, err = redisClientFromURL(redisURL)
		if err == nil {
			err = client.Ping(context.Background()).Err()
		}
		if err == nil {
			return client
		}
		fmt.Printf("Attempt %d: Redis not ready, waiting...\n", i+1)
		time.Sleep(2 * time.Second)
	}

	log.Fatal(err)
	return nil
}

func redisClientFromURL(redisURL string) (*redis.Client, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(options), nil
}

func publishMessage(msg protocol.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("marshal error:", err)
		metrics.DroppedMessagesTotal.Inc()
		return
	}

	if err := redisClient.Publish(context.Background(), redisChannel, data).Err(); err != nil {
		log.Println("redis publish error:", err)
		metrics.DroppedMessagesTotal.Inc()
	}
}

func publishSystem(content string) {
	publishMessage(protocol.Message{
		UserID:    "SYSTEM",
		RoomID:    "manuscript-lab",
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
		NodeID:    nodeID,
	})
}

func subscribeToRedis(ctx context.Context) {
	for {
		if err := subscribeOnce(ctx); err != nil {
			log.Println("redis subscribe error:", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func subscribeOnce(ctx context.Context) error {
	pubsub := redisClient.Subscribe(ctx, redisChannel)
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	channel := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case redisMessage, ok := <-channel:
			if !ok {
				return nil
			}

			var msg protocol.Message
			if err := json.Unmarshal([]byte(redisMessage.Payload), &msg); err != nil {
				log.Println("redis payload error:", err)
				metrics.DroppedMessagesTotal.Inc()
				continue
			}

			broadcast(msg)
		}
	}
}

func saveToDB(msg protocol.Message) {
	_, err := db.Exec(
		"INSERT INTO messages (user_id, room_id, content, node_id) VALUES ($1, $2, $3, $4)",
		msg.UserID, msg.RoomID, msg.Content, msg.NodeID,
	)
	if err != nil {
		log.Println("DB error:", err)
		metrics.DroppedMessagesTotal.Inc()
	}
}

func broadcast(msg protocol.Message) {
	start := time.Now()

	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	msg.Connections = len(clients)
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("broadcast marshal error:", err)
		metrics.DroppedMessagesTotal.Inc()
		return
	}

	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			// Defer in handleWebSocket will handle cleanup
			client.Close()
			metrics.DroppedMessagesTotal.Inc()
		}
	}

	metrics.MessagesTotal.Inc()
	metrics.MessageLatency.Observe(float64(time.Since(start).Milliseconds()))
}
