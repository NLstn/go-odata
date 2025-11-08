package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/query"
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
		name             string
		url              string
		expectedStatus   int
		expectedCount    string
		expectedType     string
		ensurePlainText  bool
		maxBodyLength    int
		forbidJSONPrefix bool
	}{
		{
			name:             "Basic count",
			url:              "/Products/$count",
			expectedStatus:   http.StatusOK,
			expectedCount:    "5",
			expectedType:     "text/plain",
			ensurePlainText:  true,
			maxBodyLength:    10,
			forbidJSONPrefix: true,
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
			helperReq := httptest.NewRequest(http.MethodGet, tt.url, nil)
			queryOptions, err := query.ParseQueryOptions(helperReq.URL.Query(), handler.metadata)
			if err != nil {
				t.Fatalf("failed to parse query options: %v", err)
			}

			scopes, hookErr := callBeforeReadCollection(handler.metadata, helperReq, queryOptions)
			if hookErr != nil {
				t.Fatalf("before read hook failed: %v", hookErr)
			}

			helperCount, err := handler.countEntities(context.Background(), queryOptions, scopes)
			if err != nil {
				t.Fatalf("countEntities returned error: %v", err)
			}

			helperCountStr := strconv.FormatInt(helperCount, 10)
			if helperCountStr != tt.expectedCount {
				t.Fatalf("countEntities = %s, want %s", helperCountStr, tt.expectedCount)
			}

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
			if tt.ensurePlainText {
				if contentType == "application/json" || contentType == "application/json;odata.metadata=minimal" {
					t.Error("Count endpoint should not return JSON content type")
				}
			}

			body := w.Body.String()
			if body != tt.expectedCount {
				t.Errorf("Body = %v, want %v", body, tt.expectedCount)
			}
			if body != helperCountStr {
				t.Errorf("Body = %v, want helper count %v", body, helperCountStr)
			}
			if tt.ensurePlainText {
				if tt.maxBodyLength > 0 && len(body) > tt.maxBodyLength {
					t.Errorf("Response body length = %d, want <= %d", len(body), tt.maxBodyLength)
				}
				if tt.forbidJSONPrefix && (len(body) > 0 && (body[0] == '{' || body[0] == '[')) {
					t.Errorf("Response appears to be JSON, should be plain text: %s", body)
				}
			}
		})
	}
}

func TestCountConsistencyAcrossEndpoints(t *testing.T) {
	handler, db := setupProductHandler(t)

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
		name     string
		filter   string
		expected int64
	}{
		{
			name:     "All products",
			expected: 5,
		},
		{
			name:     "Electronics only",
			filter:   "Category eq 'Electronics'",
			expected: 3,
		},
		{
			name:     "Furniture only",
			filter:   "Category eq 'Furniture'",
			expected: 2,
		},
		{
			name:     "Price less than 200",
			filter:   "Price lt 200",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{}
			if tt.filter != "" {
				values.Set("$filter", tt.filter)
			}

			countURL := "/Products/$count"
			if raw := values.Encode(); raw != "" {
				countURL += "?" + raw
			}

			helperReq := httptest.NewRequest(http.MethodGet, countURL, nil)
			queryOptions, err := query.ParseQueryOptions(helperReq.URL.Query(), handler.metadata)
			if err != nil {
				t.Fatalf("failed to parse query options: %v", err)
			}

			scopes, hookErr := callBeforeReadCollection(handler.metadata, helperReq, queryOptions)
			if hookErr != nil {
				t.Fatalf("before read hook failed: %v", hookErr)
			}

			helperCount, err := handler.countEntities(context.Background(), queryOptions, scopes)
			if err != nil {
				t.Fatalf("countEntities returned error: %v", err)
			}
			if helperCount != tt.expected {
				t.Fatalf("countEntities = %d, want %d", helperCount, tt.expected)
			}

			reqCount := httptest.NewRequest(http.MethodGet, countURL, nil)
			wCount := httptest.NewRecorder()
			handler.HandleCount(wCount, reqCount)

			if wCount.Code != http.StatusOK {
				t.Fatalf("$count status = %d, want %d", wCount.Code, http.StatusOK)
			}

			countBody := strings.TrimSpace(wCount.Body.String())
			expectedBody := strconv.FormatInt(tt.expected, 10)
			if countBody != expectedBody {
				t.Fatalf("$count body = %s, want %s", countBody, expectedBody)
			}

			collectionValues := url.Values{}
			if tt.filter != "" {
				collectionValues.Set("$filter", tt.filter)
			}
			collectionValues.Set("$count", "true")

			collectionURL := "/Products"
			if raw := collectionValues.Encode(); raw != "" {
				collectionURL += "?" + raw
			}

			reqCollection := httptest.NewRequest(http.MethodGet, collectionURL, nil)
			wCollection := httptest.NewRecorder()
			handler.HandleCollection(wCollection, reqCollection)

			if wCollection.Code != http.StatusOK {
				t.Fatalf("collection status = %d, want %d", wCollection.Code, http.StatusOK)
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(wCollection.Body.Bytes(), &payload); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			countValue, ok := payload["@odata.count"].(float64)
			if !ok {
				t.Fatalf("expected @odata.count in response, got %v", payload["@odata.count"])
			}

			if int64(countValue) != tt.expected {
				t.Fatalf("@odata.count = %d, want %d", int64(countValue), tt.expected)
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
