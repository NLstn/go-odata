package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Integration test entities
type Department struct {
	ID        uint       `json:"ID" gorm:"primaryKey" odata:"key"`
	Name      string     `json:"Name"`
	Employees []Employee `json:"Employees" gorm:"foreignKey:DepartmentID"`
}

type Employee struct {
	ID           uint        `json:"ID" gorm:"primaryKey" odata:"key"`
	Name         string      `json:"Name"`
	DepartmentID uint        `json:"DepartmentID"`
	Department   *Department `json:"Department,omitempty" gorm:"foreignKey:DepartmentID"`
}

func setupIntegrationTest(t *testing.T) *odata.Service {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Department{}, &Employee{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	departments := []Department{
		{ID: 1, Name: "Engineering"},
		{ID: 2, Name: "Sales"},
		{ID: 3, Name: "Marketing"},
	}
	db.Create(&departments)

	employees := []Employee{
		{ID: 1, Name: "Alice", DepartmentID: 1},
		{ID: 2, Name: "Bob", DepartmentID: 1},
		{ID: 3, Name: "Charlie", DepartmentID: 1},
		{ID: 4, Name: "Diana", DepartmentID: 2},
		{ID: 5, Name: "Eve", DepartmentID: 2},
		{ID: 6, Name: "Frank", DepartmentID: 3},
	}
	db.Create(&employees)

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	_ = service.RegisterEntity(&Department{})
	_ = service.RegisterEntity(&Employee{})

	return service
}

// TestIntegrationExpandCollection tests $expand on entity collection
func TestIntegrationExpandCollection(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Departments?$expand=Employees", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(values) != 3 {
		t.Errorf("Expected 3 departments, got %d", len(values))
	}

	// Check that Engineering department has 3 employees
	firstDept := values[0].(map[string]interface{})
	employees, ok := firstDept["Employees"].([]interface{})
	if !ok {
		t.Fatal("Expected Employees to be expanded")
	}

	if len(employees) != 3 {
		t.Errorf("Expected 3 employees in Engineering, got %d", len(employees))
	}
}

// TestIntegrationExpandEntity tests $expand on single entity
func TestIntegrationExpandEntity(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Departments(1)?$expand=Employees", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	employees, ok := response["Employees"].([]interface{})
	if !ok {
		t.Fatal("Expected Employees to be expanded")
	}

	if len(employees) != 3 {
		t.Errorf("Expected 3 employees, got %d", len(employees))
	}
}

// TestIntegrationNavigationPath tests accessing navigation property via path
func TestIntegrationNavigationPath(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Departments(1)/Employees", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the @odata.context follows OData V4 spec for navigation paths
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("Expected @odata.context in response")
	}
	expectedContext := "http://example.com/$metadata#Departments(1)/Employees"
	if context != expectedContext {
		t.Errorf("Expected @odata.context to be '%s', got '%s'", expectedContext, context)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(values) != 3 {
		t.Errorf("Expected 3 employees, got %d", len(values))
	}
}

// TestIntegrationExpandWithNestedTop tests $expand with nested $top
func TestIntegrationExpandWithNestedTop(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Departments(1)?$expand=Employees($top=2)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	employees, ok := response["Employees"].([]interface{})
	if !ok {
		t.Fatal("Expected Employees to be expanded")
	}

	if len(employees) != 2 {
		t.Errorf("Expected 2 employees due to $top=2, got %d", len(employees))
	}
}

// TestIntegrationNavigationSingleEntity tests accessing single entity via navigation property
func TestIntegrationNavigationSingleEntity(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Employees(1)/Department", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the @odata.context follows OData V4 spec for single entity navigation
	context, ok := response["@odata.context"].(string)
	if !ok {
		t.Fatal("Expected @odata.context in response")
	}
	expectedContext := "http://example.com/$metadata#Employees(1)/Department/$entity"
	if context != expectedContext {
		t.Errorf("Expected @odata.context to be '%s', got '%s'", expectedContext, context)
	}

	// Verify the department data
	if response["ID"].(float64) != 1 {
		t.Errorf("Expected Department ID to be 1, got %v", response["ID"])
	}
	if response["Name"].(string) != "Engineering" {
		t.Errorf("Expected Department Name to be 'Engineering', got %s", response["Name"])
	}
}

// TestIntegrationExpandReverseNavigation tests expanding from child to parent
func TestIntegrationExpandReverseNavigation(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Employees(1)?$expand=Department", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	department, ok := response["Department"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected Department to be expanded")
	}

	deptName, ok := department["Name"].(string)
	if !ok || deptName != "Engineering" {
		t.Errorf("Expected department name 'Engineering', got %v", deptName)
	}
}

// TestIntegrationExpandWithFilter tests combining $expand with $filter
func TestIntegrationExpandWithFilter(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Departments?$filter=Name%20eq%20Engineering&$expand=Employees", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(values) != 1 {
		t.Errorf("Expected 1 department after filter, got %d", len(values))
	}

	dept := values[0].(map[string]interface{})
	employees, ok := dept["Employees"].([]interface{})
	if !ok {
		t.Fatal("Expected Employees to be expanded")
	}

	if len(employees) != 3 {
		t.Errorf("Expected 3 employees, got %d", len(employees))
	}
}

// TestIntegrationExpandWithCount tests combining $expand with $count
func TestIntegrationExpandWithCount(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Departments?$count=true&$expand=Employees", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	count, ok := response["@odata.count"].(float64)
	if !ok {
		t.Fatal("Expected @odata.count in response")
	}

	if int(count) != 3 {
		t.Errorf("Expected count of 3, got %d", int(count))
	}
}

// TestIntegrationExpandWithOrderBy tests combining $expand with $orderby
func TestIntegrationExpandWithOrderBy(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Departments?$orderby=Name%20desc&$expand=Employees&$top=2", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(values) != 2 {
		t.Errorf("Expected 2 departments due to $top=2, got %d", len(values))
	}

	// First should be "Sales" (descending order)
	firstDept := values[0].(map[string]interface{})
	firstName, _ := firstDept["Name"].(string)
	if firstName != "Sales" {
		t.Errorf("Expected first department to be 'Sales', got %s", firstName)
	}
}

// TestIntegrationMetadata tests that metadata includes navigation properties
func TestIntegrationMetadata(t *testing.T) {
	service := setupIntegrationTest(t)

	t.Run("XML format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()

		// Check for navigation property definitions
		if !contains(body, "NavigationProperty") {
			t.Error("Expected metadata to contain NavigationProperty elements")
		}

		if !contains(body, "Employees") {
			t.Error("Expected metadata to contain Employees navigation property")
		}

		if !contains(body, "Department") {
			t.Error("Expected metadata to contain Department navigation property")
		}

		if !contains(body, "NavigationPropertyBinding") {
			t.Error("Expected metadata to contain NavigationPropertyBinding elements")
		}
	})

	t.Run("JSON format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		// Check for basic structure
		if _, ok := response["$Version"]; !ok {
			t.Error("Expected $Version in JSON metadata")
		}

		odataService, ok := response["ODataService"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected ODataService in JSON metadata")
		}

		// Check for entity types
		if _, ok := odataService["Department"]; !ok {
			t.Error("Expected Department entity type in JSON metadata")
		}

		if _, ok := odataService["Employee"]; !ok {
			t.Error("Expected Employee entity type in JSON metadata")
		}

		// Check for container
		container, ok := odataService["Container"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected Container in JSON metadata")
		}

		// Check for entity sets
		if _, ok := container["Departments"]; !ok {
			t.Error("Expected Departments entity set in JSON metadata")
		}

		if _, ok := container["Employees"]; !ok {
			t.Error("Expected Employees entity set in JSON metadata")
		}
	})
}

// TestIntegrationServiceDocument tests that service document lists entity sets
func TestIntegrationServiceDocument(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	values, ok := response["value"].([]interface{})
	if !ok {
		t.Fatal("Expected value array in response")
	}

	if len(values) != 2 {
		t.Errorf("Expected 2 entity sets, got %d", len(values))
	}
}

// TestIntegrationInvalidNavigationProperty tests error handling for invalid navigation properties
func TestIntegrationInvalidNavigationProperty(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Departments?$expand=InvalidProperty", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestIntegrationNavigationPropertyNotFound tests 404 for invalid navigation paths
func TestIntegrationNavigationPropertyNotFound(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Departments(1)/InvalidProperty", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestIntegrationEntityNotFound tests 404 for non-existent entity in navigation
func TestIntegrationEntityNotFound(t *testing.T) {
	service := setupIntegrationTest(t)

	req := httptest.NewRequest(http.MethodGet, "/Departments(999)/Employees", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestIntegrationExpandEntityEmptyCollection tests $expand on single entity with empty collection
func TestIntegrationExpandEntityEmptyCollection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Department{}, &Employee{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data - department with no employees
	departments := []Department{
		{ID: 1, Name: "EmptyDepartment"},
		{ID: 2, Name: "NonEmptyDepartment"},
	}
	db.Create(&departments)

	employees := []Employee{
		{ID: 1, Name: "Alice", DepartmentID: 2},
	}
	db.Create(&employees)

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	_ = service.RegisterEntity(&Department{})
	_ = service.RegisterEntity(&Employee{})

	// Test expanding empty collection
	req := httptest.NewRequest(http.MethodGet, "/Departments(1)?$expand=Employees", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// The key assertion: Employees field MUST be present even when empty
	employeesField, ok := response["Employees"]
	if !ok {
		t.Fatal("Expected Employees field to be present in response when expanded, even if empty")
	}

	// Verify it's an array
	employeesArray, ok := employeesField.([]interface{})
	if !ok {
		t.Fatalf("Expected Employees to be an array, got %T", employeesField)
	}

	// Verify it's empty
	if len(employeesArray) != 0 {
		t.Errorf("Expected 0 employees, got %d", len(employeesArray))
	}
}
