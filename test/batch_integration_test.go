package odata_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// BatchIntegrationProduct is a test entity for batch integration tests
type BatchIntegrationProduct struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name"`
	Price       float64 `json:"Price"`
	Category    string  `json:"Category"`
	Description string  `json:"Description"`
}

type BatchIntegrationCustomer struct {
	ID     uint                    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name   string                  `json:"Name"`
	Orders []BatchIntegrationOrder `json:"Orders" gorm:"foreignKey:CustomerID;references:ID"`
}

type BatchIntegrationOrder struct {
	ID         uint                      `json:"ID" gorm:"primaryKey" odata:"key"`
	Amount     float64                   `json:"Amount"`
	CustomerID uint                      `json:"CustomerID"`
	Customer   *BatchIntegrationCustomer `json:"Customer" gorm:"foreignKey:CustomerID;references:ID"`
}

func setupBatchIntegrationTest(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&BatchIntegrationProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&BatchIntegrationProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestBatchIntegration_SingleGetRequest(t *testing.T) {
	service, db := setupBatchIntegrationTest(t)

	// Insert test data
	product := BatchIntegrationProduct{
		ID:       1,
		Name:     "Test Product",
		Price:    99.99,
		Category: "Electronics",
	}
	db.Create(&product)

	// Create batch request
	boundary := "batch_36d5c8c6"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check response contains product data
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Test Product") {
		t.Errorf("Response does not contain product name. Body: %s", responseBody)
	}

	// Check multipart response format
	if !strings.Contains(responseBody, "HTTP/1.1 200") {
		t.Errorf("Response does not contain HTTP status. Body: %s", responseBody)
	}
}

func TestBatchIntegration_MultipleGetRequests(t *testing.T) {
	service, db := setupBatchIntegrationTest(t)

	// Insert test data
	products := []BatchIntegrationProduct{
		{ID: 1, Name: "Product 1", Price: 10.00, Category: "Category A"},
		{ID: 2, Name: "Product 2", Price: 20.00, Category: "Category B"},
		{ID: 3, Name: "Product 3", Price: 30.00, Category: "Category C"},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Create batch request with multiple GET requests
	boundary := "batch_36d5c8c6"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(2) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(3) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check response contains all products
	responseBody := w.Body.String()
	for _, p := range products {
		if !strings.Contains(responseBody, p.Name) {
			t.Errorf("Response does not contain product: %s", p.Name)
		}
	}

	// Count number of HTTP responses in batch
	count := strings.Count(responseBody, "HTTP/1.1 200")
	if count != 3 {
		t.Errorf("Expected 3 successful HTTP responses, got %d", count)
	}
}

func TestBatchIntegration_Changeset(t *testing.T) {
	service, db := setupBatchIntegrationTest(t)

	// Create batch request with changeset
	batchBoundary := "batch_36d5c8c6"
	changesetBoundary := "changeset_77162fcd"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchIntegrationProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 1","Price":10.00,"Category":"Electronics"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchIntegrationProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 2","Price":20.00,"Category":"Books"}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify both products were created
	var count int64
	db.Model(&BatchIntegrationProduct{}).Count(&count)
	if count != 2 {
		t.Errorf("Expected 2 products in database, got %d", count)
	}

	// Verify products exist
	var products []BatchIntegrationProduct
	db.Find(&products)
	names := make(map[string]bool)
	for _, p := range products {
		names[p.Name] = true
	}

	expectedNames := []string{"Product 1", "Product 2"}
	for _, expected := range expectedNames {
		if !names[expected] {
			t.Errorf("Expected product %s not found in database", expected)
		}
	}

	// Check response indicates success
	responseBody := w.Body.String()
	successCount := strings.Count(responseBody, "HTTP/1.1 201")
	if successCount != 2 {
		t.Errorf("Expected 2 successful creates (201), got %d. Body: %s", successCount, responseBody)
	}
}

func TestBatchIntegration_MixedRequests(t *testing.T) {
	service, db := setupBatchIntegrationTest(t)

	// Insert initial product
	product := BatchIntegrationProduct{
		ID:       1,
		Name:     "Existing Product",
		Price:    50.00,
		Category: "Electronics",
	}
	db.Create(&product)

	// Create batch request with GET and changeset
	batchBoundary := "batch_36d5c8c6"
	changesetBoundary := "changeset_77162fcd"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchIntegrationProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"New Product","Price":100.00,"Category":"Books"}

--%s--

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, batchBoundary, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify new product was created
	var count int64
	db.Model(&BatchIntegrationProduct{}).Count(&count)
	if count != 2 {
		t.Errorf("Expected 2 products in database, got %d", count)
	}

	// Check response contains both products
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Existing Product") {
		t.Errorf("Response does not contain existing product. Body: %s", responseBody)
	}

	// Check we have both a GET and POST response
	if !strings.Contains(responseBody, "HTTP/1.1 200") {
		t.Error("Response does not contain GET success (200)")
	}
	if !strings.Contains(responseBody, "HTTP/1.1 201") {
		t.Error("Response does not contain POST success (201)")
	}
}

func TestBatchIntegration_ChangesetWithNavigationAndChangeTracking(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&BatchIntegrationCustomer{}, &BatchIntegrationOrder{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&BatchIntegrationCustomer{}); err != nil {
		t.Fatalf("Failed to register customer entity: %v", err)
	}
	if err := service.RegisterEntity(&BatchIntegrationOrder{}); err != nil {
		t.Fatalf("Failed to register order entity: %v", err)
	}
	if err := service.EnableChangeTracking("BatchIntegrationOrders"); err != nil {
		t.Fatalf("Failed to enable change tracking: %v", err)
	}

	customer := BatchIntegrationCustomer{ID: 1, Name: "Contoso"}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("Failed to seed customer: %v", err)
	}

	initialReq := httptest.NewRequest(http.MethodGet, "/BatchIntegrationOrders", nil)
	initialReq.Header.Set("Prefer", "odata.track-changes")
	initialRes := httptest.NewRecorder()
	service.ServeHTTP(initialRes, initialReq)
	if initialRes.Code != http.StatusOK {
		t.Fatalf("Initial delta request failed: %d", initialRes.Code)
	}
	initialToken := extractDeltaToken(t, initialRes.Body.Bytes())

	batchBoundary := "batch_nav_ct"
	changesetBoundary := "changeset_nav_ct"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchIntegrationOrders HTTP/1.1
Host: localhost
Content-Type: application/json

{"ID":1,"Amount":42.5}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchIntegrationCustomers(1)/Orders/$ref HTTP/1.1
Host: localhost
Content-Type: application/json

{"@odata.id":"/BatchIntegrationOrders(1)"}

--%s--

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationOrders?$expand=Customer HTTP/1.1
Host: localhost
Accept: application/json

 
--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Batch request failed: status=%d body=%s", w.Code, w.Body.String())
	}

	responseBody := w.Body.String()
	if strings.Count(responseBody, "HTTP/1.1 201") != 1 {
		t.Fatalf("Expected one creation response, got body: %s", responseBody)
	}
	if strings.Count(responseBody, "HTTP/1.1 204") != 1 {
		t.Fatalf("Expected one navigation reference response, got body: %s", responseBody)
	}
	if !strings.Contains(responseBody, "HTTP/1.1 200") {
		t.Fatalf("Expected final GET response, got body: %s", responseBody)
	}
	if !strings.Contains(responseBody, "Contoso") || !strings.Contains(responseBody, "\"Customer\"") {
		t.Fatalf("Expanded customer information missing from response: %s", responseBody)
	}

	var order BatchIntegrationOrder
	if err := db.First(&order, 1).Error; err != nil {
		t.Fatalf("Failed to load created order: %v", err)
	}
	if order.CustomerID != 1 {
		t.Fatalf("Expected order to be associated with customer 1, got %d", order.CustomerID)
	}

	deltaReq := httptest.NewRequest(http.MethodGet,
		"/BatchIntegrationOrders?$deltatoken="+url.QueryEscape(initialToken), nil)
	deltaRes := httptest.NewRecorder()
	service.ServeHTTP(deltaRes, deltaReq)
	if deltaRes.Code != http.StatusOK {
		t.Fatalf("Delta request failed: %d", deltaRes.Code)
	}

	deltaPayload := decodeJSON(t, deltaRes.Body.Bytes())
	entries := valueEntries(t, deltaPayload)
	if len(entries) != 1 {
		t.Fatalf("Expected 1 change entry, got %d", len(entries))
	}
	if id, ok := entries[0]["ID"].(float64); !ok || uint(id) != order.ID {
		t.Fatalf("Expected change entry for order %d, got %v", order.ID, entries[0])
	}
}

func TestBatchIntegration_ErrorHandling(t *testing.T) {
	service, _ := setupBatchIntegrationTest(t)

	// Create batch request with invalid entity key
	boundary := "batch_36d5c8c6"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(999) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v (batch request itself should succeed)", w.Code, http.StatusOK)
	}

	// Response should contain error for the specific request
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "404") && !strings.Contains(responseBody, "not found") {
		t.Errorf("Response should contain 404 error. Body: %s", responseBody)
	}
}

// TestBatchIntegration_ChangesetRollback tests that changesets are atomic
// Note: Currently, this feature requires deeper transaction handling integration
// This test is skipped for now and represents a future enhancement
func TestBatchIntegration_ChangesetRollback(t *testing.T) {
	t.Skip("Changeset rollback on validation errors requires deeper transaction integration - future enhancement")

	service, db := setupBatchIntegrationTest(t)

	// Insert a product that will be referenced
	product := BatchIntegrationProduct{
		ID:       1,
		Name:     "Existing Product",
		Price:    50.00,
		Category: "Electronics",
	}
	db.Create(&product)

	// Create batch request with changeset that should fail
	// First request succeeds, second fails - both should be rolled back
	batchBoundary := "batch_36d5c8c6"
	changesetBoundary := "changeset_77162fcd"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchIntegrationProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 2","Price":20.00,"Category":"Books"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchIntegrationProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"InvalidField":"This should fail"}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify only the original product exists (changeset was rolled back)
	var count int64
	db.Model(&BatchIntegrationProduct{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 product in database (changeset rolled back), got %d", count)
	}
}

func TestBatchIntegration_GetCollection(t *testing.T) {
	service, db := setupBatchIntegrationTest(t)

	// Insert test data
	products := []BatchIntegrationProduct{
		{ID: 1, Name: "Product 1", Price: 10.00, Category: "Electronics"},
		{ID: 2, Name: "Product 2", Price: 20.00, Category: "Books"},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Create batch request to get collection
	boundary := "batch_36d5c8c6"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check response contains both products
	responseBody := w.Body.String()
	for _, p := range products {
		if !strings.Contains(responseBody, p.Name) {
			t.Errorf("Response does not contain product: %s", p.Name)
		}
	}
}

func TestBatchIntegration_UpdateWithPatch(t *testing.T) {
	service, db := setupBatchIntegrationTest(t)

	// Insert initial product
	product := BatchIntegrationProduct{
		ID:       1,
		Name:     "Original Name",
		Price:    50.00,
		Category: "Electronics",
	}
	db.Create(&product)

	// Create batch request with PATCH in changeset
	batchBoundary := "batch_36d5c8c6"
	changesetBoundary := "changeset_77162fcd"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

PATCH /BatchIntegrationProducts(1) HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Updated Name","Price":75.00}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify product was updated
	var updatedProduct BatchIntegrationProduct
	db.First(&updatedProduct, 1)
	if updatedProduct.Name != "Updated Name" {
		t.Errorf("Name = %v, want 'Updated Name'", updatedProduct.Name)
	}
	if updatedProduct.Price != 75.00 {
		t.Errorf("Price = %v, want 75.00", updatedProduct.Price)
	}

	// Check response indicates success
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "HTTP/1.1 204") && !strings.Contains(responseBody, "HTTP/1.1 200") {
		t.Errorf("Response should indicate success (204 or 200). Body: %s", responseBody)
	}
}

func TestBatchIntegration_DeleteInChangeset(t *testing.T) {
	service, db := setupBatchIntegrationTest(t)

	// Insert test products
	products := []BatchIntegrationProduct{
		{ID: 1, Name: "Product 1", Price: 10.00, Category: "Electronics"},
		{ID: 2, Name: "Product 2", Price: 20.00, Category: "Books"},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Create batch request with DELETE in changeset
	batchBoundary := "batch_36d5c8c6"
	changesetBoundary := "changeset_77162fcd"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

DELETE /BatchIntegrationProducts(1) HTTP/1.1
Host: localhost


--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify product was deleted
	var count int64
	db.Model(&BatchIntegrationProduct{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 product remaining in database, got %d", count)
	}

	// Verify the correct product was deleted
	var remaining BatchIntegrationProduct
	db.First(&remaining)
	if remaining.ID != 2 {
		t.Errorf("Expected product 2 to remain, got ID %d", remaining.ID)
	}

	// Check response indicates success
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "HTTP/1.1 204") && !strings.Contains(responseBody, "HTTP/1.1 200") {
		t.Errorf("Response should indicate success (204 or 200). Body: %s", responseBody)
	}
}

func TestBatchIntegration_ODataV4Compliance(t *testing.T) {
	service, db := setupBatchIntegrationTest(t)

	// Insert test data
	product := BatchIntegrationProduct{
		ID:       1,
		Name:     "Test Product",
		Price:    99.99,
		Category: "Electronics",
	}
	db.Create(&product)

	// Create batch request
	boundary := "batch_36d5c8c6"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Verify OData v4 compliance
	tests := []struct {
		name     string
		check    func() bool
		errorMsg string
	}{
		{
			name:     "Response status is 200 OK",
			check:    func() bool { return w.Code == http.StatusOK },
			errorMsg: fmt.Sprintf("Status = %v, want 200", w.Code),
		},
		{
			name:     "Content-Type is multipart/mixed",
			check:    func() bool { return strings.Contains(w.Header().Get("Content-Type"), "multipart/mixed") },
			errorMsg: fmt.Sprintf("Content-Type = %v, should contain multipart/mixed", w.Header().Get("Content-Type")),
		},
		{
			name:     "Response contains Content-Type: application/http",
			check:    func() bool { return strings.Contains(w.Body.String(), "Content-Type: application/http") },
			errorMsg: "Response should contain 'Content-Type: application/http' for individual parts",
		},
		{
			name:     "Response contains Content-Transfer-Encoding: binary",
			check:    func() bool { return strings.Contains(w.Body.String(), "Content-Transfer-Encoding: binary") },
			errorMsg: "Response should contain 'Content-Transfer-Encoding: binary'",
		},
		{
			name:     "Response contains HTTP status line",
			check:    func() bool { return strings.Contains(w.Body.String(), "HTTP/1.1") },
			errorMsg: "Response should contain HTTP status line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check() {
				t.Error(tt.errorMsg)
			}
		})
	}
}
