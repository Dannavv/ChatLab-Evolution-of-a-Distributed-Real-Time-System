package telemetry

import (
	"runtime"
	"time"

	"github.com/antigravity/chat-lab/shared/backend/metrics"
)

// StartMemoryTracking begins a background goroutine to update memory metrics
func StartMemoryTracking(interval time.Duration) {
	go func() {
		var m runtime.MemStats
		for {
			runtime.ReadMemStats(&m)
			metrics.MemoryBytes.Set(float64(m.Alloc))
			time.Sleep(interval)
		}
	}()
}
