package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestMetadataWithExplicitTypes verifies that explicit type tags are properly rendered in metadata
func TestMetadataWithExplicitTypes(t *testing.T) {
	type EntityWithExplicitTypes struct {
		ID       int     `json:"id" odata:"key"`
		Revenue  float64 `json:"revenue" odata:"type=Edm.Decimal,precision=18,scale=4"`
		Discount float64 `json:"discount" odata:"type=Edm.Double"`
	}

	meta, err := metadata.AnalyzeEntity(EntityWithExplicitTypes{})
	if err != nil {
		t.Fatalf("AnalyzeEntity() error = %v", err)
	}

	entities := map[string]*metadata.EntityMetadata{
		"EntityWithExplicitTypes": meta,
	}

	handler := NewMetadataHandler(entities)

	t.Run("XML format with explicit types", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()

		// Verify Revenue is Edm.Decimal with Precision and Scale
		if !strings.Contains(body, `<Property Name="revenue" Type="Edm.Decimal"`) {
			t.Error("Expected revenue to be Type=\"Edm.Decimal\"")
		}
		if !strings.Contains(body, `Precision="18"`) {
			t.Error("Expected revenue to have Precision=\"18\"")
		}
		if !strings.Contains(body, `Scale="4"`) {
			t.Error("Expected revenue to have Scale=\"4\"")
		}

		// Verify Discount is Edm.Double (not Edm.Decimal)
		if !strings.Contains(body, `<Property Name="discount" Type="Edm.Double"`) {
			t.Error("Expected discount to be Type=\"Edm.Double\"")
		}
	})

	t.Run("JSON format with explicit types", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
		w := httptest.NewRecorder()

		handler.HandleMetadata(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()

		// Verify Revenue is Edm.Decimal with $Precision and $Scale
		if !strings.Contains(body, `"revenue"`) {
			t.Error("Expected revenue property in JSON")
		}
		// Check for Edm.Decimal (with flexible whitespace)
		if !strings.Contains(body, `"$Type": "Edm.Decimal"`) && !strings.Contains(body, `"$Type":"Edm.Decimal"`) {
			t.Error("Expected revenue to have $Type: Edm.Decimal")
		}
		// Check for Precision and Scale
		if !strings.Contains(body, `"$Precision": 18`) && !strings.Contains(body, `"$Precision":18`) {
			t.Error("Expected revenue to have $Precision: 18")
		}
		if !strings.Contains(body, `"$Scale": 4`) && !strings.Contains(body, `"$Scale":4`) {
			t.Error("Expected revenue to have $Scale: 4")
		}

		// Verify Discount is Edm.Double
		if !strings.Contains(body, `"discount"`) {
			t.Error("Expected discount property in JSON")
		}
		if !strings.Contains(body, `"$Type": "Edm.Double"`) && !strings.Contains(body, `"$Type":"Edm.Double"`) {
			t.Error("Expected discount to have $Type: Edm.Double")
		}
	})
}
