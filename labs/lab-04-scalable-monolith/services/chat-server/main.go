package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

	messageQueue = make(chan protocol.Message, envIntDefault("QUEUE_SIZE", 256))
	nodeID       = envOrDefault("NODE_ID", "monolith-async-01")
	port         = envOrDefault("PORT", "8080")
	workerCount  = envIntDefault("WORKERS", 4)
	procDelay    = time.Duration(envIntDefault("PROCESSING_DELAY_MS", 60)) * time.Millisecond

	activeClientCount int64
	busyWorkerCount   int64

	queueDepthGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_queue_depth",
		Help: "Number of chat messages waiting in the async queue",
	})

	workersBusyGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_workers_busy",
		Help: "Number of worker goroutines actively processing messages",
	})

	messagesQueuedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_messages_queued_total",
		Help: "Total number of messages accepted into the async queue",
	})

	messagesProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_messages_processed_total",
		Help: "Total number of messages processed by worker goroutines",
	})

	queueWaitLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_queue_wait_ms",
		Help:    "Time a message spends waiting in the async queue",
		Buckets: prometheus.LinearBuckets(0, 10, 20),
	})
)

type statusResponse struct {
	NodeID          string `json:"node_id"`
	ActiveConnections int   `json:"active_connections"`
	QueueDepth      int    `json:"queue_depth"`
	QueueCapacity   int    `json:"queue_capacity"`
	BusyWorkers     int    `json:"busy_workers"`
	Workers        int    `json:"workers"`
	ProcessingDelay int    `json:"processing_delay_ms"`
}

func main() {
	for i := 0; i < workerCount; i++ {
		go workerLoop(i + 1)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/send", handleSendMessage)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/status", handleStatus)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)

	fmt.Printf("Chat Server %s with async workers starting on :%s\n", nodeID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envIntDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
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
	atomic.AddInt64(&activeClientCount, 1)
	metrics.ActiveConnections.Inc()
	clientsMutex.Unlock()

	for {
		_, msgData, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg protocol.Message
		if err := json.Unmarshal(msgData, &msg); err != nil {
			continue
		}

		prepareMessage(&msg)
		enqueueMessage(msg)
	}

	clientsMutex.Lock()
	delete(clients, conn)
	atomic.AddInt64(&activeClientCount, -1)
	metrics.ActiveConnections.Dec()
	clientsMutex.Unlock()
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	prepareMessage(&msg)
	if !enqueueMessage(msg) {
		http.Error(w, "queue is full", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(statusResponse{
		NodeID:            nodeID,
		ActiveConnections: int(atomic.LoadInt64(&activeClientCount)),
		QueueDepth:        len(messageQueue),
		QueueCapacity:     cap(messageQueue),
		BusyWorkers:       int(atomic.LoadInt64(&busyWorkerCount)),
		Workers:           workerCount,
		ProcessingDelay:   int(procDelay / time.Millisecond),
	})
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

func enqueueMessage(msg protocol.Message) bool {
	select {
	case messageQueue <- msg:
		messagesQueuedTotal.Inc()
		queueDepthGauge.Set(float64(len(messageQueue)))
		return true
	default:
		metrics.DroppedMessagesTotal.Inc()
		return false
	}
}

func workerLoop(id int) {
	for msg := range messageQueue {
		queueDepthGauge.Set(float64(len(messageQueue)))
		atomic.AddInt64(&busyWorkerCount, 1)
		workersBusyGauge.Set(float64(atomic.LoadInt64(&busyWorkerCount)))

		if procDelay > 0 {
			time.Sleep(procDelay)
		}

		broadcast(msg)
		messagesProcessedTotal.Inc()
		queueWaitLatency.Observe(float64(time.Since(time.UnixMilli(msg.Timestamp)).Milliseconds()))

		atomic.AddInt64(&busyWorkerCount, -1)
		workersBusyGauge.Set(float64(atomic.LoadInt64(&busyWorkerCount)))
	}
}

func broadcast(msg protocol.Message) {
	start := time.Now()

	clientsMutex.Lock()
	msg.Connections = len(clients)
	data, err := json.Marshal(msg)
	if err != nil {
		clientsMutex.Unlock()
		log.Println("broadcast marshal error:", err)
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