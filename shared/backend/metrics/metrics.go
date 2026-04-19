package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_active_connections",
		Help: "Number of active WebSocket connections",
	})

	MessagesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_messages_total",
		Help: "Total number of chat messages processed",
	})

	MessageLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_message_latency_ms",
		Help:    "Latency of chat messages in milliseconds",
		Buckets: prometheus.LinearBuckets(0, 5, 20), // 0 to 100ms
	})

	MemoryBytes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chat_memory_bytes",
		Help: "Current memory usage in bytes",
	})

	DroppedMessagesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_dropped_messages_total",
		Help: "Total number of messages dropped due to backpressure or errors",
	})

	ReconnectsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chat_reconnects_total",
		Help: "Total number of WebSocket reconnections",
	})
)
