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

// CustomClub - test entity with custom TableName
type CustomClub struct {
	ID   string `json:"ID" gorm:"primaryKey;type:varchar(36)" odata:"key"`
	Name string `json:"Name" gorm:"not null" odata:"required"`
}

func (CustomClub) TableName() string {
	return "Clubs"
}

func (CustomClub) EntitySetName() string {
	return "CustomClubs"
}

// CustomTeam - test entity with custom TableName
type CustomTeam struct {
	ID     string      `json:"ID" gorm:"primaryKey;type:varchar(36)" odata:"key"`
	Name   string      `json:"Name" gorm:"not null" odata:"required"`
	ClubID string      `json:"ClubID" gorm:"type:varchar(36);not null" odata:"required"`
	Club   *CustomClub `json:"Club" gorm:"foreignKey:ClubID;references:ID" odata:"nav"`
}

func (CustomTeam) TableName() string {
	return "Teams"
}

func (CustomTeam) EntitySetName() string {
	return "CustomTeams"
}

// CustomTeamMember - test entity with custom TableName
type CustomTeamMember struct {
	ID     string      `json:"ID" gorm:"primaryKey;type:varchar(36)" odata:"key"`
	UserID string      `json:"UserID" gorm:"type:varchar(36);not null" odata:"required"`
	Role   string      `json:"Role" gorm:"not null" odata:"required"`
	TeamID string      `json:"TeamID" gorm:"type:varchar(36);not null" odata:"required"`
	Team   *CustomTeam `json:"Team" gorm:"foreignKey:TeamID;references:ID" odata:"nav"`
}

func (CustomTeamMember) TableName() string {
	return "TeamMembers"
}

func (CustomTeamMember) EntitySetName() string {
	return "CustomTeamMembers"
}

// TestCustomTableNameWithNavigationFilter tests that custom TableName() methods are respected in JOIN clauses
// This reproduces the issue where navigation property filters generate incorrect table names
func TestCustomTableNameWithNavigationFilter(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate - this will create tables with custom names
	if err := db.AutoMigrate(&CustomClub{}, &CustomTeam{}, &CustomTeamMember{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	club1 := CustomClub{ID: "club-1", Name: "Club Alpha"}
	club2 := CustomClub{ID: "club-2", Name: "Club Beta"}
	db.Create(&club1)
	db.Create(&club2)

	team1 := CustomTeam{ID: "team-1", Name: "Team A", ClubID: "club-1"}
	team2 := CustomTeam{ID: "team-2", Name: "Team B", ClubID: "club-2"}
	db.Create(&team1)
	db.Create(&team2)

	member1 := CustomTeamMember{ID: "member-1", UserID: "user-1", Role: "admin", TeamID: "team-1"}
	member2 := CustomTeamMember{ID: "member-2", UserID: "user-2", Role: "member", TeamID: "team-2"}
	member3 := CustomTeamMember{ID: "member-3", UserID: "user-1", Role: "member", TeamID: "team-2"}
	db.Create(&member1)
	db.Create(&member2)
	db.Create(&member3)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&CustomClub{})
	service.RegisterEntity(&CustomTeam{})
	service.RegisterEntity(&CustomTeamMember{})

	// Test: Filter by navigation property with custom table names
	// This ensures proper table name resolution in JOIN and ON clauses
	testURL := "/CustomTeamMembers?$filter=" + url.QueryEscape("UserID eq 'user-1' and Team/ClubID eq 'club-1'")
	req := httptest.NewRequest(http.MethodGet, testURL, nil)
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

	values := response["value"].([]interface{})
	if len(values) != 1 {
		t.Errorf("Expected 1 team member (member-1 with user-1 in team-1 which is in club-1), got %d", len(values))
		return
	}

	member := values[0].(map[string]interface{})
	if member["ID"] != "member-1" {
		t.Errorf("Expected member-1, got %v", member["ID"])
	}
}

// TestCustomTableNameWithExpand tests that custom TableName() methods work with $expand
func TestCustomTableNameWithExpand(t *testing.T) {
	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrate
	if err := db.AutoMigrate(&CustomClub{}, &CustomTeam{}, &CustomTeamMember{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create test data
	club1 := CustomClub{ID: "club-1", Name: "Club Alpha"}
	db.Create(&club1)

	team1 := CustomTeam{ID: "team-1", Name: "Team A", ClubID: "club-1"}
	db.Create(&team1)

	member1 := CustomTeamMember{ID: "member-1", UserID: "user-1", Role: "admin", TeamID: "team-1"}
	db.Create(&member1)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	service.RegisterEntity(&CustomClub{})
	service.RegisterEntity(&CustomTeam{})
	service.RegisterEntity(&CustomTeamMember{})

	// Test: Filter with both navigation filter and expand
	testURL := "/CustomTeamMembers?$filter=" + url.QueryEscape("Team/ClubID eq 'club-1'") + "&$expand=Team"
	req := httptest.NewRequest(http.MethodGet, testURL, nil)
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

	values := response["value"].([]interface{})
	if len(values) != 1 {
		t.Errorf("Expected 1 team member, got %d", len(values))
		return
	}

	member := values[0].(map[string]interface{})
	if member["ID"] != "member-1" {
		t.Errorf("Expected member-1, got %v", member["ID"])
	}

	// Check that Team is expanded
	team, ok := member["Team"].(map[string]interface{})
	if !ok || team == nil {
		t.Error("Expected Team to be expanded")
		return
	}

	if team["Name"] != "Team A" {
		t.Errorf("Expected Team A, got %v", team["Name"])
	}
}
