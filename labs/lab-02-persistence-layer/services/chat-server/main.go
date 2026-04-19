package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

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
	nodeID       = "monolith-db-01"
	db           *sql.DB
)

func main() {
	// DB initialization
	dbConn := os.Getenv("DB_URL")
	if dbConn == "" {
		dbConn = os.Getenv("DATABASE_URL")
	}
	if dbConn == "" {
		dbConn = "postgres://user:pass@db:5432/chat?sslmode=disable"
	}

	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dbConn)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			break
		}
		fmt.Printf("Attempt %d: DB not ready, waiting...\n", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/send", handleSendMessage)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	// Start shared memory tracking
	telemetry.StartMemoryTracking(2 * time.Second)

	fmt.Printf("Chat Server %s with DB starting on :8080\n", nodeID)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	clientsMutex.Lock()
	clients[conn] = true
	metrics.ActiveConnections.Inc()
	clientsMutex.Unlock()
	broadcastSystem("Client connected to persistence node.")

	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		metrics.ActiveConnections.Dec()
		clientsMutex.Unlock()
		broadcastSystem("Client disconnected from persistence node.")
	}()

	for {
		_, msgData, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg protocol.Message
		json.Unmarshal(msgData, &msg)
		msg.Timestamp = time.Now().UnixMilli()
		msg.NodeID = nodeID

		saveToDB(msg)
		broadcast(msg)
	}
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var msg protocol.Message
	json.NewDecoder(r.Body).Decode(&msg)
	msg.Timestamp = time.Now().UnixMilli()
	msg.NodeID = nodeID

	saveToDB(msg)
	broadcast(msg)
	w.WriteHeader(http.StatusAccepted)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := protocol.HealthResponse{Status: "healthy", NodeID: nodeID}
	json.NewEncoder(w).Encode(resp)
}

func saveToDB(msg protocol.Message) {
	_, err := db.Exec(
		"INSERT INTO messages (user_id, room_id, content, node_id) VALUES ($1, $2, $3, $4)",
		msg.UserID, msg.RoomID, msg.Content, msg.NodeID,
	)
	if err != nil {
		log.Println("DB error:", err)
	}
}

func broadcast(msg protocol.Message) {
	start := time.Now()

	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	msg.Connections = len(clients)
	data, _ := json.Marshal(msg)

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			// Defer in handleWebSocket will handle cleanup
			client.Close()
		}
	}

	metrics.MessagesTotal.Inc()
	metrics.MessageLatency.Observe(float64(time.Since(start).Milliseconds()))
}

func broadcastSystem(content string) {
	msg := protocol.Message{
		UserID:    "SYSTEM",
		RoomID:    "manuscript-lab",
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
		NodeID:    nodeID,
	}
	broadcast(msg)
}
