package odata

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestVirtualProduct is a test entity for virtual entity tests
type TestVirtualProduct struct {
	ID    int     `json:"id" odata:"key"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func setupVirtualTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	return db
}

func TestRegisterVirtualEntity(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	// Verify the entity was registered
	if _, exists := service.entities["TestVirtualProducts"]; !exists {
		t.Error("Virtual entity was not registered")
	}

	// Verify the IsVirtual flag is set
	metadata := service.entities["TestVirtualProducts"]
	if !metadata.IsVirtual {
		t.Error("IsVirtual flag was not set on virtual entity")
	}
}

func TestRegisterVirtualEntity_DuplicateEntitySet(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	// Register once
	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	// Try to register again
	err = service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err == nil {
		t.Fatal("Expected error when registering duplicate entity set")
	}

	expected := "entity set 'TestVirtualProducts' is already registered"
	if err.Error() != expected {
		t.Errorf("Unexpected error: got %q, want %q", err.Error(), expected)
	}
}

func TestVirtualEntity_GetCollection_WithoutOverwrite(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	resp, err := http.Get(server.URL + "/TestVirtualProducts")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("Expected status 405, got %d: %s", resp.StatusCode, body)
	}

	// Verify that an error response is returned
	var errorResponse map[string]interface{}
	if err := json.Unmarshal(body, &errorResponse); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if _, ok := errorResponse["error"]; !ok {
		t.Error("Expected error object in response")
	}
}

func TestVirtualEntity_GetEntity_WithoutOverwrite(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	resp, err := http.Get(server.URL + "/TestVirtualProducts(1)")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 405, got %d: %s", resp.StatusCode, body)
	}
}

func TestVirtualEntity_Create_WithoutOverwrite(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	body := `{"name": "Test Product", "price": 99.99}`
	resp, err := http.Post(server.URL+"/TestVirtualProducts", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		bodyContent, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 405, got %d: %s", resp.StatusCode, bodyContent)
	}
}

func TestVirtualEntity_Update_WithoutOverwrite(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	body := `{"name": "Updated Product"}`
	req, _ := http.NewRequest(http.MethodPatch, server.URL+"/TestVirtualProducts(1)", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		bodyContent, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 405, got %d: %s", resp.StatusCode, bodyContent)
	}
}

func TestVirtualEntity_Delete_WithoutOverwrite(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/TestVirtualProducts(1)", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		bodyContent, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 405, got %d: %s", resp.StatusCode, bodyContent)
	}
}

func TestVirtualEntity_WithOverwriteHandlers(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	// Set up overwrite handlers
	virtualData := []TestVirtualProduct{
		{ID: 1, Name: "Virtual Product 1", Price: 99.99},
		{ID: 2, Name: "Virtual Product 2", Price: 149.99},
	}

	err = service.SetEntityOverwrite("TestVirtualProducts", &EntityOverwrite{
		GetCollection: func(ctx *OverwriteContext) (*CollectionResult, error) {
			return &CollectionResult{Items: virtualData}, nil
		},
		GetEntity: func(ctx *OverwriteContext) (interface{}, error) {
			if ctx.EntityKey == "1" {
				return &virtualData[0], nil
			} else if ctx.EntityKey == "2" {
				return &virtualData[1], nil
			}
			return nil, gorm.ErrRecordNotFound
		},
		Create: func(ctx *OverwriteContext, entity interface{}) (interface{}, error) {
			product := entity.(*TestVirtualProduct)
			product.ID = 3
			return product, nil
		},
		Update: func(ctx *OverwriteContext, data map[string]interface{}, isFullReplace bool) (interface{}, error) {
			if ctx.EntityKey == "1" {
				updated := virtualData[0]
				if name, ok := data["name"].(string); ok {
					updated.Name = name
				}
				return &updated, nil
			}
			return nil, gorm.ErrRecordNotFound
		},
		Delete: func(ctx *OverwriteContext) error {
			if ctx.EntityKey == "1" || ctx.EntityKey == "2" {
				return nil
			}
			return gorm.ErrRecordNotFound
		},
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite handlers: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// Test GetCollection
	t.Run("GetCollection", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/TestVirtualProducts")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, body)
		}

		var result struct {
			Value []TestVirtualProduct `json:"value"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(result.Value) != 2 {
			t.Errorf("Expected 2 products, got %d", len(result.Value))
		}
	})

	// Test GetEntity
	t.Run("GetEntity", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/TestVirtualProducts(1)")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, body)
		}

		var result TestVirtualProduct
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.ID != 1 {
			t.Errorf("Expected ID 1, got %d", result.ID)
		}
	})

	// Test Create
	t.Run("Create", func(t *testing.T) {
		body := `{"name": "New Product", "price": 199.99}`
		resp, err := http.Post(server.URL+"/TestVirtualProducts", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			bodyContent, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, bodyContent)
		}
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		body := `{"name": "Updated Product"}`
		req, _ := http.NewRequest(http.MethodPatch, server.URL+"/TestVirtualProducts(1)", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Prefer", "return=representation")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyContent, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, bodyContent)
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, server.URL+"/TestVirtualProducts(1)", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			bodyContent, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected status 204, got %d: %s", resp.StatusCode, bodyContent)
		}
	})
}

func TestVirtualEntity_PartialOverwriteHandlers(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	// Only set GetCollection handler
	virtualData := []TestVirtualProduct{
		{ID: 1, Name: "Virtual Product 1", Price: 99.99},
	}

	err = service.SetGetCollectionOverwrite("TestVirtualProducts", func(ctx *OverwriteContext) (*CollectionResult, error) {
		return &CollectionResult{Items: virtualData}, nil
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite handler: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	// GetCollection should work
	resp, err := http.Get(server.URL + "/TestVirtualProducts")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, body)
	}

	// GetEntity should return 405 (no handler set)
	resp2, err := http.Get(server.URL + "/TestVirtualProducts(1)")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusMethodNotAllowed {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("Expected status 405 for GetEntity without handler, got %d: %s", resp2.StatusCode, body)
	}
}

func TestVirtualEntity_AppearsInServiceDocument(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, body)
	}

	var serviceDoc struct {
		Value []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&serviceDoc); err != nil {
		t.Fatalf("Failed to decode service document: %v", err)
	}

	// Check if virtual entity appears in service document
	found := false
	for _, entity := range serviceDoc.Value {
		if entity.Name == "TestVirtualProducts" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Virtual entity was not found in service document")
	}
}

func TestVirtualEntity_ErrorHandling(t *testing.T) {
	db := setupVirtualTestDB(t)
	service := NewService(db)

	err := service.RegisterVirtualEntity(&TestVirtualProduct{})
	if err != nil {
		t.Fatalf("Failed to register virtual entity: %v", err)
	}

	// Set up overwrite handler that returns an error
	err = service.SetGetCollectionOverwrite("TestVirtualProducts", func(ctx *OverwriteContext) (*CollectionResult, error) {
		return nil, errors.New("custom error from virtual handler")
	})
	if err != nil {
		t.Fatalf("Failed to set overwrite handler: %v", err)
	}

	server := httptest.NewServer(service)
	defer server.Close()

	resp, err := http.Get(server.URL + "/TestVirtualProducts")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", resp.StatusCode)
	}
}
