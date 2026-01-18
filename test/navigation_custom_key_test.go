package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestNavigationFilterWithCustomPrimaryKey tests filtering on navigation properties
// where the target entity has a custom primary key (not "id").
// This test validates that joins are created correctly even when the references: tag is not specified.
func TestNavigationFilterWithCustomPrimaryKey(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Define test entities with custom primary keys
	type Department struct {
		Code        string `json:"Code" gorm:"primaryKey;type:varchar(10)" odata:"key"`
		Name        string `json:"Name" gorm:"not null" odata:"required"`
		Description string `json:"Description"`
	}

	type Employee struct {
		ID             uint   `json:"ID" gorm:"primaryKey" odata:"key"`
		Name           string `json:"Name" gorm:"not null" odata:"required"`
		DepartmentCode string `json:"DepartmentCode" gorm:"type:varchar(10);not null" odata:"required"`
		// Note: No explicit references: tag - should auto-detect from Department's primary key
		Department *Department `json:"Department" gorm:"foreignKey:DepartmentCode"`
	}

	// Migrate and seed
	if err := db.AutoMigrate(&Department{}, &Employee{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	deptIT := Department{Code: "IT", Name: "Information Technology", Description: "IT Department"}
	deptHR := Department{Code: "HR", Name: "Human Resources", Description: "HR Department"}
	deptFN := Department{Code: "FN", Name: "Finance", Description: "Finance Department"}
	db.Create(&deptIT)
	db.Create(&deptHR)
	db.Create(&deptFN)

	empAlice := Employee{ID: 1, Name: "Alice", DepartmentCode: "IT"}
	empBob := Employee{ID: 2, Name: "Bob", DepartmentCode: "IT"}
	empCharlie := Employee{ID: 3, Name: "Charlie", DepartmentCode: "HR"}
	empDiana := Employee{ID: 4, Name: "Diana", DepartmentCode: "FN"}
	db.Create(&empAlice)
	db.Create(&empBob)
	db.Create(&empCharlie)
	db.Create(&empDiana)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&Department{})
	service.RegisterEntity(&Employee{})

	// Helper to build filter URLs
	buildFilterURL := func(path, filter string) string {
		return path + "?$filter=" + url.QueryEscape(filter)
	}

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedCount  int
		validate       func(t *testing.T, body []byte)
	}{
		{
			name:           "Filter by navigation property with custom primary key - Department/Name",
			url:            buildFilterURL("/Employees", "Department/Name eq 'Information Technology'"),
			expectedStatus: http.StatusOK,
			expectedCount:  2, // Alice and Bob are in IT
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 2 {
					t.Errorf("Expected 2 employees in IT, got %d", len(values))
					return
				}
				// Verify that both employees are in IT department
				for _, v := range values {
					emp := v.(map[string]interface{})
					if emp["DepartmentCode"] != "IT" {
						t.Errorf("Expected DepartmentCode 'IT', got %v", emp["DepartmentCode"])
					}
				}
			},
		},
		{
			name:           "Filter by navigation property with custom primary key - Department/Code",
			url:            buildFilterURL("/Employees", "Department/Code eq 'HR'"),
			expectedStatus: http.StatusOK,
			expectedCount:  1, // Only Charlie is in HR
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 1 {
					t.Errorf("Expected 1 employee in HR, got %d", len(values))
					return
				}
				emp := values[0].(map[string]interface{})
				if emp["Name"] != "Charlie" {
					t.Errorf("Expected employee 'Charlie', got %v", emp["Name"])
				}
			},
		},
		{
			name:           "Filter by navigation property - not equals",
			url:            buildFilterURL("/Employees", "Department/Code ne 'IT'"),
			expectedStatus: http.StatusOK,
			expectedCount:  2, // Charlie (HR) and Diana (FN) are not in IT
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				values, ok := response["value"].([]interface{})
				if !ok {
					t.Fatal("Expected value array in response")
				}

				if tt.expectedCount > 0 && len(values) != tt.expectedCount {
					t.Errorf("Expected %d results, got %d", tt.expectedCount, len(values))
				}

				if tt.validate != nil {
					tt.validate(t, w.Body.Bytes())
				}
			}
		})
	}
}
