package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// OrderWithDate represents an order entity with date fields for testing
type OrderWithDate struct {
	ID        uint      `json:"ID" gorm:"primaryKey" odata:"key"`
	OrderNo   string    `json:"OrderNo" gorm:"not null"`
	Amount    float64   `json:"Amount" gorm:"not null"`
	OrderDate time.Time `json:"OrderDate" gorm:"not null"`
}

func setupDateFunctionTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the models
	if err := db.AutoMigrate(&OrderWithDate{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data with specific dates and times
	orders := []OrderWithDate{
		{
			ID:        1,
			OrderNo:   "ORD001",
			Amount:    100.00,
			OrderDate: time.Date(2024, 12, 25, 14, 30, 0, 0, time.UTC),
		},
		{
			ID:        2,
			OrderNo:   "ORD002",
			Amount:    200.00,
			OrderDate: time.Date(2024, 12, 26, 10, 15, 30, 0, time.UTC),
		},
		{
			ID:        3,
			OrderNo:   "ORD003",
			Amount:    150.00,
			OrderDate: time.Date(2024, 6, 15, 9, 0, 0, 0, time.UTC),
		},
		{
			ID:        4,
			OrderNo:   "ORD004",
			Amount:    300.00,
			OrderDate: time.Date(2023, 12, 25, 16, 45, 15, 0, time.UTC),
		},
		{
			ID:        5,
			OrderNo:   "ORD005",
			Amount:    250.00,
			OrderDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	if err := db.Create(&orders).Error; err != nil {
		t.Fatalf("Failed to seed orders: %v", err)
	}

	return db
}

// TestDateFunctions_Year tests the year() function
func TestDateFunctions_Year(t *testing.T) {
	db := setupDateFunctionTestDB(t)

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&OrderWithDate{}); err != nil {
		t.Fatalf("Failed to register OrderWithDate entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "Filter by year equals 2024",
			filter:        "year(OrderDate) eq 2024",
			expectedCount: 4, // ORD001, ORD002, ORD003, ORD005
			description:   "Should return orders from year 2024",
		},
		{
			name:          "Filter by year equals 2023",
			filter:        "year(OrderDate) eq 2023",
			expectedCount: 1, // ORD004
			description:   "Should return orders from year 2023",
		},
		{
			name:          "Filter by year greater than 2023",
			filter:        "year(OrderDate) gt 2023",
			expectedCount: 4, // ORD001, ORD002, ORD003, ORD005
			description:   "Should return orders from years after 2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/OrderWithDates?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatalf("Response missing 'value' field or not an array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("%s: Expected %d results, got %d", tt.description, tt.expectedCount, len(value))
			}
		})
	}
}

// TestDateFunctions_Month tests the month() function
func TestDateFunctions_Month(t *testing.T) {
	db := setupDateFunctionTestDB(t)

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&OrderWithDate{}); err != nil {
		t.Fatalf("Failed to register OrderWithDate entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "Filter by month equals 12",
			filter:        "month(OrderDate) eq 12",
			expectedCount: 3, // ORD001, ORD002, ORD004
			description:   "Should return orders from December",
		},
		{
			name:          "Filter by month equals 6",
			filter:        "month(OrderDate) eq 6",
			expectedCount: 1, // ORD003
			description:   "Should return orders from June",
		},
		{
			name:          "Filter by month greater than or equal to 6",
			filter:        "month(OrderDate) ge 6",
			expectedCount: 4, // ORD001, ORD002, ORD003, ORD004
			description:   "Should return orders from June onwards",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/OrderWithDates?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatalf("Response missing 'value' field or not an array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("%s: Expected %d results, got %d", tt.description, tt.expectedCount, len(value))
			}
		})
	}
}

// TestDateFunctions_Day tests the day() function
func TestDateFunctions_Day(t *testing.T) {
	db := setupDateFunctionTestDB(t)

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&OrderWithDate{}); err != nil {
		t.Fatalf("Failed to register OrderWithDate entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "Filter by day equals 25",
			filter:        "day(OrderDate) eq 25",
			expectedCount: 2, // ORD001, ORD004
			description:   "Should return orders on day 25",
		},
		{
			name:          "Filter by day greater than 20",
			filter:        "day(OrderDate) gt 20",
			expectedCount: 3, // ORD001, ORD002, ORD004
			description:   "Should return orders after day 20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/OrderWithDates?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatalf("Response missing 'value' field or not an array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("%s: Expected %d results, got %d", tt.description, tt.expectedCount, len(value))
			}
		})
	}
}

// TestDateFunctions_Hour tests the hour() function
func TestDateFunctions_Hour(t *testing.T) {
	db := setupDateFunctionTestDB(t)

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&OrderWithDate{}); err != nil {
		t.Fatalf("Failed to register OrderWithDate entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "Filter by hour equals 14",
			filter:        "hour(OrderDate) eq 14",
			expectedCount: 1, // ORD001
			description:   "Should return orders at hour 14",
		},
		{
			name:          "Filter by hour less than 12",
			filter:        "hour(OrderDate) lt 12",
			expectedCount: 3, // ORD002, ORD003, ORD005
			description:   "Should return orders in the morning",
		},
		{
			name:          "Filter by hour greater than or equal to 10",
			filter:        "hour(OrderDate) ge 10",
			expectedCount: 3, // ORD001, ORD002, ORD004
			description:   "Should return orders at or after 10 AM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/OrderWithDates?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatalf("Response missing 'value' field or not an array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("%s: Expected %d results, got %d", tt.description, tt.expectedCount, len(value))
			}
		})
	}
}

// TestDateFunctions_Combined tests combined date function filters
func TestDateFunctions_Combined(t *testing.T) {
	db := setupDateFunctionTestDB(t)

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&OrderWithDate{}); err != nil {
		t.Fatalf("Failed to register OrderWithDate entity: %v", err)
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		description   string
	}{
		{
			name:          "Filter by year and month",
			filter:        "year(OrderDate) eq 2024 and month(OrderDate) eq 12",
			expectedCount: 2, // ORD001, ORD002
			description:   "Should return orders from December 2024",
		},
		{
			name:          "Filter by date components",
			filter:        "year(OrderDate) eq 2024 and month(OrderDate) eq 12 and day(OrderDate) eq 25",
			expectedCount: 1, // ORD001
			description:   "Should return orders on December 25, 2024",
		},
		{
			name:          "Filter by time components",
			filter:        "hour(OrderDate) ge 9 and hour(OrderDate) le 17",
			expectedCount: 4, // Business hours: ORD001, ORD002, ORD003, ORD004
			description:   "Should return orders during business hours",
		},
		{
			name:          "Complex date filter with OR",
			filter:        "month(OrderDate) eq 12 or month(OrderDate) eq 1",
			expectedCount: 4, // ORD001, ORD002, ORD004, ORD005
			description:   "Should return orders from December or January",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/OrderWithDates?$filter="+url.QueryEscape(tt.filter), nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			value, ok := response["value"].([]interface{})
			if !ok {
				t.Fatalf("Response missing 'value' field or not an array")
			}

			if len(value) != tt.expectedCount {
				t.Errorf("%s: Expected %d results, got %d", tt.description, tt.expectedCount, len(value))
			}
		})
	}
}
