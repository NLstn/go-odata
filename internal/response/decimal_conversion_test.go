package response

import (
	"encoding/json"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/shopspring/decimal"
)

func TestConvertFieldValueDecimal(t *testing.T) {
	// Create mock metadata with Edm.Decimal property
	fullMetadata := &metadata.EntityMetadata{
		Properties: []metadata.PropertyMetadata{
			{
				FieldName: "Revenue",
				Name:      "Revenue",
				EdmType:   "Edm.Decimal",
			},
			{
				FieldName: "Status",
				Name:      "Status",
				EdmType:   "Edm.String",
			},
		},
	}

	t.Run("Converts decimal.Decimal to JSON number", func(t *testing.T) {
		value := decimal.NewFromFloat(1488.44)
		result := convertFieldValue(value, fullMetadata, "Revenue")

		// Should return json.RawMessage (raw number)
		rawMsg, ok := result.(json.RawMessage)
		if !ok {
			t.Fatalf("Expected json.RawMessage, got %T", result)
		}

		// Verify it's a valid number without quotes
		expected := "1488.44"
		if string(rawMsg) != expected {
			t.Errorf("Expected %q, got %q", expected, string(rawMsg))
		}

		// Verify it can be unmarshaled as a number
		var num float64
		err := json.Unmarshal(rawMsg, &num)
		if err != nil {
			t.Errorf("Failed to unmarshal as number: %v", err)
		}
		if num != 1488.44 {
			t.Errorf("Expected 1488.44, got %f", num)
		}
	})

	t.Run("Handles pointer to decimal.Decimal", func(t *testing.T) {
		value := decimal.NewFromFloat(2500.50)
		ptrValue := &value
		result := convertFieldValue(ptrValue, fullMetadata, "Revenue")

		// Pointer to decimal.Decimal also implements json.Marshaler
		rawMsg, ok := result.(json.RawMessage)
		if !ok {
			t.Fatalf("Expected json.RawMessage, got %T", result)
		}

		expected := "2500.5"
		if string(rawMsg) != expected {
			t.Errorf("Expected %q, got %q", expected, string(rawMsg))
		}
	})

	t.Run("Preserves negative decimals", func(t *testing.T) {
		value := decimal.NewFromFloat(-123.45)
		result := convertFieldValue(value, fullMetadata, "Revenue")

		rawMsg := result.(json.RawMessage)
		expected := "-123.45"
		if string(rawMsg) != expected {
			t.Errorf("Expected %q, got %q", expected, string(rawMsg))
		}
	})

	t.Run("Preserves zero", func(t *testing.T) {
		value := decimal.Zero
		result := convertFieldValue(value, fullMetadata, "Revenue")

		rawMsg := result.(json.RawMessage)
		expected := "0"
		if string(rawMsg) != expected {
			t.Errorf("Expected %q, got %q", expected, string(rawMsg))
		}
	})

	t.Run("Does not convert non-Edm.Decimal fields", func(t *testing.T) {
		value := "test string"
		result := convertFieldValue(value, fullMetadata, "Status")

		// Should return original value unchanged
		strValue, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}
		if strValue != "test string" {
			t.Errorf("Expected 'test string', got %q", strValue)
		}
	})

	t.Run("Returns original value if no metadata", func(t *testing.T) {
		value := decimal.NewFromFloat(100.50)
		result := convertFieldValue(value, nil, "Revenue")

		// Should return original decimal.Decimal
		decValue, ok := result.(decimal.Decimal)
		if !ok {
			t.Fatalf("Expected decimal.Decimal, got %T", result)
		}
		if !decValue.Equal(value) {
			t.Errorf("Expected %s, got %s", value.String(), decValue.String())
		}
	})

	t.Run("Returns original value if field not found", func(t *testing.T) {
		value := decimal.NewFromFloat(100.50)
		result := convertFieldValue(value, fullMetadata, "UnknownField")

		// Should return original decimal.Decimal
		decValue, ok := result.(decimal.Decimal)
		if !ok {
			t.Fatalf("Expected decimal.Decimal, got %T", result)
		}
		if !decValue.Equal(value) {
			t.Errorf("Expected %s, got %s", value.String(), decValue.String())
		}
	})

	t.Run("Handles large precision decimals", func(t *testing.T) {
		value := decimal.RequireFromString("123456789.12345678")
		result := convertFieldValue(value, fullMetadata, "Revenue")

		rawMsg := result.(json.RawMessage)
		expected := "123456789.12345678"
		if string(rawMsg) != expected {
			t.Errorf("Expected %q, got %q", expected, string(rawMsg))
		}
	})
}
