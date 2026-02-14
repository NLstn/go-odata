package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/observability"
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
	entityHandler := NewEntityHandler(db, entityMeta, nil)
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

	batchHandler := NewBatchHandler(db, handlers, serviceHandler, 100)
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

func TestParseHTTPRequestHandlesLargeBody(t *testing.T) {
	handler := &BatchHandler{}

	largeBody := strings.Repeat("x", 70*1024)
	request := fmt.Sprintf("POST /BatchTestProducts HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(largeBody), largeBody)

	parsed, err := handler.parseHTTPRequest(strings.NewReader(request))
	if err != nil {
		t.Fatalf("parseHTTPRequest returned error: %v", err)
	}

	if parsed.Method != http.MethodPost {
		t.Fatalf("Method = %s, want %s", parsed.Method, http.MethodPost)
	}

	if parsed.URL != "/BatchTestProducts" {
		t.Fatalf("URL = %s, want /BatchTestProducts", parsed.URL)
	}

	if got := parsed.Headers.Get("Content-Length"); got != fmt.Sprintf("%d", len(largeBody)) {
		t.Fatalf("Content-Length header = %s, want %d", got, len(largeBody))
	}

	if len(parsed.Body) != len(largeBody) {
		t.Fatalf("Body length = %d, want %d", len(parsed.Body), len(largeBody))
	}

	if string(parsed.Body) != largeBody {
		t.Fatalf("Body mismatch: first 32 bytes %q", string(parsed.Body)[:32])
	}
}

func TestGenerateBoundaryProducesUniqueValues(t *testing.T) {
	boundaries := make(map[string]struct{})

	for i := 0; i < 10; i++ {
		boundary := generateBoundary()
		if boundary == "" {
			t.Fatalf("expected boundary to be non-empty")
		}

		if _, exists := boundaries[boundary]; exists {
			t.Fatalf("duplicate boundary generated: %s", boundary)
		}
		boundaries[boundary] = struct{}{}
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
			name: "GET request without leading slash",
			input: `GET BatchTestProducts(1) HTTP/1.1
Host: localhost
Accept: application/json

`,
			expectError: false,
			method:      "GET",
			url:         "BatchTestProducts(1)",
		},
		{
			name: "GET request with filter containing spaces",
			input: `GET /BatchTestProducts?$filter=Name eq 'Widget' HTTP/1.1
Host: localhost
Accept: application/json

`,
			expectError: false,
			method:      "GET",
			url:         "/BatchTestProducts?$filter=Name eq 'Widget'",
		},
		{
			name: "GET request with filter and expand containing spaces",
			input: `GET /BatchTestProducts?$expand=Category&$filter=Price gt 100 HTTP/1.1
Host: localhost
Accept: application/json

`,
			expectError: false,
			method:      "GET",
			url:         "/BatchTestProducts?$expand=Category&$filter=Price gt 100",
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

	// Verify OData-Version header using direct map access (non-canonical capitalization)
	//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
	odataVersionValues := resp.Headers["OData-Version"]
	if len(odataVersionValues) == 0 || odataVersionValues[0] != "4.01" {
		t.Errorf("OData-Version = %v, want [4.01]", odataVersionValues)
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

func TestBatchHandler_ContentIDEcho(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Insert test data
	product := BatchTestProduct{
		ID:       1,
		Name:     "Test Product",
		Price:    99.99,
		Category: "Electronics",
	}
	db.Create(&product)

	// Create batch request with Content-ID header in the MIME part
	boundary := "batch_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: request1

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
		t.Fatalf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Per OData v4 spec, Content-ID MUST be echoed back in the response MIME part envelope
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Content-ID: request1") {
		t.Errorf("Response does not contain echoed Content-ID header. Body: %s", responseBody)
	}
}

func TestBatchHandler_ContentIDEcho_MultipleRequests(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Insert test data
	products := []BatchTestProduct{
		{ID: 1, Name: "Product 1", Price: 10.00, Category: "Category A"},
		{ID: 2, Name: "Product 2", Price: 20.00, Category: "Category B"},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Create batch request with multiple Content-IDs
	boundary := "batch_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: getProduct1

GET /BatchTestProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: getProduct2

GET /BatchTestProducts(2) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Per OData v4 spec, Content-ID MUST be echoed back in the response MIME part envelopes
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Content-ID: getProduct1") {
		t.Errorf("Response does not contain first Content-ID header. Body: %s", responseBody)
	}
	if !strings.Contains(responseBody, "Content-ID: getProduct2") {
		t.Errorf("Response does not contain second Content-ID header. Body: %s", responseBody)
	}
}

func TestBatchHandler_ContentIDEcho_Changeset(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Create batch request with changeset containing Content-IDs
	batchBoundary := "batch_boundary"
	changesetBoundary := "changeset_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /BatchTestProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 1","Price":10.00,"Category":"Electronics"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

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
		t.Fatalf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify products were created
	var count int64
	db.Model(&BatchTestProduct{}).Count(&count)
	if count != 2 {
		t.Fatalf("Expected 2 products in database, got %d", count)
	}

	// Per OData v4 spec, Content-ID MUST be echoed back in changeset response parts
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Content-ID: 1") {
		t.Errorf("Response does not contain first Content-ID header. Body: %s", responseBody)
	}
	if !strings.Contains(responseBody, "Content-ID: 2") {
		t.Errorf("Response does not contain second Content-ID header. Body: %s", responseBody)
	}
}

func TestBatchHandler_ContentIDEcho_NoContentID(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Insert test data
	product := BatchTestProduct{
		ID:       1,
		Name:     "Test Product",
		Price:    99.99,
		Category: "Electronics",
	}
	db.Create(&product)

	// Create batch request WITHOUT Content-ID header
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
		t.Fatalf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// When no Content-ID is provided, none should be in the response
	responseBody := w.Body.String()

	// Count occurrences of Content-ID in response - should only see Content-Type, not Content-ID
	if strings.Contains(responseBody, "Content-ID:") {
		t.Errorf("Response should not contain Content-ID header when none was provided. Body: %s", responseBody)
	}
}

func TestBatchHandler_URLWithoutLeadingSlash(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Insert test data
	product := BatchTestProduct{
		ID:       1,
		Name:     "Test Product",
		Price:    99.99,
		Category: "Electronics",
	}
	db.Create(&product)

	// Create batch request WITHOUT leading slash
	// This should not crash the server
	boundary := "batch_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET BatchTestProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	// This should not panic
	handler.HandleBatch(w, req)

	// Should get a valid response (either success or error, but not a crash)
	if w.Code == 0 {
		t.Error("No response code set, handler may have panicked")
	}

	// The request should either succeed or return a proper error response
	// but should never crash
	if w.Code != http.StatusOK {
		t.Logf("Response code: %d, Body: %s", w.Code, w.Body.String())
	}

	// Check response contains product data
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Test Product") {
		t.Errorf("Response does not contain product name. Body: %s", responseBody)
	}
}

func TestBatchHandler_ChangesetWithoutLeadingSlash(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Create batch request with changeset WITHOUT leading slash
	batchBoundary := "batch_boundary"
	changesetBoundary := "changeset_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST BatchTestProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 1","Price":10.00,"Category":"Electronics"}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	// This should not panic
	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify product was created
	var count int64
	db.Model(&BatchTestProduct{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 product in database, got %d", count)
	}
}

func TestBatchHandler_SetLogger(t *testing.T) {
	handler, _, _ := setupBatchTestHandler(t)

	// Test with custom logger
	customLogger := slog.Default()
	handler.SetLogger(customLogger)
	if handler.logger != customLogger {
		t.Error("Expected custom logger to be set")
	}

	// Test with nil logger (should use default)
	handler.SetLogger(nil)
	if handler.logger == nil {
		t.Error("Expected default logger when nil is passed")
	}
}

func TestBatchHandler_SetObservability(t *testing.T) {
	handler, _, _ := setupBatchTestHandler(t)

	cfg := &observability.Config{}
	handler.SetObservability(cfg)

	if handler.observability != cfg {
		t.Error("Expected observability config to be set")
	}
}

func TestBatchHandler_SetPreRequestHook(t *testing.T) {
	handler, _, _ := setupBatchTestHandler(t)

	hookCalled := false
	hook := func(r *http.Request) (context.Context, error) {
		hookCalled = true
		return r.Context(), nil
	}

	handler.SetPreRequestHook(hook)

	if handler.preRequestHook == nil {
		t.Error("Expected pre-request hook to be set")
	}

	// Test that the hook was set correctly by invoking it
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx, err := handler.preRequestHook(req)
	if err != nil {
		t.Errorf("Unexpected error from hook: %v", err)
	}
	if ctx == nil {
		t.Error("Expected context from hook")
	}
	if !hookCalled {
		t.Error("Expected hook to be called")
	}
}

func TestBatchHandler_MaxBatchSizeEnforcement(t *testing.T) {
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
	entityHandler := NewEntityHandler(db, entityMeta, nil)
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

	// Create handler with small max batch size for testing
	batchHandler := NewBatchHandler(db, handlers, serviceHandler, 2)

	// Insert test data
	products := []BatchTestProduct{
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

	batchHandler.HandleBatch(w, req)

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

func TestBatchHandler_MaxBatchSizeWithinLimit(t *testing.T) {
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
	entityHandler := NewEntityHandler(db, entityMeta, nil)
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

	// Create handler with max batch size of 2
	batchHandler := NewBatchHandler(db, handlers, serviceHandler, 2)

	// Insert test data
	products := []BatchTestProduct{
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

GET /BatchTestProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchTestProducts(2) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	batchHandler.HandleBatch(w, req)

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

func TestBatchHandler_MaxBatchSizeWithChangeset(t *testing.T) {
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
	entityHandler := NewEntityHandler(db, entityMeta, nil)
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

	// Create handler with max batch size of 2
	batchHandler := NewBatchHandler(db, handlers, serviceHandler, 2)

	// Create batch request with changeset containing 3 POST requests (exceeds limit)
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

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /BatchTestProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 3","Price":30.00,"Category":"Toys"}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	batchHandler.HandleBatch(w, req)

	// Should return 413 Request Entity Too Large
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusRequestEntityTooLarge, w.Body.String())
	}

	// Verify no products were created (transaction should be rolled back)
	var count int64
	db.Model(&BatchTestProduct{}).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 products in database due to batch size limit, got %d", count)
	}
}

func TestBatchHandler_FilterQueryWithSpaces(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)

	// Insert test data
	products := []BatchTestProduct{
		{ID: 1, Name: "Widget", Price: 10.00, Category: "Electronics"},
		{ID: 2, Name: "Gadget", Price: 20.00, Category: "Electronics"},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Create batch request with $filter containing spaces (e.g., eq 'Widget')
	// This previously caused a panic in httptest.NewRequest
	boundary := "batch_boundary"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /BatchTestProducts?$filter=Name eq 'Widget' HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	// This should not panic
	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "Widget") {
		t.Errorf("Response should contain 'Widget'. Body: %s", responseBody)
	}
}
