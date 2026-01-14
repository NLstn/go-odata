package odata_test

import (
	"encoding/json"
	odata "github.com/nlstn/go-odata"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ExpandTestProduct represents a product entity for testing expand with filter
type ExpandTestProduct struct {
	ID           uint                           `json:"ID" gorm:"primaryKey" odata:"key"`
	Name         string                         `json:"Name" gorm:"not null"`
	Price        float64                        `json:"Price" gorm:"not null"`
	Category     string                         `json:"Category" gorm:"not null"`
	Descriptions []ExpandTestProductDescription `json:"Descriptions" gorm:"foreignKey:ProductID"`
}

// ExpandTestProductDescription represents a multilingual product description entity
type ExpandTestProductDescription struct {
	ProductID   uint   `json:"ProductID" gorm:"primaryKey" odata:"key"`
	LanguageKey string `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key"`
	Description string `json:"Description" gorm:"not null"`
	LongText    string `json:"LongText" gorm:"type:text"`
}

func setupExpandFilterTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the models
	if err := db.AutoMigrate(&ExpandTestProduct{}, &ExpandTestProductDescription{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	products := []ExpandTestProduct{
		{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics"},
	}

	descriptions := []ExpandTestProductDescription{
		{ProductID: 1, LanguageKey: "EN", Description: "High-performance laptop", LongText: "A great laptop for work"},
		{ProductID: 1, LanguageKey: "DE", Description: "Hochleistungs-Laptop", LongText: "Ein großartiger Laptop für die Arbeit"},
		{ProductID: 1, LanguageKey: "FR", Description: "Ordinateur portable haute performance", LongText: "Un excellent ordinateur pour le travail"},
		{ProductID: 2, LanguageKey: "EN", Description: "Wireless mouse", LongText: "Ergonomic wireless mouse"},
		{ProductID: 2, LanguageKey: "DE", Description: "Kabellose Maus", LongText: "Ergonomische kabellose Maus"},
	}

	if err := db.Create(&products).Error; err != nil {
		t.Fatalf("Failed to seed products: %v", err)
	}

	if err := db.Create(&descriptions).Error; err != nil {
		t.Fatalf("Failed to seed descriptions: %v", err)
	}

	return db
}

func TestExpandWithFilterIntegration(t *testing.T) {
	db := setupExpandFilterTestDB(t)

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&ExpandTestProduct{}); err != nil {
		t.Fatalf("Failed to register ExpandTestProduct entity: %v", err)
	}
	if err := service.RegisterEntity(&ExpandTestProductDescription{}); err != nil {
		t.Fatalf("Failed to register ExpandTestProductDescription entity: %v", err)
	}

	// Test case: Expand descriptions with filter for German language
	t.Run("Filter German descriptions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ExpandTestProducts?$expand=Descriptions($filter=LanguageKey%20eq%20%27DE%27)", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		value, ok := response["value"].([]interface{})
		if !ok {
			t.Fatal("Response does not contain value array")
		}

		if len(value) != 2 {
			t.Fatalf("Expected 2 products, got %d", len(value))
		}

		// Check first product (Laptop)
		product1 := value[0].(map[string]interface{})
		descriptions1, ok := product1["Descriptions"].([]interface{})
		if !ok {
			t.Fatal("Product 1 does not contain Descriptions array")
		}

		if len(descriptions1) != 1 {
			t.Fatalf("Expected 1 German description for product 1, got %d", len(descriptions1))
		}

		desc1 := descriptions1[0].(map[string]interface{})
		if desc1["LanguageKey"] != "DE" {
			t.Errorf("Expected German description, got %s", desc1["LanguageKey"])
		}
		if desc1["Description"] != "Hochleistungs-Laptop" {
			t.Errorf("Unexpected description: %s", desc1["Description"])
		}

		// Check second product (Mouse)
		product2 := value[1].(map[string]interface{})
		descriptions2, ok := product2["Descriptions"].([]interface{})
		if !ok {
			t.Fatal("Product 2 does not contain Descriptions array")
		}

		if len(descriptions2) != 1 {
			t.Fatalf("Expected 1 German description for product 2, got %d", len(descriptions2))
		}

		desc2 := descriptions2[0].(map[string]interface{})
		if desc2["LanguageKey"] != "DE" {
			t.Errorf("Expected German description, got %s", desc2["LanguageKey"])
		}
	})

	// Test case: Expand descriptions with filter for English language
	t.Run("Filter English descriptions", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ExpandTestProducts?$expand=Descriptions($filter=LanguageKey%20eq%20%27EN%27)", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		value := response["value"].([]interface{})
		for _, v := range value {
			product := v.(map[string]interface{})
			descriptions := product["Descriptions"].([]interface{})

			for _, d := range descriptions {
				desc := d.(map[string]interface{})
				if desc["LanguageKey"] != "EN" {
					t.Errorf("Expected only English descriptions, got %s", desc["LanguageKey"])
				}
			}
		}
	})

	// Test case: Expand with filter and top
	t.Run("Filter with top", func(t *testing.T) {
		// First test with just filter (no top) to see if it works
		req := httptest.NewRequest("GET", "/ExpandTestProducts(1)?$expand=Descriptions($filter=LanguageKey%20ne%20%27FR%27)", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		descriptions, ok := response["Descriptions"].([]interface{})
		if !ok {
			t.Fatalf("Response does not contain Descriptions array. Response: %+v", response)
		}

		// Should have 2 descriptions (EN and DE), not FR
		if len(descriptions) != 2 {
			t.Fatalf("Expected 2 descriptions (EN and DE, not FR), got %d", len(descriptions))
		}

		// Verify no French descriptions
		for _, d := range descriptions {
			desc := d.(map[string]interface{})
			if desc["LanguageKey"] == "FR" {
				t.Error("Expected non-French description due to filter")
			}
		}
	})

	// Test case: Expand without filter (should return all descriptions)
	t.Run("No filter returns all", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ExpandTestProducts(1)?$expand=Descriptions", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		descriptions, ok := response["Descriptions"].([]interface{})
		if !ok {
			t.Fatal("Response does not contain Descriptions array")
		}

		// Product 1 has 3 descriptions (EN, DE, FR)
		if len(descriptions) != 3 {
			t.Fatalf("Expected 3 descriptions without filter, got %d", len(descriptions))
		}
	})

	// Test case: Expand with orderby
	t.Run("Filter with orderby", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ExpandTestProducts(1)?$expand=Descriptions($orderby=LanguageKey%20desc)", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		descriptions, ok := response["Descriptions"].([]interface{})
		if !ok {
			t.Fatal("Response does not contain Descriptions array")
		}

		// Should be ordered by LanguageKey descending: FR, EN, DE
		if len(descriptions) == 3 {
			firstLang := descriptions[0].(map[string]interface{})["LanguageKey"].(string)
			lastLang := descriptions[2].(map[string]interface{})["LanguageKey"].(string)

			// FR should come before DE when sorted descending
			if firstLang == "DE" || lastLang == "FR" {
				t.Errorf("Expected descending order by LanguageKey, got first=%s, last=%s", firstLang, lastLang)
			}
		}
	})

	t.Run("Select navigation property includes navigation link but does not expand", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/ExpandTestProducts?$select=Name,Descriptions", nil)
		req.Header.Set("Accept", "application/json;odata.metadata=minimal")
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

		// Should have Name property
		if _, ok := item["Name"]; !ok {
			t.Error("Expected Name property")
		}

		// Should NOT have Descriptions data (not expanded without $expand)
		if descriptions, ok := item["Descriptions"]; ok {
			if _, isArray := descriptions.([]interface{}); isArray {
				t.Error("Expected Descriptions to not be expanded without $expand")
			}
		}

		// Should have Descriptions@odata.navigationLink
		if navLink, ok := item["Descriptions@odata.navigationLink"]; !ok {
			t.Error("Expected Descriptions@odata.navigationLink to be present for minimal metadata with $select")
		} else if navLinkStr, ok := navLink.(string); ok {
			if navLinkStr == "" {
				t.Error("Expected Descriptions@odata.navigationLink to be non-empty")
			}
		} else {
			t.Errorf("Expected Descriptions@odata.navigationLink to be a string, got %T", navLink)
		}
	})
}
