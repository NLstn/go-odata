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

// TestProductPropertyAccess is for testing property access
type TestProductPropertyAccess struct {
	ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string  `json:"Name" gorm:"not null" odata:"required,maxlength=100"`
	Price    float64 `json:"Price" gorm:"not null" odata:"required"`
	Category string  `json:"Category" odata:"maxlength=50"`
}

// TestCategoryPropertyAccess is for testing navigation property distinction
type TestCategoryPropertyAccess struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name" odata:"required"`
}

// TestProductWithNavPropertyAccess tests property access with navigation properties
type TestProductWithNavPropertyAccess struct {
	ID         uint                            `json:"ID" gorm:"primaryKey" odata:"key"`
	Name       string                          `json:"Name" odata:"required"`
	CategoryID uint                            `json:"CategoryID"`
	Category   *TestCategoryPropertyAccess `json:"Category" gorm:"foreignKey:CategoryID"`
}

func setupPropertyAccessTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestProductPropertyAccess{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&TestProductPropertyAccess{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func setupPropertyAccessWithNavTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&TestProductWithNavPropertyAccess{}, &TestCategoryPropertyAccess{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(&TestProductWithNavPropertyAccess{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}
	if err := service.RegisterEntity(&TestCategoryPropertyAccess{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestPropertyAccess_StructuralProperty verifies that structural properties can be accessed correctly
func TestPropertyAccess_StructuralProperty(t *testing.T) {
	service, db := setupPropertyAccessTestService(t)

	product := TestProductPropertyAccess{
		ID:       1,
		Name:     "Laptop",
		Price:    999.99,
		Category: "Electronics",
	}
	db.Create(&product)

	tests := []struct {
		name       string
		url        string
		wantStatus int
		wantValue  interface{}
	}{
		{
			name:       "Access Name property",
			url:        "/TestProductPropertyAccesses(1)/Name",
			wantStatus: http.StatusOK,
			wantValue:  "Laptop",
		},
		{
			name:       "Access Price property",
			url:        "/TestProductPropertyAccesses(1)/Price",
			wantStatus: http.StatusOK,
			wantValue:  999.99,
		},
		{
			name:       "Access Category property",
			url:        "/TestProductPropertyAccesses(1)/Category",
			wantStatus: http.StatusOK,
			wantValue:  "Electronics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, tt.wantStatus, w.Body.String())
				return
			}

			if tt.wantStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Verify @odata.context is present
				if _, ok := response["@odata.context"]; !ok {
					t.Error("Response missing @odata.context")
				}

				// Verify value
				if response["value"] != tt.wantValue {
					t.Errorf("value = %v, want %v", response["value"], tt.wantValue)
				}
			}
		})
	}
}

// TestPropertyAccess_ValueEndpoint verifies that $value endpoint works correctly
func TestPropertyAccess_ValueEndpoint(t *testing.T) {
	service, db := setupPropertyAccessTestService(t)

	product := TestProductPropertyAccess{
		ID:       1,
		Name:     "Laptop",
		Price:    999.99,
		Category: "Electronics",
	}
	db.Create(&product)

	tests := []struct {
		name        string
		url         string
		wantStatus  int
		wantBody    string
		wantContent string
	}{
		{
			name:        "Access Name/$value",
			url:         "/TestProductPropertyAccesses(1)/Name/$value",
			wantStatus:  http.StatusOK,
			wantBody:    "Laptop",
			wantContent: "text/plain",
		},
		{
			name:        "Access Price/$value",
			url:         "/TestProductPropertyAccesses(1)/Price/$value",
			wantStatus:  http.StatusOK,
			wantBody:    "999.99",
			wantContent: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, tt.wantStatus, w.Body.String())
				return
			}

			if tt.wantStatus == http.StatusOK {
				contentType := w.Header().Get("Content-Type")
				if contentType != tt.wantContent+"; charset=utf-8" {
					t.Errorf("Content-Type = %v, want %v", contentType, tt.wantContent)
				}

				body := w.Body.String()
				if body != tt.wantBody {
					t.Errorf("Body = %v, want %v", body, tt.wantBody)
				}
			}
		})
	}
}

// TestPropertyAccess_NavigationVsStructural verifies proper distinction between navigation and structural properties
func TestPropertyAccess_NavigationVsStructural(t *testing.T) {
	service, db := setupPropertyAccessWithNavTestService(t)

	category := TestCategoryPropertyAccess{ID: 1, Name: "Electronics"}
	db.Create(&category)

	product := TestProductWithNavPropertyAccess{
		ID:         1,
		Name:       "Laptop",
		CategoryID: 1,
	}
	db.Create(&product)

	tests := []struct {
		name       string
		url        string
		wantStatus int
		isNav      bool
	}{
		{
			name:       "Structural property Name should return value wrapper",
			url:        "/TestProductWithNavPropertyAccesses(1)/Name",
			wantStatus: http.StatusOK,
			isNav:      false,
		},
		{
			name:       "Navigation property Category should return entity",
			url:        "/TestProductWithNavPropertyAccesses(1)/Category",
			wantStatus: http.StatusOK,
			isNav:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, tt.wantStatus, w.Body.String())
				return
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if tt.isNav {
				// Navigation property should return entity with properties, not a value wrapper
				if _, hasValue := response["value"]; hasValue {
					// If it has "value", check if it's an entity with ID
					if _, hasID := response["ID"]; !hasID {
						t.Error("Navigation property should return entity properties")
					}
				}
				// Should have entity properties like ID and Name
				if _, hasID := response["ID"]; !hasID {
					t.Error("Navigation property response missing entity properties")
				}
			} else {
				// Structural property should have value wrapper
				if _, hasValue := response["value"]; !hasValue {
					t.Error("Structural property should have value wrapper")
				}
				// Should NOT have entity properties like ID
				if _, hasID := response["ID"]; hasID {
					t.Error("Structural property should not have entity properties")
				}
			}
		})
	}
}

// TestPropertyAccess_ValueOnNavigationProperty verifies that $value is rejected on navigation properties
func TestPropertyAccess_ValueOnNavigationProperty(t *testing.T) {
	service, db := setupPropertyAccessWithNavTestService(t)

	category := TestCategoryPropertyAccess{ID: 1, Name: "Electronics"}
	db.Create(&category)

	product := TestProductWithNavPropertyAccess{
		ID:         1,
		Name:       "Laptop",
		CategoryID: 1,
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductWithNavPropertyAccesses(1)/Category/$value", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v for $value on navigation property. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}
}

// TestPropertyAccess_NonexistentProperty verifies proper error for nonexistent properties
func TestPropertyAccess_NonexistentProperty(t *testing.T) {
	service, db := setupPropertyAccessTestService(t)

	product := TestProductPropertyAccess{
		ID:       1,
		Name:     "Laptop",
		Price:    999.99,
		Category: "Electronics",
	}
	db.Create(&product)

	req := httptest.NewRequest(http.MethodGet, "/TestProductPropertyAccesses(1)/NonexistentProperty", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v for nonexistent property. Body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Response missing error field")
	}

	// Verify the error message mentions it's not a valid property (not specifically "navigation property")
	errorMap, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Error is not a map")
	}

	message, ok := errorMap["message"].(string)
	if !ok {
		t.Fatal("Error message is not a string")
	}

	if message != "Property not found" {
		t.Errorf("Error message = %v, want 'Property not found'", message)
	}
}

// TestPropertyAccess_MethodNotAllowed verifies that only GET is allowed for property access
func TestPropertyAccess_MethodNotAllowed(t *testing.T) {
	service, db := setupPropertyAccessTestService(t)

	product := TestProductPropertyAccess{
		ID:       1,
		Name:     "Laptop",
		Price:    999.99,
		Category: "Electronics",
	}
	db.Create(&product)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/TestProductPropertyAccesses(1)/Name", nil)
			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Status = %v, want %v for %s method. Body: %s", w.Code, http.StatusMethodNotAllowed, method, w.Body.String())
			}
		})
	}
}
