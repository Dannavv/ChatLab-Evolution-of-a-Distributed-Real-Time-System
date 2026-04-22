package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/antigravity/chat-lab/shared/backend/protocol"
	"github.com/antigravity/chat-lab/shared/backend/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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
	if collectorURL := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); collectorURL != "" {
		shutdown := telemetry.InitTracer("history-service", collectorURL)
		defer shutdown()
	}

	dbConn = connectDB(dbURL)

	http.HandleFunc("/history", handleHistory)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	
	server := &http.Server{Addr: ":" + port}
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Printf("\nShutting down history service %s...\n", nodeID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	fmt.Printf("Lab 11/10 Hardened History Service %s listening on :%s\n", nodeID, port)
	log.Fatal(server.ListenAndServe())
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	tr := otel.Tracer("history-service")
	_, span := tr.Start(ctx, "handleHistory")
	defer span.End()
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

	var rows *sql.Rows
	err := withRetry(3, 50*time.Millisecond, func() error {
		var qErr error
		rows, qErr = dbConn.Query(
			"SELECT message_id, user_id, room_id, content, trace_id, source_service, created_at FROM messages WHERE room_id = $1 ORDER BY created_at DESC LIMIT $2",
			room,
			limit,
		)
		return qErr
	})
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

func withRetry(attempts int, baseBackoff time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			// Jittered exponential backoff
			jitter := time.Duration(rand.Int63n(int64(baseBackoff / 2)))
			time.Sleep(baseBackoff + jitter)
			baseBackoff *= 2
		}
	}
	return err
}
