package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEntityHandlerCount(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert test products
	products := []Product{
		{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics"},
		{ID: 3, Name: "Keyboard", Price: 149.99, Category: "Electronics"},
		{ID: 4, Name: "Chair", Price: 249.99, Category: "Furniture"},
		{ID: 5, Name: "Desk", Price: 399.99, Category: "Furniture"},
	}
	for _, product := range products {
		db.Create(&product)
	}

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedCount  string
		expectedType   string
	}{
		{
			name:           "Basic count",
			url:            "/Products/$count",
			expectedStatus: http.StatusOK,
			expectedCount:  "5",
			expectedType:   "text/plain",
		},
		{
			name:           "Count with filter - Electronics",
			url:            "/Products/$count?$filter=Category%20eq%20%27Electronics%27",
			expectedStatus: http.StatusOK,
			expectedCount:  "3",
			expectedType:   "text/plain",
		},
		{
			name:           "Count with filter - Furniture",
			url:            "/Products/$count?$filter=Category%20eq%20%27Furniture%27",
			expectedStatus: http.StatusOK,
			expectedCount:  "2",
			expectedType:   "text/plain",
		},
		{
			name:           "Count with filter - Price gt 100",
			url:            "/Products/$count?$filter=Price%20gt%20100",
			expectedStatus: http.StatusOK,
			expectedCount:  "4",
			expectedType:   "text/plain",
		},
		{
			name:           "Count with filter - Price lt 50",
			url:            "/Products/$count?$filter=Price%20lt%2050",
			expectedStatus: http.StatusOK,
			expectedCount:  "1",
			expectedType:   "text/plain",
		},
		{
			name:           "Count with filter - No matches",
			url:            "/Products/$count?$filter=Category%20eq%20%27NonExistent%27",
			expectedStatus: http.StatusOK,
			expectedCount:  "0",
			expectedType:   "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.HandleCount(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.expectedStatus)
				t.Logf("Response body: %s", w.Body.String())
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("Content-Type = %v, want %v", contentType, tt.expectedType)
			}

			body := w.Body.String()
			if body != tt.expectedCount {
				t.Errorf("Body = %v, want %v", body, tt.expectedCount)
			}
		})
	}
}

func TestEntityHandlerCountInvalidMethod(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert one product
	db.Create(&Product{ID: 1, Name: "Test", Price: 99.99, Category: "Test"})

	req := httptest.NewRequest(http.MethodPost, "/Products/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleCount(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestEntityHandlerCountInvalidFilter(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert one product
	db.Create(&Product{ID: 1, Name: "Test", Price: 99.99, Category: "Test"})

	req := httptest.NewRequest(http.MethodGet, "/Products/$count?$filter=InvalidProperty%20eq%20%27value%27", nil)
	w := httptest.NewRecorder()

	handler.HandleCount(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestEntityHandlerCountEmptyCollection(t *testing.T) {
	handler, _ := setupProductHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/Products/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleCount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body != "0" {
		t.Errorf("Body = %v, want %v", body, "0")
	}
}

// Test that $count endpoint returns plain text, not JSON
func TestEntityHandlerCountReturnsPlainText(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert test products
	products := []Product{
		{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics"},
		{ID: 3, Name: "Chair", Price: 249.99, Category: "Furniture"},
	}
	for _, product := range products {
		db.Create(&product)
	}

	req := httptest.NewRequest(http.MethodGet, "/Products/$count", nil)
	w := httptest.NewRecorder()

	handler.HandleCount(w, req)

	// Verify status is OK
	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify Content-Type is text/plain, not JSON
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Content-Type = %v, want text/plain", contentType)
	}
	if contentType == "application/json" || contentType == "application/json;odata.metadata=minimal" {
		t.Error("Count endpoint should not return JSON")
	}

	// Verify body is just a number, not a JSON object
	body := w.Body.String()
	if body != "3" {
		t.Errorf("Body = %v, want 3", body)
	}

	// Verify it's not a JSON response with a "value" property
	if len(body) > 10 || body[0] == '{' || body[0] == '[' {
		t.Errorf("Response appears to be JSON, should be plain text: %s", body)
	}
}

// Test that $count endpoint works with complex filters
func TestEntityHandlerCountComplexFilter(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert test products
	products := []Product{
		{ID: 1, Name: "Laptop Pro", Price: 1999.99, Category: "Electronics"},
		{ID: 2, Name: "Laptop Basic", Price: 799.99, Category: "Electronics"},
		{ID: 3, Name: "Mouse Wireless", Price: 49.99, Category: "Electronics"},
		{ID: 4, Name: "Chair Pro", Price: 499.99, Category: "Furniture"},
		{ID: 5, Name: "Chair Basic", Price: 149.99, Category: "Furniture"},
	}
	for _, product := range products {
		db.Create(&product)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount string
	}{
		{
			name:          "Contains function",
			filter:        "contains(Name,%27Laptop%27)",
			expectedCount: "2",
		},
		{
			name:          "StartsWith function",
			filter:        "startswith(Name,%27Chair%27)",
			expectedCount: "2",
		},
		{
			name:          "EndsWith function",
			filter:        "endswith(Name,%27Pro%27)",
			expectedCount: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products/$count?$filter="+tt.filter, nil)
			w := httptest.NewRecorder()

			handler.HandleCount(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
				t.Logf("Response body: %s", w.Body.String())
			}

			body := w.Body.String()
			if body != tt.expectedCount {
				t.Errorf("Body = %v, want %v", body, tt.expectedCount)
			}
		})
	}
}

// Test that $count endpoint ignores query options other than $filter
// According to OData v4 spec, only $filter and $search should apply to $count
func TestEntityHandlerCountIgnoresOtherQueryOptions(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert test products
	products := []Product{
		{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics"},
		{ID: 3, Name: "Keyboard", Price: 149.99, Category: "Electronics"},
		{ID: 4, Name: "Chair", Price: 249.99, Category: "Furniture"},
		{ID: 5, Name: "Desk", Price: 399.99, Category: "Furniture"},
	}
	for _, product := range products {
		db.Create(&product)
	}

	tests := []struct {
		name          string
		url           string
		expectedCount string
		description   string
	}{
		{
			name:          "With $top - should be ignored",
			url:           "/Products/$count?$top=2",
			expectedCount: "5",
			description:   "$top should not affect count",
		},
		{
			name:          "With $skip - should be ignored",
			url:           "/Products/$count?$skip=2",
			expectedCount: "5",
			description:   "$skip should not affect count",
		},
		{
			name:          "With $orderby - should be ignored",
			url:           "/Products/$count?$orderby=Name",
			expectedCount: "5",
			description:   "$orderby should not affect count",
		},
		{
			name:          "With $select - should be ignored",
			url:           "/Products/$count?$select=Name",
			expectedCount: "5",
			description:   "$select should not affect count",
		},
		{
			name:          "With $filter and $top - only filter should apply",
			url:           "/Products/$count?$filter=Category%20eq%20%27Electronics%27&$top=2",
			expectedCount: "3",
			description:   "Only $filter should apply, $top should be ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.HandleCount(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
				t.Logf("Response body: %s", w.Body.String())
			}

			body := w.Body.String()
			if body != tt.expectedCount {
				t.Errorf("Body = %v, want %v (%s)", body, tt.expectedCount, tt.description)
			}
		})
	}
}
