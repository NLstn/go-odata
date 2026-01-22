package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/shopspring/decimal"
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

// TestDecimalTypeShouldHavePrecisionScale verifies that when using decimal.Decimal type,
// Precision and Scale ARE emitted in the metadata. This test verifies the expected behavior
// when using a type that maps to Edm.Decimal.
func TestDecimalTypeShouldHavePrecisionScale(t *testing.T) {
	// Define entity with decimal.Decimal type
	entityDef := struct {
		ID    int             `json:"id" odata:"key"`
		Total decimal.Decimal `json:"total" odata:"precision=18,scale=4"`
	}{}

	entities := make(map[string]*metadata.EntityMetadata)
	entityMeta, err := metadata.AnalyzeEntity(entityDef)
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
	propStart := strings.Index(body, `Name="total"`)
	if propStart == -1 {
		t.Fatalf("Property 'total' not found in metadata")
	}

	// Extract the property line (up to the closing />)
	propEnd := strings.Index(body[propStart:], "/>")
	if propEnd == -1 {
		t.Fatalf("Property closing tag not found")
	}
	propertyLine := body[propStart : propStart+propEnd]

	t.Logf("Property line: %s", propertyLine)

	// Verify it's Edm.Decimal type
	if !strings.Contains(propertyLine, `Type="Edm.Decimal"`) {
		t.Errorf("Expected Type=Edm.Decimal in: %s", propertyLine)
	}

	// Verify Precision is present
	if !strings.Contains(propertyLine, `Precision="18"`) {
		t.Errorf("Expected Precision=18 for Edm.Decimal, got: %s", propertyLine)
	}

	// Verify Scale is present
	if !strings.Contains(propertyLine, `Scale="4"`) {
		t.Errorf("Expected Scale=4 for Edm.Decimal, got: %s", propertyLine)
	}
}

// TestDecimalTypeJSONMetadata verifies that decimal.Decimal type
// emits Precision and Scale in JSON metadata format as well.
func TestDecimalTypeJSONMetadata(t *testing.T) {
	// Define entity with decimal.Decimal type
	entityDef := struct {
		ID     int             `json:"id" odata:"key"`
		Amount decimal.Decimal `json:"amount" odata:"precision=10,scale=2"`
	}{}

	entities := make(map[string]*metadata.EntityMetadata)
	entityMeta, err := metadata.AnalyzeEntity(entityDef)
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	entities["TestEntity"] = entityMeta

	handler := NewMetadataHandler(entities)
	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.HandleMetadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	t.Logf("JSON Metadata: %s", body)

	// Verify the JSON contains the decimal type with precision and scale
	// Note: JSON may have whitespace differences
	if !strings.Contains(body, `"$Type": "Edm.Decimal"`) && !strings.Contains(body, `"$Type":"Edm.Decimal"`) {
		t.Errorf("Expected $Type:Edm.Decimal in JSON metadata, got: %s", body)
	}

	if !strings.Contains(body, `"$Precision": 10`) && !strings.Contains(body, `"$Precision":10`) {
		t.Errorf("Expected $Precision:10 for Edm.Decimal in JSON metadata, got: %s", body)
	}

	if !strings.Contains(body, `"$Scale": 2`) && !strings.Contains(body, `"$Scale":2`) {
		t.Errorf("Expected $Scale:2 for Edm.Decimal in JSON metadata, got: %s", body)
	}
}
