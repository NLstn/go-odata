package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ErrorTestProduct is a test entity for error response validation
type ErrorTestProduct struct {
	ID    int     `json:"id" gorm:"primarykey" odata:"key"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func setupErrorTestService(t *testing.T) *odata.Service {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ErrorTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(ErrorTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Create a test entity
	db.Create(&ErrorTestProduct{
		ID:    1,
		Name:  "Laptop",
		Price: 999.99,
	})

	return service
}

func TestErrorResponse_EntityNotFound(t *testing.T) {
	service := setupErrorTestService(t)

	// Try to fetch non-existent entity
	req := httptest.NewRequest(http.MethodGet, "/ErrorTestProducts(999)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	// Verify OData headers
	if version := w.Header().Get("OData-Version"); version != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", version)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}

	// Parse and validate error structure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify error field exists
	errorData, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error field is missing or not an object")
	}

	// Verify code field
	if errorData["code"] != "404" {
		t.Errorf("error.code = %v, want 404", errorData["code"])
	}

	// Verify message field
	if errorData["message"] != "Entity not found" {
		t.Errorf("error.message = %v, want 'Entity not found'", errorData["message"])
	}

	// Verify target field
	if errorData["target"] != "ErrorTestProducts(999)" {
		t.Errorf("error.target = %v, want 'ErrorTestProducts(999)'", errorData["target"])
	}

	// Verify details array exists
	details, ok := errorData["details"].([]interface{})
	if !ok {
		t.Fatal("error.details is missing or not an array")
	}

	if len(details) != 1 {
		t.Fatalf("len(error.details) = %v, want 1", len(details))
	}

	// Verify first detail
	firstDetail, ok := details[0].(map[string]interface{})
	if !ok {
		t.Fatal("error.details[0] is not an object")
	}

	if firstDetail["target"] != "ErrorTestProducts(999)" {
		t.Errorf("error.details[0].target = %v, want 'ErrorTestProducts(999)'", firstDetail["target"])
	}

	if firstDetail["message"] != "The entity with key '999' does not exist" {
		t.Errorf("error.details[0].message = %v, want 'The entity with key '999' does not exist'", firstDetail["message"])
	}
}

func TestErrorResponse_InvalidQuery(t *testing.T) {
	service := setupErrorTestService(t)

	// Try to use invalid filter syntax
	req := httptest.NewRequest(http.MethodGet, "/ErrorTestProducts?$filter=price%20invalid%20syntax", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	// Parse error response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorData, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error field is missing or not an object")
	}

	// Verify basic error structure
	if errorData["code"] != "400" {
		t.Errorf("error.code = %v, want 400", errorData["code"])
	}

	if errorData["message"] != "Invalid query options" {
		t.Errorf("error.message = %v, want 'Invalid query options'", errorData["message"])
	}

	// Verify details exist
	details, ok := errorData["details"].([]interface{})
	if !ok {
		t.Fatal("error.details is missing or not an array")
	}

	if len(details) < 1 {
		t.Fatal("error.details should contain at least one detail")
	}
}

func TestErrorResponse_EntitySetNotFound(t *testing.T) {
	service := setupErrorTestService(t)

	// Try to access non-existent entity set
	req := httptest.NewRequest(http.MethodGet, "/NonExistentEntitySet", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	// Parse error response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorData, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error field is missing or not an object")
	}

	// Verify error structure
	if errorData["code"] != "404" {
		t.Errorf("error.code = %v, want 404", errorData["code"])
	}

	if errorData["message"] != "Entity set not found" {
		t.Errorf("error.message = %v, want 'Entity set not found'", errorData["message"])
	}
}

func TestErrorResponse_MethodNotAllowed(t *testing.T) {
	service := setupErrorTestService(t)

	// Try unsupported method on collection
	req := httptest.NewRequest(http.MethodPut, "/ErrorTestProducts", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify status code
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}

	// Parse error response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	errorData, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error field is missing or not an object")
	}

	// Verify error structure
	if errorData["code"] != "405" {
		t.Errorf("error.code = %v, want 405", errorData["code"])
	}

	if errorData["message"] != "Method not allowed" {
		t.Errorf("error.message = %v, want 'Method not allowed'", errorData["message"])
	}
}

func TestErrorResponse_InvalidQueryOption(t *testing.T) {
	service := setupErrorTestService(t)

	tests := []struct {
		name        string
		url         string
		expectCode  int
		expectMsg   string
		expectError string
	}{
		{
			name:        "Single invalid query option",
			url:         "/ErrorTestProducts?$invalidQuery=1234",
			expectCode:  http.StatusBadRequest,
			expectMsg:   "Invalid query options",
			expectError: "unknown query option: '$invalidQuery'",
		},
		{
			name:        "Multiple invalid query options",
			url:         "/ErrorTestProducts?$invalidOption=value&$anotherInvalid=test",
			expectCode:  http.StatusBadRequest,
			expectMsg:   "Invalid query options",
			expectError: "", // Either error could come first
		},
		{
			name:        "Valid and invalid mixed",
			url:         "/ErrorTestProducts?$filter=price%20gt%2050&$invalidQuery=1234",
			expectCode:  http.StatusBadRequest,
			expectMsg:   "Invalid query options",
			expectError: "unknown query option: '$invalidQuery'",
		},
		{
			name:       "Non-$ prefixed parameter should work",
			url:        "/ErrorTestProducts?customParam=value",
			expectCode: http.StatusOK,
		},
		{
			name:       "All valid query options",
			url:        "/ErrorTestProducts?$filter=price%20gt%2050&$select=name&$top=10",
			expectCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			// Verify status code
			if w.Code != tt.expectCode {
				t.Errorf("Status = %v, want %v", w.Code, tt.expectCode)
			}

			// For error responses, validate structure
			if tt.expectCode != http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				errorData, ok := response["error"].(map[string]interface{})
				if !ok {
					t.Fatal("error field is missing or not an object")
				}

				// Verify error message
				if errorData["message"] != tt.expectMsg {
					t.Errorf("error.message = %v, want '%s'", errorData["message"], tt.expectMsg)
				}

				// Verify details contain the expected error
				if tt.expectError != "" {
					details, ok := errorData["details"].([]interface{})
					if !ok || len(details) == 0 {
						t.Fatal("error.details is missing or empty")
					}

					firstDetail, ok := details[0].(map[string]interface{})
					if !ok {
						t.Fatal("error.details[0] is not an object")
					}

					detailMsg, ok := firstDetail["message"].(string)
					if !ok {
						t.Fatal("error.details[0].message is not a string")
					}

					if detailMsg != tt.expectError {
						t.Errorf("error.details[0].message = %v, want '%s'", detailMsg, tt.expectError)
					}
				}
			}
		})
	}
}
