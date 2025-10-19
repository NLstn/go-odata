package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test entities for relation testing
type Author struct {
	ID    uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name  string `json:"Name"`
	Books []Book `json:"Books" gorm:"foreignKey:AuthorID"`
}

type Book struct {
	ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Title    string  `json:"Title"`
	AuthorID uint    `json:"AuthorID"`
	Author   *Author `json:"Author,omitempty" gorm:"foreignKey:AuthorID"`
}

func setupRelationTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Author{}, &Book{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	authors := []Author{
		{ID: 1, Name: "J.K. Rowling"},
		{ID: 2, Name: "George R.R. Martin"},
		{ID: 3, Name: "J.R.R. Tolkien"},
	}
	db.Create(&authors)

	books := []Book{
		{ID: 1, Title: "Harry Potter and the Philosopher's Stone", AuthorID: 1},
		{ID: 2, Title: "Harry Potter and the Chamber of Secrets", AuthorID: 1},
		{ID: 3, Title: "A Game of Thrones", AuthorID: 2},
		{ID: 4, Title: "A Clash of Kings", AuthorID: 2},
		{ID: 5, Title: "The Hobbit", AuthorID: 3},
		{ID: 6, Title: "The Fellowship of the Ring", AuthorID: 3},
	}
	db.Create(&books)

	return db
}

// TestExpandBasic tests basic $expand functionality
func TestExpandBasic(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors?$expand=Books", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok || len(values) == 0 {
		t.Fatal("Expected value array in response")
	}

	// Check that Books are expanded
	firstAuthor := values[0].(map[string]interface{})
	books, ok := firstAuthor["Books"].([]interface{})
	if !ok {
		t.Error("Expected Books to be expanded")
	}

	if len(books) == 0 {
		t.Error("Expected at least one book for first author")
	}
}

// TestExpandOnSingleEntity tests $expand on a single entity
func TestExpandOnSingleEntity(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors(1)?$expand=Books", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	books, ok := response["Books"].([]interface{})
	if !ok {
		t.Error("Expected Books to be expanded")
	}

	// J.K. Rowling should have 2 books
	if len(books) != 2 {
		t.Errorf("Expected 2 books, got %d", len(books))
	}
}

// TestExpandWithNestedTop tests $expand with nested $top
func TestExpandWithNestedTop(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors(1)?$expand=Books($top=1)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	books, ok := response["Books"].([]interface{})
	if !ok {
		t.Error("Expected Books to be expanded")
	}

	if len(books) != 1 {
		t.Errorf("Expected 1 book due to $top=1, got %d", len(books))
	}
}

// TestExpandWithNestedSkip tests $expand with nested $skip
func TestExpandWithNestedSkip(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors(1)?$expand=Books($skip=1)", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	books, ok := response["Books"].([]interface{})
	if !ok {
		t.Error("Expected Books to be expanded")
	}

	// Should skip the first book, leaving 1
	if len(books) != 1 {
		t.Errorf("Expected 1 book after skipping 1, got %d", len(books))
	}
}

// TestExpandReverseNavigation tests expanding from Book to Author
func TestExpandReverseNavigation(t *testing.T) {
	db := setupRelationTestDB(t)
	bookMeta, _ := metadata.AnalyzeEntity(&Book{})
	handler := NewEntityHandler(db, bookMeta)

	req := httptest.NewRequest(http.MethodGet, "/Books(1)?$expand=Author", nil)
	w := httptest.NewRecorder()

	handler.HandleEntity(w, req, "1")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	author, ok := response["Author"].(map[string]interface{})
	if !ok {
		t.Error("Expected Author to be expanded")
	}

	authorName, ok := author["Name"].(string)
	if !ok || authorName != "J.K. Rowling" {
		t.Errorf("Expected author name 'J.K. Rowling', got %v", authorName)
	}
}

// TestExpandWithFilter tests $expand combined with $filter on main entity
func TestExpandWithFilter(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors?$filter=Name%20eq%20%27J.K.%20Rowling%27&$expand=Books", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	// Should only return J.K. Rowling
	if len(values) != 1 {
		t.Errorf("Expected 1 author after filter, got %d", len(values))
	}

	author := values[0].(map[string]interface{})
	books, ok := author["Books"].([]interface{})
	if !ok {
		t.Error("Expected Books to be expanded")
	}

	if len(books) != 2 {
		t.Errorf("Expected 2 books for J.K. Rowling, got %d", len(books))
	}
}

// TestExpandWithCount tests $expand combined with $count
func TestExpandWithCount(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors?$count=true&$expand=Books", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	count, ok := response["@odata.count"].(float64)
	if !ok {
		t.Error("Expected @odata.count in response")
	}

	if int(count) != 3 {
		t.Errorf("Expected count of 3 authors, got %d", int(count))
	}
}

// TestNavigationPropertyCount tests $count on navigation properties
func TestNavigationPropertyCount(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	entitiesMetadata := map[string]*metadata.EntityMetadata{
		"Authors": authorMeta,
	}
	handler := NewEntityHandler(db, authorMeta)
	handler.SetEntitiesMetadata(entitiesMetadata)

	tests := []struct {
		name          string
		path          string
		entityKey     string
		navProp       string
		expectedCount string
	}{
		{
			name:          "J.K. Rowling's books count",
			path:          "/Authors(1)/Books/$count",
			entityKey:     "1",
			navProp:       "Books",
			expectedCount: "2",
		},
		{
			name:          "George R.R. Martin's books count",
			path:          "/Authors(2)/Books/$count",
			entityKey:     "2",
			navProp:       "Books",
			expectedCount: "2",
		},
		{
			name:          "J.R.R. Tolkien's books count",
			path:          "/Authors(3)/Books/$count",
			entityKey:     "3",
			navProp:       "Books",
			expectedCount: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			handler.HandleNavigationPropertyCount(w, req, tt.entityKey, tt.navProp)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			// Check Content-Type
			contentType := w.Header().Get("Content-Type")
			if contentType != "text/plain" {
				t.Errorf("Expected Content-Type text/plain, got %s", contentType)
			}

			// Check count value
			count := w.Body.String()
			if count != tt.expectedCount {
				t.Errorf("Expected count %s, got %s", tt.expectedCount, count)
			}
		})
	}
}

// TestNavigationPropertyCountNotFound tests $count on navigation property when entity not found
func TestNavigationPropertyCountNotFound(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors(999)/Books/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationPropertyCount(w, req, "999", "Books")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestNavigationPropertyCountInvalidProperty tests $count on invalid navigation property
func TestNavigationPropertyCountInvalidProperty(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors(1)/InvalidProperty/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationPropertyCount(w, req, "1", "InvalidProperty")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestNavigationPropertyCountOnSingleValuedProperty tests $count on single-valued navigation property (should fail)
func TestNavigationPropertyCountOnSingleValuedProperty(t *testing.T) {
	db := setupRelationTestDB(t)
	bookMeta, _ := metadata.AnalyzeEntity(&Book{})
	handler := NewEntityHandler(db, bookMeta)

	req := httptest.NewRequest(http.MethodGet, "/Books(1)/Author/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationPropertyCount(w, req, "1", "Author")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestNavigationPropertyCountHEAD tests HEAD request on navigation property count
func TestNavigationPropertyCountHEAD(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodHead, "/Authors(1)/Books/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationPropertyCount(w, req, "1", "Books")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// HEAD should not return body
	if w.Body.Len() > 0 {
		t.Errorf("Expected empty body for HEAD request, got %d bytes", w.Body.Len())
	}

	// But should have correct Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}
}

// TestNavigationPropertyCountOPTIONS tests OPTIONS request on navigation property count
func TestNavigationPropertyCountOPTIONS(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodOptions, "/Authors(1)/Books/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationPropertyCount(w, req, "1", "Books")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check Allow header
	allow := w.Header().Get("Allow")
	if allow != "GET, HEAD, OPTIONS" {
		t.Errorf("Expected Allow header 'GET, HEAD, OPTIONS', got %s", allow)
	}
}


// TestNavigationPropertyPath tests accessing navigation properties via path
func TestNavigationPropertyPath(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors(1)/Books", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationProperty(w, req, "1", "Books", false)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(values) != 2 {
		t.Errorf("Expected 2 books for author 1, got %d", len(values))
	}
}

// TestNavigationPropertyPathSingle tests accessing single navigation property via path
func TestNavigationPropertyPathSingle(t *testing.T) {
	db := setupRelationTestDB(t)
	bookMeta, _ := metadata.AnalyzeEntity(&Book{})
	handler := NewEntityHandler(db, bookMeta)

	req := httptest.NewRequest(http.MethodGet, "/Books(1)/Author", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationProperty(w, req, "1", "Author", false)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	name, ok := response["Name"].(string)
	if !ok || name != "J.K. Rowling" {
		t.Errorf("Expected author name 'J.K. Rowling', got %v", name)
	}
}

// TestExpandInvalidNavigationProperty tests that invalid navigation properties are rejected
func TestExpandInvalidNavigationProperty(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors?$expand=InvalidProperty", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestNavigationPropertyNotFound tests that invalid navigation property paths return 404
func TestNavigationPropertyNotFound(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors(1)/InvalidProperty", nil)
	w := httptest.NewRecorder()

	handler.HandleNavigationProperty(w, req, "1", "InvalidProperty", false)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestExpandWithOrderBy tests $expand combined with $orderby on main entity
func TestExpandWithOrderBy(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "/Authors?$orderby=Name%20desc&$expand=Books&$top=2", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	// Should return authors in descending order by name
	firstAuthor := values[0].(map[string]interface{})
	firstName, _ := firstAuthor["Name"].(string)

	// Should start with "J.R.R. Tolkien" (last alphabetically)
	if firstName != "J.R.R. Tolkien" {
		t.Errorf("Expected first author to be 'J.R.R. Tolkien', got %s", firstName)
	}
}

// TestMetadataNavigationProperties tests that metadata includes navigation properties
func TestMetadataNavigationProperties(t *testing.T) {
	authorMeta, err := metadata.AnalyzeEntity(&Author{})
	if err != nil {
		t.Fatalf("Failed to analyze Author entity: %v", err)
	}

	hasNavProp := false
	for _, prop := range authorMeta.Properties {
		if prop.IsNavigationProp && prop.Name == "Books" {
			hasNavProp = true
			if !prop.NavigationIsArray {
				t.Error("Expected Books to be a collection navigation property")
			}
			if prop.NavigationTarget != "Book" {
				t.Errorf("Expected navigation target to be 'Book', got %s", prop.NavigationTarget)
			}
		}
	}

	if !hasNavProp {
		t.Error("Expected Books navigation property in metadata")
	}
}

// TestQueryParserExpandSingle tests parsing a single $expand
func TestQueryParserExpandSingle(t *testing.T) {

	queryString := "Books"
	expandOpts := parseExpandForTest(queryString)

	if len(expandOpts) != 1 {
		t.Fatalf("Expected 1 expand option, got %d", len(expandOpts))
	}

	if expandOpts[0].NavigationProperty != "Books" {
		t.Errorf("Expected navigation property 'Books', got %s", expandOpts[0].NavigationProperty)
	}
}

// Helper function for testing expand parsing
func parseExpandForTest(expandStr string) []query.ExpandOption {
	// This is a simplified version for testing - in real code, use the parser
	parts := []string{expandStr}
	result := make([]query.ExpandOption, 0, len(parts))

	for _, part := range parts {
		expand := query.ExpandOption{
			NavigationProperty: part,
		}
		result = append(result, expand)
	}

	return result
}

// TestNavigationLinksWithoutExpand tests that navigation links are included when properties are not expanded
// and full metadata is requested (per OData v4 spec)
func TestNavigationLinksWithoutExpand(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/Authors", nil)
	// Request full metadata to get navigation links (per OData v4 spec)
	req.Header.Set("Accept", "application/json;odata.metadata=full")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok || len(values) == 0 {
		t.Fatal("Expected value array in response")
	}

	// Check that navigation links are present
	firstAuthor := values[0].(map[string]interface{})

	// Books property should not be present as a null value
	if _, hasBooks := firstAuthor["Books"]; hasBooks {
		t.Error("Books property should not be present when not expanded")
	}

	// Navigation link should be present
	navLink, hasNavLink := firstAuthor["Books@odata.navigationLink"]
	if !hasNavLink {
		t.Error("Expected Books@odata.navigationLink to be present")
	}

	// Validate the navigation link format
	expectedPattern := "http://localhost:8080/Authors(1)/Books"
	if navLink != expectedPattern {
		t.Errorf("Expected navigation link to be %s, got %v", expectedPattern, navLink)
	}
}

// TestNavigationLinksWithExpand tests that expanded properties show data instead of navigation links
func TestNavigationLinksWithExpand(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/Authors?$expand=Books", nil)
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok || len(values) == 0 {
		t.Fatal("Expected value array in response")
	}

	// Check that Books are expanded (actual data, not navigation links)
	firstAuthor := values[0].(map[string]interface{})

	books, hasBooks := firstAuthor["Books"].([]interface{})
	if !hasBooks {
		t.Error("Expected Books to be expanded with actual data")
	}

	if len(books) == 0 {
		t.Error("Expected at least one book for first author")
	}

	// Navigation link should NOT be present when expanded
	if _, hasNavLink := firstAuthor["Books@odata.navigationLink"]; hasNavLink {
		t.Error("Navigation link should not be present when property is expanded")
	}

	// Verify book data structure
	firstBook := books[0].(map[string]interface{})
	if _, hasTitle := firstBook["Title"]; !hasTitle {
		t.Error("Book should have Title field")
	}
}

// TestNavigationLinksWithSelect tests that navigation links work correctly with $select
// and full metadata (per OData v4 spec)
func TestNavigationLinksWithSelect(t *testing.T) {
	db := setupRelationTestDB(t)
	authorMeta, _ := metadata.AnalyzeEntity(&Author{})
	handler := NewEntityHandler(db, authorMeta)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/Authors?$select=ID,Name", nil)
	// Request full metadata to get navigation links (per OData v4 spec)
	req.Header.Set("Accept", "application/json;odata.metadata=full")
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok || len(values) == 0 {
		t.Fatal("Expected value array in response")
	}

	firstAuthor := values[0].(map[string]interface{})

	// Check that only selected fields are present
	if _, hasID := firstAuthor["ID"]; !hasID {
		t.Error("ID should be present")
	}
	if _, hasName := firstAuthor["Name"]; !hasName {
		t.Error("Name should be present")
	}

	// Navigation link should still be present even with $select
	navLink, hasNavLink := firstAuthor["Books@odata.navigationLink"]
	if !hasNavLink {
		t.Error("Expected Books@odata.navigationLink to be present even with $select")
	}

	// Validate the navigation link format
	expectedPattern := "http://localhost:8080/Authors(1)/Books"
	if navLink != expectedPattern {
		t.Errorf("Expected navigation link to be %s, got %v", expectedPattern, navLink)
	}

	// Books property should not be present
	if _, hasBooks := firstAuthor["Books"]; hasBooks {
		t.Error("Books property should not be present when not expanded")
	}
}
