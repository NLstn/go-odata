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

// TestBatchIntegration_SubRequestsGoThroughMiddleware verifies that batch sub-requests
// pass through middleware when SetBatchSubRequestHandler is configured.
// This ensures compliance with the OData spec that sub-requests should be treated
// as independent requests.
func TestBatchIntegration_SubRequestsGoThroughMiddleware(t *testing.T) {
	service, db := setupBatchIntegrationTest(t)

	// Insert test data
	product := BatchIntegrationProduct{
		ID:       1,
		Name:     "Test Product",
		Price:    99.99,
		Category: "Electronics",
	}
	db.Create(&product)

	// Track whether middleware was called for each request
	var middlewareCalls int

	// Create middleware that adds a header and tracks calls
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalls++
			// Add a custom header to prove middleware ran
			w.Header().Set("X-Middleware-Ran", "true")
			next.ServeHTTP(w, r)
		})
	}

	// Wrap the service with middleware
	handler := middleware(service)

	// Configure batch sub-requests to use the middleware-wrapped handler
	service.SetBatchSubRequestHandler(handler)

	// Create batch request with a single GET
	boundary := "batch_middleware_test"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary)

	// Reset middleware call count
	middlewareCalls = 0

	// Execute batch request through the middleware-wrapped handler
	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Middleware should have been called at least twice:
	// 1. For the outer batch request
	// 2. For the inner sub-request
	if middlewareCalls < 2 {
		t.Errorf("Expected middleware to be called at least 2 times (batch + sub-request), got %d", middlewareCalls)
	}

	// Verify the sub-request succeeded
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Test Product") {
		t.Errorf("Response does not contain product name. Body: %s", responseBody)
	}
}

// TestBatchIntegration_SubRequestsWithContextMiddleware verifies that batch sub-requests
// can access context values set by middleware.
func TestBatchIntegration_SubRequestsWithContextMiddleware(t *testing.T) {
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

	// Insert test data
	product := BatchIntegrationProduct{
		ID:       1,
		Name:     "Test Product",
		Price:    99.99,
		Category: "Electronics",
	}
	db.Create(&product)

	// Track whether the Authorization header was seen by the policy (proving sub-request went through middleware)
	var authHeaderSeen bool

	// Create a policy that checks for the Authorization header
	// This proves that the sub-request headers are being processed
	contextCheckPolicy := &batchAuthCheckPolicy{
		onCheck: func(found bool) {
			authHeaderSeen = found
		},
	}
	service.SetPolicy(contextCheckPolicy)

	// Create middleware that we'll verify sub-requests pass through
	middlewareCallCount := 0
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCallCount++
			next.ServeHTTP(w, r)
		})
	}

	// Wrap the service with middleware
	handler := middleware(service)

	// Configure batch sub-requests to use the middleware-wrapped handler
	service.SetBatchSubRequestHandler(handler)

	// Create batch request with Authorization header in the sub-request
	boundary := "batch_context_test"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchIntegrationProducts(1) HTTP/1.1
Host: localhost
Accept: application/json
Authorization: Bearer test-token


--%s--
`, boundary, boundary)

	// Reset tracking
	authHeaderSeen = false
	middlewareCallCount = 0

	// Execute batch request
	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify middleware was called for both batch request and sub-request
	if middlewareCallCount < 2 {
		t.Errorf("Expected middleware to be called at least 2 times, got %d", middlewareCallCount)
	}

	// Verify the Authorization header from sub-request was visible to the policy
	if !authHeaderSeen {
		t.Error("Authorization header from sub-request was not visible to policy")
	}

	// Verify the sub-request succeeded
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Test Product") {
		t.Errorf("Response does not contain product name. Body: %s", responseBody)
	}
}

// batchAuthCheckPolicy is a test policy that checks for Authorization header in requests
type batchAuthCheckPolicy struct {
	onCheck func(found bool)
}

func (p *batchAuthCheckPolicy) Authorize(ctx odata.AuthContext, resource odata.ResourceDescriptor, operation odata.Operation) odata.Decision {
	// Check if the Authorization header exists in the request
	if ctx.Request.Headers != nil && ctx.Request.Headers.Get("Authorization") != "" {
		p.onCheck(true)
	}
	return odata.Allow()
}
