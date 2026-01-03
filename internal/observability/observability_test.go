package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig(
		WithServiceName("test-service"),
		WithDetailedDBTracing(),
		WithQueryOptionTracing(),
	)

	if cfg.ServiceName != "test-service" {
		t.Errorf("expected service name 'test-service', got '%s'", cfg.ServiceName)
	}
	if !cfg.EnableDetailedDBTracing {
		t.Error("expected detailed DB tracing to be enabled")
	}
	if !cfg.EnableQueryOptionTracing {
		t.Error("expected query option tracing to be enabled")
	}
}

func TestConfigInitialize(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	mp := noop.NewMeterProvider()

	cfg := NewConfig(
		WithTracerProvider(tp),
		WithMeterProvider(mp),
		WithServiceName("test-service"),
	)

	err := cfg.Initialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Tracer() == nil {
		t.Error("expected tracer to be initialized")
	}
	if cfg.Metrics() == nil {
		t.Error("expected metrics to be initialized")
	}
}

func TestConfigInitializeNoProviders(t *testing.T) {
	cfg := NewConfig(WithServiceName("test-service"))

	err := cfg.Initialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should get noop implementations
	if cfg.Tracer() == nil {
		t.Error("expected noop tracer to be returned")
	}
	if cfg.Metrics() == nil {
		t.Error("expected noop metrics to be returned")
	}
}

func TestNoopTracer(t *testing.T) {
	tracer := NewNoopTracer()

	ctx := context.Background()

	// Test various span creation methods don't panic
	ctx, span := tracer.StartSpan(ctx, "test")
	span.End()

	ctx, span = tracer.StartEntityRead(ctx, "Products", "1", false)
	span.End()

	ctx, span = tracer.StartEntityCreate(ctx, "Products")
	span.End()

	ctx, span = tracer.StartEntityUpdate(ctx, "Products", "1")
	span.End()

	ctx, span = tracer.StartEntityPatch(ctx, "Products", "1")
	span.End()

	ctx, span = tracer.StartEntityDelete(ctx, "Products", "1")
	span.End()

	ctx, span = tracer.StartBatch(ctx, 5)
	span.End()

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	_, span = tracer.StartRequest(ctx, req)
	span.End()
}

func TestNoopMetrics(t *testing.T) {
	metrics := NewNoopMetrics()

	ctx := context.Background()

	// Test various record methods don't panic
	metrics.RecordRequest(ctx, "Products", "read", 200, time.Second)
	metrics.RecordResultCount(ctx, "Products", 10)
	metrics.RecordDBQuery(ctx, "SELECT", time.Millisecond*100)
	metrics.RecordBatchSize(ctx, 5)
	metrics.RecordError(ctx, "Products", "read", "not_found")
	metrics.RecordRequestStart(ctx)
	metrics.RecordRequestEnd(ctx, time.Second, 200)
	metrics.RecordRequestDuration(ctx, time.Second, "Products", "read", 200)
}

func TestIsEnabled(t *testing.T) {
	// Empty config is not enabled
	cfg := NewConfig()
	if cfg.IsEnabled() {
		t.Error("expected empty config to not be enabled")
	}

	// With tracer provider is enabled
	cfg = NewConfig(WithTracerProvider(tracenoop.NewTracerProvider()))
	if !cfg.IsEnabled() {
		t.Error("expected config with tracer to be enabled")
	}

	// With meter provider is enabled
	cfg = NewConfig(WithMeterProvider(noop.NewMeterProvider()))
	if !cfg.IsEnabled() {
		t.Error("expected config with meter to be enabled")
	}
}

func TestTracerAddQueryOptions(t *testing.T) {
	tracer := NewNoopTracer()

	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, "test")

	// Should not panic
	tracer.AddQueryOptions(span, "Price gt 100", "Category", "Name,Price", "Name asc", 10, 20)
	span.End()
}

func TestTracerRecordError(t *testing.T) {
	tracer := NewNoopTracer()

	ctx := context.Background()
	_, span := tracer.StartSpan(ctx, "test")

	// Should not panic
	tracer.RecordError(span, nil)
	tracer.RecordError(span, context.Canceled)
	span.End()
}

func TestAttributes(t *testing.T) {
	// Test attribute helper functions don't panic
	_ = EntitySetAttr("Products")
	_ = EntityKeyAttr("1")
	_ = OperationAttr("read")
	_ = QueryFilterAttr("Price gt 100")
	_ = QuerySelectAttr("Name,Price")
	_ = QueryExpandAttr("Category")
	_ = QueryOrderByAttr("Name asc")
	_ = QueryTopAttr(10)
	_ = QuerySkipAttr(20)
	_ = ResultCountAttr(100)
	_ = BatchSizeAttr(5)
	_ = ChangesetSizeAttr(3)
}

func TestServerTimingOption(t *testing.T) {
	cfg := NewConfig(WithServerTiming())

	if !cfg.EnableServerTiming {
		t.Error("expected server timing to be enabled")
	}

	if !cfg.ServerTimingEnabled() {
		t.Error("expected ServerTimingEnabled() to return true")
	}
}

func TestServerTimingEnabledDefault(t *testing.T) {
	cfg := NewConfig()

	if cfg.EnableServerTiming {
		t.Error("expected server timing to be disabled by default")
	}

	if cfg.ServerTimingEnabled() {
		t.Error("expected ServerTimingEnabled() to return false by default")
	}
}

func TestServerTimingEnabledNilConfig(t *testing.T) {
	var cfg *Config
	if cfg.ServerTimingEnabled() {
		t.Error("expected ServerTimingEnabled() to return false for nil config")
	}
}

func TestStartServerTimingNoContext(t *testing.T) {
	// Test that StartServerTiming doesn't panic when timing is not in context
	ctx := context.Background()
	metric := StartServerTiming(ctx, "test")
	metric.Stop() // Should not panic
}

func TestStartServerTimingWithDescNoContext(t *testing.T) {
	// Test that StartServerTimingWithDesc doesn't panic when timing is not in context
	ctx := context.Background()
	metric := StartServerTimingWithDesc(ctx, "test", "Test description")
	metric.Stop() // Should not panic
}

func TestServerTimingMetricNilStop(t *testing.T) {
	// Test that Stop doesn't panic on nil metric
	var metric *ServerTimingMetric
	metric.Stop() // Should not panic
}

func TestServerTimingMetricEmptyStop(t *testing.T) {
	// Test that Stop doesn't panic on empty metric
	metric := &ServerTimingMetric{}
	metric.Stop() // Should not panic
}
