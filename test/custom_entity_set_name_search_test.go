package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewsItem represents a news entity with custom EntitySetName
// This reproduces the issue where custom entity set names cause incorrect table name resolution
type NewsItem struct {
	ID        string    `json:"ID" gorm:"primaryKey;type:varchar(36)" odata:"key"`
	ClubID    string    `json:"ClubID" gorm:"type:varchar(36);not null" odata:"required"`
	Title     string    `json:"Title" gorm:"not null" odata:"required,searchable"`
	Content   string    `json:"Content" gorm:"type:text;not null" odata:"required,searchable"`
	CreatedAt time.Time `json:"CreatedAt" odata:"immutable"`
	CreatedBy string    `json:"CreatedBy" gorm:"type:varchar(36)" odata:"required"`
	UpdatedAt time.Time `json:"UpdatedAt"`
	UpdatedBy string    `json:"UpdatedBy" gorm:"type:varchar(36)" odata:"required"`
}

// EntitySetName returns custom entity set name "News" instead of default "NewsItems"
func (NewsItem) EntitySetName() string {
	return "News"
}

// TableName returns the custom table name for GORM
func (NewsItem) TableName() string {
	return "news"
}

// TestCustomEntitySetNameWithSelectFilter tests that custom EntitySetName works with $select, $filter, $orderby
// This reproduces the exact scenario from the issue where SQL fails with "missing FROM-clause entry"
func TestCustomEntitySetNameWithSelectFilter(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate - this will create the table "news" (respecting TableName() method)
	if err := db.AutoMigrate(&NewsItem{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	clubID := "3ab5f63e-4c91-4022-aed0-69bbf62af353"
	now := time.Now()
	news1 := NewsItem{
		ID:        "news-1",
		ClubID:    clubID,
		Title:     "Breaking News",
		Content:   "This is breaking news content",
		CreatedAt: now.Add(-24 * time.Hour),
		CreatedBy: "user-1",
		UpdatedAt: now,
		UpdatedBy: "user-1",
	}
	news2 := NewsItem{
		ID:        "news-2",
		ClubID:    clubID,
		Title:     "Another Story",
		Content:   "This is another story content",
		CreatedAt: now,
		CreatedBy: "user-1",
		UpdatedAt: now,
		UpdatedBy: "user-1",
	}
	news3 := NewsItem{
		ID:        "news-3",
		ClubID:    "different-club",
		Title:     "Other Club News",
		Content:   "News from a different club",
		CreatedAt: now,
		CreatedBy: "user-2",
		UpdatedAt: now,
		UpdatedBy: "user-2",
	}

	db.Create(&news1)
	db.Create(&news2)
	db.Create(&news3)

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&NewsItem{}); err != nil {
		t.Fatalf("Failed to register NewsItem entity: %v", err)
	}

	// Test: Request with $select, $filter, and $orderby (exact scenario from the issue)
	testURL := "/News?$select=ID,Title,Content,CreatedAt,UpdatedAt&$filter=" + url.QueryEscape("ClubID eq '"+clubID+"'") + "&$orderby=" + url.QueryEscape("CreatedAt desc")
	req := httptest.NewRequest(http.MethodGet, testURL, nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatalf("Response does not contain 'value' array")
	}

	if len(values) != 2 {
		t.Errorf("Expected 2 news items for club %s, got %d", clubID, len(values))
		return
	}

	// Verify ordering (CreatedAt desc) - news2 should come before news1
	item1 := values[0].(map[string]interface{})
	item2 := values[1].(map[string]interface{})

	if item1["ID"] != "news-2" {
		t.Errorf("Expected first item to be news-2 (most recent), got %v", item1["ID"])
	}
	if item2["ID"] != "news-1" {
		t.Errorf("Expected second item to be news-1 (older), got %v", item2["ID"])
	}

	// Verify that only selected fields are returned
	if _, hasClubID := item1["ClubID"]; hasClubID {
		t.Error("ClubID should not be in response when not selected")
	}
	if _, hasCreatedBy := item1["CreatedBy"]; hasCreatedBy {
		t.Error("CreatedBy should not be in response when not selected")
	}

	// Verify that selected fields are present
	if _, hasID := item1["ID"]; !hasID {
		t.Error("ID should be in response when selected")
	}
	if _, hasTitle := item1["Title"]; !hasTitle {
		t.Error("Title should be in response when selected")
	}
}

// TestCustomEntitySetNameWithSearch tests that custom EntitySetName works with $search
// This tests the FTS table name resolution which was the source of the bug
func TestCustomEntitySetNameWithSearch(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate
	if err := db.AutoMigrate(&NewsItem{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	now := time.Now()
	news1 := NewsItem{
		ID:        "news-1",
		ClubID:    "club-1",
		Title:     "Breaking News About Technology",
		Content:   "This is breaking news content about technology",
		CreatedAt: now,
		CreatedBy: "user-1",
		UpdatedAt: now,
		UpdatedBy: "user-1",
	}
	news2 := NewsItem{
		ID:        "news-2",
		ClubID:    "club-1",
		Title:     "Sports Update",
		Content:   "Latest sports results and updates",
		CreatedAt: now,
		CreatedBy: "user-1",
		UpdatedAt: now,
		UpdatedBy: "user-1",
	}

	db.Create(&news1)
	db.Create(&news2)

	// Create OData service
	service := odata.NewService(db)
	if err := service.RegisterEntity(&NewsItem{}); err != nil {
		t.Fatalf("Failed to register NewsItem entity: %v", err)
	}

	// Test: Request with $search (will use FTS table name)
	testURL := "/News?$search=technology"
	req := httptest.NewRequest(http.MethodGet, testURL, nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatalf("Response does not contain 'value' array")
	}

	// Should find at least one item matching "technology"
	// Note: Search may work in-memory if FTS is not available, so we check for results
	if len(values) == 0 {
		t.Error("Expected at least 1 news item matching 'technology'")
		return
	}

	// Verify the found item contains the search term
	item := values[0].(map[string]interface{})
	if item["ID"] != "news-1" {
		t.Errorf("Expected to find news-1 (technology news), got %v", item["ID"])
	}
}

// TestCustomEntitySetNameMetadata verifies that entity metadata correctly stores table name
func TestCustomEntitySetNameMetadata(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&NewsItem{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&NewsItem{}); err != nil {
		t.Fatalf("Failed to register NewsItem entity: %v", err)
	}

	// Note: This test validates that the service correctly handles custom entity set names
	// The service should have entity set "News" mapped to table "news"

	// Verify entity was registered with correct names
	// The service should have entity set "News" mapped to table "news"
	// This is implementation-dependent, but we can at least verify the service works
	req := httptest.NewRequest(http.MethodGet, "/News", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Service should handle /News endpoint, got status %d: %s", w.Code, w.Body.String())
	}
}
