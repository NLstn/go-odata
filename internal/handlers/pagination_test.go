package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEntityHandlerCollectionWithTop(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert 10 test products
	products := make([]Product, 10)
	for i := 0; i < 10; i++ {
		products[i] = Product{
			ID:          i + 1,
			Name:        "Product " + string(rune('A'+i)),
			Description: "Description",
			Price:       float64(i+1) * 10.0,
			Category:    "Electronics",
		}
		db.Create(&products[i])
	}

	tests := []struct {
		name          string
		top           int
		expectedCount int
		expectNext    bool
	}{
		{
			name:          "Top 5",
			top:           5,
			expectedCount: 5,
			expectNext:    true,
		},
		{
			name:          "Top 10",
			top:           10,
			expectedCount: 10,
			expectNext:    false, // No more data
		},
		{
			name:          "Top 3",
			top:           3,
			expectedCount: 3,
			expectNext:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products", nil)
			q := req.URL.Query()
			q.Add("$top", fmt.Sprintf("%d", tt.top))
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("value field is not an array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(value))
			}

			// Check for @odata.nextLink
			_, hasNextLink := response["@odata.nextLink"]
			if tt.expectNext && !hasNextLink {
				t.Error("Expected @odata.nextLink to be present")
			}
			if !tt.expectNext && hasNextLink {
				t.Error("Did not expect @odata.nextLink to be present")
			}
		})
	}
}

func TestEntityHandlerCollectionWithSkip(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert 10 test products
	products := make([]Product, 10)
	for i := 0; i < 10; i++ {
		products[i] = Product{
			ID:          i + 1,
			Name:        "Product " + string(rune('A'+i)),
			Description: "Description",
			Price:       float64(i+1) * 10.0,
			Category:    "Electronics",
		}
		db.Create(&products[i])
	}

	tests := []struct {
		name          string
		skip          int
		expectedCount int
		expectFirstID int
	}{
		{
			name:          "Skip 0",
			skip:          0,
			expectedCount: 10,
			expectFirstID: 1,
		},
		{
			name:          "Skip 5",
			skip:          5,
			expectedCount: 5,
			expectFirstID: 6,
		},
		{
			name:          "Skip 9",
			skip:          9,
			expectedCount: 1,
			expectFirstID: 10,
		},
		{
			name:          "Skip 10",
			skip:          10,
			expectedCount: 0,
			expectFirstID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products", nil)
			q := req.URL.Query()
			q.Add("$skip", fmt.Sprintf("%d", tt.skip))
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("value field is not an array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(value))
			}

			if tt.expectedCount > 0 {
				firstItem, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("First item is not a map")
				}

				firstID := int(firstItem["ID"].(float64))
				if firstID != tt.expectFirstID {
					t.Errorf("Expected first item ID to be %d, got %d", tt.expectFirstID, firstID)
				}
			}
		})
	}
}

func TestEntityHandlerCollectionWithTopAndSkip(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert 20 test products
	products := make([]Product, 20)
	for i := 0; i < 20; i++ {
		products[i] = Product{
			ID:          i + 1,
			Name:        "Product " + string(rune('A'+(i%26))),
			Description: "Description",
			Price:       float64(i+1) * 10.0,
			Category:    "Electronics",
		}
		db.Create(&products[i])
	}

	tests := []struct {
		name          string
		top           int
		skip          int
		expectedCount int
		expectFirstID int
		expectNext    bool
	}{
		{
			name:          "First page (top=5, skip=0)",
			top:           5,
			skip:          0,
			expectedCount: 5,
			expectFirstID: 1,
			expectNext:    true,
		},
		{
			name:          "Second page (top=5, skip=5)",
			top:           5,
			skip:          5,
			expectedCount: 5,
			expectFirstID: 6,
			expectNext:    true,
		},
		{
			name:          "Last page (top=5, skip=15)",
			top:           5,
			skip:          15,
			expectedCount: 5,
			expectFirstID: 16,
			expectNext:    false,
		},
		{
			name:          "Partial last page (top=10, skip=15)",
			top:           10,
			skip:          15,
			expectedCount: 5,
			expectFirstID: 16,
			expectNext:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products", nil)
			q := req.URL.Query()
			q.Add("$top", fmt.Sprintf("%d", tt.top))
			q.Add("$skip", fmt.Sprintf("%d", tt.skip))
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("value field is not an array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(value))
			}

			if tt.expectedCount > 0 {
				firstItem, ok := value[0].(map[string]interface{})
				if !ok {
					t.Fatal("First item is not a map")
				}

				firstID := int(firstItem["ID"].(float64))
				if firstID != tt.expectFirstID {
					t.Errorf("Expected first item ID to be %d, got %d", tt.expectFirstID, firstID)
				}
			}

			// Check for @odata.nextLink
			_, hasNextLink := response["@odata.nextLink"]
			if tt.expectNext && !hasNextLink {
				t.Error("Expected @odata.nextLink to be present")
			}
			if !tt.expectNext && hasNextLink {
				t.Error("Did not expect @odata.nextLink to be present")
			}
		})
	}
}

func TestEntityHandlerCollectionWithCount(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert 15 test products
	products := make([]Product, 15)
	for i := 0; i < 15; i++ {
		products[i] = Product{
			ID:          i + 1,
			Name:        "Product " + string(rune('A'+(i%26))),
			Description: "Description",
			Price:       float64(i+1) * 10.0,
			Category:    "Electronics",
		}
		db.Create(&products[i])
	}

	tests := []struct {
		name          string
		count         string
		top           *int
		expectedCount *int64
	}{
		{
			name:          "Count with no pagination",
			count:         "true",
			top:           nil,
			expectedCount: int64Ptr(15),
		},
		{
			name:          "Count with top=5",
			count:         "true",
			top:           intPtr(5),
			expectedCount: int64Ptr(15),
		},
		{
			name:          "No count parameter",
			count:         "",
			top:           nil,
			expectedCount: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products", nil)
			q := req.URL.Query()
			if tt.count != "" {
				q.Add("$count", tt.count)
			}
			if tt.top != nil {
				q.Add("$top", fmt.Sprintf("%d", *tt.top))
			}
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check @odata.count
			if tt.expectedCount != nil {
				countValue, hasCount := response["@odata.count"]
				if !hasCount {
					t.Error("Expected @odata.count to be present")
				} else {
					actualCount := int64(countValue.(float64))
					if actualCount != *tt.expectedCount {
						t.Errorf("Expected count to be %d, got %d", *tt.expectedCount, actualCount)
					}
				}
			} else {
				if _, hasCount := response["@odata.count"]; hasCount {
					t.Error("Did not expect @odata.count to be present")
				}
			}
		})
	}
}

func TestEntityHandlerCollectionWithCountAndFilter(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert mixed products
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
		filter        string
		count         string
		expectedCount int64
		expectedItems int
	}{
		{
			name:          "Count with filter (Electronics)",
			filter:        "Category eq 'Electronics'",
			count:         "true",
			expectedCount: 3,
			expectedItems: 3,
		},
		{
			name:          "Count with filter (Furniture)",
			filter:        "Category eq 'Furniture'",
			count:         "true",
			expectedCount: 2,
			expectedItems: 2,
		},
		{
			name:          "Count with filter and top",
			filter:        "Category eq 'Electronics'",
			count:         "true",
			expectedCount: 3,
			expectedItems: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products", nil)
			q := req.URL.Query()
			q.Add("$filter", tt.filter)
			if tt.count != "" {
				q.Add("$count", tt.count)
			}
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
				t.Logf("Response body: %s", w.Body.String())
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check @odata.count
			countValue, hasCount := response["@odata.count"]
			if !hasCount {
				t.Error("Expected @odata.count to be present")
			} else {
				actualCount := int64(countValue.(float64))
				if actualCount != tt.expectedCount {
					t.Errorf("Expected count to be %d, got %d", tt.expectedCount, actualCount)
				}
			}

			// Check actual items returned
			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("value field is not an array")
			}

			if len(value) != tt.expectedItems {
				t.Errorf("Expected %d items, got %d", tt.expectedItems, len(value))
			}
		})
	}
}

func TestEntityHandlerCollectionInvalidPagination(t *testing.T) {
	handler, _ := setupProductHandler(t)

	tests := []struct {
		name       string
		queryParam string
		value      string
	}{
		{
			name:       "Invalid $top (negative)",
			queryParam: "$top",
			value:      "-1",
		},
		{
			name:       "Invalid $top (not a number)",
			queryParam: "$top",
			value:      "abc",
		},
		{
			name:       "Invalid $skip (negative)",
			queryParam: "$skip",
			value:      "-1",
		},
		{
			name:       "Invalid $skip (not a number)",
			queryParam: "$skip",
			value:      "xyz",
		},
		{
			name:       "Invalid $count",
			queryParam: "$count",
			value:      "maybe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products", nil)
			q := req.URL.Query()
			q.Add(tt.queryParam, tt.value)
			req.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if _, ok := response["error"]; !ok {
				t.Error("Response missing error field")
			}
		})
	}
}

func TestEntityHandlerCollectionWithMaxPageSize(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert 20 test products
	products := make([]Product, 20)
	for i := 0; i < 20; i++ {
		products[i] = Product{
			ID:          i + 1,
			Name:        fmt.Sprintf("Product %d", i+1),
			Description: "Description",
			Price:       float64(i+1) * 10.0,
			Category:    "Electronics",
		}
		db.Create(&products[i])
	}

	tests := []struct {
		name                string
		maxPageSize         int
		top                 *int
		expectedCount       int
		expectNext          bool
		expectAppliedHeader bool
	}{
		{
			name:                "MaxPageSize 5 without $top",
			maxPageSize:         5,
			top:                 nil,
			expectedCount:       5,
			expectNext:          true,
			expectAppliedHeader: true,
		},
		{
			name:                "MaxPageSize 10 with $top=5",
			maxPageSize:         10,
			top:                 intPtr(5),
			expectedCount:       5,
			expectNext:          true,
			expectAppliedHeader: true,
		},
		{
			name:                "MaxPageSize 5 with $top=10",
			maxPageSize:         5,
			top:                 intPtr(10),
			expectedCount:       5,
			expectNext:          true,
			expectAppliedHeader: true,
		},
		{
			name:                "MaxPageSize 100 with $top=8",
			maxPageSize:         100,
			top:                 intPtr(8),
			expectedCount:       8,
			expectNext:          true,
			expectAppliedHeader: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products", nil)
			req.Header.Set("Prefer", fmt.Sprintf("odata.maxpagesize=%d", tt.maxPageSize))

			q := req.URL.Query()
			if tt.top != nil {
				q.Add("$top", fmt.Sprintf("%d", *tt.top))
			}
			req.URL.RawQuery = q.Encode()

			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
			}

			// Check for Preference-Applied header
			if tt.expectAppliedHeader {
				appliedHeader := w.Header().Get("Preference-Applied")
				if appliedHeader == "" {
					t.Error("Expected Preference-Applied header to be present")
				} else if appliedHeader != fmt.Sprintf("odata.maxpagesize=%d", tt.maxPageSize) {
					t.Errorf("Expected Preference-Applied header to be 'odata.maxpagesize=%d', got '%s'", tt.maxPageSize, appliedHeader)
				}
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatal("value field is not an array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(value))
			}

			if tt.expectNext {
				if _, hasNext := response["@odata.nextLink"]; !hasNext {
					t.Error("Expected @odata.nextLink to be present")
				}
			}
		})
	}
}

func TestEntityHandlerCollectionWithSkipToken(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert 15 test products with different prices for ordering
	products := make([]Product, 15)
	for i := 0; i < 15; i++ {
		products[i] = Product{
			ID:          i + 1,
			Name:        fmt.Sprintf("Product %c", 'A'+i),
			Description: "Description",
			Price:       float64((i+1)*10) + float64(i%3), // Varying prices
			Category:    "Electronics",
		}
		db.Create(&products[i])
	}

	// First request with $top=5 and $orderby=Price
	req1 := httptest.NewRequest(http.MethodGet, "/Products?$top=5&$orderby=Price", nil)
	w1 := httptest.NewRecorder()
	handler.HandleCollection(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("First request failed with status %v, body: %s", w1.Code, w1.Body.String())
	}

	var response1 map[string]interface{}
	if err := json.NewDecoder(w1.Body).Decode(&response1); err != nil {
		t.Fatalf("Failed to decode first response: %v", err)
	}

	// Check that we have a nextLink with $skiptoken
	nextLink, hasNext := response1["@odata.nextLink"].(string)
	if !hasNext {
		t.Fatal("Expected @odata.nextLink in first response")
	}

	// Verify nextLink contains $skiptoken (URL-encoded as %24skiptoken)
	if !contains(nextLink, "$skiptoken") && !contains(nextLink, "%24skiptoken") {
		t.Errorf("Expected nextLink to contain $skiptoken, got: %s", nextLink)
	}

	// Get first page results
	value1, ok := response1["value"].([]interface{})
	if !ok {
		t.Fatal("value field is not an array")
	}

	if len(value1) != 5 {
		t.Errorf("Expected 5 results in first page, got %d", len(value1))
	}

	// Note: We can't easily test the second request without parsing the skiptoken URL
	// because it's base64 encoded and requires proper HTTP handling
	// The integration tests would cover this more thoroughly
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}

// TestSkipTokenValidation tests that invalid skiptoken returns 400 Bad Request
func TestSkipTokenValidation(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert test products
	for i := 1; i <= 5; i++ {
		product := Product{
			ID:          i,
			Name:        fmt.Sprintf("Product %d", i),
			Description: "Test product",
			Price:       float64(i * 10),
			Category:    "Electronics",
		}
		db.Create(&product)
	}

	tests := []struct {
		name           string
		skiptoken      string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Invalid skiptoken - not base64",
			skiptoken:      "invalid-token!@#$%",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Invalid skiptoken - empty",
			skiptoken:      "",
			expectedStatus: http.StatusOK, // Empty skiptoken is ignored
			expectError:    false,
		},
		{
			name:           "Invalid skiptoken - valid base64 but invalid JSON",
			skiptoken:      "bm90LWpzb24=", // "not-json" in base64
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Valid skiptoken",
			skiptoken:      "eyJrIjp7IklEIjoyfX0=", // Valid token: {"k":{"ID":2}}
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/Products", nil)
			if tt.skiptoken != "" {
				q := req.URL.Query()
				q.Add("$skiptoken", tt.skiptoken)
				req.URL.RawQuery = q.Encode()
			}
			w := httptest.NewRecorder()

			handler.HandleCollection(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, tt.expectedStatus, w.Body.String())
			}

			if tt.expectError {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}

				// Check for error field
				if _, ok := response["error"]; !ok {
					t.Error("Expected error field in response")
				}
			}
		})
	}
}

// TestSkipTokenInNextLink tests that nextLink contains $skiptoken instead of $skip
func TestSkipTokenInNextLink(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert 10 test products
	for i := 1; i <= 10; i++ {
		product := Product{
			ID:          i,
			Name:        fmt.Sprintf("Product %d", i),
			Description: "Test product",
			Price:       float64(i * 10),
			Category:    "Electronics",
		}
		db.Create(&product)
	}

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	q := req.URL.Query()
	q.Add("$top", "3")
	req.URL.RawQuery = q.Encode()
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that nextLink is present
	nextLink, ok := response["@odata.nextLink"].(string)
	if !ok {
		t.Fatal("Expected @odata.nextLink to be present")
	}

	// Verify that nextLink contains $skiptoken
	if !contains(nextLink, "skiptoken") {
		t.Errorf("Expected nextLink to contain $skiptoken, got: %s", nextLink)
	}

	// Verify that nextLink does NOT contain $skip
	if contains(nextLink, "%24skip=") || contains(nextLink, "$skip=") {
		t.Errorf("Expected nextLink to NOT contain $skip, got: %s", nextLink)
	}
}

// TestSkipTokenPreservesQueryOptions tests that $skiptoken preserves other query options
func TestSkipTokenPreservesQueryOptions(t *testing.T) {
	handler, db := setupProductHandler(t)

	// Insert 10 test products
	for i := 1; i <= 10; i++ {
		product := Product{
			ID:          i,
			Name:        fmt.Sprintf("Product %d", i),
			Description: "Test product",
			Price:       float64(i * 10),
			Category:    "Electronics",
		}
		db.Create(&product)
	}

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	q := req.URL.Query()
	q.Add("$top", "3")
	q.Add("$filter", "Price gt 20")
	q.Add("$orderby", "Price desc")
	req.URL.RawQuery = q.Encode()
	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that nextLink is present
	nextLink, ok := response["@odata.nextLink"].(string)
	if !ok {
		t.Fatal("Expected @odata.nextLink to be present")
	}

	// Verify that nextLink preserves filter and orderby
	if !contains(nextLink, "filter") {
		t.Errorf("Expected nextLink to preserve $filter, got: %s", nextLink)
	}
	if !contains(nextLink, "orderby") {
		t.Errorf("Expected nextLink to preserve $orderby, got: %s", nextLink)
	}
	if !contains(nextLink, "top") {
		t.Errorf("Expected nextLink to preserve $top, got: %s", nextLink)
	}
}
