package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/version"
	"gorm.io/gorm"
)

func TestSetODataHeader(t *testing.T) {
	w := httptest.NewRecorder()

	SetODataHeader(w, "OData-Version", "4.01")

	// Verify header was set - use direct map access since OData headers have non-canonical keys
	// OData spec requires exact "OData-Version" capitalization which is non-canonical in Go
	header := w.Header()
	values := header["OData-Version"] //nolint:staticcheck // OData headers require non-canonical keys
	if len(values) == 0 || values[0] != "4.01" {
		t.Errorf("Expected OData-Version header to be 4.01, got %v", values)
	}
}

func TestSetODataVersionHeaderForRequest(t *testing.T) {
	tests := []struct {
		name            string
		maxVersion      string
		expectedVersion string
	}{
		{
			name:            "No MaxVersion header",
			maxVersion:      "",
			expectedVersion: "4.01",
		},
		{
			name:            "MaxVersion 4.0",
			maxVersion:      "4.0",
			expectedVersion: "4.0",
		},
		{
			name:            "MaxVersion 4.01",
			maxVersion:      "4.01",
			expectedVersion: "4.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.maxVersion != "" {
				req.Header.Set("OData-MaxVersion", tt.maxVersion)
			}

			// Negotiate version and store in context
			negotiatedVersion := version.NegotiateVersion(tt.maxVersion)
			req = req.WithContext(version.WithVersion(req.Context(), negotiatedVersion))
			response.SetODataVersionHeaderFromRequest(w, req)

			// Check for the header - use Get() since Set() was used
			value := w.Header().Get("OData-Version")
			if value == "" {
				t.Error("Expected OData-Version header to be set")
				return
			}
			if value != tt.expectedVersion {
				t.Errorf("Expected OData-Version %s, got %s", tt.expectedVersion, value)
			}
		})
	}
}

func TestGetNegotiatedODataVersion(t *testing.T) {
	tests := []struct {
		name            string
		maxVersion      string
		expectedVersion string
	}{
		{
			name:            "No MaxVersion header",
			maxVersion:      "",
			expectedVersion: "4.01",
		},
		{
			name:            "MaxVersion 4.0",
			maxVersion:      "4.0",
			expectedVersion: "4.0",
		},
		{
			name:            "MaxVersion 4.01",
			maxVersion:      "4.01",
			expectedVersion: "4.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.maxVersion != "" {
				req.Header.Set("OData-MaxVersion", tt.maxVersion)
			}

			negotiatedVersion := version.NegotiateVersion(tt.maxVersion)
			versionStr := negotiatedVersion.String()
			if versionStr != tt.expectedVersion {
				t.Errorf("version.NegotiateVersion() = %s, want %s", versionStr, tt.expectedVersion)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ProductID", "product_id"},
		{"productId", "product_id"},
		{"XMLParser", "xml_parser"},
		{"SimpleTest", "simple_test"},
		{"ID", "id"},
		{"Test", "test"},
		{"test", "test"},
		{"TestXMLParser", "test_xml_parser"},
		{"HTTPResponse", "http_response"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseCompositeKey(t *testing.T) {
	tests := []struct {
		name       string
		keyPart    string
		wantKeyMap map[string]string
		wantErr    bool
	}{
		{
			name:       "Simple key",
			keyPart:    "123",
			wantKeyMap: nil,
			wantErr:    true, // Not a composite key
		},
		{
			name:    "Single composite key",
			keyPart: "ID=123",
			wantKeyMap: map[string]string{
				"ID": "123",
			},
			wantErr: false,
		},
		{
			name:    "Multiple composite keys",
			keyPart: "ID=123,Name=test",
			wantKeyMap: map[string]string{
				"ID":   "123",
				"Name": "test",
			},
			wantErr: false,
		},
		{
			name:    "Quoted string value",
			keyPart: "ID=123,Name='test value'",
			wantKeyMap: map[string]string{
				"ID":   "123",
				"Name": "test value",
			},
			wantErr: false,
		},
		{
			name:    "Double quoted value",
			keyPart: "ID=123,Name=\"test value\"",
			wantKeyMap: map[string]string{
				"ID":   "123",
				"Name": "test value",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := &response.ODataURLComponents{
				EntityKeyMap: make(map[string]string),
			}

			err := parseCompositeKey(tt.keyPart, components)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCompositeKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantKeyMap != nil {
				for k, v := range tt.wantKeyMap {
					if components.EntityKeyMap[k] != v {
						t.Errorf("parseCompositeKey() key %q = %q, want %q", k, components.EntityKeyMap[k], v)
					}
				}
			}
		})
	}
}

func TestEntityHandlerHandleFetchError(t *testing.T) {
	entitySetName := "Products"
	entityKey := "42"
	handler := &EntityHandler{
		metadata: &metadata.EntityMetadata{
			EntitySetName: entitySetName,
			KeyProperties: []metadata.PropertyMetadata{
				{
					Name:       "ID",
					JsonName:   "ID",
					ColumnName: "id",
					IsKey:      true,
				},
			},
		},
	}

	decodeError := func(t *testing.T, recorder *httptest.ResponseRecorder) response.ODataError {
		t.Helper()

		var payload struct {
			Error response.ODataError `json:"error"`
		}

		if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
			t.Fatalf("decode error response: %v", err)
		}

		return payload.Error
	}

	t.Run("record not found", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/Products(42)", nil)

		handler.handleFetchError(recorder, req, gorm.ErrRecordNotFound, entityKey)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
		}

		odataErr := decodeError(t, recorder)
		if odataErr.Code != fmt.Sprintf("%d", http.StatusNotFound) {
			t.Fatalf("expected error code %d, got %s", http.StatusNotFound, odataErr.Code)
		}
		if odataErr.Message != ErrMsgEntityNotFound {
			t.Fatalf("expected error message %q, got %q", ErrMsgEntityNotFound, odataErr.Message)
		}

		expectedTarget := fmt.Sprintf(ODataEntityKeyFormat, entitySetName, entityKey)
		if !strings.Contains(odataErr.Target, expectedTarget) {
			t.Fatalf("expected target %q to include %q", odataErr.Target, expectedTarget)
		}

		if len(odataErr.Details) != 1 {
			t.Fatalf("expected 1 detail message, got %d", len(odataErr.Details))
		}
		expectedDetail := fmt.Sprintf(EntityKeyNotExistFmt, entityKey)
		if odataErr.Details[0].Message != expectedDetail {
			t.Fatalf("expected detail message %q, got %q", expectedDetail, odataErr.Details[0].Message)
		}
	})

	t.Run("database error", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/Products(42)", nil)
		fetchErr := errors.New("database unavailable")

		handler.handleFetchError(recorder, req, fetchErr, entityKey)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
		}

		odataErr := decodeError(t, recorder)
		if odataErr.Code != fmt.Sprintf("%d", http.StatusInternalServerError) {
			t.Fatalf("expected error code %d, got %s", http.StatusInternalServerError, odataErr.Code)
		}
		if odataErr.Message != ErrMsgDatabaseError {
			t.Fatalf("expected error message %q, got %q", ErrMsgDatabaseError, odataErr.Message)
		}
		if len(odataErr.Details) != 1 {
			t.Fatalf("expected 1 detail message, got %d", len(odataErr.Details))
		}
		if odataErr.Details[0].Message != fetchErr.Error() {
			t.Fatalf("expected detail message %q, got %q", fetchErr.Error(), odataErr.Details[0].Message)
		}
	})
}

func TestParseEntityKeyValues(t *testing.T) {
	tests := []struct {
		name          string
		entityKey     string
		keyProperties []metadata.PropertyMetadata
		expected      map[string]interface{}
	}{
		{
			name:      "Empty entity key",
			entityKey: "",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "ID", Name: "ID", Type: reflect.TypeOf(int(0))},
			},
			expected: nil,
		},
		{
			name:      "Single numeric key",
			entityKey: "42",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "ID", Name: "ID", Type: reflect.TypeOf(int64(0))},
			},
			expected: map[string]interface{}{
				"ID": int64(42),
			},
		},
		{
			name:      "Single string key",
			entityKey: "abc123",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "Code", Name: "Code", Type: reflect.TypeOf(string(""))},
			},
			expected: map[string]interface{}{
				"Code": "abc123",
			},
		},
		{
			name:      "Composite key with two integers",
			entityKey: "OrderID=1,ProductID=5",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "OrderID", Name: "OrderID", Type: reflect.TypeOf(int64(0))},
				{JsonName: "ProductID", Name: "ProductID", Type: reflect.TypeOf(int64(0))},
			},
			expected: map[string]interface{}{
				"OrderID":   int64(1),
				"ProductID": int64(5),
			},
		},
		{
			name:      "Composite key with integer and string",
			entityKey: "ProductID=1,LanguageKey='EN'",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "ProductID", Name: "ProductID", Type: reflect.TypeOf(uint(0))},
				{JsonName: "LanguageKey", Name: "LanguageKey", Type: reflect.TypeOf(string(""))},
			},
			expected: map[string]interface{}{
				"ProductID":   uint(1),
				"LanguageKey": "EN",
			},
		},
		{
			name:      "Composite key with quoted strings",
			entityKey: "FirstName='John',LastName='Doe'",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "FirstName", Name: "FirstName", Type: reflect.TypeOf(string(""))},
				{JsonName: "LastName", Name: "LastName", Type: reflect.TypeOf(string(""))},
			},
			expected: map[string]interface{}{
				"FirstName": "John",
				"LastName":  "Doe",
			},
		},
		{
			name:      "Single key with equals sign (treated as composite)",
			entityKey: "ID=123",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "ID", Name: "ID", Type: reflect.TypeOf(int(0))},
			},
			expected: map[string]interface{}{
				"ID": int(123),
			},
		},
		{
			name:          "Non-empty entity key with nil keyProperties",
			entityKey:     "42",
			keyProperties: nil,
			expected:      map[string]interface{}{},
		},
		{
			name:          "Non-empty entity key with empty keyProperties",
			entityKey:     "123",
			keyProperties: []metadata.PropertyMetadata{},
			expected:      map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEntityKeyValues(tt.entityKey, tt.keyProperties)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Expected non-nil result, got nil")
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d key-value pairs, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("Expected key %s not found in result", key)
					continue
				}
				if actualValue != expectedValue {
					t.Errorf("For key %s: expected %v (%T), got %v (%T)",
						key, expectedValue, expectedValue, actualValue, actualValue)
				}
			}
		})
	}
}

func TestConvertKeyValue(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		keyName       string
		keyProperties []metadata.PropertyMetadata
		expected      interface{}
	}{
		{
			name:    "Convert to int",
			value:   "42",
			keyName: "ID",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "ID", Name: "ID", Type: reflect.TypeOf(int(0))},
			},
			expected: int(42),
		},
		{
			name:    "Convert to int8",
			value:   "42",
			keyName: "ID",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "ID", Name: "ID", Type: reflect.TypeOf(int8(0))},
			},
			expected: int8(42),
		},
		{
			name:    "Convert to int16",
			value:   "100",
			keyName: "ID",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "ID", Name: "ID", Type: reflect.TypeOf(int16(0))},
			},
			expected: int16(100),
		},
		{
			name:    "Convert to int32",
			value:   "1000",
			keyName: "ID",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "ID", Name: "ID", Type: reflect.TypeOf(int32(0))},
			},
			expected: int32(1000),
		},
		{
			name:    "Convert to int64",
			value:   "42",
			keyName: "ID",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "ID", Name: "ID", Type: reflect.TypeOf(int64(0))},
			},
			expected: int64(42),
		},
		{
			name:    "Convert to uint",
			value:   "50",
			keyName: "Count",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "Count", Name: "Count", Type: reflect.TypeOf(uint(0))},
			},
			expected: uint(50),
		},
		{
			name:    "Convert to uint8",
			value:   "255",
			keyName: "Count",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "Count", Name: "Count", Type: reflect.TypeOf(uint8(0))},
			},
			expected: uint8(255),
		},
		{
			name:    "Convert to uint16",
			value:   "1000",
			keyName: "Count",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "Count", Name: "Count", Type: reflect.TypeOf(uint16(0))},
			},
			expected: uint16(1000),
		},
		{
			name:    "Convert to uint32",
			value:   "50000",
			keyName: "Count",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "Count", Name: "Count", Type: reflect.TypeOf(uint32(0))},
			},
			expected: uint32(50000),
		},
		{
			name:    "Convert to uint64",
			value:   "100",
			keyName: "Count",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "Count", Name: "Count", Type: reflect.TypeOf(uint64(0))},
			},
			expected: uint64(100),
		},
		{
			name:    "Convert to float32",
			value:   "3.14",
			keyName: "Price",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "Price", Name: "Price", Type: reflect.TypeOf(float32(0))},
			},
			expected: float32(3.14),
		},
		{
			name:    "Convert to float64",
			value:   "3.14",
			keyName: "Price",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "Price", Name: "Price", Type: reflect.TypeOf(float64(0))},
			},
			expected: float64(3.14),
		},
		{
			name:    "Convert to bool",
			value:   "true",
			keyName: "IsActive",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "IsActive", Name: "IsActive", Type: reflect.TypeOf(bool(false))},
			},
			expected: true,
		},
		{
			name:    "Keep as string",
			value:   "abc123",
			keyName: "Code",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "Code", Name: "Code", Type: reflect.TypeOf(string(""))},
			},
			expected: "abc123",
		},
		{
			name:    "Invalid int - return string",
			value:   "not-a-number",
			keyName: "ID",
			keyProperties: []metadata.PropertyMetadata{
				{JsonName: "ID", Name: "ID", Type: reflect.TypeOf(int64(0))},
			},
			expected: "not-a-number",
		},
		{
			name:          "Unknown type - return string",
			value:         "value",
			keyName:       "Unknown",
			keyProperties: []metadata.PropertyMetadata{},
			expected:      "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertKeyValue(tt.value, tt.keyName, tt.keyProperties)
			if result != tt.expected {
				t.Errorf("Expected %v (%T), got %v (%T)",
					tt.expected, tt.expected, result, result)
			}
		})
	}
}
