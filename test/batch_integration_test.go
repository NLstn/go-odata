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

// TestBatchIntegration_ChangesetRollback verifies that transactional changesets
// commit or roll back database rows and change tracking events together.
func TestBatchIntegration_ChangesetRollback(t *testing.T) {
	t.Run("success commits and records change events", func(t *testing.T) {
		service, db := setupBatchIntegrationTest(t)

		if err := service.EnableChangeTracking("BatchIntegrationProducts"); err != nil {
			t.Fatalf("enable change tracking: %v", err)
		}

		initialReq := httptest.NewRequest(http.MethodGet, "/BatchIntegrationProducts", nil)
		initialReq.Header.Set("Prefer", "odata.track-changes")
		initialRes := httptest.NewRecorder()
		service.ServeHTTP(initialRes, initialReq)
		if initialRes.Code != http.StatusOK {
			t.Fatalf("initial delta request failed: %d", initialRes.Code)
		}
		initialToken := extractDeltaToken(t, initialRes.Body.Bytes())

		batchBoundary := "batch_changeset_success"
		changesetBoundary := "changeset_success"
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
		res := httptest.NewRecorder()

		service.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("batch request failed: %d body: %s", res.Code, res.Body.String())
		}

		var products []BatchIntegrationProduct
		if err := db.Order("id").Find(&products).Error; err != nil {
			t.Fatalf("query products: %v", err)
		}
		if len(products) != 2 {
			t.Fatalf("expected 2 products, got %d", len(products))
		}
		names := []string{products[0].Name, products[1].Name}
		expectedNames := []string{"Product 1", "Product 2"}
		for i, expected := range expectedNames {
			if names[i] != expected {
				t.Fatalf("expected product %d to be %s, got %s", i+1, expected, names[i])
			}
		}

		deltaReq := httptest.NewRequest(http.MethodGet,
			"/BatchIntegrationProducts?$deltatoken="+url.QueryEscape(initialToken), nil)
		deltaRes := httptest.NewRecorder()
		service.ServeHTTP(deltaRes, deltaReq)
		if deltaRes.Code != http.StatusOK {
			t.Fatalf("delta request failed: %d", deltaRes.Code)
		}

		deltaPayload := decodeJSON(t, deltaRes.Body.Bytes())
		entries := valueEntries(t, deltaPayload)
		if len(entries) != 2 {
			t.Fatalf("expected 2 change events, got %d", len(entries))
		}
		observed := map[string]bool{}
		for _, entry := range entries {
			name, _ := entry["Name"].(string)
			observed[name] = true
		}
		for _, expected := range expectedNames {
			if !observed[expected] {
				t.Fatalf("missing change tracking event for %s", expected)
			}
		}
	})

	t.Run("failure rolls back database and change events", func(t *testing.T) {
		service, db := setupBatchIntegrationTest(t)

		if err := service.EnableChangeTracking("BatchIntegrationProducts"); err != nil {
			t.Fatalf("enable change tracking: %v", err)
		}

		initialReq := httptest.NewRequest(http.MethodGet, "/BatchIntegrationProducts", nil)
		initialReq.Header.Set("Prefer", "odata.track-changes")
		initialRes := httptest.NewRecorder()
		service.ServeHTTP(initialRes, initialReq)
		if initialRes.Code != http.StatusOK {
			t.Fatalf("initial delta request failed: %d", initialRes.Code)
		}
		initialToken := extractDeltaToken(t, initialRes.Body.Bytes())

		batchBoundary := "batch_changeset_failure"
		changesetBoundary := "changeset_failure"
		body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchIntegrationProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"ShouldRollback","Price":30.00,"Category":"Garden"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchIntegrationProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

		req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
		req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
		res := httptest.NewRecorder()

		service.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("batch request failed: %d body: %s", res.Code, res.Body.String())
		}

		var count int64
		if err := db.Model(&BatchIntegrationProduct{}).Count(&count).Error; err != nil {
			t.Fatalf("count products: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected no products after rollback, found %d", count)
		}

		deltaReq := httptest.NewRequest(http.MethodGet,
			"/BatchIntegrationProducts?$deltatoken="+url.QueryEscape(initialToken), nil)
		deltaRes := httptest.NewRecorder()
		service.ServeHTTP(deltaRes, deltaReq)
		if deltaRes.Code != http.StatusOK {
			t.Fatalf("delta request failed: %d", deltaRes.Code)
		}

		deltaPayload := decodeJSON(t, deltaRes.Body.Bytes())
		entries := valueEntries(t, deltaPayload)
		if len(entries) != 0 {
			t.Fatalf("expected no change tracking events after rollback, got %d", len(entries))
		}
	})
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

// TestBatchIntegration_ContentIDEcho verifies that Content-ID headers are echoed back
// in batch responses per OData v4 spec section 11.7.4
func TestBatchIntegration_ContentIDEcho(t *testing.T) {
	service, db := setupBatchIntegrationTest(t)

	// Insert test data
	product := BatchIntegrationProduct{
		ID:       1,
		Name:     "Test Product",
		Price:    99.99,
		Category: "Electronics",
	}
	db.Create(&product)

	t.Run("single request with Content-ID", func(t *testing.T) {
		boundary := "batch_36d5c8c6"
		body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: myRequest1

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
			t.Fatalf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		responseBody := w.Body.String()
		if !strings.Contains(responseBody, "Content-ID: myRequest1") {
			t.Errorf("Response does not contain echoed Content-ID header. Body: %s", responseBody)
		}
	})

	t.Run("changeset with Content-IDs", func(t *testing.T) {
		// Clean up and reset data
		db.Exec("DELETE FROM batch_integration_products")

		batchBoundary := "batch_36d5c8c6"
		changesetBoundary := "changeset_77162fcd"
		body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /BatchIntegrationProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 1","Price":10.00,"Category":"Electronics"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

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
			t.Fatalf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		responseBody := w.Body.String()
		if !strings.Contains(responseBody, "Content-ID: 1") {
			t.Errorf("Response does not contain first Content-ID header. Body: %s", responseBody)
		}
		if !strings.Contains(responseBody, "Content-ID: 2") {
			t.Errorf("Response does not contain second Content-ID header. Body: %s", responseBody)
		}
	})

	t.Run("no Content-ID when not provided", func(t *testing.T) {
		// Insert test data
		db.Exec("DELETE FROM batch_integration_products")
		product := BatchIntegrationProduct{
			ID:       1,
			Name:     "Test Product",
			Price:    99.99,
			Category: "Electronics",
		}
		db.Create(&product)

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
			t.Fatalf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		responseBody := w.Body.String()
		if strings.Contains(responseBody, "Content-ID:") {
			t.Errorf("Response should not contain Content-ID when none was provided. Body: %s", responseBody)
		}
	})
}

func TestBatchIntegration_MaxBatchSizeEnforcement(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&BatchIntegrationProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create service with small max batch size
	service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{
		MaxBatchSize: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&BatchIntegrationProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	products := []BatchIntegrationProduct{
		{ID: 1, Name: "Product 1", Price: 10.00, Category: "Category A"},
		{ID: 2, Name: "Product 2", Price: 20.00, Category: "Category B"},
		{ID: 3, Name: "Product 3", Price: 30.00, Category: "Category C"},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Create batch request with 3 GET requests (exceeds limit of 2)
	boundary := "batch_boundary"
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

	// Should return 413 Request Entity Too Large
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}

	// Check error message contains limit information
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Maximum allowed: 2") {
		t.Errorf("Response should contain limit information. Body: %s", responseBody)
	}
}

func TestBatchIntegration_MaxBatchSizeWithinLimit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&BatchIntegrationProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create service with max batch size of 2
	service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{
		MaxBatchSize: 2,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if err := service.RegisterEntity(&BatchIntegrationProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	products := []BatchIntegrationProduct{
		{ID: 1, Name: "Product 1", Price: 10.00, Category: "Category A"},
		{ID: 2, Name: "Product 2", Price: 20.00, Category: "Category B"},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Create batch request with 2 GET requests (within limit)
	boundary := "batch_boundary"
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


--%s--
`, boundary, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should succeed with 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check response contains both products
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Product 1") || !strings.Contains(responseBody, "Product 2") {
		t.Errorf("Response should contain both products. Body: %s", responseBody)
	}
}

func TestBatchIntegration_DefaultMaxBatchSize(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&BatchIntegrationProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create service without specifying MaxBatchSize (should use default)
	service := odata.NewService(db)
	if err := service.RegisterEntity(&BatchIntegrationProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	for i := 1; i <= 100; i++ {
		product := BatchIntegrationProduct{
			ID:       uint(i),
			Name:     fmt.Sprintf("Product %d", i),
			Price:    float64(i) * 10.0,
			Category: "Test",
		}
		db.Create(&product)
	}

	// Create batch request with exactly 100 requests (should be allowed with default limit)
	boundary := "batch_boundary"
	bodyBuilder := strings.Builder{}
	for i := 1; i <= 100; i++ {
		bodyBuilder.WriteString(fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(%d) HTTP/1.1
Host: localhost
Accept: application/json


`, boundary, i))
	}
	bodyBuilder.WriteString(fmt.Sprintf("--%s--\n", boundary))

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(bodyBuilder.String()))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should succeed with 200 OK (default limit is 100)
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Count number of HTTP responses in batch
	responseBody := w.Body.String()
	count := strings.Count(responseBody, "HTTP/1.1")
	if count != 100 {
		t.Errorf("Expected 100 HTTP responses, got %d", count)
	}
}

func TestBatchIntegration_ExceedDefaultMaxBatchSize(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&BatchIntegrationProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Create service without specifying MaxBatchSize (should use default of 100)
	service := odata.NewService(db)
	if err := service.RegisterEntity(&BatchIntegrationProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Insert test data
	for i := 1; i <= 101; i++ {
		product := BatchIntegrationProduct{
			ID:       uint(i),
			Name:     fmt.Sprintf("Product %d", i),
			Price:    float64(i) * 10.0,
			Category: "Test",
		}
		db.Create(&product)
	}

	// Create batch request with 101 requests (exceeds default limit of 100)
	boundary := "batch_boundary"
	bodyBuilder := strings.Builder{}
	for i := 1; i <= 101; i++ {
		bodyBuilder.WriteString(fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(%d) HTTP/1.1
Host: localhost
Accept: application/json


`, boundary, i))
	}
	bodyBuilder.WriteString(fmt.Sprintf("--%s--\n", boundary))

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(bodyBuilder.String()))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Should return 413 Request Entity Too Large
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}

	// Check error message contains limit information
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Maximum allowed: 100") {
		t.Errorf("Response should contain default limit information. Body: %s", responseBody)
	}
}
