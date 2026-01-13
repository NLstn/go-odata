package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSearchTestHandler(t *testing.T, withFTS bool) (*EntityHandler, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&SearchTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(SearchTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	if withFTS {
		handler.SetFTSManager(query.NewFTSManager(db))
	}

	return handler, db
}

func TestEntityHandlerCountWithSearchFallback(t *testing.T) {
	handler, db := setupSearchTestHandler(t, false)

	testData := []SearchTestProduct{
		{ID: 1, Name: "Laptop Pro", Description: "High-performance laptop", Category: "Electronics", Price: 1200},
		{ID: 2, Name: "Mouse", Description: "Wireless mouse", Category: "Laptop Accessories", Price: 25},
		{ID: 3, Name: "Desk", Description: "Office desk", Category: "Furniture", Price: 200},
	}
	if err := db.Create(&testData).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/SearchTestProducts/$count?$search=laptop", nil)
	queryOptions, err := query.ParseQueryOptions(req.URL.Query(), handler.metadata)
	if err != nil {
		t.Fatalf("failed to parse query options: %v", err)
	}

	scopes, hookErr := callBeforeReadCollection(handler.metadata, req, queryOptions)
	if hookErr != nil {
		t.Fatalf("before read hook failed: %v", hookErr)
	}

	helperCount, err := handler.countEntities(context.Background(), queryOptions, scopes)
	if err != nil {
		t.Fatalf("countEntities returned error: %v", err)
	}
	if helperCount != 1 {
		t.Fatalf("countEntities = %d, want %d", helperCount, 1)
	}

	w := httptest.NewRecorder()
	handler.HandleCount(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}
	if body := w.Body.String(); body != "1" {
		t.Fatalf("Body = %v, want %v", body, "1")
	}
}

func TestEntityHandlerCountWithSearchFTS(t *testing.T) {
	handler, db := setupSearchTestHandler(t, true)

	testData := []SearchTestProduct{
		{ID: 1, Name: "Laptop Pro", Description: "High-performance laptop", Category: "Electronics", Price: 1200},
		{ID: 2, Name: "Laptop Stand", Description: "Ergonomic stand", Category: "Accessories", Price: 50},
		{ID: 3, Name: "Desk", Description: "Office desk", Category: "Electronics", Price: 200},
	}
	if err := db.Create(&testData).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/SearchTestProducts/$count?$search=laptop&$filter=Category%20eq%20%27Electronics%27", nil)
	queryOptions, err := query.ParseQueryOptions(req.URL.Query(), handler.metadata)
	if err != nil {
		t.Fatalf("failed to parse query options: %v", err)
	}

	scopes, hookErr := callBeforeReadCollection(handler.metadata, req, queryOptions)
	if hookErr != nil {
		t.Fatalf("before read hook failed: %v", hookErr)
	}

	helperCount, err := handler.countEntities(context.Background(), queryOptions, scopes)
	if err != nil {
		t.Fatalf("countEntities returned error: %v", err)
	}
	if helperCount != 1 {
		t.Fatalf("countEntities = %d, want %d", helperCount, 1)
	}

	w := httptest.NewRecorder()
	handler.HandleCount(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}
	if body := w.Body.String(); body != strconv.FormatInt(helperCount, 10) {
		t.Fatalf("Body = %v, want %v", body, helperCount)
	}
}
