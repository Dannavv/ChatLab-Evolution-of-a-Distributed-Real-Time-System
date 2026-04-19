package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type eventEnvelope struct {
	EventID      string `json:"event_id"`
	Type         string `json:"type"`
	UserID       string `json:"user_id"`
	RoomID       string `json:"room_id"`
	Content      string `json:"content,omitempty"`
	NodeID       string `json:"node_id"`
	OriginRegion string `json:"origin_region"`
	SourceRegion string `json:"source_region"`
	Timestamp    int64  `json:"timestamp"`
}

type bridgeStatus struct {
	NodeID          string   `json:"node_id"`
	ForwardedEvents int64    `json:"forwarded_events"`
	DroppedEvents   int64    `json:"dropped_events"`
	SyncRegions     []string `json:"sync_regions"`
}

var (
	port      = envOrDefault("PORT", "8082")
	channel   = envOrDefault("EVENTS_CHANNEL", "chat:global-events")
	redisUS   = envOrDefault("REDIS_US_URL", "redis://redis-us:6379")
	redisEU   = envOrDefault("REDIS_EU_URL", "redis://redis-eu:6379")
	clientUS  *redis.Client
	clientEU  *redis.Client

	seenEvents = make(map[string]int64)
	seenMu     sync.Mutex

	forwardedEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_global_forwarded_events_total",
		Help: "Events forwarded between regions",
	})
	droppedForwardTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_global_forward_dropped_total",
		Help: "Events dropped during forwarding",
	})

	forwardedCount int64
	droppedCount   int64
)

func main() {
	clientUS = connectRedis(redisUS)
	clientEU = connectRedis(redisEU)

	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/status", handleStatus)
	http.Handle("/metrics", promhttp.Handler())

	go bridgeRegion(context.Background(), "us", clientUS, clientEU)
	go bridgeRegion(context.Background(), "eu", clientEU, clientUS)
	go pruneSeenEvents()

	fmt.Printf("Lab 08 bridge listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "node_id": "global-bridge"})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(bridgeStatus{
		NodeID:          "global-bridge",
		ForwardedEvents: atomic.LoadInt64(&forwardedCount),
		DroppedEvents:   atomic.LoadInt64(&droppedCount),
		SyncRegions:     []string{"us", "eu"},
	})
}

func bridgeRegion(ctx context.Context, source string, sourceClient *redis.Client, targetClient *redis.Client) {
	for {
		if err := bridgeOnce(ctx, source, sourceClient, targetClient); err != nil {
			log.Println("bridge error:", source, err)
			time.Sleep(2 * time.Second)
		}
	}
}

func bridgeOnce(ctx context.Context, source string, sourceClient *redis.Client, targetClient *redis.Client) error {
	pubsub := sourceClient.Subscribe(ctx, channel)
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	for msg := range pubsub.Channel() {
		var evt eventEnvelope
		if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
			incDropped()
			continue
		}

		if evt.EventID == "" {
			incDropped()
			continue
		}
		if isDuplicate(evt.EventID) {
			continue
		}
		markSeen(evt.EventID)

		evt.SourceRegion = source
		payload, err := json.Marshal(evt)
		if err != nil {
			incDropped()
			continue
		}

		if err := targetClient.Publish(ctx, channel, payload).Err(); err != nil {
			incDropped()
			continue
		}
		incForwarded()
	}

	return nil
}

func incForwarded() {
	forwardedEventsTotal.Inc()
	atomic.AddInt64(&forwardedCount, 1)
}

func incDropped() {
	droppedForwardTotal.Inc()
	atomic.AddInt64(&droppedCount, 1)
}

func isDuplicate(eventID string) bool {
	seenMu.Lock()
	_, ok := seenEvents[eventID]
	seenMu.Unlock()
	return ok
}

func markSeen(eventID string) {
	seenMu.Lock()
	seenEvents[eventID] = time.Now().UnixMilli()
	seenMu.Unlock()
}

func pruneSeenEvents() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-2 * time.Minute).UnixMilli()
		seenMu.Lock()
		for id, ts := range seenEvents {
			if ts < cutoff {
				delete(seenEvents, id)
			}
		}
		seenMu.Unlock()
	}
}

func connectRedis(redisAddr string) *redis.Client {
	for i := 0; i < 15; i++ {
		opts, err := redis.ParseURL(redisAddr)
		if err == nil {
			client := redis.NewClient(opts)
			if err := client.Ping(context.Background()).Err(); err == nil {
				return client
			}
		}
		time.Sleep(2 * time.Second)
	}
	log.Fatal("redis not available")
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
