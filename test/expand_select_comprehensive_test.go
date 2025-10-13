package odata_test

import (
	"encoding/json"
	odata "github.com/nlstn/go-odata"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestExpandSelectComprehensive tests various combinations of expand and select
func TestExpandSelectComprehensive(t *testing.T) {
	db := setupTestDB(t)
	service := odata.NewService(db)
	if err := service.RegisterEntity(&Product{}); err != nil {
		t.Fatalf("Failed to register Product entity: %v", err)
	}
	if err := service.RegisterEntity(&ProductDescription{}); err != nil {
		t.Fatalf("Failed to register ProductDescription entity: %v", err)
	}

	t.Run("Expand without select returns all properties", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ProductDescriptions?$expand=Product", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		value := response["value"].([]interface{})
		if len(value) == 0 {
			t.Fatal("Expected at least one result")
		}

		item := value[0].(map[string]interface{})

		// Should have all properties from ProductDescription
		if _, ok := item["LanguageKey"]; !ok {
			t.Error("Expected LanguageKey property")
		}
		if _, ok := item["Description"]; !ok {
			t.Error("Expected Description property")
		}

		// Should have full Product entity
		product, ok := item["Product"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected Product to be expanded")
		}

		if _, ok := product["ID"]; !ok {
			t.Error("Expected Product.ID property")
		}
		if _, ok := product["Name"]; !ok {
			t.Error("Expected Product.Name property")
		}
		if _, ok := product["Price"]; !ok {
			t.Error("Expected Product.Price property")
		}
	})

	t.Run("Select multiple properties without expand", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ProductDescriptions?$select=LanguageKey,Description", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		value := response["value"].([]interface{})
		if len(value) == 0 {
			t.Fatal("Expected at least one result")
		}

		item := value[0].(map[string]interface{})

		// Should have selected properties plus key
		if _, ok := item["LanguageKey"]; !ok {
			t.Error("Expected LanguageKey property")
		}
		if _, ok := item["Description"]; !ok {
			t.Error("Expected Description property")
		}
		if _, ok := item["ProductID"]; !ok {
			t.Error("Expected ProductID key property")
		}
	})

	t.Run("Select multiple navigation paths", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ProductDescriptions?$expand=Product&$select=LanguageKey,Product/Name,Product/Price", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		value := response["value"].([]interface{})
		if len(value) == 0 {
			t.Fatal("Expected at least one result")
		}

		item := value[0].(map[string]interface{})

		// Should have LanguageKey
		if _, ok := item["LanguageKey"]; !ok {
			t.Error("Expected LanguageKey property")
		}

		// Should have Product with Name and Price
		product, ok := item["Product"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected Product to be expanded")
		}

		if _, ok := product["Name"]; !ok {
			t.Error("Expected Product.Name property")
		}
		if _, ok := product["Price"]; !ok {
			t.Error("Expected Product.Price property")
		}
		if _, ok := product["ID"]; !ok {
			t.Error("Expected Product.ID key property")
		}
	})

	t.Run("Expand with nested select and regular select on main entity", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ProductDescriptions?$expand=Product($select=Name)&$select=LanguageKey,Description", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		value := response["value"].([]interface{})
		if len(value) == 0 {
			t.Fatal("Expected at least one result")
		}

		item := value[0].(map[string]interface{})

		// Should have selected properties from main entity
		if _, ok := item["LanguageKey"]; !ok {
			t.Error("Expected LanguageKey property")
		}
		if _, ok := item["Description"]; !ok {
			t.Error("Expected Description property")
		}

		// Should have Product with only Name
		product, ok := item["Product"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected Product to be expanded")
		}

		if _, ok := product["Name"]; !ok {
			t.Error("Expected Product.Name property")
		}
		if _, ok := product["Price"]; ok {
			t.Error("Did not expect Product.Price property (not selected)")
		}
	})

	t.Run("Navigation path select without explicit expand", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ProductDescriptions?$select=LanguageKey,Product/Name", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		value := response["value"].([]interface{})
		if len(value) == 0 {
			t.Fatal("Expected at least one result")
		}

		item := value[0].(map[string]interface{})

		// Should have LanguageKey
		if _, ok := item["LanguageKey"]; !ok {
			t.Error("Expected LanguageKey property")
		}

		// Should have Product automatically expanded with Name
		product, ok := item["Product"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected Product to be auto-expanded when using Product/Name in select")
		}

		if _, ok := product["Name"]; !ok {
			t.Error("Expected Product.Name property")
		}
		if _, ok := product["Price"]; ok {
			t.Error("Did not expect Product.Price property (not selected)")
		}
	})

	t.Run("Single entity with expand and select", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ProductDescriptions(ProductID=1,LanguageKey='EN')?$expand=Product&$select=LanguageKey", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should have LanguageKey
		if _, ok := response["LanguageKey"]; !ok {
			t.Error("Expected LanguageKey property")
		}

		// Should have full Product entity
		product, ok := response["Product"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected Product to be expanded")
		}

		if _, ok := product["ID"]; !ok {
			t.Error("Expected Product.ID property")
		}
		if _, ok := product["Name"]; !ok {
			t.Error("Expected Product.Name property")
		}
		if _, ok := product["Price"]; !ok {
			t.Error("Expected Product.Price property")
		}
	})

	t.Run("Single entity with expand nested select", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ProductDescriptions(ProductID=1,LanguageKey='EN')?$expand=Product($select=Name)&$select=LanguageKey", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should have LanguageKey
		if _, ok := response["LanguageKey"]; !ok {
			t.Error("Expected LanguageKey property")
		}

		// Should have Product with only Name
		product, ok := response["Product"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected Product to be expanded")
		}

		if _, ok := product["Name"]; !ok {
			t.Error("Expected Product.Name property")
		}
		if _, ok := product["Price"]; ok {
			t.Error("Did not expect Product.Price property (not selected)")
		}
	})
}
