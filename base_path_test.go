package odata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	service := NewService(db)

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
			err := service.SetBasePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetBasePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetBasePath_GetBasePath(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	service := NewService(db)

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

	service := NewService(db)
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

	service := NewService(db)
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

	service := NewService(db)
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

	service := NewService(db)
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
