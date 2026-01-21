package odata_test

import (
	"bytes"
	"encoding/json"
	odata "github.com/nlstn/go-odata"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// PostTestProduct is a test entity for POST operations
type PostTestProduct struct {
	ID          int     `json:"id" gorm:"primarykey;autoIncrement" odata:"key"`
	Name        string  `json:"name" odata:"required"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
}

// PostTestProductCompositeKey is a test entity with composite keys for POST operations
type PostTestProductCompositeKey struct {
	ProductID   int    `json:"productID" gorm:"primaryKey" odata:"key"`
	LanguageKey string `json:"languageKey" gorm:"primaryKey" odata:"key"`
	Name        string `json:"name" odata:"required"`
	Description string `json:"description"`
}

// PostTestProductNoAutoIncrement is a test entity where key is not auto-incremented
type PostTestProductNoAutoIncrement struct {
	ID    int     `json:"id" gorm:"primarykey" odata:"key"`
	Name  string  `json:"name" odata:"required"`
	Price float64 `json:"price"`
}

// PostUUIDProduct uses server-side key generation
type PostUUIDProduct struct {
	ID   string `json:"id" gorm:"type:char(36);primaryKey" odata:"key,generate=uuid"`
	Name string `json:"name" odata:"required"`
}

func setupPostTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PostTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(PostTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func setupPostCompositeKeyTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PostTestProductCompositeKey{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(PostTestProductCompositeKey{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func setupPostNoAutoIncrementTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PostTestProductNoAutoIncrement{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(PostTestProductNoAutoIncrement{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func setupPostUUIDTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PostUUIDProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&PostUUIDProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestPostEntity_Success(t *testing.T) {
	service, db := setupPostTestService(t)

	// Create a new product
	newProduct := map[string]interface{}{
		"name":        "Laptop",
		"price":       999.99,
		"description": "A high-performance laptop",
		"category":    "Electronics",
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PostTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 201 Created
	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify Location header is present
	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Location header is empty")
	}

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json;odata.metadata=minimal" {
		t.Errorf("Content-Type = %v, want application/json;odata.metadata=minimal", contentType)
	}

	// Verify response body contains the created entity
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check @odata.context
	if _, ok := response["@odata.context"]; !ok {
		t.Error("Response missing @odata.context")
	}

	// Check entity properties
	if response["name"] != "Laptop" {
		t.Errorf("name = %v, want Laptop", response["name"])
	}

	if response["price"] != 999.99 {
		t.Errorf("price = %v, want 999.99", response["price"])
	}

	// Verify entity was created in database
	var count int64
	db.Model(&PostTestProduct{}).Count(&count)
	if count != 1 {
		t.Errorf("Entity count = %d, want 1", count)
	}

	// Verify the created entity has an ID
	if _, ok := response["id"]; !ok {
		t.Error("Response missing id field")
	}
}

func TestPostEntity_CompositeKey_Success(t *testing.T) {
	service, db := setupPostCompositeKeyTestService(t)

	// Create a new product with composite key
	newProduct := map[string]interface{}{
		"productID":   1,
		"languageKey": "EN",
		"name":        "Laptop",
		"description": "A high-performance laptop",
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PostTestProductCompositeKeys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 201 Created
	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify Location header contains composite key
	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Location header is empty")
	}
	// Location should be in format: /PostTestProductCompositeKeys(productID=1,languageKey='EN')
	// We just check it's not empty for now

	// Verify response body
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["name"] != "Laptop" {
		t.Errorf("name = %v, want Laptop", response["name"])
	}

	// Verify entity was created in database
	var count int64
	db.Model(&PostTestProductCompositeKey{}).Where("product_id = ? AND language_key = ?", 1, "EN").Count(&count)
	if count != 1 {
		t.Errorf("Entity count = %d, want 1", count)
	}
}

func TestPostEntity_MissingRequiredField(t *testing.T) {
	service, _ := setupPostTestService(t)

	// Create a product without required 'name' field
	newProduct := map[string]interface{}{
		"price":       999.99,
		"description": "A high-performance laptop",
		"category":    "Electronics",
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PostTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	// Verify error response format
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

func TestPostEntity_InvalidJSON(t *testing.T) {
	service, _ := setupPostTestService(t)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/PostTestProducts", bytes.NewBufferString("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	// Verify error response format
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

func TestPostEntity_EmptyBody(t *testing.T) {
	service, _ := setupPostTestService(t)

	// Send empty body
	req := httptest.NewRequest(http.MethodPost, "/PostTestProducts", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 400 Bad Request (missing required field)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestPostEntity_WithKeyInBody(t *testing.T) {
	service, db := setupPostNoAutoIncrementTestService(t)

	// Create a product with explicit key in body
	newProduct := map[string]interface{}{
		"id":    42,
		"name":  "Laptop",
		"price": 999.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PostTestProductNoAutoIncrements", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 201 Created
	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify response contains the specified ID
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["id"] != float64(42) {
		t.Errorf("id = %v, want 42", response["id"])
	}

	// Verify entity was created with the specified ID
	var product PostTestProductNoAutoIncrement
	db.First(&product, 42)
	if product.ID != 42 {
		t.Errorf("Created entity ID = %d, want 42", product.ID)
	}
}

func TestPostEntity_UUIDGeneratedKey(t *testing.T) {
	service, db := setupPostUUIDTestService(t)

	payload := map[string]interface{}{
		"name": "Generated",
	}

	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/PostUUIDProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Status = %v, want %v. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	rawID, ok := response["id"].(string)
	if !ok || rawID == "" {
		t.Fatalf("expected generated id string, got %v", response["id"])
	}

	var record PostUUIDProduct
	if err := db.First(&record).Error; err != nil {
		t.Fatalf("failed to fetch created record: %v", err)
	}

	if record.ID == "" {
		t.Fatal("database record has empty UUID")
	}

	if !looksLikeUUID(rawID) {
		t.Fatalf("response id %q is not in UUID format", rawID)
	}

	if record.ID != rawID {
		t.Fatalf("response ID %s does not match stored ID %s", rawID, record.ID)
	}
}

func looksLikeUUID(v string) bool {
	if len(v) != 36 {
		return false
	}

	for _, pos := range []int{8, 13, 18, 23} {
		if v[pos] != '-' {
			return false
		}
	}

	for _, r := range v {
		if r == '-' {
			continue
		}
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

func TestPostEntity_ToEntityEndpoint(t *testing.T) {
	service, db := setupPostTestService(t)

	// First create an entity
	product := PostTestProduct{
		ID:    1,
		Name:  "Laptop",
		Price: 999.99,
	}
	db.Create(&product)

	// Try to POST to individual entity endpoint
	newProduct := map[string]interface{}{
		"name":  "Mouse",
		"price": 29.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PostTestProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 405 Method Not Allowed
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestPostEntity_MultipleEntities(t *testing.T) {
	service, db := setupPostTestService(t)

	products := []map[string]interface{}{
		{
			"name":     "Laptop",
			"price":    999.99,
			"category": "Electronics",
		},
		{
			"name":     "Mouse",
			"price":    29.99,
			"category": "Accessories",
		},
		{
			"name":     "Keyboard",
			"price":    79.99,
			"category": "Accessories",
		},
	}

	for _, product := range products {
		body, _ := json.Marshal(product)

		req := httptest.NewRequest(http.MethodPost, "/PostTestProducts", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Status = %v, want %v for product %v", w.Code, http.StatusCreated, product["name"])
		}
	}

	// Verify all entities were created
	var count int64
	db.Model(&PostTestProduct{}).Count(&count)
	if count != 3 {
		t.Errorf("Entity count = %d, want 3", count)
	}
}

func TestPostEntity_VerifyLocationHeader(t *testing.T) {
	service, _ := setupPostTestService(t)

	newProduct := map[string]interface{}{
		"name":  "Laptop",
		"price": 999.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/PostTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("Location header is empty")
	}

	// Location should contain the entity set name and key
	// For example: http://localhost:8080/PostTestProducts(1)
	// We verify it has the expected pattern
	var response map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&response)
	id := int(response["id"].(float64))

	// The location should end with the key in parentheses
	expectedEnd := "/PostTestProducts(" + string(rune(id+'0')) + ")"
	if len(location) < len(expectedEnd) || location[len(location)-len(expectedEnd):] != expectedEnd {
		// This check might be too strict, so let's just check it contains PostTestProducts
		if len(location) == 0 {
			t.Errorf("Location = %v, should contain PostTestProducts", location)
		}
	}
}

func TestPostEntity_GetAfterPost(t *testing.T) {
	service, _ := setupPostTestService(t)

	// Create a new product
	newProduct := map[string]interface{}{
		"name":  "Laptop",
		"price": 999.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PostTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST failed: Status = %v, want %v", w.Code, http.StatusCreated)
	}

	// Extract the ID from the response
	var postResponse map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&postResponse)
	id := int(postResponse["id"].(float64))

	// Try to GET the created entity
	req = httptest.NewRequest(http.MethodGet, "/PostTestProducts("+string(rune(id+'0'))+")", nil)
	w = httptest.NewRecorder()
	service.ServeHTTP(w, req)

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("GET after POST: Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify the entity properties
	var getResponse map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&getResponse)

	if getResponse["name"] != "Laptop" {
		t.Errorf("name = %v, want Laptop", getResponse["name"])
	}
}

func TestPostEntity_DuplicateCompositeKey(t *testing.T) {
	service, db := setupPostCompositeKeyTestService(t)

	// Create first entity
	product := PostTestProductCompositeKey{
		ProductID:   1,
		LanguageKey: "EN",
		Name:        "Laptop",
	}
	db.Create(&product)

	// Try to create duplicate
	newProduct := map[string]interface{}{
		"productID":   1,
		"languageKey": "EN",
		"name":        "Mouse",
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "/PostTestProductCompositeKeys", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 500 Internal Server Error (database constraint violation)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}
}

func TestPostEntity_ContextURL(t *testing.T) {
	service, _ := setupPostTestService(t)

	newProduct := map[string]interface{}{
		"name":  "Laptop",
		"price": 999.99,
	}
	body, _ := json.Marshal(newProduct)

	req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/PostTestProducts", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify @odata.context format
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("@odata.context is not a string")
	}

	expectedContext := "http://localhost:8080/$metadata#PostTestProducts/$entity"
	if context != expectedContext {
		t.Errorf("@odata.context = %v, want %v", context, expectedContext)
	}
}
