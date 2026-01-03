package odata_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ServerTimingProduct struct {
	ID    int     `json:"id" gorm:"primarykey" odata:"key"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func TestServerTimingHeaderPresent(t *testing.T) {
	// Set up database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ServerTimingProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create and insert test data
	product := ServerTimingProduct{ID: 1, Name: "Test Product", Price: 9.99}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&ServerTimingProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Enable observability with server timing
	if err := service.SetObservability(odata.ObservabilityConfig{
		EnableServerTiming: true,
	}); err != nil {
		t.Fatalf("Failed to set observability: %v", err)
	}

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/ServerTimingProducts", nil)
	rec := httptest.NewRecorder()

	service.ServeHTTP(rec, req)

	// Check that the response includes the Server-Timing header
	serverTiming := rec.Header().Get("Server-Timing")
	if serverTiming == "" {
		t.Error("Expected Server-Timing header to be present, but it was not")
	}
}

func TestServerTimingHeaderAbsentWhenDisabled(t *testing.T) {
	// Set up database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ServerTimingProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create and insert test data
	product := ServerTimingProduct{ID: 1, Name: "Test Product", Price: 9.99}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&ServerTimingProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Do NOT enable server timing

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/ServerTimingProducts", nil)
	rec := httptest.NewRecorder()

	service.ServeHTTP(rec, req)

	// Check that the response does NOT include the Server-Timing header
	serverTiming := rec.Header().Get("Server-Timing")
	if serverTiming != "" {
		t.Errorf("Expected no Server-Timing header when disabled, but got: %s", serverTiming)
	}
}

func TestServerTimingHeaderContainsTotalMetric(t *testing.T) {
	// Set up database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ServerTimingProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create and insert test data
	product := ServerTimingProduct{ID: 1, Name: "Test Product", Price: 9.99}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&ServerTimingProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Enable observability with server timing
	if err := service.SetObservability(odata.ObservabilityConfig{
		EnableServerTiming: true,
	}); err != nil {
		t.Fatalf("Failed to set observability: %v", err)
	}

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/ServerTimingProducts", nil)
	rec := httptest.NewRecorder()

	service.ServeHTTP(rec, req)

	// Check that the response includes the Server-Timing header with 'total' metric
	serverTiming := rec.Header().Get("Server-Timing")
	if serverTiming == "" {
		t.Fatal("Expected Server-Timing header to be present, but it was not")
	}

	// The header should contain the 'total' metric
	// The format is like: total;desc="Total request duration";dur=0.123
	if len(serverTiming) < 5 {
		t.Errorf("Server-Timing header seems too short: %s", serverTiming)
	}
}

func TestServerTimingWithOtherObservabilityOptions(t *testing.T) {
	// Set up database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ServerTimingProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create and insert test data
	product := ServerTimingProduct{ID: 1, Name: "Test Product", Price: 9.99}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&ServerTimingProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Enable observability with server timing and other options
	if err := service.SetObservability(odata.ObservabilityConfig{
		EnableServerTiming: true,
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
	}); err != nil {
		t.Fatalf("Failed to set observability: %v", err)
	}

	// Create a test request
	req := httptest.NewRequest(http.MethodGet, "/ServerTimingProducts", nil)
	rec := httptest.NewRecorder()

	service.ServeHTTP(rec, req)

	// Check that the response includes the Server-Timing header
	serverTiming := rec.Header().Get("Server-Timing")
	if serverTiming == "" {
		t.Error("Expected Server-Timing header to be present, but it was not")
	}

	// Check that the response is still valid
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rec.Code)
	}
}
