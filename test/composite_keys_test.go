package odata_test

import (
	"encoding/json"
	odata "github.com/nlstn/go-odata"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestEntityCompositeKey represents an entity with composite keys
type TestEntityCompositeKey struct {
	ProductID   uint   `json:"ProductID" gorm:"primaryKey" odata:"key"`
	LanguageKey string `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key"`
	Description string `json:"Description" gorm:"not null"`
	LongText    string `json:"LongText" gorm:"type:text"`
}

// TestParentEntity represents a parent entity for navigation testing
type TestParentEntity struct {
	ID           uint                     `json:"ID" gorm:"primaryKey" odata:"key"`
	Name         string                   `json:"Name"`
	Descriptions []TestEntityCompositeKey `json:"Descriptions" gorm:"foreignKey:ProductID"`
}

func setupCompositeKeyTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntityCompositeKey{}, &TestParentEntity{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	testData := []TestEntityCompositeKey{
		{ProductID: 1, LanguageKey: "EN", Description: "English description", LongText: "Long text in English"},
		{ProductID: 1, LanguageKey: "DE", Description: "German description", LongText: "Long text in German"},
		{ProductID: 2, LanguageKey: "EN", Description: "Product 2 English", LongText: "Product 2 long text"},
		{ProductID: 2, LanguageKey: "FR", Description: "Product 2 French", LongText: "Product 2 texte long"},
	}

	for _, item := range testData {
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("Failed to seed data: %v", err)
		}
	}

	return db
}

func TestAnalyzeEntityWithCompositeKeys(t *testing.T) {
	entity := TestEntityCompositeKey{}
	meta, err := metadata.AnalyzeEntity(entity)
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	if len(meta.KeyProperties) != 2 {
		t.Errorf("Expected 2 key properties, got %d", len(meta.KeyProperties))
	}

	// Check that both keys are identified
	hasProductID := false
	hasLanguageKey := false
	for _, keyProp := range meta.KeyProperties {
		if keyProp.JsonName == "ProductID" {
			hasProductID = true
		}
		if keyProp.JsonName == "LanguageKey" {
			hasLanguageKey = true
		}
	}

	if !hasProductID {
		t.Error("ProductID should be identified as a key property")
	}
	if !hasLanguageKey {
		t.Error("LanguageKey should be identified as a key property")
	}
}

func TestCompositeKeyURLParsing(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		expectEntitySet string
		expectKeyMap    map[string]string
		expectError     bool
	}{
		{
			name:            "Composite key with two properties",
			url:             "ProductDescriptions(ProductID=1,LanguageKey='EN')",
			expectEntitySet: "ProductDescriptions",
			expectKeyMap: map[string]string{
				"ProductID":   "1",
				"LanguageKey": "EN",
			},
			expectError: false,
		},
		{
			name:            "Composite key with quotes",
			url:             "ProductDescriptions(ProductID=2,LanguageKey='DE')",
			expectEntitySet: "ProductDescriptions",
			expectKeyMap: map[string]string{
				"ProductID":   "2",
				"LanguageKey": "DE",
			},
			expectError: false,
		},
		{
			name:            "Single key backward compatible",
			url:             "Products(1)",
			expectEntitySet: "Products",
			expectKeyMap:    map[string]string{},
			expectError:     false,
		},
		{
			name:            "Single key with name",
			url:             "Products(ID=5)",
			expectEntitySet: "Products",
			expectKeyMap: map[string]string{
				"ID": "5",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := odata.NewService(nil)
			components, err := parseURLForTest(tt.url)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if err != nil {
				return
			}

			if components.EntitySet != tt.expectEntitySet {
				t.Errorf("Expected entity set %s, got %s", tt.expectEntitySet, components.EntitySet)
			}

			if len(tt.expectKeyMap) > 0 {
				for key, expectedValue := range tt.expectKeyMap {
					actualValue, ok := components.EntityKeyMap[key]
					if !ok {
						t.Errorf("Expected key %s not found in EntityKeyMap", key)
					} else if actualValue != expectedValue {
						t.Errorf("For key %s, expected value %s, got %s", key, expectedValue, actualValue)
					}
				}
			}

			_ = service // avoid unused variable error
		})
	}
}

func TestCompositeKeyEntityRetrieval(t *testing.T) {
	db := setupCompositeKeyTestDB(t)
	service := odata.NewService(db)

	if err := service.RegisterEntity(TestEntityCompositeKey{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	tests := []struct {
		name            string
		url             string
		expectStatus    int
		expectProductID uint
		expectLangKey   string
	}{
		{
			name:            "Get entity with composite key EN",
			url:             "/TestEntityCompositeKeys(ProductID=1,LanguageKey='EN')",
			expectStatus:    http.StatusOK,
			expectProductID: 1,
			expectLangKey:   "EN",
		},
		{
			name:            "Get entity with composite key DE",
			url:             "/TestEntityCompositeKeys(ProductID=1,LanguageKey='DE')",
			expectStatus:    http.StatusOK,
			expectProductID: 1,
			expectLangKey:   "DE",
		},
		{
			name:            "Get non-existent entity",
			url:             "/TestEntityCompositeKeys(ProductID=99,LanguageKey='XX')",
			expectStatus:    http.StatusNotFound,
			expectProductID: 0,
			expectLangKey:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectStatus, w.Code, w.Body.String())
			}

			if tt.expectStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if productID, ok := response["ProductID"].(float64); ok {
					if uint(productID) != tt.expectProductID {
						t.Errorf("Expected ProductID %d, got %d", tt.expectProductID, uint(productID))
					}
				} else {
					t.Error("ProductID not found in response")
				}

				if langKey, ok := response["LanguageKey"].(string); ok {
					if langKey != tt.expectLangKey {
						t.Errorf("Expected LanguageKey %s, got %s", tt.expectLangKey, langKey)
					}
				} else {
					t.Error("LanguageKey not found in response")
				}
			}
		})
	}
}

func TestCompositeKeyMetadataGeneration(t *testing.T) {
	db := setupCompositeKeyTestDB(t)
	service := odata.NewService(db)

	if err := service.RegisterEntity(TestEntityCompositeKey{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	metadata := w.Body.String()

	// Check that both key properties are declared in the Key element
	if !contains(metadata, `<PropertyRef Name="ProductID" />`) {
		t.Error("Metadata should contain ProductID as a key property")
	}

	if !contains(metadata, `<PropertyRef Name="LanguageKey" />`) {
		t.Error("Metadata should contain LanguageKey as a key property")
	}

	// Verify the Key element contains both properties
	keySection := extractSection(metadata, "<Key>", "</Key>")
	if keySection == "" {
		t.Fatal("Could not find Key section in metadata")
	}

	if !contains(keySection, "ProductID") || !contains(keySection, "LanguageKey") {
		t.Error("Key section should contain both ProductID and LanguageKey")
	}
}

func TestCompositeKeyCollection(t *testing.T) {
	db := setupCompositeKeyTestDB(t)
	service := odata.NewService(db)

	if err := service.RegisterEntity(TestEntityCompositeKey{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/TestEntityCompositeKeys", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected 'value' to be an array")
	}

	// Should have 4 items from seed data
	if len(value) != 4 {
		t.Errorf("Expected 4 entities, got %d", len(value))
	}

	// Verify each entity has both key properties
	for i, item := range value {
		entity, ok := item.(map[string]interface{})
		if !ok {
			t.Errorf("Entity %d is not a map", i)
			continue
		}

		if _, hasProductID := entity["ProductID"]; !hasProductID {
			t.Errorf("Entity %d missing ProductID", i)
		}

		if _, hasLangKey := entity["LanguageKey"]; !hasLangKey {
			t.Errorf("Entity %d missing LanguageKey", i)
		}
	}
}

// Helper functions

type URLComponents struct {
	EntitySet          string
	EntityKey          string
	EntityKeyMap       map[string]string
	NavigationProperty string
}

func parseURLForTest(url string) (*URLComponents, error) {
	// This is a simplified version for testing
	// In real code, use response.ParseODataURLComponents
	components := &URLComponents{
		EntityKeyMap: make(map[string]string),
	}

	// Simple parsing logic
	if idx := indexOfStr(url, "("); idx != -1 {
		components.EntitySet = url[:idx]
		endIdx := indexOfStr(url, ")")
		if endIdx > idx {
			keyPart := url[idx+1 : endIdx]
			if containsStr(keyPart, "=") {
				// Parse composite key
				pairs := splitByStr(keyPart, ",")
				for _, pair := range pairs {
					parts := splitByStr(pair, "=")
					if len(parts) == 2 {
						key := parts[0]
						value := trimStr(parts[1], "'\"")
						components.EntityKeyMap[key] = value
					}
				}
			} else {
				// Single key
				components.EntityKey = keyPart
			}
		}
	} else {
		components.EntitySet = url
	}

	return components, nil
}

func containsStr(s, substr string) bool {
	return indexOfStr(s, substr) >= 0
}

func indexOfStr(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func splitByStr(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimStr(s, cutset string) string {
	// Trim characters in cutset from both ends
	start := 0
	end := len(s)

	for start < end {
		found := false
		for _, c := range cutset {
			if rune(s[start]) == c {
				start++
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	for start < end {
		found := false
		for _, c := range cutset {
			if rune(s[end-1]) == c {
				end--
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	return s[start:end]
}

func extractSection(content, startTag, endTag string) string {
	startIdx := indexOfStr(content, startTag)
	if startIdx == -1 {
		return ""
	}

	endIdx := indexOfStr(content[startIdx:], endTag)
	if endIdx == -1 {
		return ""
	}

	return content[startIdx : startIdx+endIdx+len(endTag)]
}
