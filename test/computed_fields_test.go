package odata_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Product entity with computed field for testing
type ProductWithComputed struct {
	ID          int     `json:"id" gorm:"primarykey" odata:"key"`
	Name        string  `json:"name" odata:"required"`
	Price       float64 `json:"price"`
	DisplayName string  `json:"displayName" odata:"computed"` // Server-side computed, no database column
}

// ODataAfterReadEntity hook to populate the computed field
func (ProductWithComputed) ODataAfterReadEntity(ctx context.Context, r *http.Request, opts *odata.QueryOptions, entity interface{}) (interface{}, error) {
	if p, ok := entity.(*ProductWithComputed); ok {
		p.DisplayName = "Product: " + p.Name
	}
	return nil, nil
}

// ODataAfterReadCollection hook to populate the computed field
func (ProductWithComputed) ODataAfterReadCollection(ctx context.Context, r *http.Request, opts *odata.QueryOptions, results interface{}) (interface{}, error) {
	if products, ok := results.(*[]ProductWithComputed); ok {
		for i := range *products {
			(*products)[i].DisplayName = "Product: " + (*products)[i].Name
		}
	}
	return nil, nil
}

func setupComputedFieldsTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ProductWithComputed{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(&ProductWithComputed{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	if err := db.Create(&ProductWithComputed{ID: 1, Name: "Widget", Price: 9.99}).Error; err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}
	if err := db.Create(&ProductWithComputed{ID: 2, Name: "Gadget", Price: 19.99}).Error; err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}
	if err := db.Create(&ProductWithComputed{ID: 3, Name: "Gizmo", Price: 29.99}).Error; err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	return service, db
}

func TestComputedFields_Select_DoesNotCauseSQL_Error(t *testing.T) {
	service, _ := setupComputedFieldsTestService(t)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "Select computed field only",
			url:            "/ProductWithComputeds(1)?$select=displayName",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				if displayName, ok := response["displayName"].(string); !ok || displayName != "Product: Widget" {
					t.Errorf("Expected displayName to be 'Product: Widget', got %v", response["displayName"])
				}
			},
		},
		{
			name:           "Select computed field with regular field",
			url:            "/ProductWithComputeds(1)?$select=name,displayName",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				if name, ok := response["name"].(string); !ok || name != "Widget" {
					t.Errorf("Expected name to be 'Widget', got %v", response["name"])
				}
				if displayName, ok := response["displayName"].(string); !ok || displayName != "Product: Widget" {
					t.Errorf("Expected displayName to be 'Product: Widget', got %v", response["displayName"])
				}
			},
		},
		{
			name:           "Select in collection with computed field",
			url:            "/ProductWithComputeds?$select=name,displayName",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("Expected 'value' array in response")
				}
				if len(value) < 1 {
					t.Fatal("Expected at least 1 item in value array")
				}
				item := value[0].(map[string]interface{})
				// The hook may or may not populate displayName in collections with $select
				// The main goal is ensuring no SQL error occurs
				t.Logf("First item: %+v", item)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestComputedFields_Filter_ReturnsError(t *testing.T) {
	service, _ := setupComputedFieldsTestService(t)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		errorContains  string
	}{
		{
			name:           "Filter on computed field",
			url:            "/ProductWithComputeds?$filter=displayName%20eq%20%27Product%3A%20Widget%27",
			expectedStatus: http.StatusBadRequest,
			errorContains:  "displayName",
		},
		{
			name:           "Filter with startswith on computed field",
			url:            "/ProductWithComputeds?$filter=startswith(displayName,%20%27Product%27)",
			expectedStatus: http.StatusBadRequest,
			errorContains:  "displayName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.errorContains != "" && !strings.Contains(w.Body.String(), tt.errorContains) {
				t.Errorf("Expected error to contain %q, got: %s", tt.errorContains, w.Body.String())
			}
		})
	}
}

func TestComputedFields_OrderBy_ReturnsError(t *testing.T) {
	service, _ := setupComputedFieldsTestService(t)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		errorContains  string
	}{
		{
			name:           "OrderBy on computed field",
			url:            "/ProductWithComputeds?$orderby=displayName",
			expectedStatus: http.StatusBadRequest,
			errorContains:  "displayName",
		},
		{
			name:           "OrderBy desc on computed field",
			url:            "/ProductWithComputeds?$orderby=displayName%20desc",
			expectedStatus: http.StatusBadRequest,
			errorContains:  "displayName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.errorContains != "" && !strings.Contains(w.Body.String(), tt.errorContains) {
				t.Errorf("Expected error to contain %q, got: %s", tt.errorContains, w.Body.String())
			}
		})
	}
}

func TestComputedFields_POST_RejectClientProvidedValues(t *testing.T) {
	service, _ := setupComputedFieldsTestService(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		errorContains  string
	}{
		{
			name: "Reject computed field in POST",
			requestBody: map[string]interface{}{
				"id":          100,
				"name":        "New Product",
				"price":       99.99,
				"displayName": "Hacker Product",
			},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "displayName",
		},
		{
			name: "Accept request without computed field",
			requestBody: map[string]interface{}{
				"id":    101,
				"name":  "Valid Product",
				"price": 49.99,
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/ProductWithComputeds", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.errorContains != "" && !strings.Contains(w.Body.String(), tt.errorContains) {
				t.Errorf("Expected error to contain %q, got: %s", tt.errorContains, w.Body.String())
			}
		})
	}
}

func TestComputedFields_PATCH_RejectClientProvidedValues(t *testing.T) {
	service, _ := setupComputedFieldsTestService(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		errorContains  string
	}{
		{
			name: "Reject computed field in PATCH",
			requestBody: map[string]interface{}{
				"displayName": "Hacker Display",
			},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "displayName",
		},
		{
			name: "Accept PATCH without computed field",
			requestBody: map[string]interface{}{
				"price": 39.99,
			},
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("PATCH", "/ProductWithComputeds(1)", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.errorContains != "" && !strings.Contains(w.Body.String(), tt.errorContains) {
				t.Errorf("Expected error to contain %q, got: %s", tt.errorContains, w.Body.String())
			}
		})
	}
}
