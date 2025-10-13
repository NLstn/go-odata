package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
)

// TestDateFunctions_ComputeIntegration tests date function extraction using $compute
func TestDateFunctions_ComputeIntegration(t *testing.T) {
	db := setupDateFunctionTestDB(t)

	service := odata.NewService(db)
	if err := service.RegisterEntity(&OrderWithDate{}); err != nil {
		t.Fatalf("Failed to register OrderWithDate entity: %v", err)
	}

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		validate       func(*testing.T, map[string]interface{})
	}{
		{
			name:           "Extract year from date",
			query:          "$compute=year(OrderDate)%20as%20OrderYear&$select=OrderNo,OrderYear",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) == 0 {
					t.Fatal("Expected non-empty result")
				}
				
				// Check first item has OrderYear field
				firstItem := value[0].(map[string]interface{})
				if _, hasOrderYear := firstItem["OrderYear"]; !hasOrderYear {
					t.Error("Expected OrderYear field in result")
				}
				if _, hasOrderNo := firstItem["OrderNo"]; !hasOrderNo {
					t.Error("Expected OrderNo field in result")
				}
			},
		},
		{
			name:           "Extract month from date",
			query:          "$compute=month(OrderDate)%20as%20OrderMonth&$select=OrderNo,OrderMonth",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) == 0 {
					t.Fatal("Expected non-empty result")
				}
				
				firstItem := value[0].(map[string]interface{})
				if _, hasOrderMonth := firstItem["OrderMonth"]; !hasOrderMonth {
					t.Error("Expected OrderMonth field in result")
				}
			},
		},
		{
			name:           "Extract multiple date components",
			query:          "$compute=year(OrderDate)%20as%20Year,month(OrderDate)%20as%20Month,day(OrderDate)%20as%20Day&$select=OrderNo,Year,Month,Day",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) == 0 {
					t.Fatal("Expected non-empty result")
				}
				
				firstItem := value[0].(map[string]interface{})
				if _, hasYear := firstItem["Year"]; !hasYear {
					t.Error("Expected Year field in result")
				}
				if _, hasMonth := firstItem["Month"]; !hasMonth {
					t.Error("Expected Month field in result")
				}
				if _, hasDay := firstItem["Day"]; !hasDay {
					t.Error("Expected Day field in result")
				}
			},
		},
		{
			name:           "Extract hour from datetime",
			query:          "$compute=hour(OrderDate)%20as%20Hour&$select=OrderNo,Hour",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) == 0 {
					t.Fatal("Expected non-empty result")
				}
				
				firstItem := value[0].(map[string]interface{})
				if _, hasHour := firstItem["Hour"]; !hasHour {
					t.Error("Expected Hour field in result")
				}
			},
		},
		{
			name:           "Extract minute from datetime",
			query:          "$compute=minute(OrderDate)%20as%20Minute&$select=OrderNo,Minute",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) == 0 {
					t.Fatal("Expected non-empty result")
				}
				
				firstItem := value[0].(map[string]interface{})
				if _, hasMinute := firstItem["Minute"]; !hasMinute {
					t.Error("Expected Minute field in result")
				}
			},
		},
		{
			name:           "Extract second from datetime",
			query:          "$compute=second(OrderDate)%20as%20Second&$select=OrderNo,Second",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) == 0 {
					t.Fatal("Expected non-empty result")
				}
				
				firstItem := value[0].(map[string]interface{})
				if _, hasSecond := firstItem["Second"]; !hasSecond {
					t.Error("Expected Second field in result")
				}
			},
		},
		{
			name:           "Extract date part",
			query:          "$compute=date(OrderDate)%20as%20DatePart&$select=OrderNo,DatePart",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) == 0 {
					t.Fatal("Expected non-empty result")
				}
				
				firstItem := value[0].(map[string]interface{})
				if _, hasDatePart := firstItem["DatePart"]; !hasDatePart {
					t.Error("Expected DatePart field in result")
				}
			},
		},
		{
			name:           "Extract time part",
			query:          "$compute=time(OrderDate)%20as%20TimePart&$select=OrderNo,TimePart",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) == 0 {
					t.Fatal("Expected non-empty result")
				}
				
				firstItem := value[0].(map[string]interface{})
				if _, hasTimePart := firstItem["TimePart"]; !hasTimePart {
					t.Error("Expected TimePart field in result")
				}
			},
		},
		{
			name:           "Compute without select (should return all fields plus computed)",
			query:          "$compute=year(OrderDate)%20as%20OrderYear",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				if len(value) == 0 {
					t.Fatal("Expected non-empty result")
				}
				
				firstItem := value[0].(map[string]interface{})
				// Should have all original fields plus OrderYear
				if _, hasID := firstItem["ID"]; !hasID {
					t.Error("Expected ID field in result")
				}
				if _, hasOrderNo := firstItem["OrderNo"]; !hasOrderNo {
					t.Error("Expected OrderNo field in result")
				}
				if _, hasOrderYear := firstItem["OrderYear"]; !hasOrderYear {
					t.Error("Expected OrderYear field in result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, "/OrderWithDates?"+tt.query, nil)
			w := httptest.NewRecorder()

			// Handle request
			service.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.expectedStatus != http.StatusOK {
				return
			}

			// Parse response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Run validation
			if tt.validate != nil {
				tt.validate(t, response)
			}
		})
	}
}

// TestDateFunctions_ComputeWithFilter tests compute combined with filter
func TestDateFunctions_ComputeWithFilter(t *testing.T) {
	db := setupDateFunctionTestDB(t)

	service := odata.NewService(db)
	if err := service.RegisterEntity(&OrderWithDate{}); err != nil {
		t.Fatalf("Failed to register OrderWithDate entity: %v", err)
	}

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		validate       func(*testing.T, map[string]interface{})
	}{
		{
			name:           "Compute with filter on year",
			query:          "$compute=year(OrderDate)%20as%20Year&$filter=year(OrderDate)%20eq%202024&$select=OrderNo,Year",
			expectedStatus: http.StatusOK,
			validate: func(t *testing.T, response map[string]interface{}) {
				value, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("value is not an array")
				}
				// Should return only 2024 orders
				for _, item := range value {
					itemMap := item.(map[string]interface{})
					year, ok := itemMap["Year"].(float64)
					if !ok {
						t.Error("Year field is not a number")
						continue
					}
					if int(year) != 2024 {
						t.Errorf("Expected year 2024, got %d", int(year))
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/OrderWithDates?"+tt.query, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.expectedStatus != http.StatusOK {
				return
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, response)
			}
		})
	}
}
