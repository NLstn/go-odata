package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestSetODataVersionHeader(t *testing.T) {
	w := httptest.NewRecorder()

	SetODataVersionHeader(w)

	// Verify header was set with correct value - use direct map access
	// OData spec requires exact "OData-Version" capitalization which is non-canonical in Go
	header := w.Header()
	values := header["OData-Version"] //nolint:staticcheck // OData headers require non-canonical keys
	if len(values) == 0 {
		t.Error("Expected OData-Version header to be set")
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
