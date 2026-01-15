package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// PropertyHelperTestEntity is a test entity for property helper tests
type PropertyHelperTestEntity struct {
	ID              uint             `json:"ID" gorm:"primaryKey" odata:"key"`
	Name            string           `json:"Name"`
	StructProp      string           `json:"StructuralProperty"`
	ComplexProp     Address          `json:"ComplexProperty" gorm:"embedded"`
	StreamProp      []byte           `json:"StreamProperty" odata:"stream"`
	RelatedProducts []RelatedProduct `json:"RelatedProducts" gorm:"foreignKey:ParentID"`
}

type Address struct {
	Street string `json:"Street"`
	City   string `json:"City"`
}

type RelatedProduct struct {
	ID       uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string `json:"Name"`
	ParentID uint   `json:"ParentID"`
}

func setupPropertyHelperTest(t *testing.T) (*EntityHandler, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PropertyHelperTestEntity{}, &RelatedProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(PropertyHelperTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	relatedMeta, err := metadata.AnalyzeEntity(RelatedProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze related entity: %v", err)
	}

	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"PropertyHelperTestEntities": entityMeta,
		"RelatedProducts":            relatedMeta,
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	handler.SetEntitiesMetadata(entitiesMetadata)
	return handler, db
}

func TestEntityHandler_IsNavigationProperty(t *testing.T) {
	handler, _ := setupPropertyHelperTest(t)

	tests := []struct {
		name         string
		propertyName string
		want         bool
	}{
		{
			name:         "Valid navigation property",
			propertyName: "RelatedProducts",
			want:         true,
		},
		{
			name:         "Navigation property with key",
			propertyName: "RelatedProducts(1)",
			want:         true,
		},
		{
			name:         "Structural property",
			propertyName: "Name",
			want:         false,
		},
		{
			name:         "Non-existent property",
			propertyName: "NonExistent",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.IsNavigationProperty(tt.propertyName)
			if got != tt.want {
				t.Errorf("IsNavigationProperty(%s) = %v, want %v", tt.propertyName, got, tt.want)
			}
		})
	}
}

func TestEntityHandler_IsStructuralProperty(t *testing.T) {
	handler, _ := setupPropertyHelperTest(t)

	tests := []struct {
		name         string
		propertyName string
		want         bool
	}{
		{
			name:         "Valid structural property",
			propertyName: "Name",
			want:         true,
		},
		{
			name:         "Another structural property",
			propertyName: "StructuralProperty",
			want:         true,
		},
		{
			name:         "Navigation property",
			propertyName: "RelatedProducts",
			want:         false,
		},
		{
			name:         "Non-existent property",
			propertyName: "NonExistent",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.IsStructuralProperty(tt.propertyName)
			if got != tt.want {
				t.Errorf("IsStructuralProperty(%s) = %v, want %v", tt.propertyName, got, tt.want)
			}
		})
	}
}

func TestEntityHandler_IsComplexTypeProperty(t *testing.T) {
	handler, _ := setupPropertyHelperTest(t)

	tests := []struct {
		name         string
		propertyName string
		want         bool
	}{
		{
			name:         "Complex property",
			propertyName: "ComplexProp",
			want:         true,
		},
		{
			name:         "Structural property",
			propertyName: "Name",
			want:         false,
		},
		{
			name:         "Non-existent property",
			propertyName: "NonExistent",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.IsComplexTypeProperty(tt.propertyName)
			if got != tt.want {
				t.Errorf("IsComplexTypeProperty(%s) = %v, want %v", tt.propertyName, got, tt.want)
			}
		})
	}
}

func TestEntityHandler_IsStreamProperty(t *testing.T) {
	handler, _ := setupPropertyHelperTest(t)

	tests := []struct {
		name         string
		propertyName string
		want         bool
	}{
		{
			name:         "Stream property",
			propertyName: "StreamProp",
			want:         true,
		},
		{
			name:         "Structural property",
			propertyName: "Name",
			want:         false,
		},
		{
			name:         "Non-existent property",
			propertyName: "NonExistent",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.IsStreamProperty(tt.propertyName)
			if got != tt.want {
				t.Errorf("IsStreamProperty(%s) = %v, want %v", tt.propertyName, got, tt.want)
			}
		})
	}
}

func TestEntityHandler_NavigationTargetSet(t *testing.T) {
	handler, _ := setupPropertyHelperTest(t)

	tests := []struct {
		name         string
		propertyName string
		wantName     string
		wantOk       bool
	}{
		{
			name:         "Valid navigation property",
			propertyName: "RelatedProducts",
			wantName:     "RelatedProducts",
			wantOk:       true,
		},
		{
			name:         "Navigation property with key",
			propertyName: "RelatedProducts(1)",
			wantName:     "RelatedProducts",
			wantOk:       true,
		},
		{
			name:         "Non-navigation property",
			propertyName: "Name",
			wantName:     "",
			wantOk:       false,
		},
		{
			name:         "Non-existent property",
			propertyName: "NonExistent",
			wantName:     "",
			wantOk:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotOk := handler.NavigationTargetSet(tt.propertyName)
			if gotName != tt.wantName || gotOk != tt.wantOk {
				t.Errorf("NavigationTargetSet(%s) = (%s, %v), want (%s, %v)",
					tt.propertyName, gotName, gotOk, tt.wantName, tt.wantOk)
			}
		})
	}
}

func TestEntityHandler_writeMethodNotAllowedError(t *testing.T) {
	handler, _ := setupPropertyHelperTest(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)

	handler.writeMethodNotAllowedError(w, r, "POST", "structural properties")

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected error response body, got empty")
	}

	// Validate error message contains expected details
	if !strings.Contains(body, "POST") {
		t.Error("Expected error message to contain 'POST'")
	}
	if !strings.Contains(body, "structural properties") {
		t.Error("Expected error message to contain 'structural properties'")
	}
}

func TestEntityHandler_writePropertyNotFoundError(t *testing.T) {
	handler, _ := setupPropertyHelperTest(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test/NonExistentProperty", nil)

	handler.writePropertyNotFoundError(w, r, "NonExistentProperty")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected error response body, got empty")
	}

	// Validate error message contains expected details
	if !strings.Contains(body, "Property not found") {
		t.Error("Expected error message to contain 'Property not found'")
	}
	if !strings.Contains(body, "NonExistentProperty") {
		t.Error("Expected error message to contain 'NonExistentProperty'")
	}
}
