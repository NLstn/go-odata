package observability

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	noopmetric "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestNewTracer(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	if tracer == nil {
		t.Fatal("NewTracer() should return non-nil tracer")
		return
	}
	if tracer.serviceName != "test-service" {
		t.Errorf("serviceName = %q, want %q", tracer.serviceName, "test-service")
	}
}

func TestTracer_StartEntityRead_WithKey(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	ctx, span := tracer.StartEntityRead(context.Background(), "Products", "123", false)
	defer span.End()

	if ctx == nil {
		t.Error("StartEntityRead() should return non-nil context")
	}
}

func TestTracer_StartEntityRead_WithoutKey(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	ctx, span := tracer.StartEntityRead(context.Background(), "Products", "", false)
	defer span.End()

	if ctx == nil {
		t.Error("StartEntityRead() should return non-nil context")
	}
}

func TestTracer_StartAction(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	ctx, span := tracer.StartAction(context.Background(), "UpdatePrice", "Products", true)
	defer span.End()

	if ctx == nil {
		t.Error("StartAction() should return non-nil context")
	}
}

func TestTracer_StartAction_Unbound(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	ctx, span := tracer.StartAction(context.Background(), "GlobalAction", "", false)
	defer span.End()

	if ctx == nil {
		t.Error("StartAction() should return non-nil context")
	}
}

func TestTracer_StartFunction(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	ctx, span := tracer.StartFunction(context.Background(), "GetPrice", "Products", true)
	defer span.End()

	if ctx == nil {
		t.Error("StartFunction() should return non-nil context")
	}
}

func TestTracer_StartFunction_Unbound(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	ctx, span := tracer.StartFunction(context.Background(), "GlobalFunction", "", false)
	defer span.End()

	if ctx == nil {
		t.Error("StartFunction() should return non-nil context")
	}
}

func TestTracer_StartChangeset(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	ctx, span := tracer.StartChangeset(context.Background(), "changeset-123", 5)
	defer span.End()

	if ctx == nil {
		t.Error("StartChangeset() should return non-nil context")
	}
}

func TestTracer_StartDBQuery(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	ctx, span := tracer.StartDBQuery(context.Background(), "SELECT")
	defer span.End()

	if ctx == nil {
		t.Error("StartDBQuery() should return non-nil context")
	}
}

func TestTracer_SetHTTPStatus_Success(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	ctx, span := tracer.StartSpan(context.Background(), "test")
	defer span.End()

	// Should not panic
	tracer.SetHTTPStatus(ctx, http.StatusOK)
}

func TestTracer_SetHTTPStatus_Error(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	ctx, span := tracer.StartSpan(context.Background(), "test")
	defer span.End()

	// Should not panic and should set error status
	tracer.SetHTTPStatus(ctx, http.StatusInternalServerError)
}

func TestTracer_AddQueryOptions_All(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	_, span := tracer.StartSpan(context.Background(), "test")
	defer span.End()

	// Should not panic with all options set
	tracer.AddQueryOptions(span, "Name eq 'Test'", "Category", "Name,Price", "Name asc", 10, 5)
}

func TestTracer_AddQueryOptions_None(t *testing.T) {
	tp := tracenoop.NewTracerProvider()
	tracer := NewTracer(tp, "test-service")

	_, span := tracer.StartSpan(context.Background(), "test")
	defer span.End()

	// Should not panic with no options set
	tracer.AddQueryOptions(span, "", "", "", "", 0, 0)
}

func TestLoggerWithTrace(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Without valid trace context
	enrichedLogger := LoggerWithTrace(context.Background(), logger)
	if enrichedLogger == nil {
		t.Error("LoggerWithTrace() should return non-nil logger")
	}
}

func TestNewMetrics(t *testing.T) {
	// Test with noop provider from otel library
	mp := noopmetric.NewMeterProvider()
	metrics := NewMetrics(mp)

	if metrics == nil {
		t.Fatal("NewMetrics() should return non-nil metrics")
	}
}

func TestWithServiceVersion(t *testing.T) {
	cfg := NewConfig(
		WithServiceVersion("1.0.0"),
	)

	if cfg.ServiceVersion != "1.0.0" {
		t.Errorf("ServiceVersion = %q, want %q", cfg.ServiceVersion, "1.0.0")
	}
}

func TestWithLogger(t *testing.T) {
	// WithLogger is currently a no-op, but should not panic
	cfg := NewConfig(
		WithLogger(nil),
	)

	if cfg == nil {
		t.Error("NewConfig() should return non-nil config")
	}
}

func TestConfig_Tracer_Nil(t *testing.T) {
	var cfg *Config

	tracer := cfg.Tracer()
	if tracer == nil {
		t.Error("Tracer() should return noop tracer for nil config")
	}
}

func TestConfig_Metrics_Nil(t *testing.T) {
	var cfg *Config

	metrics := cfg.Metrics()
	if metrics == nil {
		t.Error("Metrics() should return noop metrics for nil config")
	}
}

func TestConfig_Tracer_NotInitialized(t *testing.T) {
	cfg := NewConfig()

	tracer := cfg.Tracer()
	if tracer == nil {
		t.Error("Tracer() should return noop tracer when not initialized")
	}
}

func TestConfig_Metrics_NotInitialized(t *testing.T) {
	cfg := NewConfig()

	metrics := cfg.Metrics()
	if metrics == nil {
		t.Error("Metrics() should return noop metrics when not initialized")
	}
}

func TestMetrics_RecordRequest(t *testing.T) {
	metrics := NewNoopMetrics()

	// Should not panic
	metrics.RecordRequest(context.Background(), "Products", "read", http.StatusOK, time.Second)
}

func TestMetrics_RecordResultCount(t *testing.T) {
	metrics := NewNoopMetrics()

	// Should not panic
	metrics.RecordResultCount(context.Background(), "Products", 100)
}

func TestMetrics_RecordDBQuery(t *testing.T) {
	metrics := NewNoopMetrics()

	// Should not panic
	metrics.RecordDBQuery(context.Background(), "SELECT", time.Millisecond*50)
}

func TestMetrics_RecordBatchSize(t *testing.T) {
	metrics := NewNoopMetrics()

	// Should not panic
	metrics.RecordBatchSize(context.Background(), 10)
}

func TestMetrics_RecordError(t *testing.T) {
	metrics := NewNoopMetrics()

	// Should not panic
	metrics.RecordError(context.Background(), "Products", "read", "not_found")
}

func TestMetrics_RecordRequestStart(t *testing.T) {
	metrics := NewNoopMetrics()

	// Should not panic
	metrics.RecordRequestStart(context.Background())
}

func TestMetrics_RecordRequestEnd(t *testing.T) {
	metrics := NewNoopMetrics()

	// Should not panic
	metrics.RecordRequestEnd(context.Background(), time.Second, http.StatusOK)
}

func TestMetrics_RecordRequestDuration(t *testing.T) {
	metrics := NewNoopMetrics()

	// Should not panic
	metrics.RecordRequestDuration(context.Background(), time.Second, "Products", "read", http.StatusOK)
}

func TestNoopTracer_AllOperations(t *testing.T) {
	tracer := NewNoopTracer()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "StartSpan",
			fn: func() {
				_, span := tracer.StartSpan(ctx, "test")
				span.End()
			},
		},
		{
			name: "StartEntityRead",
			fn: func() {
				_, span := tracer.StartEntityRead(ctx, "Products", "1", false)
				span.End()
			},
		},
		{
			name: "StartEntityCreate",
			fn: func() {
				_, span := tracer.StartEntityCreate(ctx, "Products")
				span.End()
			},
		},
		{
			name: "StartEntityUpdate",
			fn: func() {
				_, span := tracer.StartEntityUpdate(ctx, "Products", "1")
				span.End()
			},
		},
		{
			name: "StartEntityPatch",
			fn: func() {
				_, span := tracer.StartEntityPatch(ctx, "Products", "1")
				span.End()
			},
		},
		{
			name: "StartEntityDelete",
			fn: func() {
				_, span := tracer.StartEntityDelete(ctx, "Products", "1")
				span.End()
			},
		},
		{
			name: "StartBatch",
			fn: func() {
				_, span := tracer.StartBatch(ctx, 5)
				span.End()
			},
		},
		{
			name: "StartRequest",
			fn: func() {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				_, span := tracer.StartRequest(ctx, req)
				span.End()
			},
		},
		{
			name: "StartChangeset",
			fn: func() {
				_, span := tracer.StartChangeset(ctx, "cs-1", 3)
				span.End()
			},
		},
		{
			name: "StartAction",
			fn: func() {
				_, span := tracer.StartAction(ctx, "TestAction", "Products", true)
				span.End()
			},
		},
		{
			name: "StartFunction",
			fn: func() {
				_, span := tracer.StartFunction(ctx, "TestFunc", "Products", false)
				span.End()
			},
		},
		{
			name: "StartDBQuery",
			fn: func() {
				_, span := tracer.StartDBQuery(ctx, "SELECT")
				span.End()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			tt.fn()
		})
	}
}
