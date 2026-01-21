package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestProductValidation is a test entity for select validation tests
type TestProductValidation struct {
	ID          int     `json:"id" gorm:"primarykey" odata:"key"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
}

func setupValidationTestService(t *testing.T) *odata.Service {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestProductValidation{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(TestProductValidation{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	db.Create(&TestProductValidation{ID: 1, Name: "Laptop", Price: 999.99, Description: "High-performance laptop", Category: "Electronics"})
	db.Create(&TestProductValidation{ID: 2, Name: "Mouse", Price: 29.99, Description: "Wireless mouse", Category: "Electronics"})

	return service
}

// TestSelectWithInvalidProperty verifies that selecting an invalid property returns an error
func TestSelectWithInvalidProperty(t *testing.T) {
	service := setupValidationTestService(t)

	tests := []struct {
		name          string
		url           string
		expectedError string
	}{
		{
			name:          "Single invalid property",
			url:           "/TestProductValidations?$select=invalidproperty",
			expectedError: "property 'invalidproperty' does not exist in entity type",
		},
		{
			name:          "Valid and invalid properties mixed",
			url:           "/TestProductValidations?$select=name,invalidprop",
			expectedError: "property 'invalidprop' does not exist in entity type",
		},
		{
			name:          "Multiple invalid properties",
			url:           "/TestProductValidations?$select=invalid1,invalid2",
			expectedError: "property 'invalid1' does not exist in entity type",
		},
		{
			name:          "Invalid property with spaces",
			url:           "/TestProductValidations?$select=name,%20invalidprop",
			expectedError: "property 'invalidprop' does not exist in entity type",
		},
		{
			name:          "Typo in property name",
			url:           "/TestProductValidations?$select=nam",
			expectedError: "property 'nam' does not exist in entity type",
		},
		{
			name:          "Case sensitive property name",
			url:           "/TestProductValidations?$select=NAME",
			expectedError: "property 'NAME' does not exist in entity type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			// Should return 400 Bad Request
			if w.Code != http.StatusBadRequest {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
				return
			}

			// Parse error response
			var errorResponse map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			// Check error structure
			errorObj, ok := errorResponse["error"].(map[string]interface{})
			if !ok {
				t.Fatal("Response does not contain error object")
			}

			// Check error code
			if code, ok := errorObj["code"].(string); !ok || code != "400" {
				t.Errorf("Expected error code '400', got %v", errorObj["code"])
			}

			// Check error message
			if message, ok := errorObj["message"].(string); !ok || message != "Invalid query options" {
				t.Errorf("Expected error message 'Invalid query options', got %v", errorObj["message"])
			}

			// Check error details contain the expected error
			details, ok := errorObj["details"].([]interface{})
			if !ok || len(details) == 0 {
				t.Fatal("Error response does not contain details")
			}

			detailObj, ok := details[0].(map[string]interface{})
			if !ok {
				t.Fatal("Error detail is not a map")
			}

			detailMessage, ok := detailObj["message"].(string)
			if !ok {
				t.Fatal("Error detail does not contain message")
			}

			if detailMessage != tt.expectedError {
				t.Errorf("Expected error detail '%s', got '%s'", tt.expectedError, detailMessage)
			}
		})
	}
}

// TestSelectWithValidProperties verifies that selecting valid properties works correctly
func TestSelectWithValidProperties(t *testing.T) {
	service := setupValidationTestService(t)

	tests := []struct {
		name           string
		url            string
		expectedFields []string
	}{
		{
			name:           "Single valid property",
			url:            "/TestProductValidations?$select=name",
			expectedFields: []string{"name"},
		},
		{
			name:           "Multiple valid properties",
			url:            "/TestProductValidations?$select=name,price,category",
			expectedFields: []string{"name", "price", "category"},
		},
		{
			name:           "All properties",
			url:            "/TestProductValidations?$select=id,name,price,description,category",
			expectedFields: []string{"id", "name", "price", "description", "category"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			// Should return 200 OK
			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
				return
			}

			// Parse response
			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check that response contains value array
			value, ok := response["value"].([]interface{})
			if !ok || len(value) == 0 {
				t.Fatal("Response does not contain value array or is empty")
			}

			// Verify that the first item contains only the expected fields
			firstItem, ok := value[0].(map[string]interface{})
			if !ok {
				t.Fatal("First value item is not a map")
			}

			// Verify all expected fields are present
			for _, field := range tt.expectedFields {
				if _, exists := firstItem[field]; !exists {
					t.Errorf("Expected field %s not found in response", field)
				}
			}
		})
	}
}

// TestSelectWithSingleEntity verifies that selecting properties on a single entity is validated
func TestSelectWithSingleEntity(t *testing.T) {
	service := setupValidationTestService(t)

	tests := []struct {
		name       string
		url        string
		expectCode int
	}{
		{
			name:       "Valid property on single entity",
			url:        "/TestProductValidations(1)?$select=name",
			expectCode: http.StatusOK,
		},
		{
			name:       "Invalid property on single entity",
			url:        "/TestProductValidations(1)?$select=invalidprop",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "Multiple properties on single entity",
			url:        "/TestProductValidations(1)?$select=name,price",
			expectCode: http.StatusOK,
		},
		{
			name:       "Mixed valid and invalid on single entity",
			url:        "/TestProductValidations(1)?$select=name,invalidprop",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, tt.expectCode, w.Body.String())
			}
		})
	}
}

// TestSelectWithEmptyValue verifies that empty $select parameter is handled correctly
func TestSelectWithEmptyValue(t *testing.T) {
	service := setupValidationTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/TestProductValidations?$select=", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Empty $select should return all fields (or be handled gracefully)
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// TestSelectValidationWithOtherQueryOptions verifies that $select validation works with other options
func TestSelectValidationWithOtherQueryOptions(t *testing.T) {
	service := setupValidationTestService(t)

	tests := []struct {
		name       string
		url        string
		expectCode int
	}{
		{
			name:       "Invalid select with valid filter",
			url:        "/TestProductValidations?$select=invalidprop&$filter=price%20gt%2050",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "Valid select with valid filter",
			url:        "/TestProductValidations?$select=name&$filter=price%20gt%2050",
			expectCode: http.StatusOK,
		},
		{
			name:       "Invalid select with orderby",
			url:        "/TestProductValidations?$select=invalidprop&$orderby=price",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "Valid select with orderby",
			url:        "/TestProductValidations?$select=name,price&$orderby=price%20desc",
			expectCode: http.StatusOK,
		},
		{
			name:       "Invalid select with pagination",
			url:        "/TestProductValidations?$select=invalidprop&$top=10&$skip=5",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, tt.expectCode, w.Body.String())
			}
		})
	}
}

// TestSelectDoesNotReturnEmptyObjects verifies the original issue is fixed
func TestSelectDoesNotReturnEmptyObjects(t *testing.T) {
	service := setupValidationTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/TestProductValidations?$select=nonexistentfield", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400, not 200 with empty objects
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %v", w.Code)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should not have a "value" array with empty objects
	if value, ok := response["value"]; ok {
		t.Errorf("Response should not contain 'value' array when $select has invalid properties, got: %v", value)
	}

	// Should have an error object
	if _, ok := response["error"]; !ok {
		t.Error("Response should contain 'error' object for invalid $select")
	}
}

// TestSelectOnSingleEntity ensures $select works correctly for individual entity requests
func TestSelectOnSingleEntity(t *testing.T) {
	service := setupValidationTestService(t)

	tests := []struct {
		name              string
		entityID          string
		selectParam       string
		expectedProps     []string
		unexpectedProps   []string
		expectedPropCount int
	}{
		{
			name:              "Single entity - select single property",
			entityID:          "1",
			selectParam:       "name",
			expectedProps:     []string{"name", "id"}, // ID (key) is always included per OData spec
			unexpectedProps:   []string{"price", "description", "category"},
			expectedPropCount: 2, // name + id (key is always included)
		},
		{
			name:              "Single entity - select multiple properties",
			entityID:          "1",
			selectParam:       "name,price",
			expectedProps:     []string{"name", "price", "id"}, // ID (key) is always included per OData spec
			unexpectedProps:   []string{"description", "category"},
			expectedPropCount: 3, // name + price + id (key is always included)
		},
		{
			name:              "Single entity - select all properties except one",
			entityID:          "2",
			selectParam:       "id,name,price,category",
			expectedProps:     []string{"id", "name", "price", "category"},
			unexpectedProps:   []string{"description"},
			expectedPropCount: 4,
		},
		{
			name:              "Single entity - select with whitespace",
			entityID:          "1",
			selectParam:       "name, price, category",
			expectedProps:     []string{"name", "price", "category", "id"},
			unexpectedProps:   []string{"description"},
			expectedPropCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build URL with proper encoding
			baseURL := "/TestProductValidations(" + tt.entityID + ")"
			req := httptest.NewRequest(http.MethodGet, baseURL, nil)
			q := req.URL.Query()
			q.Add("$select", tt.selectParam)
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
				return
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Single entity should not have a "value" array
			if _, hasValue := response["value"]; hasValue {
				t.Error("Single entity response should not have 'value' array")
			}

			// Should have @odata.context
			if _, hasContext := response["@odata.context"]; !hasContext {
				t.Error("Response should have '@odata.context'")
			}

			// Verify exact number of properties (excluding OData control annotations)
			actualPropCount := 0
			for key := range response {
				// Exclude OData control annotations (starting with @odata.)
				if !strings.HasPrefix(key, "@odata.") {
					actualPropCount++
				}
			}

			if actualPropCount != tt.expectedPropCount {
				t.Errorf("Expected exactly %d properties (excluding @odata.* annotations), got %d. Properties: %v",
					tt.expectedPropCount, actualPropCount, response)
			}

			// Verify expected properties are present
			for _, prop := range tt.expectedProps {
				if _, exists := response[prop]; !exists {
					t.Errorf("Expected property %s not found in response", prop)
				}
			}

			// Verify unexpected properties are NOT present
			for _, prop := range tt.unexpectedProps {
				if _, exists := response[prop]; exists {
					t.Errorf("Unexpected property %s found in response (should not be present)", prop)
				}
			}
		})
	}
}

// TestSelectReturnsOnlySelectedProperties ensures $select query properly filters properties
func TestSelectReturnsOnlySelectedProperties(t *testing.T) {
	service := setupValidationTestService(t)

	tests := []struct {
		name              string
		selectParam       string
		expectedProps     []string
		unexpectedProps   []string
		expectedPropCount int
	}{
		{
			name:              "Single property",
			selectParam:       "name",
			expectedProps:     []string{"name", "id"}, // ID (key) is always included per OData spec
			unexpectedProps:   []string{"price", "description", "category"},
			expectedPropCount: 2, // name + id (key is always included)
		},
		{
			name:              "Multiple properties",
			selectParam:       "name,price",
			expectedProps:     []string{"name", "price", "id"}, // ID (key) is always included per OData spec
			unexpectedProps:   []string{"description", "category"},
			expectedPropCount: 3, // name + price + id (key is always included)
		},
		{
			name:              "All properties except one",
			selectParam:       "id,name,price,category",
			expectedProps:     []string{"id", "name", "price", "category"},
			unexpectedProps:   []string{"description"},
			expectedPropCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/TestProductValidations?$select=" + tt.selectParam
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
				return
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok || len(value) == 0 {
				t.Fatal("Response does not contain value array or is empty")
			}

			firstItem, ok := value[0].(map[string]interface{})
			if !ok {
				t.Fatal("First value item is not a map")
			}

			// Verify exact number of properties (excluding @odata.* control information)
			nonControlFields := 0
			for key := range firstItem {
				if !strings.HasPrefix(key, "@odata.") {
					nonControlFields++
				}
			}
			if nonControlFields != tt.expectedPropCount {
				t.Errorf("Expected exactly %d properties, got %d. Properties: %v",
					tt.expectedPropCount, nonControlFields, firstItem)
			}

			// Verify expected properties are present
			for _, prop := range tt.expectedProps {
				if _, exists := firstItem[prop]; !exists {
					t.Errorf("Expected property %s not found in response", prop)
				}
			}

			// Verify unexpected properties are NOT present
			for _, prop := range tt.unexpectedProps {
				if _, exists := firstItem[prop]; exists {
					t.Errorf("Unexpected property %s found in response (should not be present)", prop)
				}
			}
		})
	}
}

// TestSelectAlwaysIncludesKeyProperties verifies that key properties are always returned
// even when not explicitly selected, per OData v4 spec section 11.2.4.1
func TestSelectAlwaysIncludesKeyProperties(t *testing.T) {
	service := setupValidationTestService(t)

	tests := []struct {
		name        string
		url         string
		description string
	}{
		{
			name:        "Single entity without key in select",
			url:         "/TestProductValidations(1)?$select=name,price",
			description: "Key property 'id' should be included even though not selected",
		},
		{
			name:        "Collection without key in select",
			url:         "/TestProductValidations?$select=name",
			description: "Key property 'id' should be included in all items even though not selected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
				return
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check if this is a single entity or collection response
			if value, isCollection := response["value"].([]interface{}); isCollection {
				// Collection response
				if len(value) == 0 {
					t.Fatal("Response value array is empty")
				}

				firstItem, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("First value item is not a map")
				}

				if _, hasID := firstItem["id"]; !hasID {
					t.Errorf("%s: Key property 'id' not found in collection item. Properties: %v",
						tt.description, firstItem)
				}
			} else {
				// Single entity response
				if _, hasID := response["id"]; !hasID {
					t.Errorf("%s: Key property 'id' not found in response. Properties: %v",
						tt.description, response)
				}
			}
		})
	}
}

// TestSelectWithETag verifies that ETag functionality still works when using $select
func TestSelectWithETag(t *testing.T) {
	service := setupValidationTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/TestProductValidations(1)?$select=name", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
		return
	}

	// Note: ETag generation depends on the entity having an ETag property configured
	// This test verifies that $select doesn't break the ETag header if it's present
	etag := w.Header().Get("ETag")

	// If the entity type has an ETag configured, it should be present
	// If not, this test just verifies the request succeeded
	t.Logf("ETag header value: %q (may be empty if no ETag property configured)", etag)

	// Verify response body is valid JSON with selected properties
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have name and id (key)
	if _, hasName := response["name"]; !hasName {
		t.Error("Expected 'name' property in response")
	}

	if _, hasID := response["id"]; !hasID {
		t.Error("Expected 'id' property (key) in response")
	}

	// Should NOT have unselected properties
	if _, hasPrice := response["price"]; hasPrice {
		t.Error("Did not expect 'price' property in response (not selected)")
	}
}
