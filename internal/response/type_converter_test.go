package response

import (
	"encoding/json"
	"testing"
	"time"
)

// Test basic time.Time to Edm.Date conversion
func TestConvertToEdmDate(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "basic date conversion",
			input: time.Date(2026, 1, 24, 15, 30, 0, 0, time.UTC),
			want:  "2026-01-24",
		},
		{
			name:  "date with timezone offset extracts UTC date",
			input: time.Date(2026, 1, 24, 23, 30, 0, 0, time.FixedZone("EST", -5*3600)),
			want:  "2026-01-25", // Next day in UTC
		},
		{
			name:    "wrong type returns error",
			input:   "2026-01-24",
			wantErr: true,
		},
		{
			name:  "empty string returns nil",
			input: "",
			want:  "", // Empty want means we expect nil
		},
		{
			name:  "zero time returns nil",
			input: time.Time{},
			want:  "", // Empty want means we expect nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToEdmType(tt.input, "Edm.Date")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Handle nil result for empty string/zero time tests
			if tt.want == "" {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else if result != tt.want {
				t.Errorf("expected %s, got %s", tt.want, result)
			}
		})
	}
}

// Test time.Time to Edm.TimeOfDay conversion
func TestConvertToEdmTimeOfDay(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "basic time conversion",
			input: time.Date(2026, 1, 24, 15, 30, 45, 123456000, time.UTC),
			want:  "15:30:45.123456",
		},
		{
			name:  "midnight",
			input: time.Date(2026, 1, 24, 0, 0, 0, 0, time.UTC),
			want:  "00:00:00.000000",
		},
		{
			name:    "wrong type returns error",
			input:   "15:30:00",
			wantErr: true,
		},
		{
			name:  "empty string returns nil",
			input: "",
			want:  "",
		},
		{
			name:  "zero time returns nil",
			input: time.Time{},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToEdmType(tt.input, "Edm.TimeOfDay")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Handle nil result for empty string/zero time tests
			if tt.want == "" {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else if result != tt.want {
				t.Errorf("expected %s, got %s", tt.want, result)
			}
		})
	}
}

// Test time.Duration to Edm.Duration conversion
func TestConvertToEdmDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "2 hours",
			input: 2 * time.Hour,
			want:  "PT2H",
		},
		{
			name:  "2 hours 30 minutes",
			input: 2*time.Hour + 30*time.Minute,
			want:  "PT2H30M",
		},
		{
			name:  "1 hour 15 minutes 30 seconds",
			input: time.Hour + 15*time.Minute + 30*time.Second,
			want:  "PT1H15M30S",
		},
		{
			name:    "wrong type returns error",
			input:   "PT2H",
			wantErr: true,
		},
		{
			name:  "empty string returns nil",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToEdmType(tt.input, "Edm.Duration")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Handle nil result for empty string tests
			if tt.want == "" {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else if result != tt.want {
				t.Errorf("expected %s, got %s", tt.want, result)
			}
		})
	}
}

// Test decimal with interface implementation
type TestDecimal struct {
	value string
}

func (d TestDecimal) EdmDecimalString() string {
	return d.value
}

func TestConvertToEdmDecimal_WithInterface(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "decimal with interface",
			input: TestDecimal{value: "123.45"},
			want:  "123.45",
		},
		{
			name:  "decimal with high precision",
			input: TestDecimal{value: "999999999999.123456789"},
			want:  "999999999999.123456789",
		},
		{
			name:  "negative decimal",
			input: TestDecimal{value: "-1842.53"},
			want:  "-1842.53",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToEdmType(tt.input, "Edm.Decimal")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Result should be json.Number
			num, ok := result.(json.Number)
			if !ok {
				t.Fatalf("expected json.Number, got %T", result)
			}

			if num.String() != tt.want {
				t.Errorf("expected %s, got %s", tt.want, num.String())
			}
		})
	}
}

// Test decimal with primitive types
func TestConvertToEdmDecimal_Primitive(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:  "float64",
			input: float64(123.45),
		},
		{
			name:  "float32",
			input: float32(123.45),
		},
		{
			name:  "int",
			input: int(123),
		},
		{
			name:  "int64",
			input: int64(123),
		},
		{
			name:  "uint",
			input: uint(123),
		},
		{
			name:  "empty string returns nil",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToEdmType(tt.input, "Edm.Decimal")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Handle empty string returning nil
			if tt.input == "" {
				if result != nil {
					t.Errorf("expected nil for empty string, got %v", result)
				}
			} else if result != tt.input {
				t.Errorf("expected %v, got %v", tt.input, result)
			}
		})
	}
}

// Test error cases
func TestConvertToEdmDecimal_UnsupportedType(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "string",
			input: "not a number",
		},
		{
			name:  "bool",
			input: true,
		},
		{
			name:  "slice",
			input: []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ConvertToEdmType(tt.input, "Edm.Decimal")

			if err == nil {
				t.Fatal("expected error for unsupported type")
			}
		})
	}
}

// Test nil handling
func TestConvertToEdmType_Nil(t *testing.T) {
	result, err := ConvertToEdmType(nil, "Edm.Date")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// Test unknown EDM type (should return as-is)
func TestConvertToEdmType_UnknownType(t *testing.T) {
	input := "some value"
	result, err := ConvertToEdmType(input, "Edm.Unknown")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != input {
		t.Errorf("expected %v, got %v", input, result)
	}
}

// TestEmptyStringHandling tests the specific Excel/OData client scenario
// where empty strings are provided instead of null for typed fields
func TestEmptyStringHandling(t *testing.T) {
	testCases := []struct {
		name    string
		edmType string
		input   interface{}
		wantNil bool
	}{
		{
			name:    "Edm.Date with empty string",
			edmType: "Edm.Date",
			input:   "",
			wantNil: true,
		},
		{
			name:    "Edm.TimeOfDay with empty string",
			edmType: "Edm.TimeOfDay",
			input:   "",
			wantNil: true,
		},
		{
			name:    "Edm.Duration with empty string",
			edmType: "Edm.Duration",
			input:   "",
			wantNil: true,
		},
		{
			name:    "Edm.Decimal with empty string",
			edmType: "Edm.Decimal",
			input:   "",
			wantNil: true,
		},
		{
			name:    "Edm.Date with zero time.Time",
			edmType: "Edm.Date",
			input:   time.Time{},
			wantNil: true,
		},
		{
			name:    "Edm.TimeOfDay with zero time.Time",
			edmType: "Edm.TimeOfDay",
			input:   time.Time{},
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ConvertToEdmType(tc.input, tc.edmType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantNil {
				if result != nil {
					t.Errorf("expected nil for %s with input %v, got %v", tc.edmType, tc.input, result)
				}
			} else {
				if result == nil {
					t.Errorf("expected non-nil result, got nil")
				}
			}
		})
	}
}
