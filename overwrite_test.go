package odata

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestProduct is a test entity for overwrite handler tests
type TestOverwriteProduct struct {
	ID    int     `json:"id" odata:"key"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func setupOverwriteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestOverwriteProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}

func setupOverwriteTestService(t *testing.T) *Service {
	t.Helper()
	db := setupOverwriteTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(&TestOverwriteProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service
}

func TestSetEntityOverwrite_EntityNotFound(t *testing.T) {
	db := setupOverwriteTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	err = service.SetEntityOverwrite("NonExistent", &EntityOverwrite{})
	if err == nil {
		t.Fatal("expected error for non-existent entity set")
	}

	expected := "entity set 'NonExistent' is not registered"
	if err.Error() != expected {
		t.Fatalf("unexpected error: got %q, want %q", err.Error(), expected)
	}
}

func TestSetEntityOverwrite_NilOverwrite(t *testing.T) {
	service := setupOverwriteTestService(t)

	// Setting nil should not error
	err := service.SetEntityOverwrite("TestOverwriteProducts", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetCollectionOverwrite(t *testing.T) {
	service := setupOverwriteTestService(t)

	// Set up custom data
	customProducts := []TestOverwriteProduct{
		{ID: 100, Name: "Custom Product 1", Price: 99.99},
		{ID: 200, Name: "Custom Product 2", Price: 149.99},
	}

	// Register overwrite handler
	err := service.SetGetCollectionOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext) (*CollectionResult, error) {
		return &CollectionResult{Items: customProducts}, nil
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	// Create test server
	server := httptest.NewServer(service)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/TestOverwriteProducts")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, body)
	}

	// Parse response
	var result struct {
		Value []TestOverwriteProduct `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Value) != 2 {
		t.Fatalf("Expected 2 products, got %d", len(result.Value))
	}

	if result.Value[0].ID != 100 || result.Value[0].Name != "Custom Product 1" {
		t.Errorf("Unexpected first product: %+v", result.Value[0])
	}
}

func TestGetCollectionOverwrite_WithCount(t *testing.T) {
	service := setupOverwriteTestService(t)

	customProducts := []TestOverwriteProduct{
		{ID: 1, Name: "Product 1", Price: 10.00},
		{ID: 2, Name: "Product 2", Price: 20.00},
	}
	totalCount := int64(100) // Simulate a filtered count different from returned items

	err := service.SetGetCollectionOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext) (*CollectionResult, error) {
		result := &CollectionResult{Items: customProducts}
		if ctx.QueryOptions.Count {
			result.Count = &totalCount
		}
		return result, nil
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	resp, err := http.Get(server.URL + "/TestOverwriteProducts?$count=true")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Value []TestOverwriteProduct `json:"value"`
		Count *int64                 `json:"@odata.count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Count == nil || *result.Count != 100 {
		t.Errorf("Expected count 100, got %v", result.Count)
	}
}

func TestGetCollectionOverwrite_Error(t *testing.T) {
	service := setupOverwriteTestService(t)

	err := service.SetGetCollectionOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext) (*CollectionResult, error) {
		return nil, errors.New("custom error from handler")
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	resp, err := http.Get(server.URL + "/TestOverwriteProducts")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestGetCollectionOverwrite_HookError_CustomStatusCode(t *testing.T) {
	service := setupOverwriteTestService(t)

	// Register overwrite handler that returns HookError with custom status code
	err := service.SetGetCollectionOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext) (*CollectionResult, error) {
		// Simulate authentication check failure
		return nil, NewHookError(http.StatusUnauthorized, "unauthorized: user not authenticated")
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/TestOverwriteProducts")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body once
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Verify status code is 401, not 500
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d: %s", resp.StatusCode, bodyBytes)
	}

	// Verify error response contains the custom message
	var result struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if result.Error.Code != "401" {
		t.Errorf("Expected error code '401', got %q", result.Error.Code)
	}

	if result.Error.Message != "unauthorized: user not authenticated" {
		t.Errorf("Expected error message 'unauthorized: user not authenticated', got %q", result.Error.Message)
	}
}

func TestGetEntityOverwrite(t *testing.T) {
	service := setupOverwriteTestService(t)

	customProduct := TestOverwriteProduct{ID: 42, Name: "Custom Product", Price: 999.99}

	err := service.SetGetEntityOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext) (interface{}, error) {
		if ctx.EntityKey == "42" {
			return &customProduct, nil
		}
		return nil, gorm.ErrRecordNotFound
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// Test found entity
	resp, err := http.Get(server.URL + "/TestOverwriteProducts(42)")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, body)
	}

	var result TestOverwriteProduct
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.ID != 42 || result.Name != "Custom Product" {
		t.Errorf("Unexpected product: %+v", result)
	}

	// Test not found entity
	resp2, err := http.Get(server.URL + "/TestOverwriteProducts(999)")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp2.StatusCode)
	}
}

func TestCreateOverwrite(t *testing.T) {
	service := setupOverwriteTestService(t)

	var createdProduct *TestOverwriteProduct

	err := service.SetCreateOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext, entity interface{}) (interface{}, error) {
		product := entity.(*TestOverwriteProduct)
		product.ID = 12345 // Assign server-generated ID
		createdProduct = product
		return product, nil
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// Create a new product
	body := `{"name": "New Product", "price": 59.99}`
	resp, err := http.Post(server.URL+"/TestOverwriteProducts", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyContent, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, bodyContent)
	}

	// Verify Location header
	location := resp.Header.Get("Location")
	if !strings.Contains(location, "TestOverwriteProducts(12345)") {
		t.Errorf("Expected Location header to contain 'TestOverwriteProducts(12345)', got %s", location)
	}

	// Verify the entity was passed to the handler
	if createdProduct == nil || createdProduct.Name != "New Product" {
		t.Errorf("Unexpected created product: %+v", createdProduct)
	}
}

func TestUpdateOverwrite_Patch(t *testing.T) {
	service := setupOverwriteTestService(t)

	var updateData map[string]interface{}
	var wasFullReplace bool

	err := service.SetUpdateOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext, data map[string]interface{}, isFullReplace bool) (interface{}, error) {
		updateData = data
		wasFullReplace = isFullReplace

		id, _ := strconv.Atoi(ctx.EntityKey)
		return &TestOverwriteProduct{
			ID:    id,
			Name:  data["name"].(string),
			Price: data["price"].(float64),
		}, nil
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// PATCH request
	body := `{"name": "Updated Name", "price": 79.99}`
	req, _ := http.NewRequest(http.MethodPatch, server.URL+"/TestOverwriteProducts(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		bodyContent, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 204, got %d: %s", resp.StatusCode, bodyContent)
	}

	if wasFullReplace {
		t.Error("Expected isFullReplace to be false for PATCH")
	}

	if updateData["name"] != "Updated Name" {
		t.Errorf("Unexpected update data: %+v", updateData)
	}
}

func TestUpdateOverwrite_Put(t *testing.T) {
	service := setupOverwriteTestService(t)

	var wasFullReplace bool

	err := service.SetUpdateOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext, data map[string]interface{}, isFullReplace bool) (interface{}, error) {
		wasFullReplace = isFullReplace
		return &TestOverwriteProduct{ID: 1, Name: "Replaced", Price: 100.00}, nil
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// PUT request
	body := `{"name": "Replaced", "price": 100.00}`
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/TestOverwriteProducts(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		bodyContent, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 204, got %d: %s", resp.StatusCode, bodyContent)
	}

	if !wasFullReplace {
		t.Error("Expected isFullReplace to be true for PUT")
	}
}

func TestDeleteOverwrite(t *testing.T) {
	service := setupOverwriteTestService(t)

	var deletedKey string

	err := service.SetDeleteOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext) error {
		deletedKey = ctx.EntityKey
		if ctx.EntityKey == "999" {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// Successful delete
	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/TestOverwriteProducts(42)", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected status 204, got %d", resp.StatusCode)
	}

	if deletedKey != "42" {
		t.Errorf("Expected deleted key '42', got '%s'", deletedKey)
	}

	// Delete not found
	req2, _ := http.NewRequest(http.MethodDelete, server.URL+"/TestOverwriteProducts(999)", nil)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp2.StatusCode)
	}
}

func TestGetCountOverwrite(t *testing.T) {
	service := setupOverwriteTestService(t)

	err := service.SetGetCountOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext) (int64, error) {
		// Return a custom count
		return 42, nil
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	resp, err := http.Get(server.URL + "/TestOverwriteProducts/$count")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	if string(body) != "42" {
		t.Errorf("Expected count '42', got '%s'", body)
	}
}

func TestGetCountOverwrite_Error(t *testing.T) {
	service := setupOverwriteTestService(t)

	err := service.SetGetCountOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext) (int64, error) {
		return 0, errors.New("count error")
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	resp, err := http.Get(server.URL + "/TestOverwriteProducts/$count")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestSetEntityOverwrite_AllHandlers(t *testing.T) {
	service := setupOverwriteTestService(t)

	// Set all handlers at once using EntityOverwrite
	err := service.SetEntityOverwrite("TestOverwriteProducts", &EntityOverwrite{
		GetCollection: func(ctx *OverwriteContext) (*CollectionResult, error) {
			return &CollectionResult{Items: []TestOverwriteProduct{{ID: 1, Name: "Test", Price: 10.0}}}, nil
		},
		GetEntity: func(ctx *OverwriteContext) (interface{}, error) {
			return &TestOverwriteProduct{ID: 1, Name: "Test", Price: 10.0}, nil
		},
		Create: func(ctx *OverwriteContext, entity interface{}) (interface{}, error) {
			p := entity.(*TestOverwriteProduct)
			p.ID = 99
			return p, nil
		},
		Update: func(ctx *OverwriteContext, data map[string]interface{}, isFullReplace bool) (interface{}, error) {
			return &TestOverwriteProduct{ID: 1, Name: "Updated", Price: 20.0}, nil
		},
		Delete: func(ctx *OverwriteContext) error {
			return nil
		},
		GetCount: func(ctx *OverwriteContext) (int64, error) {
			return 100, nil
		},
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// Test GetCollection
	resp, _ := http.Get(server.URL + "/TestOverwriteProducts")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GetCollection: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Test GetEntity
	resp, _ = http.Get(server.URL + "/TestOverwriteProducts(1)")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GetEntity: expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Test Create
	resp, _ = http.Post(server.URL+"/TestOverwriteProducts", "application/json", strings.NewReader(`{"name":"New"}`))
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Create: expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Test Update (PATCH)
	req, _ := http.NewRequest(http.MethodPatch, server.URL+"/TestOverwriteProducts(1)", strings.NewReader(`{"name":"Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Update: expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Test Delete
	req, _ = http.NewRequest(http.MethodDelete, server.URL+"/TestOverwriteProducts(1)", nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Delete: expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Test GetCount
	resp, _ = http.Get(server.URL + "/TestOverwriteProducts/$count")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GetCount: expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "100" {
		t.Errorf("GetCount: expected '100', got '%s'", body)
	}
	resp.Body.Close()
}

func TestOverwriteContext_QueryOptionsAccess(t *testing.T) {
	service := setupOverwriteTestService(t)

	var capturedFilter string
	var capturedTop *int
	var capturedSelect []string

	err := service.SetGetCollectionOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext) (*CollectionResult, error) {
		if ctx.QueryOptions.Filter != nil {
			capturedFilter = fmt.Sprintf("%v", ctx.QueryOptions.Filter.Property)
		}
		capturedTop = ctx.QueryOptions.Top
		capturedSelect = ctx.QueryOptions.Select
		return &CollectionResult{Items: []TestOverwriteProduct{}}, nil
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// Build query URL using url.Values for better readability
	queryParams := url.Values{}
	queryParams.Set("$filter", "name eq 'Test'")
	queryParams.Set("$top", "5")
	queryParams.Set("$select", "id,name")

	resp, err := http.Get(server.URL + "/TestOverwriteProducts?" + queryParams.Encode())
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyContent, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, bodyContent)
	}

	if capturedFilter != "name" {
		t.Errorf("Expected filter property 'name', got '%s'", capturedFilter)
	}

	if capturedTop == nil || *capturedTop != 5 {
		t.Errorf("Expected top 5, got %v", capturedTop)
	}

	if len(capturedSelect) != 2 {
		t.Errorf("Expected 2 select properties, got %d", len(capturedSelect))
	}
}

func TestOverwriteWithInvalidQueryOptionStillValidated(t *testing.T) {
	service := setupOverwriteTestService(t)

	// Register overwrite handler
	err := service.SetGetCollectionOverwrite("TestOverwriteProducts", func(ctx *OverwriteContext) (*CollectionResult, error) {
		// This should never be called because query validation should fail first
		t.Error("Handler should not be called for invalid query")
		return &CollectionResult{Items: []TestOverwriteProduct{}}, nil
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// Test with invalid $top value
	resp, err := http.Get(server.URL + "/TestOverwriteProducts?$top=invalid")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid query option, got %d", resp.StatusCode)
	}
}

func TestSetOverwriteMethodsForUnregisteredEntity(t *testing.T) {
	db := setupOverwriteTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	tests := []struct {
		name   string
		setter func() error
	}{
		{
			name: "SetGetCollectionOverwrite",
			setter: func() error {
				return service.SetGetCollectionOverwrite("NonExistent", func(ctx *OverwriteContext) (*CollectionResult, error) {
					return nil, nil
				})
			},
		},
		{
			name: "SetGetEntityOverwrite",
			setter: func() error {
				return service.SetGetEntityOverwrite("NonExistent", func(ctx *OverwriteContext) (interface{}, error) {
					return nil, nil
				})
			},
		},
		{
			name: "SetCreateOverwrite",
			setter: func() error {
				return service.SetCreateOverwrite("NonExistent", func(ctx *OverwriteContext, entity interface{}) (interface{}, error) {
					return nil, nil
				})
			},
		},
		{
			name: "SetUpdateOverwrite",
			setter: func() error {
				return service.SetUpdateOverwrite("NonExistent", func(ctx *OverwriteContext, data map[string]interface{}, isFullReplace bool) (interface{}, error) {
					return nil, nil
				})
			},
		},
		{
			name: "SetDeleteOverwrite",
			setter: func() error {
				return service.SetDeleteOverwrite("NonExistent", func(ctx *OverwriteContext) error {
					return nil
				})
			},
		},
		{
			name: "SetGetCountOverwrite",
			setter: func() error {
				return service.SetGetCountOverwrite("NonExistent", func(ctx *OverwriteContext) (int64, error) {
					return 0, nil
				})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.setter()
			if err == nil {
				t.Fatal("expected error for non-existent entity set")
			}
			expected := "entity set 'NonExistent' is not registered"
			if err.Error() != expected {
				t.Errorf("unexpected error: got %q, want %q", err.Error(), expected)
			}
		})
	}
}
