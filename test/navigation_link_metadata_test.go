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

// TestNavigationLinkMetadataLevels tests that navigation links are only included
// when odata.metadata=full per OData v4 specification
func TestNavigationLinkMetadataLevels(t *testing.T) {
	// Setup test database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	type Category struct {
		ID   int    `json:"id" gorm:"primaryKey" odata:"key"`
		Name string `json:"name"`
	}

	type Product struct {
		ID         int       `json:"id" gorm:"primaryKey" odata:"key"`
		Name       string    `json:"name"`
		CategoryID int       `json:"categoryId"`
		Category   *Category `json:"Category,omitempty" gorm:"foreignKey:CategoryID"`
	}

	if err := db.AutoMigrate(&Category{}, &Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	db.Create(&Category{ID: 1, Name: "Electronics"})
	db.Create(&Product{ID: 1, Name: "Laptop", CategoryID: 1})
	db.Create(&Product{ID: 2, Name: "Mouse", CategoryID: 1})

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	service.RegisterEntity(&Category{})
	service.RegisterEntity(&Product{})

	tests := []struct {
		name                  string
		url                   string
		acceptHeader          string
		formatParam           string
		shouldHaveNavLinks    bool
		shouldHaveODataType   bool
		expectedMetadataLevel string
		description           string
	}{
		{
			name:                  "Default (minimal) - no navigation links",
			url:                   "/Products",
			shouldHaveNavLinks:    false,
			shouldHaveODataType:   false,
			expectedMetadataLevel: "minimal",
			description:           "Minimal metadata (default) should NOT include navigation links",
		},
		{
			name:                  "Explicit minimal - no navigation links",
			url:                   "/Products",
			acceptHeader:          "application/json;odata.metadata=minimal",
			shouldHaveNavLinks:    false,
			shouldHaveODataType:   false,
			expectedMetadataLevel: "minimal",
			description:           "Minimal metadata should NOT include navigation links",
		},
		{
			name:                  "Full metadata - with navigation links",
			url:                   "/Products",
			acceptHeader:          "application/json;odata.metadata=full",
			shouldHaveNavLinks:    true,
			shouldHaveODataType:   true,
			expectedMetadataLevel: "full",
			description:           "Full metadata SHOULD include navigation links",
		},
		{
			name:                  "None metadata - no navigation links",
			url:                   "/Products",
			acceptHeader:          "application/json;odata.metadata=none",
			shouldHaveNavLinks:    false,
			shouldHaveODataType:   false,
			expectedMetadataLevel: "none",
			description:           "None metadata should NOT include navigation links",
		},
		{
			name:                  "Full metadata via $format - with navigation links",
			url:                   "/Products",
			formatParam:           "application/json;odata.metadata=full",
			shouldHaveNavLinks:    true,
			shouldHaveODataType:   true,
			expectedMetadataLevel: "full",
			description:           "Full metadata via $format SHOULD include navigation links",
		},
		{
			name:                  "Single entity - default (minimal) - no navigation links",
			url:                   "/Products(1)",
			shouldHaveNavLinks:    false,
			shouldHaveODataType:   false,
			expectedMetadataLevel: "minimal",
			description:           "Single entity with minimal metadata should NOT include navigation links",
		},
		{
			name:                  "Single entity - full metadata - with navigation links",
			url:                   "/Products(1)",
			acceptHeader:          "application/json;odata.metadata=full",
			shouldHaveNavLinks:    true,
			shouldHaveODataType:   true,
			expectedMetadataLevel: "full",
			description:           "Single entity with full metadata SHOULD include navigation links",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
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
				t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
				return
			}

			// Verify Content-Type header includes correct metadata level
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "odata.metadata="+tt.expectedMetadataLevel) {
				t.Errorf("Expected Content-Type to contain odata.metadata=%s, got %s",
					tt.expectedMetadataLevel, contentType)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Check navigation links and @odata.type based on whether it's a collection or single entity
			if strings.Contains(tt.url, "Products(") {
				// Single entity response
				checkEntityForNavigationLinks(t, response, tt.shouldHaveNavLinks, tt.shouldHaveODataType, tt.description)
			} else {
				// Collection response
				value, ok := response["value"].([]interface{})
				if !ok || len(value) == 0 {
					t.Fatal("Expected value array in response")
				}

				// Check first entity
				firstEntity, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("Expected first entity to be a map")
				}

				checkEntityForNavigationLinks(t, firstEntity, tt.shouldHaveNavLinks, tt.shouldHaveODataType, tt.description)
			}
		})
	}
}

// checkEntityForNavigationLinks checks if an entity has navigation links as expected
func checkEntityForNavigationLinks(t *testing.T, entity map[string]interface{}, shouldHaveNavLinks, shouldHaveODataType bool, description string) {
	t.Helper()

	// Check for @odata.type
	_, hasODataType := entity["@odata.type"]
	if shouldHaveODataType && !hasODataType {
		t.Errorf("Expected @odata.type in entity (test: %s)", description)
	}
	if !shouldHaveODataType && hasODataType {
		t.Errorf("Did not expect @odata.type in entity (test: %s)", description)
	}

	// Check for navigation links (identified by @odata.navigationLink suffix)
	hasNavLinks := false
	for key := range entity {
		if strings.HasSuffix(key, "@odata.navigationLink") {
			hasNavLinks = true
			break
		}
	}

	if shouldHaveNavLinks && !hasNavLinks {
		t.Errorf("Expected navigation links in entity (test: %s)", description)
		t.Logf("Entity keys: %v", getKeys(entity))
	}
	if !shouldHaveNavLinks && hasNavLinks {
		t.Errorf("Did not expect navigation links in entity (test: %s)", description)
		t.Logf("Entity keys: %v", getKeys(entity))
	}

	// If navigation links are present, verify they're properly formatted
	if hasNavLinks {
		for key, value := range entity {
			if strings.HasSuffix(key, "@odata.navigationLink") {
				navLink, ok := value.(string)
				if !ok {
					t.Errorf("Navigation link %s should be a string", key)
					continue
				}

				// Verify the navigation link has expected format
				// Should be like: http://example.com/Products(1)/Category
				if !strings.Contains(navLink, "/Products(") || !strings.Contains(navLink, ")/") {
					t.Errorf("Navigation link %s has unexpected format: %s", key, navLink)
				}
			}
		}
	}
}

// getKeys returns the keys of a map as a slice
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestNavigationLinkWithExpand tests that navigation links are NOT included
// when the property is expanded, regardless of metadata level
func TestNavigationLinkWithExpand(t *testing.T) {
	// Setup test database and service
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	type Category struct {
		ID   int    `json:"id" gorm:"primaryKey" odata:"key"`
		Name string `json:"name"`
	}

	type Product struct {
		ID         int       `json:"id" gorm:"primaryKey" odata:"key"`
		Name       string    `json:"name"`
		CategoryID int       `json:"categoryId"`
		Category   *Category `json:"Category,omitempty" gorm:"foreignKey:CategoryID"`
	}

	if err := db.AutoMigrate(&Category{}, &Product{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Insert test data
	db.Create(&Category{ID: 1, Name: "Electronics"})
	db.Create(&Product{ID: 1, Name: "Laptop", CategoryID: 1})

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	service.RegisterEntity(&Category{})
	service.RegisterEntity(&Product{})

	tests := []struct {
		name         string
		url          string
		acceptHeader string
		description  string
	}{
		{
			name:        "Expanded with minimal metadata - no nav link",
			url:         "/Products?$expand=Category",
			description: "Expanded properties should not have navigation links even in minimal",
		},
		{
			name:         "Expanded with full metadata - no nav link",
			url:          "/Products?$expand=Category",
			acceptHeader: "application/json;odata.metadata=full",
			description:  "Expanded properties should not have navigation links even in full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
				return
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok || len(value) == 0 {
				t.Fatal("Expected value array in response")
			}

			firstEntity, ok := value[0].(map[string]interface{})
			if !ok {
				t.Fatal("Expected first entity to be a map")
			}

			// Category should be expanded (present as object)
			category, ok := firstEntity["Category"]
			if !ok {
				t.Error("Expected Category to be expanded")
			} else {
				// Verify it's an object, not a navigation link
				if _, isMap := category.(map[string]interface{}); !isMap {
					t.Errorf("Expected Category to be an object, got %T", category)
				}
			}

			// Should NOT have Category@odata.navigationLink
			if _, hasNavLink := firstEntity["Category@odata.navigationLink"]; hasNavLink {
				t.Error("Should NOT have Category@odata.navigationLink when property is expanded")
			}
		})
	}
}
