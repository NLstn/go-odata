package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Document is an entity with Edm.Untyped properties.
type DocumentEntity struct {
	ID      string          `json:"id" gorm:"primarykey" odata:"key"`
	Title   string          `json:"title" odata:"required"`
	Payload json.RawMessage `json:"payload" gorm:"type:text"`
	Meta    interface{}     `json:"meta" gorm:"-" odata:"untyped"`
}

func setupUntypedTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&DocumentEntity{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(&DocumentEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestUntypedPropertyInXMLMetadata verifies that json.RawMessage and interface{} fields
// appear as Edm.Untyped in the XML $metadata document.
func TestUntypedPropertyInXMLMetadata(t *testing.T) {
	service, _ := setupUntypedTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// The Payload field (json.RawMessage) must appear as Edm.Untyped
	if !strings.Contains(body, `Type="Edm.Untyped"`) {
		t.Errorf("expected Edm.Untyped in XML metadata, got:\n%s", body)
	}
}

// TestUntypedPropertyInJSONMetadata verifies that json.RawMessage and interface{} fields
// appear as Edm.Untyped in the JSON $metadata (CSDL JSON) document.
func TestUntypedPropertyInJSONMetadata(t *testing.T) {
	service, _ := setupUntypedTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Fatalf("failed to parse JSON metadata: %v", err)
	}

	// Navigate to the DocumentEntity type definition
	svc, ok := meta["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected ODataService key in metadata, got: %v", meta)
	}
	docEntity, ok := svc["DocumentEntity"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected DocumentEntity in ODataService, got keys: %v", keys(svc))
	}

	// Payload: json.RawMessage should be Edm.Untyped
	payloadDef, ok := docEntity["payload"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected payload property in DocumentEntity, got keys: %v", keys(docEntity))
	}
	if payloadDef["$Type"] != "Edm.Untyped" {
		t.Errorf("expected payload.$Type == \"Edm.Untyped\", got %v", payloadDef["$Type"])
	}

	// Meta: interface{} with odata:"untyped" tag should also be Edm.Untyped
	metaDef, ok := docEntity["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected meta property in DocumentEntity, got keys: %v", keys(docEntity))
	}
	if metaDef["$Type"] != "Edm.Untyped" {
		t.Errorf("expected meta.$Type == \"Edm.Untyped\", got %v", metaDef["$Type"])
	}
}

// TestUntypedTagOverride verifies that odata:"untyped" forces Edm.Untyped in metadata
// even on a field whose Go type would normally map to something else.
func TestUntypedTagOverride(t *testing.T) {
	type EntityWithUntypedTag struct {
		ID   int    `json:"id" gorm:"primarykey" odata:"key"`
		Data string `json:"data" odata:"untyped"` // string field explicitly marked as untyped
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&EntityWithUntypedTag{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&EntityWithUntypedTag{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Fatalf("failed to parse JSON metadata: %v", err)
	}

	svc, ok := meta["ODataService"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected ODataService in metadata")
	}
	entity, ok := svc["EntityWithUntypedTag"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected EntityWithUntypedTag in ODataService")
	}
	dataDef, ok := entity["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data property in EntityWithUntypedTag")
	}
	if dataDef["$Type"] != "Edm.Untyped" {
		t.Errorf("expected data.$Type == \"Edm.Untyped\", got %v", dataDef["$Type"])
	}
}

// keys returns the keys of a map for diagnostic output.
func keys(m map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
