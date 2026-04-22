package telemetry

import (
	"context"
	"log"
	"runtime"
	"time"

	"github.com/antigravity/chat-lab/shared/backend/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
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

// InitTracer initializes an OTLP exporter and a global trace provider
func InitTracer(serviceName, collectorURL string) func() {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
	}

	// Default collectorURL usually something like jaeger:4318 for OTLP/HTTP
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(collectorURL),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("failed to stop tracer provider: %v", err)
		}
	}
}
