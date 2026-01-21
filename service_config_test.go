package odata

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// TestSetLogger verifies custom logger configuration
func TestSetLogger(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test setting a custom logger
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	service.SetLogger(logger)

	// Verify logger was set by checking that debug logs are written
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// The logger should have been used for the registration debug message
	if buf.Len() == 0 {
		t.Error("Expected logger to be used, but no logs were written")
	}

	// Test setting nil logger (should use default)
	service.SetLogger(nil)
	// Service should still work with default logger
}

// TestSetObservabilityAndObservability verifies observability configuration
func TestSetObservabilityAndObservability(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Initially no observability configured
	if service.Observability() != nil {
		t.Error("Expected observability to be nil initially")
	}

	// Test setting observability with tracer provider
	tp := tracenoop.NewTracerProvider()
	mp := noop.NewMeterProvider()

	err = service.SetObservability(ObservabilityConfig{
		TracerProvider:          tp,
		MeterProvider:           mp,
		ServiceName:             "test-service",
		ServiceVersion:          "1.0.0",
		EnableDetailedDBTracing: true,
		EnableServerTiming:      true,
	})
	if err != nil {
		t.Fatalf("Failed to set observability: %v", err)
	}

	// Verify observability was configured
	obs := service.Observability()
	if obs == nil {
		t.Fatal("Expected observability to be configured")
	}

	// Test with minimal config (no tracer/meter providers)
	service2, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create second service: %v", err)
	}

	err = service2.SetObservability(ObservabilityConfig{
		ServiceName: "minimal-service",
	})
	if err != nil {
		t.Fatalf("Failed to set minimal observability: %v", err)
	}
}

// TestStartServerTiming verifies server timing metrics
func TestStartServerTiming(t *testing.T) {
	// Test with no server timing configured
	ctx := context.Background()
	metric := StartServerTiming(ctx, "test-operation")
	if metric == nil {
		t.Fatal("Expected non-nil metric")
	}
	metric.Stop() // Should be safe to call even without timing configured

	// Test with description
	metricWithDesc := StartServerTimingWithDesc(ctx, "test-op", "Test operation description")
	if metricWithDesc == nil {
		t.Fatal("Expected non-nil metric with description")
	}
	metricWithDesc.Stop()
}

// TestEnableAsyncProcessing verifies async processing configuration
func TestEnableAsyncProcessing(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Initially no async manager
	if service.AsyncManager() != nil {
		t.Error("Expected no async manager initially")
	}

	// Enable async processing with custom config
	err = service.EnableAsyncProcessing(AsyncConfig{
		MonitorPathPrefix:    "/api/async/",
		DefaultRetryInterval: 5 * time.Second,
		MaxQueueSize:         10,
		JobRetention:         10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Failed to enable async processing: %v", err)
	}

	// Verify async manager was created
	if service.AsyncManager() == nil {
		t.Error("Expected async manager to be created")
	}

	// Verify monitor prefix was set
	prefix := service.AsyncMonitorPrefix()
	if prefix != "/api/async/" {
		t.Errorf("Expected monitor prefix '/api/async/', got %q", prefix)
	}

	// Test with default config values
	service2, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create second service: %v", err)
	}

	err = service2.EnableAsyncProcessing(AsyncConfig{})
	if err != nil {
		t.Fatalf("Failed to enable async with defaults: %v", err)
	}

	// Verify default monitor prefix
	defaultPrefix := service2.AsyncMonitorPrefix()
	if defaultPrefix != "/$async/jobs/" {
		t.Errorf("Expected default monitor prefix '/$async/jobs/', got %q", defaultPrefix)
	}

	// Test with disable retention
	service3, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create third service: %v", err)
	}

	err = service3.EnableAsyncProcessing(AsyncConfig{
		DisableRetention: true,
	})
	if err != nil {
		t.Fatalf("Failed to enable async with retention disabled: %v", err)
	}
}

// TestSetPreRequestHook verifies pre-request hook configuration
func TestSetPreRequestHook(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register a test entity
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Set a pre-request hook that adds context value
	type contextKey string
	const userKey contextKey = "user"

	hookCalled := false
	service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		hookCalled = true
		return context.WithValue(r.Context(), userKey, "test-user"), nil
	})

	// Make a request to trigger the hook
	req := httptest.NewRequest("GET", "/Products", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if !hookCalled {
		t.Error("Expected pre-request hook to be called")
	}

	// Test hook that returns error
	service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		return nil, http.ErrAbortHandler
	})

	req = httptest.NewRequest("GET", "/Products", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

// TestClose verifies service cleanup
func TestClose(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Enable async processing to have something to close
	err = service.EnableAsyncProcessing(AsyncConfig{})
	if err != nil {
		t.Fatalf("Failed to enable async: %v", err)
	}

	if service.AsyncManager() == nil {
		t.Fatal("Expected async manager to be set")
	}

	// Close the service
	err = service.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Verify async manager was cleaned up
	if service.AsyncManager() != nil {
		t.Error("Expected async manager to be nil after close")
	}

	// Test closing again should be safe
	err = service.Close()
	if err != nil {
		t.Errorf("Second close returned error: %v", err)
	}

	// Test closing nil service
	var nilService *Service
	err = nilService.Close()
	if err != nil {
		t.Errorf("Close on nil service returned error: %v", err)
	}
}

// TestSetNamespace verifies namespace configuration
func TestSetNamespace(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Register an entity
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Set custom namespace
	err = service.SetNamespace("MyCompany.MyService")
	if err != nil {
		t.Fatalf("Failed to set namespace: %v", err)
	}

	// Verify namespace was set by checking metadata
	req := httptest.NewRequest("GET", "/$metadata", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("MyCompany.MyService")) {
		t.Error("Expected namespace 'MyCompany.MyService' in metadata")
	}

	// Test empty namespace returns error
	err = service.SetNamespace("")
	if err == nil {
		t.Error("Expected error for empty namespace")
	}

	// Test whitespace-only namespace returns error
	err = service.SetNamespace("   ")
	if err == nil {
		t.Error("Expected error for whitespace-only namespace")
	}

	// Test setting same namespace again (should be no-op)
	err = service.SetNamespace("MyCompany.MyService")
	if err != nil {
		t.Errorf("Setting same namespace returned error: %v", err)
	}
}

// TestResetFTS verifies FTS cache reset
func TestResetFTS(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Should not panic even if FTS is not configured
	service.ResetFTS()

	// No assertion needed - just verifying it doesn't panic
}

// TestHandler verifies the HTTP handler function
func TestHandler(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Get the handler
	handler := service.Handler()
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	// Register entity to test
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Test that handler works
	req := httptest.NewRequest("GET", "/Products", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestTransactionFromContext verifies transaction context extraction
func TestTransactionFromContext(t *testing.T) {
	// Test with context that has no transaction
	ctx := context.Background()
	tx, ok := TransactionFromContext(ctx)
	if ok {
		t.Error("Expected no transaction from empty context")
	}
	if tx != nil {
		t.Error("Expected nil transaction from empty context")
	}
}
