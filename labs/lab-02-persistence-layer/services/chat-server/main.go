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
	dbConn := os.Getenv("DB_URL")
	if dbConn == "" {
		dbConn = "postgres://user:pass@db:5432/chat?sslmode=disable"
	}

	// Robust DB Connection with Backoff
	var err error
	for i := 0; i < 15; i++ {
		db, err = sql.Open("postgres", dbConn)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			break
		}
		fmt.Printf("Attempt %d: DB not ready (%v), retrying...\n", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Fatal: Could not connect to database after 15 attempts: %v", err)
	}

	// Schema Initialization (Ensuring proper indexing for Read Path)
	initSchema()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/send", handleSendMessage)
	http.HandleFunc("/history", handleGetHistory) // Evaluates Read Path
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)

	fmt.Printf("Chat Server %s (Durable) starting on :8080\n", nodeID)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initSchema() {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			user_id TEXT,
			room_id TEXT,
			content TEXT,
			node_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_room_id ON messages(room_id);
	`)
	if err != nil {
		log.Printf("Warning: Schema init failed: %v", err)
	}
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

	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		metrics.ActiveConnections.Dec()
		clientsMutex.Unlock()
	}()

	for {
		_, msgData, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg protocol.Message
		json.Unmarshal(msgData, &msg)
		processMessage(msg)
	}
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	processMessage(msg)
	w.WriteHeader(http.StatusAccepted)
}

func handleGetHistory(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		roomID = "manuscript-lab"
	}

	start := time.Now()
	rows, err := db.Query("SELECT user_id, room_id, content, created_at FROM messages WHERE room_id = $1 ORDER BY created_at DESC LIMIT 50", roomID)
	if err != nil {
		metrics.DBErrorsTotal.Inc()
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var msgs []protocol.Message
	for rows.Next() {
		var m protocol.Message
		var createdAt time.Time
		rows.Scan(&m.UserID, &m.RoomID, &m.Content, &createdAt)
		m.Timestamp = createdAt.UnixMilli()
		msgs = append(msgs, m)
	}

	duration := float64(time.Since(start).Milliseconds())
	metrics.DBQueryDuration.Observe(duration) // Track Read Latency

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

func processMessage(msg protocol.Message) {
	pStart := time.Now()
	enrichMessage(&msg)
	metrics.ProcessingLatency.Observe(float64(time.Since(pStart).Seconds() * 1000))

	// Persistence with Retry Logic (Write-Through Durability)
	go func(m protocol.Message) {
		success := false
		for i := 0; i < 3; i++ { // 3 Retries
			start := time.Now()
			_, err := db.Exec(
				"INSERT INTO messages (user_id, room_id, content, node_id) VALUES ($1, $2, $3, $4)",
				m.UserID, m.RoomID, m.Content, m.NodeID,
			)
			writeMs := float64(time.Since(start).Milliseconds())
			metrics.DBQueryDuration.Observe(writeMs)

			if err == nil {
				success = true
				break
			}
			metrics.DBErrorsTotal.Inc()
			time.Sleep(100 * time.Millisecond)
		}
		if !success {
			log.Printf("ERROR: Critical Data Loss - Message persistence failed after 3 retries: %+v", m)
		}
	}(msg)

	// Broadcast (Parallelized from persistence to minimize client-facing latency)
	broadcast(msg)
}

func enrichMessage(msg *protocol.Message) {
	now := time.Now().UnixMilli()
	if msg.MessageID == "" {
		if msg.TraceID != "" {
			msg.MessageID = msg.TraceID
		} else {
			msg.MessageID = fmt.Sprintf("%s-%d", nodeID, now)
		}
	}
	msg.ServerReceiveTimestamp = now
	msg.Timestamp = now
	msg.NodeID = nodeID
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "connected"
	if err := db.Ping(); err != nil {
		dbStatus = "disconnected"
	}
	resp := map[string]string{
		"status":   "healthy",
		"node_id":  nodeID,
		"database": dbStatus,
	}
	json.NewEncoder(w).Encode(resp)
}

func broadcast(msg protocol.Message) {
	start := time.Now()
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	msg.Connections = len(clients)
	msg.ServerBroadcastTimestamp = time.Now().UnixMilli()
	msg.Timestamp = msg.ServerBroadcastTimestamp
	data, _ := json.Marshal(msg)

	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			client.Close()
		}
	}

	metrics.MessagesTotal.Inc()
	metrics.MessageLatency.Observe(float64(time.Since(start).Milliseconds()))
}

func broadcastSystem(content string) {
	msg := protocol.Message{
		UserID:  "SYSTEM",
		RoomID:  "manuscript-lab",
		Content: content,
	}
	processMessage(msg)
}
