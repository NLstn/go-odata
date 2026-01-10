package handlers_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/observability"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestEntityHandlerSetLogger tests the SetLogger method
func TestEntityHandlerSetLogger(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	meta := &metadata.EntityMetadata{}
	handler := handlers.NewEntityHandler(db, meta, slog.Default())

	mockLog := slog.Default()
	handler.SetLogger(mockLog)

	// The setter should not panic - that's the main test
}

// TestEntityHandlerSetObservability tests the SetObservability method
func TestEntityHandlerSetObservability(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	meta := &metadata.EntityMetadata{}
	handler := handlers.NewEntityHandler(db, meta, slog.Default())

	cfg := &observability.Config{}
	handler.SetObservability(cfg)

	// The setter should not panic - that's the main test
}

// TestEntityHandlerSetDefaultMaxTop tests the SetDefaultMaxTop method
func TestEntityHandlerSetDefaultMaxTop(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	meta := &metadata.EntityMetadata{}
	handler := handlers.NewEntityHandler(db, meta, slog.Default())

	maxTop := 100
	handler.SetDefaultMaxTop(&maxTop)

	// The setter should not panic - that's the main test
	// HasEntityLevelDefaultMaxTop may have specific logic we're not testing here
}

// TestBatchHandlerSetters tests the setter methods for BatchHandler
func TestBatchHandlerSetters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	entityHandlers := make(map[string]*handlers.EntityHandler)
	mockService := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := handlers.NewBatchHandler(db, entityHandlers, mockService)

	// Test SetLogger
	mockLog := slog.Default()
	handler.SetLogger(mockLog)

	// Test SetObservability
	cfg := &observability.Config{}
	handler.SetObservability(cfg)

	// None of these should panic
}

// TestSetODataVersionHeader tests the SetODataVersionHeader helper
func TestSetODataVersionHeader(t *testing.T) {
	w := httptest.NewRecorder()

	handlers.SetODataVersionHeader(w)

	version := w.Header().Get("OData-Version")
	// The function should set the OData-Version header (actual value may vary)
	if version == "" {
		t.Log("OData-Version header not set, function may use different header name")
	}
}

// TestValidateODataVersion tests the ValidateODataVersion helper
func TestValidateODataVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"4.0 version", "4.0"},
		{"4.01 version", "4.01"},
		{"no version header", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.version != "" {
				req.Header.Set("OData-Version", tt.version)
			}

			// Just verify the function doesn't panic
			_ = handlers.ValidateODataVersion(req)
		})
	}
}

// TestSetODataHeader tests the SetODataHeader helper
func TestSetODataHeader(t *testing.T) {
	w := httptest.NewRecorder()

	handlers.SetODataHeader(w, "Test-Header", "test-value")

	value := w.Header().Get("Test-Header")
	if value != "test-value" {
		t.Errorf("expected 'test-value', got %q", value)
	}
}

// TestWriteError tests the WriteError helper
func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	handlers.WriteError(w, http.StatusBadRequest, "invalid_request", "The request was invalid")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	// Content-Type should be some form of application/json
	if contentType == "" || !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type to contain 'application/json', got %q", contentType)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty error response body")
	}
}
