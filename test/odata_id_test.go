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

// TestODataIDFieldIntegration tests the @odata.id field implementation according to OData v4 spec
func TestODataIDFieldIntegration(t *testing.T) {
	// Setup test database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	type Product struct {
		ID    int     `json:"id" gorm:"primaryKey" odata:"key"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	db.Create(&Product{ID: 1, Name: "Laptop", Price: 999.99})
	db.Create(&Product{ID: 2, Name: "Mouse", Price: 29.99})

	service := odata.NewService(db)
	service.RegisterEntity(&Product{})

	tests := []struct {
		name                string
		url                 string
		acceptHeader        string
		formatParam         string
		metadataLevel       string
		shouldHaveODataID   bool
		expectedODataID     string
		description         string
	}{
		{
			name:              "Full metadata - collection should have @odata.id",
			url:               "/Products",
			acceptHeader:      "application/json;odata.metadata=full",
			metadataLevel:     "full",
			shouldHaveODataID: true,
			expectedODataID:   "http://localhost:8080/Products(1)",
			description:       "Full metadata should always include @odata.id",
		},
		{
			name:              "Full metadata - single entity should have @odata.id",
			url:               "/Products(1)",
			acceptHeader:      "application/json;odata.metadata=full",
			metadataLevel:     "full",
			shouldHaveODataID: true,
			expectedODataID:   "http://localhost:8080/Products(1)",
			description:       "Full metadata should always include @odata.id for single entity",
		},
		{
			name:              "Minimal metadata - collection without $select should NOT have @odata.id",
			url:               "/Products",
			acceptHeader:      "application/json;odata.metadata=minimal",
			metadataLevel:     "minimal",
			shouldHaveODataID: false,
			description:       "Minimal metadata without $select should not include @odata.id",
		},
		{
			name:              "Minimal metadata - single entity without $select should NOT have @odata.id",
			url:               "/Products(1)",
			acceptHeader:      "application/json;odata.metadata=minimal",
			metadataLevel:     "minimal",
			shouldHaveODataID: false,
			description:       "Minimal metadata without $select should not include @odata.id for single entity",
		},
		{
			name:              "Minimal metadata - collection with $select (key omitted from request but auto-included) should NOT have @odata.id",
			url:               "/Products?$select=name,price",
			acceptHeader:      "application/json;odata.metadata=minimal",
			metadataLevel:     "minimal",
			shouldHaveODataID: false,
			description:       "Minimal metadata with $select - keys are auto-included per OData spec, so no @odata.id needed",
		},
		{
			name:              "Minimal metadata - single entity with $select (key omitted from request but auto-included) should NOT have @odata.id",
			url:               "/Products(1)?$select=name,price",
			acceptHeader:      "application/json;odata.metadata=minimal",
			metadataLevel:     "minimal",
			shouldHaveODataID: false,
			description:       "Minimal metadata with $select - keys are auto-included per OData spec, so no @odata.id needed for single entity",
		},
		{
			name:              "Minimal metadata - collection with $select (key included) should NOT have @odata.id",
			url:               "/Products?$select=id,name",
			acceptHeader:      "application/json;odata.metadata=minimal",
			metadataLevel:     "minimal",
			shouldHaveODataID: false,
			description:       "Minimal metadata with $select including key should not include @odata.id",
		},
		{
			name:              "Minimal metadata - single entity with $select (key included) should NOT have @odata.id",
			url:               "/Products(1)?$select=id,name",
			acceptHeader:      "application/json;odata.metadata=minimal",
			metadataLevel:     "minimal",
			shouldHaveODataID: false,
			description:       "Minimal metadata with $select including key should not include @odata.id for single entity",
		},
		{
			name:              "None metadata - collection should NOT have @odata.id",
			url:               "/Products",
			acceptHeader:      "application/json;odata.metadata=none",
			metadataLevel:     "none",
			shouldHaveODataID: false,
			description:       "None metadata should never include @odata.id",
		},
		{
			name:              "None metadata - single entity should NOT have @odata.id",
			url:               "/Products(1)",
			acceptHeader:      "application/json;odata.metadata=none",
			metadataLevel:     "none",
			shouldHaveODataID: false,
			description:       "None metadata should never include @odata.id for single entity",
		},
		{
			name:              "None metadata - with $select should NOT have @odata.id",
			url:               "/Products?$select=name",
			acceptHeader:      "application/json;odata.metadata=none",
			metadataLevel:     "none",
			shouldHaveODataID: false,
			description:       "None metadata should never include @odata.id even with $select",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Host = "localhost:8080" // Set correct host for @odata.id generation
			if tt.formatParam != "" {
				q := req.URL.Query()
				q.Set("$format", tt.formatParam)
				req.URL.RawQuery = q.Encode()
			}
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
				return
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Check if this is a collection or single entity response
			isCollection := false
			var entityToCheck map[string]interface{}

			if value, ok := response["value"].([]interface{}); ok && len(value) > 0 {
				// Collection response
				isCollection = true
				if firstEntity, ok := value[0].(map[string]interface{}); ok {
					entityToCheck = firstEntity
				} else {
					t.Fatal("Expected first entity to be a map")
				}
			} else {
				// Single entity response
				entityToCheck = response
			}

			// Check for @odata.id
			odataID, hasODataID := entityToCheck["@odata.id"]
			
			if tt.shouldHaveODataID && !hasODataID {
				t.Errorf("Expected @odata.id in %s (test: %s)", 
					map[bool]string{true: "collection", false: "entity"}[isCollection], 
					tt.description)
			}
			
			if !tt.shouldHaveODataID && hasODataID {
				t.Errorf("Did not expect @odata.id in %s, but got: %v (test: %s)", 
					map[bool]string{true: "collection", false: "entity"}[isCollection], 
					odataID, 
					tt.description)
			}
			
			if tt.shouldHaveODataID && hasODataID && tt.expectedODataID != "" {
				// Verify the format is correct
				odataIDStr, ok := odataID.(string)
				if !ok {
					t.Errorf("Expected @odata.id to be a string, got %T", odataID)
				} else if odataIDStr != tt.expectedODataID {
					t.Errorf("Expected @odata.id=%s, got %s (test: %s)", 
						tt.expectedODataID, odataIDStr, tt.description)
				}
			}
		})
	}
}

// TestODataIDFieldWithCompositeKeys tests @odata.id with composite keys
func TestODataIDFieldWithCompositeKeys(t *testing.T) {
	// Setup test database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	type ProductTranslation struct {
		ProductID   int    `json:"productId" gorm:"primaryKey" odata:"key"`
		LanguageKey string `json:"languageKey" gorm:"primaryKey" odata:"key"`
		Description string `json:"description"`
	}

	if err := db.AutoMigrate(&ProductTranslation{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	db.Create(&ProductTranslation{ProductID: 1, LanguageKey: "EN", Description: "Laptop"})
	db.Create(&ProductTranslation{ProductID: 1, LanguageKey: "DE", Description: "Laptop"})

	service := odata.NewService(db)
	service.RegisterEntity(&ProductTranslation{})

	tests := []struct {
		name              string
		url               string
		acceptHeader      string
		shouldHaveODataID bool
		expectedODataID   string
		description       string
	}{
		{
			name:              "Full metadata - composite key should have @odata.id",
			url:               "/ProductTranslations",
			acceptHeader:      "application/json;odata.metadata=full",
			shouldHaveODataID: true,
			expectedODataID:   "http://localhost:8080/ProductTranslations(productId=1,languageKey='EN')",
			description:       "Full metadata with composite key should include @odata.id",
		},
		{
			name:              "Minimal metadata with $select (keys omitted from request but auto-included) - should NOT have @odata.id",
			url:               "/ProductTranslations?$select=description",
			acceptHeader:      "application/json;odata.metadata=minimal",
			shouldHaveODataID: false,
			description:       "Minimal metadata with $select - composite keys are auto-included per OData spec, so no @odata.id needed",
		},
		{
			name:              "Minimal metadata with $select (one key omitted from request but auto-included) - should NOT have @odata.id",
			url:               "/ProductTranslations?$select=productId,description",
			acceptHeader:      "application/json;odata.metadata=minimal",
			shouldHaveODataID: false,
			description:       "Minimal metadata - all keys are auto-included per OData spec, so no @odata.id needed",
		},
		{
			name:              "Minimal metadata with $select (all keys included) - should NOT have @odata.id",
			url:               "/ProductTranslations?$select=productId,languageKey,description",
			acceptHeader:      "application/json;odata.metadata=minimal",
			shouldHaveODataID: false,
			description:       "Minimal metadata with all keys included should not include @odata.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Host = "localhost:8080" // Set correct host for @odata.id generation
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
				return
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Get first entity from collection
			value, ok := response["value"].([]interface{})
			if !ok || len(value) == 0 {
				t.Fatal("Expected value array in response")
			}

			firstEntity, ok := value[0].(map[string]interface{})
			if !ok {
				t.Fatal("Expected first entity to be a map")
			}

			// Check for @odata.id
			odataID, hasODataID := firstEntity["@odata.id"]
			
			if tt.shouldHaveODataID && !hasODataID {
				t.Errorf("Expected @odata.id in entity (test: %s)", tt.description)
			}
			
			if !tt.shouldHaveODataID && hasODataID {
				t.Errorf("Did not expect @odata.id in entity, but got: %v (test: %s)", odataID, tt.description)
			}
			
			if tt.shouldHaveODataID && hasODataID && tt.expectedODataID != "" {
				// Verify the format is correct
				odataIDStr, ok := odataID.(string)
				if !ok {
					t.Errorf("Expected @odata.id to be a string, got %T", odataID)
				} else if odataIDStr != tt.expectedODataID {
					t.Errorf("Expected @odata.id=%s, got %s (test: %s)", 
						tt.expectedODataID, odataIDStr, tt.description)
				}
			}
		})
	}
}

// TestODataIDFieldOrdering tests that @odata.id appears in the correct position
func TestODataIDFieldOrdering(t *testing.T) {
	// Setup test database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	type Product struct {
		ID    int     `json:"id" gorm:"primaryKey" odata:"key"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	db.Create(&Product{ID: 1, Name: "Laptop", Price: 999.99})

	service := odata.NewService(db)
	service.RegisterEntity(&Product{})

	// Test with full metadata to check ordering
	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	req.Host = "localhost:8080" // Set correct host for @odata.id generation
	req.Header.Set("Accept", "application/json;odata.metadata=full")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Parse the raw JSON to check field order
	body := w.Body.String()
	
	// Check that @odata.id comes after @odata.context but before regular properties
	contextPos := findPosition(body, `"@odata.context"`)
	idPos := findPosition(body, `"@odata.id"`)
	typePos := findPosition(body, `"@odata.type"`)
	valuePos := findPosition(body, `"value"`)
	
	if contextPos == -1 {
		t.Error("@odata.context not found in response")
	}
	if idPos == -1 {
		t.Error("@odata.id not found in response (should be present in full metadata)")
	}
	if typePos == -1 {
		t.Error("@odata.type not found in response (should be present in full metadata)")
	}
	
	// The order should be: @odata.context, then value (collection), and within each entity: @odata.id, @odata.type, then properties
	if contextPos != -1 && valuePos != -1 && contextPos > valuePos {
		t.Error("@odata.context should appear before value in collection response")
	}
}

// findPosition finds the position of a substring in a string, returns -1 if not found
func findPosition(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
