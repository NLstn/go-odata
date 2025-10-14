package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// HeaderCapTestProduct is a test entity for header capitalization tests
type HeaderCapTestProduct struct {
	ID    int     `json:"id" gorm:"primarykey;autoIncrement" odata:"key"`
	Name  string  `json:"name" odata:"required"`
	Price float64 `json:"price"`
}

func setupHeaderCapTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&HeaderCapTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service := odata.NewService(db)
	if err := service.RegisterEntity(HeaderCapTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

// TestODataHeaderExactCapitalization verifies that OData headers are set with exact capitalization
// including the capital 'D' in "OData-Version" and "OData-EntityId" as required by OData v4 spec.
// This test accesses headers directly using non-canonical keys to verify exact capitalization.
func TestODataHeaderExactCapitalization(t *testing.T) {
	service, db := setupHeaderCapTestService(t)

	// Create a product for update/delete tests
	product := HeaderCapTestProduct{Name: "Test Product", Price: 99.99}
	db.Create(&product)

	tests := []struct {
		name           string
		method         string
		path           string
		body           map[string]interface{}
		headers        map[string]string
		expectedStatus int
		checkVersion   bool
		checkEntityId  bool
	}{
		{
			name:           "GET Collection - OData-Version",
			method:         http.MethodGet,
			path:           "/HeaderCapTestProducts",
			expectedStatus: http.StatusOK,
			checkVersion:   true,
			checkEntityId:  false,
		},
		{
			name:           "GET Entity - OData-Version",
			method:         http.MethodGet,
			path:           "/HeaderCapTestProducts(1)",
			expectedStatus: http.StatusOK,
			checkVersion:   true,
			checkEntityId:  false,
		},
		{
			name:           "GET Service Document - OData-Version",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusOK,
			checkVersion:   true,
			checkEntityId:  false,
		},
		{
			name:           "GET Metadata - OData-Version",
			method:         http.MethodGet,
			path:           "/$metadata",
			expectedStatus: http.StatusOK,
			checkVersion:   true,
			checkEntityId:  false,
		},
		{
			name:           "POST with return=minimal - Both Headers",
			method:         http.MethodPost,
			path:           "/HeaderCapTestProducts",
			body:           map[string]interface{}{"name": "New Product", "price": 49.99},
			headers:        map[string]string{"Prefer": "return=minimal"},
			expectedStatus: http.StatusNoContent,
			checkVersion:   true,
			checkEntityId:  true,
		},
		{
			name:           "PATCH with return=minimal - Both Headers",
			method:         http.MethodPatch,
			path:           "/HeaderCapTestProducts(1)",
			body:           map[string]interface{}{"price": 129.99},
			expectedStatus: http.StatusNoContent,
			checkVersion:   true,
			checkEntityId:  true,
		},
		{
			name:           "PUT with return=minimal - Both Headers",
			method:         http.MethodPut,
			path:           "/HeaderCapTestProducts(1)",
			body:           map[string]interface{}{"name": "Updated Product", "price": 159.99},
			expectedStatus: http.StatusNoContent,
			checkVersion:   true,
			checkEntityId:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody *bytes.Buffer
			if tt.body != nil {
				bodyBytes, _ := json.Marshal(tt.body)
				reqBody = bytes.NewBuffer(bodyBytes)
			} else {
				reqBody = &bytes.Buffer{}
			}

			req := httptest.NewRequest(tt.method, tt.path, reqBody)
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v. Body: %s", w.Code, tt.expectedStatus, w.Body.String())
			}

			if tt.checkVersion {
				// Verify OData-Version header with exact capitalization
				// Access directly with exact casing (OData-Version with capital 'D')
				//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
				odataVersionValues := w.Header()["OData-Version"]
				if len(odataVersionValues) == 0 || odataVersionValues[0] != "4.0" {
					t.Errorf("OData-Version header with exact capitalization not found. Got: %v", odataVersionValues)
				}

				// Note: We also set the canonical form so Header.Get() works, which is acceptable
			}

			if tt.checkEntityId {
				// Verify OData-EntityId header with exact capitalization
				// Access directly with exact casing (OData-EntityId with capital 'D')
				//nolint:staticcheck // SA1008: intentionally using non-canonical header key per OData spec
				entityIdValues := w.Header()["OData-EntityId"]
				if len(entityIdValues) == 0 {
					t.Error("OData-EntityId header with exact capitalization not found")
				}

				// Note: We also set the canonical form so Header.Get() works, which is acceptable
			}
		})
	}
}


