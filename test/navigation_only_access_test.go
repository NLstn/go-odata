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

// NavOnlyOrder is a parent entity that has a navigation property to NavOnlyItem.
type NavOnlyOrder struct {
	ID    int           `json:"ID" gorm:"primarykey" odata:"key"`
	Name  string        `json:"Name"`
	Items []NavOnlyItem `json:"Items,omitempty" gorm:"foreignKey:OrderID"`
}

// NavOnlyItem is a child entity that is only accessible via navigation from its parent.
type NavOnlyItem struct {
	ID      int    `json:"ID" gorm:"primarykey" odata:"key"`
	OrderID int    `json:"OrderID"`
	Name    string `json:"Name"`
}

// IsAccessibleOnlyViaNavigation marks this entity as only accessible via parent navigation.
func (NavOnlyItem) IsAccessibleOnlyViaNavigation() bool {
	return true
}

func setupNavOnlyTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NavOnlyOrder{}, &NavOnlyItem{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(NavOnlyOrder{}); err != nil {
		t.Fatalf("Failed to register NavOnlyOrder: %v", err)
	}
	if err := service.RegisterEntity(NavOnlyItem{}); err != nil {
		t.Fatalf("Failed to register NavOnlyItem: %v", err)
	}

	return service, db
}

// TestNavOnly_DirectCollectionAccessReturns404 verifies that directly accessing the
// collection of an entity marked as IsAccessibleOnlyViaNavigation returns 404.
func TestNavOnly_DirectCollectionAccessReturns404(t *testing.T) {
	service, _ := setupNavOnlyTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/NavOnlyItems", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for direct collection access, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestNavOnly_DirectEntityAccessReturns404 verifies that directly accessing an individual
// entity marked as IsAccessibleOnlyViaNavigation returns 404.
func TestNavOnly_DirectEntityAccessReturns404(t *testing.T) {
	service, _ := setupNavOnlyTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/NavOnlyItems(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for direct entity access, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestNavOnly_DirectCountAccessReturns404 verifies that directly accessing the $count of
// an entity marked as IsAccessibleOnlyViaNavigation returns 404.
func TestNavOnly_DirectCountAccessReturns404(t *testing.T) {
	service, _ := setupNavOnlyTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/NavOnlyItems/$count", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for direct $count access, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestNavOnly_PostToCollectionReturns404 verifies that POST to the collection of an entity
// marked as IsAccessibleOnlyViaNavigation returns 404.
func TestNavOnly_PostToCollectionReturns404(t *testing.T) {
	service, _ := setupNavOnlyTestService(t)

	body := `{"ID":1,"OrderID":1,"Name":"item"}`
	req := httptest.NewRequest(http.MethodPost, "/NavOnlyItems", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for POST to nav-only collection, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestNavOnly_NotInServiceDocument verifies that entities marked as IsAccessibleOnlyViaNavigation
// do not appear in the service document.
func TestNavOnly_NotInServiceDocument(t *testing.T) {
	service, _ := setupNavOnlyTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Service document returned %d, want 200. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode service document: %v", err)
	}

	value, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("value is missing or not an array in service document")
	}

	for _, item := range value {
		entity, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := entity["name"].(string)
		if name == "NavOnlyItems" {
			t.Error("NavOnlyItems should not appear in the service document since it is only accessible via navigation")
		}
	}

	// Verify that the parent entity IS in the service document
	foundOrders := false
	for _, item := range value {
		entity, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := entity["name"].(string)
		if name == "NavOnlyOrders" {
			foundOrders = true
		}
	}
	if !foundOrders {
		t.Error("NavOnlyOrders should appear in the service document")
	}
}

// TestNavOnly_NotInXMLMetadataEntityContainer verifies that entities marked as
// IsAccessibleOnlyViaNavigation do not appear in the XML metadata EntityContainer.
func TestNavOnly_NotInXMLMetadataEntityContainer(t *testing.T) {
	service, _ := setupNavOnlyTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/xml")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Metadata returned %d, want 200. Body: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// The EntityContainer must not have an EntitySet for NavOnlyItems
	if strings.Contains(body, `<EntitySet Name="NavOnlyItems"`) {
		t.Error("NavOnlyItems should not appear as an EntitySet in the XML metadata EntityContainer")
	}
	// But the EntityType definition is fine (it can still be a target of navigation properties)
	if !strings.Contains(body, `<EntityType Name="NavOnlyItem"`) {
		t.Error("NavOnlyItem EntityType should still be defined in the XML metadata schema")
	}
}

// TestNavOnly_NotInJSONMetadataEntityContainer verifies that entities marked as
// IsAccessibleOnlyViaNavigation do not appear in the JSON metadata EntityContainer.
func TestNavOnly_NotInJSONMetadataEntityContainer(t *testing.T) {
	service, _ := setupNavOnlyTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("JSON metadata returned %d, want 200. Body: %s", w.Code, w.Body.String())
	}

	var metadata map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&metadata); err != nil {
		t.Fatalf("Failed to decode JSON metadata: %v", err)
	}

	// Navigate to the EntityContainer using $EntityContainer (e.g. "ODataService.Container")
	containerRef, _ := metadata["$EntityContainer"].(string)
	if containerRef == "" {
		t.Skip("No $EntityContainer key in JSON metadata - skipping container check")
	}

	// containerRef is "Namespace.Container" - split to navigate
	parts := strings.SplitN(containerRef, ".", 2)
	if len(parts) != 2 {
		t.Skipf("Unexpected $EntityContainer format: %s", containerRef)
	}

	namespaceObj, ok := metadata[parts[0]].(map[string]interface{})
	if !ok {
		t.Skipf("Namespace object '%s' not found in JSON metadata", parts[0])
	}

	container, ok := namespaceObj[parts[1]].(map[string]interface{})
	if !ok {
		t.Skipf("Container '%s' not found in namespace object", parts[1])
	}

	if _, exists := container["NavOnlyItems"]; exists {
		t.Error("NavOnlyItems should not appear as an entity set in the JSON metadata EntityContainer")
	}
}

// TestNavOnly_ParentEntityIsAccessible verifies that the parent entity that has a
// navigation property to a nav-only entity is still fully accessible.
func TestNavOnly_ParentEntityIsAccessible(t *testing.T) {
	service, db := setupNavOnlyTestService(t)

	// Create test data
	db.Create(&NavOnlyOrder{ID: 1, Name: "Order 1"})

	req := httptest.NewRequest(http.MethodGet, "/NavOnlyOrders", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for parent entity collection, got %d. Body: %s", w.Code, w.Body.String())
	}
}
