package odata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type BasePathProduct struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name"`
}

func TestSetBasePath_Validation(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"empty string", "", false},
		{"valid path", "/odata", false},
		{"valid nested", "/api/v1/odata", false},
		{"missing leading slash", "odata", true},
		{"trailing slash", "/odata/", true},
		{"only slash", "/", true}, // Should be rejected - use empty string for root
		{"whitespace trimmed", "  /odata  ", false},
		{"path traversal", "/odata/../admin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = service.SetBasePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetBasePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetBasePath_GetBasePath(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	tests := []struct {
		name     string
		setPath  string
		wantPath string
	}{
		{"empty path", "", ""},
		{"simple path", "/odata", "/odata"},
		{"nested path", "/api/v1/odata", "/api/v1/odata"},
		{"trimmed whitespace", "  /odata  ", "/odata"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := service.SetBasePath(tt.setPath); err != nil {
				t.Fatalf("SetBasePath() error = %v", err)
			}
			if got := service.GetBasePath(); got != tt.wantPath {
				t.Errorf("GetBasePath() = %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestBasePath_URLGeneration(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&BasePathProduct{})
	db.Create(&BasePathProduct{ID: 1, Name: "Test Product"})

	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&BasePathProduct{}); err != nil {
		t.Fatal(err)
	}
	if err := service.SetBasePath("/odata"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/odata/BasePathProducts", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	// Verify @odata.context includes base path
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("@odata.context not found")
	}
	expectedContext := "http://example.com/odata/$metadata#BasePathProducts"
	if context != expectedContext {
		t.Errorf("@odata.context = %q, want %q", context, expectedContext)
	}

	// Verify entity @odata.id includes base path
	values, ok := response["value"].([]interface{})
	if !ok || len(values) == 0 {
		t.Fatal("value array not found or empty")
	}
	entity := values[0].(map[string]interface{})
	id, ok := entity["@odata.id"].(string)
	if !ok {
		t.Fatal("@odata.id not found")
	}
	expectedID := "http://example.com/odata/BasePathProducts(1)"
	if id != expectedID {
		t.Errorf("@odata.id = %q, want %q", id, expectedID)
	}
}

func TestBasePath_PathStripping(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&BasePathProduct{})

	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&BasePathProduct{}); err != nil {
		t.Fatal(err)
	}
	if err := service.SetBasePath("/odata"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		requestPath    string
		expectStatus   int
		expectStripped bool
	}{
		{"with base path prefix", "/odata/BasePathProducts", http.StatusOK, true},
		{"exact base path match", "/odata", http.StatusOK, true},
		{"partial match - should not strip", "/odatax/BasePathProducts", http.StatusNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			req.Host = "example.com"
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("expected status %d, got %d", tt.expectStatus, w.Code)
			}
		})
	}
}

func TestBasePath_ServiceDocument(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&BasePathProduct{})

	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&BasePathProduct{}); err != nil {
		t.Fatal(err)
	}
	if err := service.SetBasePath("/odata"); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/odata/", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	// Verify @odata.context includes base path
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("@odata.context not found")
	}
	expectedContext := "http://example.com/odata/$metadata"
	if context != expectedContext {
		t.Errorf("@odata.context = %q, want %q", context, expectedContext)
	}
}

func TestBasePath_RootMounting(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&BasePathProduct{})
	db.Create(&BasePathProduct{ID: 1, Name: "Test"})

	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&BasePathProduct{}); err != nil {
		t.Fatal(err)
	}
	// Explicitly set empty base path to ensure clean state
	if err := service.SetBasePath(""); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/BasePathProducts", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	// Verify @odata.context does NOT include base path
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("@odata.context not found")
	}
	expectedContext := "http://example.com/$metadata#BasePathProducts"
	if context != expectedContext {
		t.Errorf("@odata.context = %q, want %q", context, expectedContext)
	}
}

func TestBasePath_ConcurrentGetBasePath(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.SetBasePath("/odata"); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			basePath := service.GetBasePath()
			if basePath != "/odata" {
				t.Errorf("expected /odata, got %s", basePath)
			}
		}()
	}
	wg.Wait()
}

func TestBasePath_MultipleServicesWithDifferentPaths(t *testing.T) {
	// Create two services with different base paths
	db1, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db1.AutoMigrate(&BasePathProduct{})
	db1.Create(&BasePathProduct{ID: 1, Name: "Service1 Product"})

	service1, err := NewService(db1)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service1.RegisterEntity(&BasePathProduct{}); err != nil {
		t.Fatal(err)
	}
	if err := service1.SetBasePath("/api/v1"); err != nil {
		t.Fatal(err)
	}

	db2, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db2.AutoMigrate(&BasePathProduct{})
	db2.Create(&BasePathProduct{ID: 2, Name: "Service2 Product"})

	service2, err := NewService(db2)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service2.RegisterEntity(&BasePathProduct{}); err != nil {
		t.Fatal(err)
	}
	if err := service2.SetBasePath("/api/v2"); err != nil {
		t.Fatal(err)
	}

	// Test service1
	req1 := httptest.NewRequest("GET", "/api/v1/BasePathProducts", nil)
	req1.Host = "example.com"
	w1 := httptest.NewRecorder()
	service1.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("service1: expected status 200, got %d: %s", w1.Code, w1.Body.String())
	}

	var response1 map[string]interface{}
	if err := json.NewDecoder(w1.Body).Decode(&response1); err != nil {
		t.Fatal(err)
	}

	context1, ok := response1["@odata.context"].(string)
	if !ok {
		t.Fatal("service1: @odata.context not found")
	}
	expectedContext1 := "http://example.com/api/v1/$metadata#BasePathProducts"
	if context1 != expectedContext1 {
		t.Errorf("service1: @odata.context = %q, want %q", context1, expectedContext1)
	}

	// Test service2
	req2 := httptest.NewRequest("GET", "/api/v2/BasePathProducts", nil)
	req2.Host = "example.com"
	w2 := httptest.NewRecorder()
	service2.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("service2: expected status 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var response2 map[string]interface{}
	if err := json.NewDecoder(w2.Body).Decode(&response2); err != nil {
		t.Fatal(err)
	}

	context2, ok := response2["@odata.context"].(string)
	if !ok {
		t.Fatal("service2: @odata.context not found")
	}
	expectedContext2 := "http://example.com/api/v2/$metadata#BasePathProducts"
	if context2 != expectedContext2 {
		t.Errorf("service2: @odata.context = %q, want %q", context2, expectedContext2)
	}

	// Verify services maintain separate base paths
	if service1.GetBasePath() != "/api/v1" {
		t.Errorf("service1: expected /api/v1, got %s", service1.GetBasePath())
	}
	if service2.GetBasePath() != "/api/v2" {
		t.Errorf("service2: expected /api/v2, got %s", service2.GetBasePath())
	}
}
