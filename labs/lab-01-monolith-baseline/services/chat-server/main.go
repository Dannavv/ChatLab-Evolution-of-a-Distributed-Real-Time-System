package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
	nodeID       = "monolith-01"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/send", handleSendMessage)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	// Start standard memory tracking
	telemetry.StartMemoryTracking(2 * time.Second)

	fmt.Printf("Chat Server %s starting on :8080\n", nodeID)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	clientsMutex.Lock()
	clients[conn] = true
	metrics.ActiveConnections.Inc()
	clientsMutex.Unlock()
	broadcastSystem("New researcher joined the manuscript.")

	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		metrics.ActiveConnections.Dec()
		clientsMutex.Unlock()
		broadcastSystem("A researcher has left the manuscript.")
	}()

	for {
		_, msgData, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg protocol.Message
		if err := json.Unmarshal(msgData, &msg); err != nil {
			continue
		}

		enrichMessage(&msg)
		broadcast(msg)
	}
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	enrichMessage(&msg)
	broadcast(msg)
	w.WriteHeader(http.StatusAccepted)
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
	resp := protocol.HealthResponse{
		Status: "healthy",
		NodeID: nodeID,
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
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			// We don't delete here to avoid double-decrement with handleWebSocket defer
			client.Close()
		}
	}

	metrics.MessagesTotal.Inc()
	metrics.MessageLatency.Observe(float64(time.Since(start).Milliseconds()))
}

func broadcastSystem(content string) {
	msg := protocol.Message{
		UserID:    "SYSTEM",
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
		NodeID:    nodeID,
	}
	enrichMessage(&msg)
	broadcast(msg)
}
