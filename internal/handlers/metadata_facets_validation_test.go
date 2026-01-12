package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestPrecisionScaleOnlyForDecimal verifies that Precision and Scale attributes
// are only emitted for Edm.Decimal types, not for Edm.Double or other types.
// This follows OData CSDL specification requirements.
func TestPrecisionScaleOnlyForDecimal(t *testing.T) {
	tests := []struct {
		name                string
		entityDef           interface{}
		shouldHavePrecision bool
		shouldHaveScale     bool
		expectedType        string
		propertyName        string
	}{
		{
			name: "float64 with precision/scale should NOT emit them (Edm.Double)",
			entityDef: struct {
				ID    int     `json:"id" odata:"key"`
				Price float64 `json:"price" odata:"precision=10,scale=2"`
			}{},
			shouldHavePrecision: false,
			shouldHaveScale:     false,
			expectedType:        "Edm.Double",
			propertyName:        "price",
		},
		{
			name: "float32 with precision/scale should NOT emit them (Edm.Single)",
			entityDef: struct {
				ID     int     `json:"id" odata:"key"`
				Amount float32 `json:"amount" odata:"precision=8,scale=2"`
			}{},
			shouldHavePrecision: false,
			shouldHaveScale:     false,
			expectedType:        "Edm.Single",
			propertyName:        "amount",
		},
		{
			name: "string with maxLength SHOULD emit it",
			entityDef: struct {
				ID   int    `json:"id" odata:"key"`
				Name string `json:"name" odata:"maxlength=100"`
			}{},
			shouldHavePrecision: false,
			shouldHaveScale:     false,
			expectedType:        "Edm.String",
			propertyName:        "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := make(map[string]*metadata.EntityMetadata)
			entityMeta, err := metadata.AnalyzeEntity(tt.entityDef)
			if err != nil {
				t.Fatalf("Failed to analyze entity: %v", err)
			}
			entities["TestEntity"] = entityMeta

			handler := NewMetadataHandler(entities)
			req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
			w := httptest.NewRecorder()

			handler.HandleMetadata(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			body := w.Body.String()

			// Find the property definition
			propStart := strings.Index(body, `Name="`+tt.propertyName+`"`)
			if propStart == -1 {
				t.Fatalf("Property %s not found in metadata", tt.propertyName)
			}

			// Extract the property line (up to the closing />)
			propEnd := strings.Index(body[propStart:], "/>")
			if propEnd == -1 {
				t.Fatalf("Property closing tag not found")
			}
			propertyLine := body[propStart : propStart+propEnd]

			// Verify expected type
			if !strings.Contains(propertyLine, `Type="`+tt.expectedType+`"`) {
				t.Errorf("Expected Type=%s in: %s", tt.expectedType, propertyLine)
			}

			// Verify Precision presence
			hasPrecision := strings.Contains(propertyLine, `Precision=`)
			if hasPrecision != tt.shouldHavePrecision {
				t.Errorf("Precision attribute presence = %v, want %v\nProperty line: %s",
					hasPrecision, tt.shouldHavePrecision, propertyLine)
			}

			// Verify Scale presence
			hasScale := strings.Contains(propertyLine, `Scale=`)
			if hasScale != tt.shouldHaveScale {
				t.Errorf("Scale attribute presence = %v, want %v\nProperty line: %s",
					hasScale, tt.shouldHaveScale, propertyLine)
			}

			// Additional check: MaxLength should be present for string with maxLength
			if tt.propertyName == "name" {
				if !strings.Contains(propertyLine, `MaxLength="100"`) {
					t.Errorf("Expected MaxLength=100 for string property, got: %s", propertyLine)
				}
			}
		})
	}
}

// TestDecimalTypeShouldHavePrecisionScale verifies that when using proper decimal type,
// Precision and Scale ARE emitted. This test documents the expected behavior when
// using a type that maps to Edm.Decimal (currently not implemented, but planned).
func TestDecimalTypeShouldHavePrecisionScale(t *testing.T) {
	t.Skip("Skipping until Edm.Decimal type mapping is implemented in metadata generation")

	// TODO: When edm.Decimal or decimal.Decimal is properly mapped to Edm.Decimal,
	// this test should verify that Precision and Scale ARE included in metadata.
	//
	// Expected behavior:
	// type Order struct {
	//     ID     int             `json:"id" odata:"key"`
	//     Total  decimal.Decimal `json:"total" odata:"precision=18,scale=4"`
	// }
	//
	// Should generate:
	// <Property Name="total" Type="Edm.Decimal" Nullable="false" Precision="18" Scale="4" />
}
