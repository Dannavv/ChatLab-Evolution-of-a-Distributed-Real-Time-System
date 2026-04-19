package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/antigravity/chat-lab/shared/backend/metrics"
	"github.com/antigravity/chat-lab/shared/backend/protocol"
	"github.com/antigravity/chat-lab/shared/backend/telemetry"
)

type secureWireEnvelope struct {
	Type        string `json:"type"`
	EventID     string `json:"event_id"`
	MessageID   string `json:"message_id"`
	UserID      string `json:"user_id"`
	RoomID      string `json:"room_id"`
	NodeID      string `json:"node_id"`
	Timestamp   int64  `json:"timestamp"`
	KeyVersion  int64  `json:"key_version"`
	Nonce       string `json:"nonce,omitempty"`
	Ciphertext  string `json:"ciphertext,omitempty"`
	Signature   string `json:"signature,omitempty"`
	Status      string `json:"status,omitempty"`
	Connections int    `json:"connections,omitempty"`
}

type securePublicEnvelope struct {
	Type         string `json:"type"`
	EventID      string `json:"event_id"`
	MessageID    string `json:"message_id"`
	UserID       string `json:"user_id"`
	RoomID       string `json:"room_id"`
	Content      string `json:"content"`
	NodeID       string `json:"node_id"`
	Timestamp    int64  `json:"timestamp"`
	KeyVersion   int64  `json:"key_version"`
	SecurityStatus string `json:"security_status"`
	SignatureOK  bool   `json:"signature_ok"`
	Connections  int    `json:"connections"`
}

type secureRequest struct {
	UserID         string `json:"user_id"`
	RoomID         string `json:"room_id"`
	Content        string `json:"content"`
	TamperSignature bool   `json:"tamper_signature"`
}

type rotationEnvelope struct {
	Type      string `json:"type"`
	EventID   string `json:"event_id"`
	NodeID    string `json:"node_id"`
	Timestamp int64  `json:"timestamp"`
	Version   int64  `json:"version"`
}

type statusResponse struct {
	NodeID            string   `json:"node_id"`
	ActiveConnections  int      `json:"active_connections"`
	CurrentKeyVersion  int64    `json:"current_key_version"`
	KnownVersions      []int64  `json:"known_versions"`
	RateLimitPerUser   int      `json:"rate_limit_per_user"`
	RateWindowSeconds  int      `json:"rate_window_seconds"`
	AcceptedMessages   int64    `json:"accepted_messages"`
	RejectedMessages   int64    `json:"rejected_messages"`
	RateLimitedMessages int64   `json:"rate_limited_messages"`
	ReplayRejected     int64    `json:"replay_rejected"`
	SignatureRejected  int64    `json:"signature_rejected"`
	KeyRotations       int64    `json:"key_rotations"`
}

type rateBucket struct {
	Tokens     float64
	LastRefill time.Time
}

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	clients      = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex

	nodeID       = envOrDefault("NODE_ID", "secure-01")
	port         = envOrDefault("PORT", "8080")
	redisURL     = envOrDefault("REDIS_URL", "redis://redis:6379")
	eventsChannel = envOrDefault("EVENTS_CHANNEL", "chat:secure-events")
	rotationChannel = envOrDefault("ROTATION_CHANNEL", "chat:secure-rotations")
	masterSecret  = envOrDefault("MASTER_SECRET", "chat-lab-master-secret")
	redisClient   *redis.Client

	activeConnections int64
	currentKeyVersion int64 = 1
	keyRotationsTotal int64
	acceptedMessagesTotal int64
	rejectedMessagesTotal int64
	rateLimitedTotal int64
	replayRejectedTotal int64
	signatureRejectedTotal int64

	rateLimitMutex sync.Mutex
	rateLimits     = map[string]*rateBucket{}
	rateLimitCap   = envIntDefault("RATE_LIMIT_CAP", 5)
	rateWindow     = time.Duration(envIntDefault("RATE_WINDOW_MS", 10000)) * time.Millisecond

	seenMutex sync.Mutex
	seenIDs   = map[string]time.Time{}
	seenTTL   = 10 * time.Minute

	secureMessagesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_secure_messages_total",
		Help: "Total secure chat messages accepted and broadcast",
	})
	secureRejectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_secure_rejected_total",
		Help: "Total secure chat messages rejected",
	})
	secureSignatureRejectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_secure_signature_rejected_total",
		Help: "Total secure chat messages rejected for invalid signatures",
	})
	secureRateLimitedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_secure_rate_limited_total",
		Help: "Total secure chat messages rejected by rate limiting",
	})
	secureReplayRejectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_secure_replay_rejected_total",
		Help: "Total secure chat messages rejected as replayed events",
	})
	secureKeyRotationsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_secure_key_rotations_total",
		Help: "Total secure key rotations processed",
	})
)

func main() {
	redisClient = connectRedis(redisURL)
	go subscribeEvents(context.Background())
	go pruneSeenIDs()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/send", handleSend)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/rotate-key", handleRotateKey)
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	telemetry.StartMemoryTracking(2 * time.Second)
	fmt.Printf("Lab 09 secure node %s listening on :%s\n", nodeID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
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
	atomic.AddInt64(&activeConnections, 1)
	metrics.ActiveConnections.Inc()
	clientsMutex.Unlock()

	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		atomic.AddInt64(&activeConnections, -1)
		metrics.ActiveConnections.Dec()
		clientsMutex.Unlock()
	}()

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var req secureRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			continue
		}
		if err := processOutgoingRequest(r.Context(), req); err != nil {
			log.Println("send error:", err)
		}
	}
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	var req secureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := processOutgoingRequest(r.Context(), req); err != nil {
		if errors.Is(err, errRateLimited) {
			http.Error(w, err.Error(), http.StatusTooManyRequests)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

var errRateLimited = errors.New("rate limited")

func processOutgoingRequest(ctx context.Context, req secureRequest) error {
	if req.UserID == "" {
		req.UserID = "guest"
	}
	if req.RoomID == "" {
		req.RoomID = "general"
	}
	if req.Content == "" {
		return errors.New("content required")
	}
	if !allowUser(req.UserID) {
		atomic.AddInt64(&rateLimitedTotal, 1)
		secureRateLimitedTotal.Inc()
		secureRejectedTotal.Inc()
		metrics.DroppedMessagesTotal.Inc()
		return errRateLimited
	}

	keyVersion := atomic.LoadInt64(&currentKeyVersion)
	messageID := generateID("msg")
	timestamp := time.Now().UnixMilli()
	ciphertext, nonce, err := encryptContent(req.RoomID, keyVersion, req.Content)
	if err != nil {
		return err
	}

	envelope := secureWireEnvelope{
		Type:       "message",
		EventID:    generateID("evt"),
		MessageID:  messageID,
		UserID:     req.UserID,
		RoomID:     req.RoomID,
		NodeID:     nodeID,
		Timestamp:  timestamp,
		KeyVersion: keyVersion,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}
	envelope.Signature = signEnvelope(envelope)
	if req.TamperSignature {
		envelope.Signature = tamperSignature(envelope.Signature)
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	if err := redisClient.Publish(ctx, eventsChannel, payload).Err(); err != nil {
		metrics.DroppedMessagesTotal.Inc()
		return err
	}

	atomic.AddInt64(&acceptedMessagesTotal, 1)
	secureMessagesTotal.Inc()
	return nil
}

func handleRotateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	version := atomic.AddInt64(&currentKeyVersion, 1)
	atomic.AddInt64(&keyRotationsTotal, 1)
	secureKeyRotationsTotal.Inc()

	event := rotationEnvelope{
		Type:      "rotation",
		EventID:   generateID("rot"),
		NodeID:    nodeID,
		Timestamp: time.Now().UnixMilli(),
		Version:   version,
	}
	payload, err := json.Marshal(event)
	if err == nil {
		if err := redisClient.Publish(r.Context(), rotationChannel, payload).Err(); err != nil {
			metrics.DroppedMessagesTotal.Inc()
		}
	}
	broadcastSystem(fmt.Sprintf("Key rotated to version %d", version))
	json.NewEncoder(w).Encode(map[string]int64{"version": version})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	versions := []int64{atomic.LoadInt64(&currentKeyVersion)}
	json.NewEncoder(w).Encode(statusResponse{
		NodeID:             nodeID,
		ActiveConnections:  int(atomic.LoadInt64(&activeConnections)),
		CurrentKeyVersion:  atomic.LoadInt64(&currentKeyVersion),
		KnownVersions:      versions,
		RateLimitPerUser:   rateLimitCap,
		RateWindowSeconds:  int(rateWindow / time.Second),
		AcceptedMessages:   atomic.LoadInt64(&acceptedMessagesTotal),
		RejectedMessages:   atomic.LoadInt64(&rejectedMessagesTotal),
		RateLimitedMessages: atomic.LoadInt64(&rateLimitedTotal),
		ReplayRejected:     atomic.LoadInt64(&replayRejectedTotal),
		SignatureRejected:  atomic.LoadInt64(&signatureRejectedTotal),
		KeyRotations:       atomic.LoadInt64(&keyRotationsTotal),
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
}

func subscribeEvents(ctx context.Context) {
	for {
		if err := subscribeOnce(ctx); err != nil {
			log.Println("subscribe error:", err)
			time.Sleep(2 * time.Second)
		}
	}
}

func subscribeOnce(ctx context.Context) error {
	pubsub := redisClient.Subscribe(ctx, eventsChannel, rotationChannel)
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	for msg := range pubsub.Channel() {
		if msg.Channel == rotationChannel {
			var rotation rotationEnvelope
			if err := json.Unmarshal([]byte(msg.Payload), &rotation); err != nil {
				metrics.DroppedMessagesTotal.Inc()
				continue
			}
			applyRotation(rotation)
			continue
		}

		var envelope secureWireEnvelope
		if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
			metrics.DroppedMessagesTotal.Inc()
			continue
		}
		processInboundEnvelope(envelope)
	}
	return nil
}

func processInboundEnvelope(envelope secureWireEnvelope) {
	if envelope.Type != "message" {
		return
	}

	if isReplay(envelope.MessageID) {
		atomic.AddInt64(&replayRejectedTotal, 1)
		secureReplayRejectedTotal.Inc()
		secureRejectedTotal.Inc()
		return
	}

	if !verifyEnvelope(envelope) {
		atomic.AddInt64(&signatureRejectedTotal, 1)
		secureSignatureRejectedTotal.Inc()
		secureRejectedTotal.Inc()
		broadcastSystem(fmt.Sprintf("Rejected message %s: invalid signature", envelope.MessageID))
		return
	}

	plaintext, err := decryptContent(envelope.RoomID, envelope.KeyVersion, envelope.Nonce, envelope.Ciphertext)
	if err != nil {
		atomic.AddInt64(&signatureRejectedTotal, 1)
		secureSignatureRejectedTotal.Inc()
		secureRejectedTotal.Inc()
		broadcastSystem(fmt.Sprintf("Rejected message %s: decrypt failed", envelope.MessageID))
		return
	}

	markSeen(envelope.MessageID)
	broadcastPublic(securePublicEnvelope{
		Type:           "message",
		EventID:        envelope.EventID,
		MessageID:      envelope.MessageID,
		UserID:         envelope.UserID,
		RoomID:         envelope.RoomID,
		Content:        plaintext,
		NodeID:         envelope.NodeID,
		Timestamp:      envelope.Timestamp,
		KeyVersion:     envelope.KeyVersion,
		SecurityStatus: "verified",
		SignatureOK:    true,
	})
}

func applyRotation(rotation rotationEnvelope) {
	if rotation.Version <= atomic.LoadInt64(&currentKeyVersion) {
		return
	}
	atomic.StoreInt64(&currentKeyVersion, rotation.Version)
	atomic.AddInt64(&keyRotationsTotal, 1)
	secureKeyRotationsTotal.Inc()
	broadcastSystem(fmt.Sprintf("Key rotated to version %d", rotation.Version))
}

func broadcastPublic(evt securePublicEnvelope) {
	start := time.Now()
	clientsMutex.Lock()
	connections := len(clients)
	evt.Connections = connections
	payload, err := json.Marshal(evt)
	if err != nil {
		clientsMutex.Unlock()
		metrics.DroppedMessagesTotal.Inc()
		return
	}
	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			conn.Close()
			delete(clients, conn)
			metrics.DroppedMessagesTotal.Inc()
		}
	}
	clientsMutex.Unlock()
	metrics.MessagesTotal.Inc()
	metrics.MessageLatency.Observe(float64(time.Since(start).Milliseconds()))
}

func broadcastSystem(content string) {
	broadcastPublic(securePublicEnvelope{
		Type:           "system",
		EventID:        generateID("sys"),
		MessageID:      generateID("msg"),
		UserID:         "SYSTEM",
		RoomID:         "security",
		Content:        content,
		NodeID:         nodeID,
		Timestamp:      time.Now().UnixMilli(),
		KeyVersion:     atomic.LoadInt64(&currentKeyVersion),
		SecurityStatus: "system",
		SignatureOK:    true,
	})
}

func allowUser(userID string) bool {
	rateLimitMutex.Lock()
	defer rateLimitMutex.Unlock()

	bucket, ok := rateLimits[userID]
	if !ok {
		bucket = &rateBucket{Tokens: float64(rateLimitCap), LastRefill: time.Now()}
		rateLimits[userID] = bucket
	}

	now := time.Now()
	elapsed := now.Sub(bucket.LastRefill)
	if elapsed > 0 {
		refill := (elapsed.Seconds() / rateWindow.Seconds()) * float64(rateLimitCap)
		bucket.Tokens = minFloat(float64(rateLimitCap), bucket.Tokens+refill)
		bucket.LastRefill = now
	}

	if bucket.Tokens < 1 {
		return false
	}
	bucket.Tokens -= 1
	return true
}

func pruneSeenIDs() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		seenMutex.Lock()
		for id, ts := range seenIDs {
			if now.Sub(ts) > seenTTL {
				delete(seenIDs, id)
			}
		}
		seenMutex.Unlock()
	}
}

func isReplay(messageID string) bool {
	seenMutex.Lock()
	defer seenMutex.Unlock()
	_, ok := seenIDs[messageID]
	return ok
}

func markSeen(messageID string) {
	seenMutex.Lock()
	seenIDs[messageID] = time.Now()
	seenMutex.Unlock()
}

func verifyEnvelope(envelope secureWireEnvelope) bool {
	expected := signEnvelope(envelope)
	return hmac.Equal([]byte(expected), []byte(envelope.Signature))
}

func signEnvelope(envelope secureWireEnvelope) string {
	mac := hmac.New(sha256.New, signingKey(envelope.RoomID, envelope.KeyVersion))
	writeMac(mac, envelope.EventID)
	writeMac(mac, envelope.MessageID)
	writeMac(mac, envelope.UserID)
	writeMac(mac, envelope.RoomID)
	writeMac(mac, envelope.NodeID)
	writeMac(mac, strconv.FormatInt(envelope.Timestamp, 10))
	writeMac(mac, envelope.Nonce)
	writeMac(mac, envelope.Ciphertext)
	writeMac(mac, strconv.FormatInt(envelope.KeyVersion, 10))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func writeMac(mac hashWriter, value string) {
	mac.Write([]byte(value))
	mac.Write([]byte("|"))
}

type hashWriter interface {
	Write([]byte) (int, error)
}

func tamperSignature(signature string) string {
	if len(signature) == 0 {
		return signature
	}
	return "X" + signature[1:]
}

func encryptContent(roomID string, version int64, plaintext string) (ciphertext string, nonce string, err error) {
	block, err := aes.NewCipher(encryptionKey(roomID, version))
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonceBytes := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonceBytes); err != nil {
		return "", "", err
	}
	cipherBytes := gcm.Seal(nil, nonceBytes, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(cipherBytes), base64.StdEncoding.EncodeToString(nonceBytes), nil
}

func decryptContent(roomID string, version int64, nonce string, ciphertext string) (string, error) {
	block, err := aes.NewCipher(encryptionKey(roomID, version))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", err
	}
	cipherBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	plainBytes, err := gcm.Open(nil, nonceBytes, cipherBytes, nil)
	if err != nil {
		return "", err
	}
	return string(plainBytes), nil
}

func encryptionKey(roomID string, version int64) []byte {
	sum := sha256.Sum256([]byte(strings.Join([]string{masterSecret, "enc", roomID, strconv.FormatInt(version, 10)}, ":")))
	return sum[:]
}

func signingKey(roomID string, version int64) []byte {
	sum := sha256.Sum256([]byte(strings.Join([]string{masterSecret, "sig", roomID, strconv.FormatInt(version, 10)}, ":")))
	return sum[:]
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), randInt())
}

func randInt() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UnixNano()
	}
	return int64(b[0])<<56 | int64(b[1])<<48 | int64(b[2])<<40 | int64(b[3])<<32 | int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7])
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
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

func handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(protocol.HealthResponse{Status: "healthy", NodeID: nodeID})
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envIntDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
