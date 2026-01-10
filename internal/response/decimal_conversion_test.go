package response

import (
	"encoding/json"
	"testing"
	"time"

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

func TestConvertFieldValueDate(t *testing.T) {
	// Create mock metadata with Edm.Date property
	fullMetadata := &metadata.EntityMetadata{
		Properties: []metadata.PropertyMetadata{
			{
				FieldName: "PickupDate",
				Name:      "PickupDate",
				EdmType:   "Edm.Date",
			},
			{
				FieldName: "DeliveryDate",
				Name:      "DeliveryDate",
				EdmType:   "Edm.Date",
			},
			{
				FieldName: "CreatedAt",
				Name:      "CreatedAt",
				EdmType:   "Edm.DateTimeOffset",
			},
		},
	}

	t.Run("Converts time.Time to date-only string", func(t *testing.T) {
		value := time.Date(2024, 1, 10, 15, 30, 45, 0, time.UTC)
		result := convertFieldValue(value, fullMetadata, "PickupDate")

		// Should return date-only string
		dateStr, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}

		expected := "2024-01-10"
		if dateStr != expected {
			t.Errorf("Expected %q, got %q", expected, dateStr)
		}
	})

	t.Run("Handles pointer to time.Time", func(t *testing.T) {
		value := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)
		ptrValue := &value
		result := convertFieldValue(ptrValue, fullMetadata, "DeliveryDate")

		dateStr, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string, got %T", result)
		}

		expected := "2024-12-25"
		if dateStr != expected {
			t.Errorf("Expected %q, got %q", expected, dateStr)
		}
	})

	t.Run("Handles nil pointer to time.Time", func(t *testing.T) {
		var ptrValue *time.Time = nil
		result := convertFieldValue(ptrValue, fullMetadata, "DeliveryDate")

		// Should return original nil pointer - result will be the pointer value (nil), not nil interface
		_, ok := result.(*time.Time)
		if !ok {
			t.Errorf("Expected *time.Time, got %T", result)
		}
	})

	t.Run("Handles zero time.Time", func(t *testing.T) {
		value := time.Time{}
		result := convertFieldValue(value, fullMetadata, "PickupDate")

		// Should return original zero value
		timeVal, ok := result.(time.Time)
		if !ok {
			t.Fatalf("Expected time.Time, got %T", result)
		}
		if !timeVal.IsZero() {
			t.Errorf("Expected zero time, got %v", timeVal)
		}
	})

	t.Run("Does not convert Edm.DateTimeOffset fields", func(t *testing.T) {
		value := time.Date(2024, 1, 10, 15, 30, 45, 0, time.UTC)
		result := convertFieldValue(value, fullMetadata, "CreatedAt")

		// Should return original time.Time unchanged
		timeVal, ok := result.(time.Time)
		if !ok {
			t.Fatalf("Expected time.Time, got %T", result)
		}
		if !timeVal.Equal(value) {
			t.Errorf("Expected %v, got %v", value, timeVal)
		}
	})

	t.Run("Formats date in different time zones correctly", func(t *testing.T) {
		// Create time in EST (UTC-5)
		est, _ := time.LoadLocation("America/New_York")
		value := time.Date(2024, 1, 10, 23, 30, 0, 0, est)
		result := convertFieldValue(value, fullMetadata, "PickupDate")

		dateStr := result.(string)
		// Should use the date from the given timezone
		expected := "2024-01-10"
		if dateStr != expected {
			t.Errorf("Expected %q, got %q", expected, dateStr)
		}
	})

	t.Run("Returns original value if field not found", func(t *testing.T) {
		value := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
		result := convertFieldValue(value, fullMetadata, "UnknownField")

		// Should return original time.Time
		timeVal, ok := result.(time.Time)
		if !ok {
			t.Fatalf("Expected time.Time, got %T", result)
		}
		if !timeVal.Equal(value) {
			t.Errorf("Expected %v, got %v", value, timeVal)
		}
	})
}
