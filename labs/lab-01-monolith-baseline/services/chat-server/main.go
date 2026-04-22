package main

	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	mux.HandleFunc("/ws", handleWebSocket)
	mux.HandleFunc("/send", handleSendMessage)
	mux.HandleFunc("/health", handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	// Start standard memory tracking
	telemetry.StartMemoryTracking(2 * time.Second)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		fmt.Printf("Chat Server %s starting on :8080\n", nodeID)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
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
