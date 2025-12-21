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

// TestSingleEntityNavigationPropertyFilter tests filtering on single-entity navigation properties
// Per OData v4.01 spec 5.1.1.15, properties of entities related with cardinality 0..1 or 1 can be accessed directly
func TestSingleEntityNavigationPropertyFilter(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Define test entities
	type Club struct {
		ID   string `json:"ID" gorm:"primaryKey;type:varchar(36)" odata:"key"`
		Name string `json:"Name" gorm:"not null" odata:"required"`
	}

	type Team struct {
		ID     string `json:"ID" gorm:"primaryKey;type:varchar(36)" odata:"key"`
		Name   string `json:"Name" gorm:"not null" odata:"required"`
		ClubID string `json:"ClubID" gorm:"type:varchar(36);not null" odata:"required"`
		Club   *Club  `json:"Club" gorm:"foreignKey:ClubID;references:ID"`
	}

	type TeamMember struct {
		ID     string `json:"ID" gorm:"primaryKey;type:varchar(36)" odata:"key"`
		Name   string `json:"Name" gorm:"not null" odata:"required"`
		TeamID string `json:"TeamID" gorm:"type:varchar(36);not null" odata:"required"`
		Team   *Team  `json:"Team" gorm:"foreignKey:TeamID;references:ID"` // Single-entity navigation property
	}

	// Migrate and seed
	if err := db.AutoMigrate(&Club{}, &Team{}, &TeamMember{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	club1 := Club{ID: "club-1", Name: "Club Alpha"}
	club2 := Club{ID: "club-2", Name: "Club Beta"}
	db.Create(&club1)
	db.Create(&club2)

	team1 := Team{ID: "team-1", Name: "Team A", ClubID: "club-1"}
	team2 := Team{ID: "team-2", Name: "Team B", ClubID: "club-2"}
	team3 := Team{ID: "team-3", Name: "Team C", ClubID: "club-1"}
	db.Create(&team1)
	db.Create(&team2)
	db.Create(&team3)

	member1 := TeamMember{ID: "member-1", Name: "Alice", TeamID: "team-1"}
	member2 := TeamMember{ID: "member-2", Name: "Bob", TeamID: "team-2"}
	member3 := TeamMember{ID: "member-3", Name: "Charlie", TeamID: "team-3"}
	db.Create(&member1)
	db.Create(&member2)
	db.Create(&member3)

	// Create OData service
	service := odata.NewService(db)
	service.RegisterEntity(&Club{})
	service.RegisterEntity(&Team{})
	service.RegisterEntity(&TeamMember{})

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
			name:           "Filter by single-entity navigation property - Team/ClubID",
			url:            buildFilterURL("/TeamMembers", "Team/ClubID eq 'club-1'"),
			expectedStatus: http.StatusOK,
			expectedCount:  2, // Alice and Charlie are in teams that belong to club-1
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 2 {
					t.Errorf("Expected 2 team members, got %d", len(values))
				}
			},
		},
		{
			name:           "Filter by single-entity navigation property - nested path",
			url:            buildFilterURL("/TeamMembers", "Team/Name eq 'Team A'"),
			expectedStatus: http.StatusOK,
			expectedCount:  1, // Only Alice is in Team A
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 1 {
					t.Errorf("Expected 1 team member, got %d", len(values))
				}
				if len(values) > 0 {
					member := values[0].(map[string]interface{})
					if member["Name"] != "Alice" {
						t.Errorf("Expected Alice, got %v", member["Name"])
					}
				}
			},
		},
		{
			name:           "Filter with expand - both filter and expand navigation property",
			url:            "/TeamMembers?$filter=" + url.QueryEscape("Team/ClubID eq 'club-2'") + "&$expand=Team",
			expectedStatus: http.StatusOK,
			expectedCount:  1, // Only Bob is in a team that belongs to club-2
			validate: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				values := response["value"].([]interface{})
				if len(values) != 1 {
					t.Errorf("Expected 1 team member, got %d", len(values))
				}
				if len(values) > 0 {
					member := values[0].(map[string]interface{})
					if member["Name"] != "Bob" {
						t.Errorf("Expected Bob, got %v", member["Name"])
					}
					// Check that Team is expanded
					if team, ok := member["Team"]; !ok || team == nil {
						t.Error("Expected Team to be expanded")
					}
				}
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

			if tt.validate != nil {
				tt.validate(t, w.Body.Bytes())
			}
		})
	}
}

// TestCollectionNavigationPropertyStillRequiresLambda ensures collection navigation properties still require any/all
func TestCollectionNavigationPropertyStillRequiresLambda(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Define test entities
	type ProductDescription struct {
		ProductID   uint   `json:"ProductID" gorm:"primaryKey" odata:"key"`
		LanguageKey string `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key,maxlength=2"`
		Description string `json:"Description" gorm:"not null" odata:"required"`
	}

	type Product struct {
		ID           uint                 `json:"ID" gorm:"primaryKey" odata:"key"`
		Name         string               `json:"Name" gorm:"not null" odata:"required"`
		Descriptions []ProductDescription `json:"Descriptions" gorm:"foreignKey:ProductID;references:ID"` // Collection navigation property
	}

	// Migrate
	if err := db.AutoMigrate(&Product{}, &ProductDescription{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create OData service
	service := odata.NewService(db)
	service.RegisterEntity(&Product{})
	service.RegisterEntity(&ProductDescription{})

	// Test that collection navigation properties still require lambda operators
	testURL := "/Products?$filter=" + url.QueryEscape("Descriptions/LanguageKey eq 'EN'")
	req := httptest.NewRequest(http.MethodGet, testURL, nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// This should return an error because Descriptions is a collection
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for collection navigation property without lambda, got %d. Body: %s",
			http.StatusBadRequest, w.Code, w.Body.String())
	}
}
