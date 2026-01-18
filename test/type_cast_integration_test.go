package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TypeCastGizmo struct {
	ID             uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name           string  `json:"Name"`
	Type           string  `json:"Type,omitempty" gorm:"default:'Gizmo'"`
	SpecialFeature *string `json:"SpecialFeature,omitempty"`
}

func TestCollectionTypeCastWithAlternateDiscriminator(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TypeCastGizmo{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}

	entries := []TypeCastGizmo{
		{ID: 1, Name: "Standard Gizmo", Type: "Gizmo"},
		{ID: 2, Name: "Special Gizmo A", Type: "SpecialGizmo", SpecialFeature: stringPtr("A")},
		{ID: 3, Name: "Regular Gizmo", Type: "Gizmo"},
		{ID: 4, Name: "Special Gizmo B", Type: "SpecialGizmo", SpecialFeature: stringPtr("B")},
	}

	for _, entry := range entries {
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("failed to create gizmo %d: %v", entry.ID, err)
		}
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&TypeCastGizmo{}); err != nil {
		t.Fatalf("failed to register entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/TypeCastGizmos/ODataService.SpecialGizmo", nil)
	resp := httptest.NewRecorder()
	service.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	values, ok := payload["value"].([]interface{})
	if !ok {
		t.Fatalf("expected response 'value' array, got %#v", payload["value"])
	}

	if len(values) != 2 {
		t.Fatalf("expected 2 derived gizmos, got %d", len(values))
	}

	expectedIDs := map[uint]struct{}{2: {}, 4: {}}
	for _, v := range values {
		entry, ok := v.(map[string]interface{})
		if !ok {
			t.Fatalf("expected object entries, got %#v", v)
		}

		idValue, ok := entry["ID"].(float64)
		if !ok {
			t.Fatalf("expected numeric ID, got %#v", entry["ID"])
		}
		id := uint(idValue)
		if _, exists := expectedIDs[id]; !exists {
			t.Fatalf("unexpected gizmo ID %d returned", id)
		}
		delete(expectedIDs, id)
	}

	if len(expectedIDs) != 0 {
		t.Fatalf("missing expected gizmo IDs: %#v", expectedIDs)
	}
}

func stringPtr(v string) *string {
	return &v
}
