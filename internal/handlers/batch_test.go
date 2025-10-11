package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// BatchTestProduct is a test entity for batch operations
type BatchTestProduct struct {
	ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name        string  `json:"Name"`
	Price       float64 `json:"Price"`
	Category    string  `json:"Category"`
	Description string  `json:"Description"`
}

func setupBatchTestHandler(t *testing.T) (*BatchHandler, *gorm.DB, http.Handler) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&BatchTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(BatchTestProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handlers := make(map[string]*EntityHandler)
	entityHandler := NewEntityHandler(db, entityMeta)
	handlers["BatchTestProducts"] = entityHandler

	// Create a simple service handler for testing
	serviceHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if strings.HasPrefix(path, "BatchTestProducts") {
			if strings.Contains(path, "(") {
				// Entity request
				keyStart := strings.Index(path, "(")
				keyEnd := strings.Index(path, ")")
				if keyStart > 0 && keyEnd > keyStart {
					key := path[keyStart+1 : keyEnd]
					entityHandler.HandleEntity(w, r, key)
				}
			} else {
				// Collection request
				entityHandler.HandleCollection(w, r)
			}
		}
	})

	batchHandler := NewBatchHandler(db, handlers, serviceHandler)
	return batchHandler, db, serviceHandler
}

func TestBatchHandler_MethodNotAllowed(t *testing.T) {
	handler, _, _ := setupBatchTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/$batch", nil)
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestBatchHandler_InvalidContentType(t *testing.T) {
	handler, _, _ := setupBatchTestHandler(t)

	tests := []struct {
		name        string
		contentType string
	}{
		{
			name:        "JSON content type",
			contentType: "application/json",
		},
		{
			name:        "Missing boundary",
			contentType: "multipart/mixed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader("test"))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			handler.HandleBatch(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestBatchHandler_SingleGetRequest(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Insert test data
	product := BatchTestProduct{
		ID:       1,
		Name:     "Test Product",
		Price:    99.99,
		Category: "Electronics",
	}
	db.Create(&product)

	// Create batch request
	boundary := "batch_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchTestProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check response content type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "multipart/mixed") {
		t.Errorf("Content-Type = %v, want multipart/mixed", contentType)
	}

	// Check OData-Version header
	odataVersion := w.Header().Get("OData-Version")
	if odataVersion != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", odataVersion)
	}

	// Check response contains product data
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Test Product") {
		t.Errorf("Response does not contain product name. Body: %s", responseBody)
	}
}

func TestBatchHandler_MultipleGetRequests(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Insert test data
	products := []BatchTestProduct{
		{ID: 1, Name: "Product 1", Price: 10.00, Category: "Category A"},
		{ID: 2, Name: "Product 2", Price: 20.00, Category: "Category B"},
		{ID: 3, Name: "Product 3", Price: 30.00, Category: "Category C"},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Create batch request with multiple GET requests
	boundary := "batch_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchTestProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchTestProducts(2) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchTestProducts(3) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check response contains all products
	responseBody := w.Body.String()
	for _, p := range products {
		if !strings.Contains(responseBody, p.Name) {
			t.Errorf("Response does not contain product: %s. Body: %s", p.Name, responseBody)
		}
	}

	// Count number of HTTP responses in batch
	count := strings.Count(responseBody, "HTTP/1.1")
	if count != 3 {
		t.Errorf("Expected 3 HTTP responses, got %d", count)
	}
}

func TestBatchHandler_Changeset(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Create batch request with changeset
	batchBoundary := "batch_boundary"
	changesetBoundary := "changeset_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchTestProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 1","Price":10.00,"Category":"Electronics"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchTestProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 2","Price":20.00,"Category":"Books"}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify both products were created
	var count int64
	db.Model(&BatchTestProduct{}).Count(&count)
	if count != 2 {
		t.Errorf("Expected 2 products in database, got %d", count)
	}

	// Verify products exist
	var products []BatchTestProduct
	db.Find(&products)
	names := []string{}
	for _, p := range products {
		names = append(names, p.Name)
	}

	expectedNames := []string{"Product 1", "Product 2"}
	for _, expected := range expectedNames {
		found := false
		for _, actual := range names {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected product %s not found in database", expected)
		}
	}
}

func TestBatchHandler_MixedGetAndChangeset(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Insert initial product
	product := BatchTestProduct{
		ID:       1,
		Name:     "Existing Product",
		Price:    50.00,
		Category: "Electronics",
	}
	db.Create(&product)

	// Create batch request with GET and changeset
	batchBoundary := "batch_boundary"
	changesetBoundary := "changeset_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchTestProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchTestProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"New Product","Price":100.00,"Category":"Books"}

--%s--

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchTestProducts HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, batchBoundary, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify new product was created
	var count int64
	db.Model(&BatchTestProduct{}).Count(&count)
	if count != 2 {
		t.Errorf("Expected 2 products in database, got %d", count)
	}

	// Check response contains both products
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Existing Product") {
		t.Errorf("Response does not contain existing product. Body: %s", responseBody)
	}
}

func TestBatchHandler_EmptyBatch(t *testing.T) {
	handler, _, _ := setupBatchTestHandler(t)

	// Create empty batch request (properly formatted multipart with no parts)
	boundary := "batch_boundary"
	body := fmt.Sprintf("--%s--\r\n", boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Response should be valid multipart but empty
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "multipart/mixed") {
		t.Errorf("Content-Type = %v, want multipart/mixed", contentType)
	}
}

func TestBatchHandler_GetCollection(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Insert test data
	products := []BatchTestProduct{
		{ID: 1, Name: "Product 1", Price: 10.00, Category: "Electronics"},
		{ID: 2, Name: "Product 2", Price: 20.00, Category: "Books"},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Create batch request to get collection
	boundary := "batch_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchTestProducts HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

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

func TestBatchHandler_ErrorInRequest(t *testing.T) {
	handler, _, _ := setupBatchTestHandler(t)

	// Create batch request with invalid entity key
	boundary := "batch_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchTestProducts(999) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v (batch request itself should succeed)", w.Code, http.StatusOK)
	}

	// Response should contain error for the specific request
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "404") && !strings.Contains(responseBody, "not found") {
		t.Errorf("Response should contain 404 error. Body: %s", responseBody)
	}
}

func TestBatchHandler_ParseHTTPRequest(t *testing.T) {
	handler, _, _ := setupBatchTestHandler(t)

	tests := []struct {
		name        string
		input       string
		expectError bool
		method      string
		url         string
	}{
		{
			name: "Simple GET request",
			input: `GET /BatchTestProducts(1) HTTP/1.1
Host: localhost
Accept: application/json

`,
			expectError: false,
			method:      "GET",
			url:         "/BatchTestProducts(1)",
		},
		{
			name: "POST request with body",
			input: `POST /BatchTestProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Test"}`,
			expectError: false,
			method:      "POST",
			url:         "/BatchTestProducts",
		},
		{
			name:        "Empty request",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			req, err := handler.parseHTTPRequest(reader)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if req.Method != tt.method {
				t.Errorf("Method = %v, want %v", req.Method, tt.method)
			}

			if req.URL != tt.url {
				t.Errorf("URL = %v, want %v", req.URL, tt.url)
			}
		})
	}
}

func TestBatchHandler_CreateErrorResponse(t *testing.T) {
	handler, _, _ := setupBatchTestHandler(t)

	resp := handler.createErrorResponse(http.StatusBadRequest, "Test error")

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %v, want %v", resp.StatusCode, http.StatusBadRequest)
	}

	if resp.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", resp.Headers.Get("Content-Type"))
	}

	if resp.Headers.Get("OData-Version") != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", resp.Headers.Get("OData-Version"))
	}

	if !strings.Contains(string(resp.Body), "Test error") {
		t.Errorf("Body does not contain error message: %s", string(resp.Body))
	}
}

func TestBatchHandler_WriteBatchResponse(t *testing.T) {
	handler, _, _ := setupBatchTestHandler(t)

	responses := []batchResponse{
		{
			StatusCode: http.StatusOK,
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       []byte(`{"result":"success"}`),
		},
		{
			StatusCode: http.StatusCreated,
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       []byte(`{"id":1}`),
		},
	}

	w := httptest.NewRecorder()
	handler.writeBatchResponse(w, responses)

	// Check headers
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "multipart/mixed") {
		t.Errorf("Content-Type = %v, want multipart/mixed", contentType)
	}

	if w.Header().Get("OData-Version") != "4.0" {
		t.Errorf("OData-Version = %v, want 4.0", w.Header().Get("OData-Version"))
	}

	// Check body contains both responses
	body := w.Body.String()
	if !strings.Contains(body, "HTTP/1.1 200") {
		t.Error("Response does not contain first status line")
	}

	if !strings.Contains(body, "HTTP/1.1 201") {
		t.Error("Response does not contain second status line")
	}

	if !strings.Contains(body, `{"result":"success"}`) {
		t.Error("Response does not contain first body")
	}

	if !strings.Contains(body, `{"id":1}`) {
		t.Error("Response does not contain second body")
	}
}


