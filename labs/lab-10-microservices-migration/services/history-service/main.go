package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/antigravity/chat-lab/shared/backend/protocol"
	"github.com/antigravity/chat-lab/shared/backend/telemetry"
)

var (
	nodeID = envOrDefault("NODE_ID", "mesh-history-01")
	port   = envOrDefault("PORT", "8082")
	dbURL  = envOrDefault("DB_URL", "postgres://user:pass@db:5432/chat?sslmode=disable")
	dbConn *sql.DB

	historyQueriesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_history_queries_total",
		Help: "Total number of history queries handled",
	})
	historyQueryLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_history_query_latency_ms",
		Help:    "Latency of history queries in milliseconds",
		Buckets: prometheus.LinearBuckets(0, 10, 20),
	})
	lastTraceID string
)

type historyEntry struct {
	MessageID     string `json:"message_id"`
	UserID        string `json:"user_id"`
	RoomID        string `json:"room_id"`
	Content       string `json:"content"`
	TraceID       string `json:"trace_id"`
	SourceService string `json:"source_service"`
	CreatedAt     string `json:"created_at"`
}

type historyResponse struct {
	Room     string         `json:"room"`
	Count    int            `json:"count"`
	Messages []historyEntry `json:"messages"`
}

type statusResponse struct {
	NodeID      string `json:"node_id"`
	LastTraceID string `json:"last_trace_id"`
}

func main() {
	dbConn = connectDB(dbURL)

	http.HandleFunc("/history", handleHistory)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	fmt.Printf("Lab 10 history service %s listening on :%s\n", nodeID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	room := r.URL.Query().Get("room")
	if room == "" {
		room = "general"
	}
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	rows, err := dbConn.Query(
		"SELECT message_id, user_id, room_id, content, trace_id, source_service, created_at FROM messages WHERE room_id = $1 ORDER BY created_at DESC LIMIT $2",
		room,
		limit,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	entries := make([]historyEntry, 0, limit)
	for rows.Next() {
		var entry historyEntry
		if err := rows.Scan(&entry.MessageID, &entry.UserID, &entry.RoomID, &entry.Content, &entry.TraceID, &entry.SourceService, &entry.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		entries = append(entries, entry)
	}

	lastTraceID = r.Header.Get("X-Trace-Id")
	historyQueriesTotal.Inc()
	historyQueryLatency.Observe(float64(time.Since(start).Milliseconds()))
	json.NewEncoder(w).Encode(historyResponse{Room: room, Count: len(entries), Messages: entries})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(statusResponse{NodeID: nodeID, LastTraceID: lastTraceID})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
}

func connectDB(dbConn string) *sql.DB {
	var database *sql.DB
	var err error

	for attempt := 0; attempt < 15; attempt++ {
		database, err = sql.Open("postgres", dbConn)
		if err == nil {
			err = database.Ping()
		}
		if err == nil {
			return database
		}
		time.Sleep(2 * time.Second)
	}

	log.Fatal(err)
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
