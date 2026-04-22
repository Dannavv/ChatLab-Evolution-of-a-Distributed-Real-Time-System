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
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
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
	nodeID        = envOrDefault("NODE_ID", "mesh-message-01")
	port          = envOrDefault("PORT", "8081")
	dbURL         = envOrDefault("DB_URL", "postgres://user:pass@db:5432/chat?sslmode=disable")
	redisURL      = envOrDefault("REDIS_URL", "redis://redis:6379")
	eventsChannel = envOrDefault("EVENTS_CHANNEL", "chat:events")

	dbConn      *sql.DB
	redisClient *redis.Client

	messageServiceRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_message_service_requests_total",
		Help: "Total number of requests handled by the message service",
	})
	messageServiceWriteLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_message_service_write_latency_ms",
		Help:    "Latency of message persistence in the message service",
		Buckets: prometheus.LinearBuckets(0, 10, 20),
	})
	messageServiceErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_message_service_errors_total",
		Help: "Total number of message service errors",
	})
)

type statusResponse struct {
	NodeID        string `json:"node_id"`
	EventsChannel string `json:"events_channel"`
	LastTraceID   string `json:"last_trace_id"`
}

type incomingEnvelope struct {
	Message   protocol.Message `json:"message"`
	TraceID   string           `json:"trace_id"`
	SourceSvc string           `json:"source_service"`
}

var lastTraceID string

func main() {
	if collectorURL := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); collectorURL != "" {
		shutdown := telemetry.InitTracer("message-service", collectorURL)
		defer shutdown()
	}

	redisClient = connectRedis(redisURL)
	dbConn = connectDB(dbURL)

	http.HandleFunc("/messages", handleMessages)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	
	server := &http.Server{Addr: ":" + port}
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Printf("\nShutting down message service %s...\n", nodeID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	fmt.Printf("Lab 11/10 Hardened Message Service %s listening on :%s\n", nodeID, port)
	log.Fatal(server.ListenAndServe())
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	tr := otel.Tracer("message-service")
	ctx, span := tr.Start(ctx, "handleMessages")
	defer span.End()

	var msg protocol.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		messageServiceErrorsTotal.Inc()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}
	if msg.SourceService == "" {
		msg.SourceService = r.Header.Get("X-Parent-Service")
		if msg.SourceService == "" {
			msg.SourceService = nodeID
		}
	}
	traceID := r.Header.Get("X-Trace-Id")
	if traceID == "" {
		traceID = msg.TraceID
	}
	msg.TraceID = traceID
	lastTraceID = traceID

	if err := saveMessage(r.Context(), msg); err != nil {
		messageServiceErrorsTotal.Inc()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := publishMessage(r.Context(), msg); err != nil {
		messageServiceErrorsTotal.Inc()
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	messageServiceRequestsTotal.Inc()
	messageServiceWriteLatency.Observe(float64(time.Since(start).Milliseconds()))
	w.WriteHeader(http.StatusAccepted)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(statusResponse{
		NodeID:        nodeID,
		EventsChannel: eventsChannel,
		LastTraceID:   lastTraceID,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
}

func saveMessage(ctx context.Context, msg protocol.Message) error {
	tr := otel.Tracer("message-service")
	_, span := tr.Start(ctx, "saveMessage")
	defer span.End()

	messageID := msg.MessageID
	if messageID == "" {
		messageID = fmt.Sprintf("%s-%d", msg.TraceID, msg.Timestamp)
	}
	
	return withRetry(3, 100*time.Millisecond, func() error {
		_, err := dbConn.Exec(
			"INSERT INTO messages (message_id, user_id, room_id, content, trace_id, source_service, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW()) ON CONFLICT (message_id) DO NOTHING",
			messageID,
			msg.UserID,
			msg.RoomID,
			msg.Content,
			msg.TraceID,
			msg.SourceService,
		)
		return err
	})
}

func publishMessage(ctx context.Context, msg protocol.Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	
	return withRetry(3, 50*time.Millisecond, func() error {
		return redisClient.Publish(ctx, eventsChannel, payload).Err()
	})
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

func connectRedis(redisAddr string) *redis.Client {
	var client *redis.Client
	var err error

	for attempt := 0; attempt < 15; attempt++ {
		client, err = redisClientFromURL(redisAddr)
		if err == nil {
			err = client.Ping(context.Background()).Err()
		}
		if err == nil {
			return client
		}
		time.Sleep(2 * time.Second)
	}

	log.Fatal(err)
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
