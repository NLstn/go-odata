package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestNavigationPropertyQueryOptions tests query options on collection navigation properties
func TestNavigationPropertyQueryOptions(t *testing.T) {
	// Setup
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Define test entities
	type ProductDescription struct {
		ProductID   uint   `json:"ProductID" gorm:"primaryKey" odata:"key"`
		LanguageKey string `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key,maxlength=2"`
		Description string `json:"Description" gorm:"not null" odata:"required"`
	}

	type Product struct {
		ID           uint                 `json:"ID" gorm:"primaryKey" odata:"key"`
		Name         string               `json:"Name" gorm:"not null" odata:"required"`
		Descriptions []ProductDescription `json:"Descriptions" gorm:"foreignKey:ProductID;references:ID"`
	}

	// Migrate and seed
	if err := db.AutoMigrate(&Product{}, &ProductDescription{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	products := []Product{
		{ID: 1, Name: "Product 1"},
		{ID: 2, Name: "Product 2"},
	}

	descriptions := []ProductDescription{
		{ProductID: 1, LanguageKey: "EN", Description: "English description"},
		{ProductID: 1, LanguageKey: "DE", Description: "German description"},
		{ProductID: 1, LanguageKey: "FR", Description: "French description"},
		{ProductID: 2, LanguageKey: "EN", Description: "Product 2 English"},
	}

	for _, p := range products {
		db.Create(&p)
	}
	for _, d := range descriptions {
		db.Create(&d)
	}

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&Product{})
	service.RegisterEntity(&ProductDescription{})

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		validate       func(t *testing.T, body []byte)
	}{
		{
			name:           "Navigation property without query options",
			url:            "/Products(1)/Descriptions",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 3 {
					t.Errorf("Expected 3 descriptions, got %d", len(values))
				}
			},
		},
		{
			name:           "Navigation property with $filter",
			url:            "/Products(1)/Descriptions?%24filter=LanguageKey%20eq%20%27EN%27",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 1 {
					t.Errorf("Expected 1 description, got %d", len(values))
				}
				if len(values) > 0 {
					desc := values[0].(map[string]interface{})
					if desc["LanguageKey"] != "EN" {
						t.Errorf("Expected LanguageKey 'EN', got '%v'", desc["LanguageKey"])
					}
				}
			},
		},
		{
			name:           "Navigation property with $count",
			url:            "/Products(1)/Descriptions?%24count=true",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				count, ok := response["@odata.count"]
				if !ok {
					t.Error("Expected @odata.count in response")
				}
				if count != float64(3) {
					t.Errorf("Expected count 3, got %v", count)
				}
			},
		},
		{
			name:           "Navigation property with $orderby asc",
			url:            "/Products(1)/Descriptions?%24orderby=LanguageKey",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) < 2 {
					t.Fatal("Expected at least 2 descriptions")
				}
				first := values[0].(map[string]interface{})["LanguageKey"].(string)
				second := values[1].(map[string]interface{})["LanguageKey"].(string)
				if first != "DE" || second != "EN" {
					t.Errorf("Expected DE, EN order, got %s, %s", first, second)
				}
			},
		},
		{
			name:           "Navigation property with $orderby desc",
			url:            "/Products(1)/Descriptions?%24orderby=LanguageKey%20desc",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) < 2 {
					t.Fatal("Expected at least 2 descriptions")
				}
				first := values[0].(map[string]interface{})["LanguageKey"].(string)
				if first != "FR" {
					t.Errorf("Expected first item to be FR, got %s", first)
				}
			},
		},
		{
			name:           "Navigation property with $top",
			url:            "/Products(1)/Descriptions?%24top=2",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 2 {
					t.Errorf("Expected 2 descriptions, got %d", len(values))
				}

				nextLink, ok := response["@odata.nextLink"].(string)
				if !ok {
					t.Fatalf("Expected @odata.nextLink in response, got %v", response["@odata.nextLink"])
				}

				parsed, err := url.Parse(nextLink)
				if err != nil {
					t.Fatalf("Failed to parse @odata.nextLink: %v", err)
				}
				if token := parsed.Query().Get("$skiptoken"); token == "" {
					t.Fatalf("Expected $skiptoken query parameter in nextLink, got %s", nextLink)
				}
			},
		},
		{
			name:           "Navigation property with $skip",
			url:            "/Products(1)/Descriptions?%24skip=1",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 2 {
					t.Errorf("Expected 2 descriptions (3 - 1 skipped), got %d", len(values))
				}
			},
		},
		{
			name:           "Navigation property with combined query options",
			url:            "/Products(1)/Descriptions?%24filter=contains(Description,%27description%27)&%24orderby=LanguageKey%20desc&%24count=true&%24top=2",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				count := response["@odata.count"].(float64)
				if count != 3 {
					t.Errorf("Expected count 3, got %v", count)
				}
				values := response["value"].([]interface{})
				if len(values) > 2 {
					t.Errorf("Expected at most 2 results due to $top, got %d", len(values))
				}
			},
		},
		{
			name:           "Navigation property on non-existent parent returns 404",
			url:            "/Products(999)/Descriptions",
			expectedStatus: http.StatusNotFound,
			validate:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.validate != nil && w.Code == http.StatusOK {
				tt.validate(t, w.Body.Bytes())
			}
		})
	}
}

// TestNavigationPropertyQueryOptionsODataContext tests that navigation properties have correct @odata.context
func TestNavigationPropertyQueryOptionsODataContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	type ProductDescription struct {
		ProductID   uint   `json:"ProductID" gorm:"primaryKey" odata:"key"`
		LanguageKey string `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key"`
		Description string `json:"Description"`
	}

	type Product struct {
		ID           uint                 `json:"ID" gorm:"primaryKey" odata:"key"`
		Name         string               `json:"Name"`
		Descriptions []ProductDescription `json:"Descriptions" gorm:"foreignKey:ProductID;references:ID"`
	}

	db.AutoMigrate(&Product{}, &ProductDescription{})
	db.Create(&Product{ID: 1, Name: "Test"})
	db.Create(&ProductDescription{ProductID: 1, LanguageKey: "EN", Description: "Test"})

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&Product{})
	service.RegisterEntity(&ProductDescription{})

	req := httptest.NewRequest(http.MethodGet, "/Products(1)/Descriptions", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("Missing @odata.context in response")
	}

	expectedContextSuffix := "Products(1)/Descriptions"
	if !contains(context, expectedContextSuffix) {
		t.Errorf("Expected context to contain '%s', got '%s'", expectedContextSuffix, context)
	}
}
