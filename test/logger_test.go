package odata_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// LoggerTestProduct is a test entity for logger tests
type LoggerTestProduct struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
}

func setupLoggerTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&LoggerTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(LoggerTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestSetLogger_CustomLogger(t *testing.T) {
	service, db := setupLoggerTestService(t)

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Set the custom logger
	service.SetLogger(logger)

	// Insert test data
	db.Create(&LoggerTestProduct{ID: 1, Name: "Test"})

	// Make a request that should be logged
	req := httptest.NewRequest(http.MethodGet, "/LoggerTestProducts", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	// Verify the request succeeded
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Note: We don't verify log output content since that's implementation detail,
	// but we verify the logger was set without error and the service still works
}

func TestSetLogger_NilLogger(t *testing.T) {
	service, db := setupLoggerTestService(t)

	// Set nil logger (should fall back to default)
	service.SetLogger(nil)

	// Insert test data
	db.Create(&LoggerTestProduct{ID: 1, Name: "Test"})

	// Make a request to verify service still works
	req := httptest.NewRequest(http.MethodGet, "/LoggerTestProducts", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	// Verify the request succeeded
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestSetLogger_AfterRegistration(t *testing.T) {
	service, db := setupLoggerTestService(t)

	// Set logger after entity registration
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	service.SetLogger(logger)

	// Insert test data
	db.Create(&LoggerTestProduct{ID: 1, Name: "Test"})

	// Make multiple types of requests
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"GET collection", http.MethodGet, "/LoggerTestProducts"},
		{"GET entity", http.MethodGet, "/LoggerTestProducts(1)"},
		{"GET service document", http.MethodGet, "/"},
		{"GET metadata", http.MethodGet, "/$metadata"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)

			// Just verify no panic occurred
			if w.Code == 0 {
				t.Error("Expected non-zero status code")
			}
		})
	}
}

func TestSetLogger_MultipleSetCalls(t *testing.T) {
	service, db := setupLoggerTestService(t)

	// Set logger multiple times
	var buf1, buf2 bytes.Buffer
	logger1 := slog.New(slog.NewTextHandler(&buf1, nil))
	logger2 := slog.New(slog.NewTextHandler(&buf2, nil))

	service.SetLogger(logger1)
	service.SetLogger(logger2)
	service.SetLogger(nil) // Back to default
	service.SetLogger(logger1)

	// Insert test data
	db.Create(&LoggerTestProduct{ID: 1, Name: "Test"})

	// Verify service still works after multiple logger changes
	req := httptest.NewRequest(http.MethodGet, "/LoggerTestProducts", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestSetLogger_WithEntityOperations(t *testing.T) {
	service, _ := setupLoggerTestService(t)

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	service.SetLogger(logger)

	// Test POST
	postBody := `{"name": "New Product"}`
	postReq := httptest.NewRequest(http.MethodPost, "/LoggerTestProducts", bytes.NewBufferString(postBody))
	postReq.Header.Set("Content-Type", "application/json")
	postW := httptest.NewRecorder()
	service.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusCreated {
		t.Errorf("POST Status = %v, want %v", postW.Code, http.StatusCreated)
	}

	// Get the created entity's ID
	var postResponse map[string]interface{}
	json.NewDecoder(postW.Body).Decode(&postResponse)
	id := int(postResponse["id"].(float64))

	// Test GET
	getReq := httptest.NewRequest(http.MethodGet, "/LoggerTestProducts("+strconv.Itoa(id)+")", nil)
	getW := httptest.NewRecorder()
	service.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Errorf("GET Status = %v, want %v", getW.Code, http.StatusOK)
	}

	// Test PATCH
	patchBody := `{"name": "Updated Product"}`
	patchReq := httptest.NewRequest(http.MethodPatch, "/LoggerTestProducts("+strconv.Itoa(id)+")", bytes.NewBufferString(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchW := httptest.NewRecorder()
	service.ServeHTTP(patchW, patchReq)

	// PATCH returns 200 or 204 depending on Prefer header
	if patchW.Code != http.StatusOK && patchW.Code != http.StatusNoContent {
		t.Errorf("PATCH Status = %v, want 200 or 204", patchW.Code)
	}

	// Test DELETE
	deleteReq := httptest.NewRequest(http.MethodDelete, "/LoggerTestProducts("+strconv.Itoa(id)+")", nil)
	deleteW := httptest.NewRecorder()
	service.ServeHTTP(deleteW, deleteReq)

	if deleteW.Code != http.StatusNoContent {
		t.Errorf("DELETE Status = %v, want %v", deleteW.Code, http.StatusNoContent)
	}
}
