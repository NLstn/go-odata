package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestSelectWithNavigationFilter tests combining $select with $filter using navigation properties
// This reproduces the ambiguous column reference issue when JOINs are present
func TestSelectWithNavigationFilter(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Define test entities matching the issue description
	type Club struct {
		ID      string `json:"ID" gorm:"type:varchar(36);primaryKey" odata:"key"`
		Name    string `json:"Name"`
		Deleted bool   `json:"Deleted"`
	}

	type Member struct {
		ID     string `json:"ID" gorm:"type:varchar(36);primaryKey" odata:"key"`
		Name   string `json:"Name"`
		ClubID string `json:"ClubID" gorm:"type:varchar(36)"`
		Club   *Club  `gorm:"foreignKey:ClubID" json:"Club,omitempty" odata:"nav"`
	}

	// Migrate and seed
	if err := db.AutoMigrate(&Club{}, &Member{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	club1 := Club{ID: "club-1", Name: "Active Club", Deleted: false}
	club2 := Club{ID: "club-2", Name: "Deleted Club", Deleted: true}
	db.Create(&club1)
	db.Create(&club2)

	member1 := Member{ID: "member-1", Name: "Alice", ClubID: "club-1"}
	member2 := Member{ID: "member-2", Name: "Bob", ClubID: "club-2"}
	member3 := Member{ID: "member-3", Name: "Charlie", ClubID: "club-1"}
	db.Create(&member1)
	db.Create(&member2)
	db.Create(&member3)

	// Create OData service
	service := odata.NewService(db)
	service.RegisterEntity(&Club{})
	service.RegisterEntity(&Member{})

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedCount  int
		validate       func(t *testing.T, body []byte)
	}{
		{
			name:           "Select ID with navigation filter",
			url:            "/Members?$select=ID&$filter=" + url.QueryEscape("Club/Deleted eq false"),
			expectedStatus: http.StatusOK,
			expectedCount:  2, // Alice and Charlie belong to active club
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values, ok := response["value"].([]interface{})
				if !ok {
					t.Fatalf("Expected 'value' array in response")
				}
				if len(values) != 2 {
					t.Errorf("Expected 2 members, got %d", len(values))
				}
				// Verify that only ID is returned (plus key properties)
				for i, v := range values {
					member := v.(map[string]interface{})
					if _, hasID := member["ID"]; !hasID {
						t.Errorf("Member %d missing ID field", i)
					}
					// Name should not be in the response since it wasn't selected
					if _, hasName := member["Name"]; hasName {
						t.Errorf("Member %d should not have Name field (not selected)", i)
					}
				}
			},
		},
		{
			name:           "Select Name with navigation filter",
			url:            "/Members?$select=Name&$filter=" + url.QueryEscape("Club/Name eq 'Active Club'"),
			expectedStatus: http.StatusOK,
			expectedCount:  2, // Alice and Charlie
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 2 {
					t.Errorf("Expected 2 members, got %d", len(values))
				}
				for i, v := range values {
					member := v.(map[string]interface{})
					// ID is always included as key property
					if _, hasID := member["ID"]; !hasID {
						t.Errorf("Member %d missing ID field (key property)", i)
					}
					if _, hasName := member["Name"]; !hasName {
						t.Errorf("Member %d missing Name field (selected)", i)
					}
				}
			},
		},
		{
			name:           "Select multiple fields with navigation filter",
			url:            "/Members?$select=ID,Name&$filter=" + url.QueryEscape("Club/Deleted eq true"),
			expectedStatus: http.StatusOK,
			expectedCount:  1, // Only Bob belongs to deleted club
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 1 {
					t.Errorf("Expected 1 member, got %d", len(values))
				}
				if len(values) > 0 {
					member := values[0].(map[string]interface{})
					if member["Name"] != "Bob" {
						t.Errorf("Expected Bob, got %v", member["Name"])
					}
				}
			},
		},
		{
			name:           "Select with navigation filter and expand",
			url:            "/Members?$select=ID,Name,Club&$filter=" + url.QueryEscape("Club/Deleted eq false") + "&$expand=Club",
			expectedStatus: http.StatusOK,
			expectedCount:  2, // Alice and Charlie
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 2 {
					t.Errorf("Expected 2 members, got %d", len(values))
				}
				// Note: The current implementation requires navigation properties to be
				// explicitly included in $select to be returned, even when $expand is used.
				// This test validates that the query doesn't fail with ambiguous column errors,
				// which was the main issue being fixed.
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.validate != nil && w.Code == http.StatusOK {
				tt.validate(t, w.Body.Bytes())
			}
		})
	}
}

// TestSelectWithoutNavigationFilter ensures that $select works correctly without navigation filters
// This is a baseline test to ensure we don't break existing functionality
func TestSelectWithoutNavigationFilter(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	type SimpleEntity struct {
		ID   string `json:"ID" gorm:"type:varchar(36);primaryKey;column:id" odata:"key"`
		Name string `json:"Name" gorm:"column:name"`
		Age  int    `json:"Age" gorm:"column:age"`
	}

	if err := db.AutoMigrate(&SimpleEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	db.Create(&SimpleEntity{ID: "1", Name: "Alice", Age: 30})
	db.Create(&SimpleEntity{ID: "2", Name: "Bob", Age: 25})

	service := odata.NewService(db)
	service.RegisterEntity(&SimpleEntity{})

	// Test that $select still works without JOINs
	req := httptest.NewRequest(http.MethodGet, "/SimpleEntities?$select=Name", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["value"] == nil {
		t.Fatalf("Response missing 'value' field")
	}

	values := response["value"].([]interface{})
	if len(values) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(values))
	}
}
