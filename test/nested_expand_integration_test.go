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

// NestedExpandUser represents a user entity for testing nested expand
type NestedExpandUser struct {
	ID      string               `json:"ID" gorm:"primaryKey" odata:"key"`
	Name    string               `json:"Name"`
	Members []NestedExpandMember `gorm:"foreignKey:UserID" json:"Members,omitempty" odata:"nav"`
}

// NestedExpandMember represents a member entity
type NestedExpandMember struct {
	ID     string            `json:"ID" gorm:"primaryKey" odata:"key"`
	UserID string            `json:"UserID"`
	ClubID string            `json:"ClubID"`
	Role   string            `json:"Role"`
	User   *NestedExpandUser `gorm:"foreignKey:UserID" json:"User,omitempty" odata:"nav"`
	Club   *NestedExpandClub `gorm:"foreignKey:ClubID" json:"Club,omitempty" odata:"nav"`
}

// NestedExpandClub represents a club entity
type NestedExpandClub struct {
	ID      string               `json:"ID" gorm:"primaryKey" odata:"key"`
	Name    string               `json:"Name"`
	Members []NestedExpandMember `gorm:"foreignKey:ClubID" json:"Members,omitempty" odata:"nav"`
}

func setupNestedExpandTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the models
	if err := db.AutoMigrate(&NestedExpandUser{}, &NestedExpandMember{}, &NestedExpandClub{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	// Seed test data
	users := []NestedExpandUser{
		{ID: "user-123", Name: "John Doe"},
		{ID: "user-456", Name: "Jane Smith"},
	}

	clubs := []NestedExpandClub{
		{ID: "club-789", Name: "Chess Club"},
		{ID: "club-101", Name: "Book Club"},
	}

	members := []NestedExpandMember{
		{ID: "member-1", UserID: "user-123", ClubID: "club-789", Role: "owner"},
		{ID: "member-2", UserID: "user-123", ClubID: "club-101", Role: "member"},
		{ID: "member-3", UserID: "user-456", ClubID: "club-789", Role: "member"},
	}

	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("Failed to seed users: %v", err)
	}

	if err := db.Create(&clubs).Error; err != nil {
		t.Fatalf("Failed to seed clubs: %v", err)
	}

	if err := db.Create(&members).Error; err != nil {
		t.Fatalf("Failed to seed members: %v", err)
	}

	return db
}

func TestNestedExpandIntegration(t *testing.T) {
	db := setupNestedExpandTestDB(t)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&NestedExpandUser{}); err != nil {
		t.Fatalf("Failed to register NestedExpandUser entity: %v", err)
	}
	if err := service.RegisterEntity(&NestedExpandMember{}); err != nil {
		t.Fatalf("Failed to register NestedExpandMember entity: %v", err)
	}
	if err := service.RegisterEntity(&NestedExpandClub{}); err != nil {
		t.Fatalf("Failed to register NestedExpandClub entity: %v", err)
	}

	t.Run("Single level expand works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/NestedExpandUsers(user-123)?$expand=Members", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify user data
		if response["ID"] != "user-123" {
			t.Errorf("Expected ID 'user-123', got %v", response["ID"])
		}

		// Verify members are expanded
		members, ok := response["Members"].([]interface{})
		if !ok {
			t.Fatal("Members should be expanded")
		}

		if len(members) != 2 {
			t.Errorf("Expected 2 members, got %d", len(members))
		}

		// Verify Club is NOT expanded (since we didn't request it)
		if len(members) > 0 {
			member := members[0].(map[string]interface{})
			if _, hasClub := member["Club"]; hasClub {
				t.Error("Club should not be expanded without nested $expand")
			}
		}
	})

	t.Run("Nested expand loads navigation properties at second level", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/NestedExpandUsers(user-123)?$expand=Members($expand=Club)", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify user data
		if response["ID"] != "user-123" {
			t.Errorf("Expected ID 'user-123', got %v", response["ID"])
		}

		// Verify members are expanded
		members, ok := response["Members"].([]interface{})
		if !ok {
			t.Fatal("Members should be expanded")
		}

		if len(members) != 2 {
			t.Errorf("Expected 2 members, got %d", len(members))
		}

		// Verify Club IS expanded for each member
		foundChessClub := false
		foundBookClub := false
		for _, m := range members {
			member := m.(map[string]interface{})

			club, hasClub := member["Club"]
			if !hasClub {
				t.Error("Club should be expanded with nested $expand")
				continue
			}

			clubMap, ok := club.(map[string]interface{})
			if !ok {
				t.Error("Club should be an object")
				continue
			}

			clubName, ok := clubMap["Name"].(string)
			if !ok {
				t.Error("Club should have a Name field")
				continue
			}

			switch clubName {
			case "Chess Club":
				foundChessClub = true
				if clubMap["ID"] != "club-789" {
					t.Errorf("Expected Chess Club ID 'club-789', got %v", clubMap["ID"])
				}
			case "Book Club":
				foundBookClub = true
				if clubMap["ID"] != "club-101" {
					t.Errorf("Expected Book Club ID 'club-101', got %v", clubMap["ID"])
				}
			}
		}

		if !foundChessClub || !foundBookClub {
			t.Error("Expected to find both Chess Club and Book Club in expanded members")
		}
	})

	t.Run("Nested expand with $select on nested level", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/NestedExpandUsers(user-123)?$expand=Members($expand=Club($select=Name))", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		members, ok := response["Members"].([]interface{})
		if !ok || len(members) == 0 {
			t.Fatal("Members should be expanded")
		}

		// Verify Club is expanded with only Name field (due to $select)
		member := members[0].(map[string]interface{})
		club, hasClub := member["Club"]
		if !hasClub {
			t.Error("Club should be expanded")
		} else {
			clubMap := club.(map[string]interface{})
			if _, hasName := clubMap["Name"]; !hasName {
				t.Error("Club should have Name field")
			}
			// Note: $select behavior might include ID field as it's a key property
		}
	})

	t.Run("Nested expand with $filter on nested level", func(t *testing.T) {
		// Create a user with multiple members in different clubs
		// Note: URL needs to be properly encoded
		req := httptest.NewRequest("GET", "/NestedExpandUsers(user-123)?$expand=Members($expand=Club($filter=Name%20eq%20%27Chess%20Club%27))", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Note: Filter on nested expand might not work as expected in GORM
		// This test documents the current behavior
		members, ok := response["Members"].([]interface{})
		if !ok || len(members) == 0 {
			t.Fatal("Members should be expanded")
		}
	})
}

func TestNestedExpandMultipleLevels(t *testing.T) {
	db := setupNestedExpandTestDB(t)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(&NestedExpandUser{}); err != nil {
		t.Fatalf("Failed to register NestedExpandUser entity: %v", err)
	}
	if err := service.RegisterEntity(&NestedExpandMember{}); err != nil {
		t.Fatalf("Failed to register NestedExpandMember entity: %v", err)
	}
	if err := service.RegisterEntity(&NestedExpandClub{}); err != nil {
		t.Fatalf("Failed to register NestedExpandClub entity: %v", err)
	}

	t.Run("Collection endpoint with nested expand", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/NestedExpandUsers?$expand=Members($expand=Club)", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		value, ok := response["value"].([]interface{})
		if !ok {
			t.Fatal("Response should have value array")
		}

		if len(value) != 2 {
			t.Errorf("Expected 2 users, got %d", len(value))
		}

		// Check first user
		user := value[0].(map[string]interface{})
		members, ok := user["Members"].([]interface{})
		if !ok {
			t.Fatal("Members should be expanded")
		}

		// Verify nested Club expansion
		if len(members) > 0 {
			member := members[0].(map[string]interface{})
			if _, hasClub := member["Club"]; !hasClub {
				t.Error("Club should be expanded with nested $expand")
			}
		}
	})

	t.Run("Direct member query with nested expand", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/NestedExpandMembers(member-1)?$expand=Club", nil)
		w := httptest.NewRecorder()

		service.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify Club is expanded
		club, hasClub := response["Club"]
		if !hasClub {
			t.Fatal("Club should be expanded")
		}

		clubMap := club.(map[string]interface{})
		if clubMap["ID"] != "club-789" {
			t.Errorf("Expected Club ID 'club-789', got %v", clubMap["ID"])
		}
		if clubMap["Name"] != "Chess Club" {
			t.Errorf("Expected Club Name 'Chess Club', got %v", clubMap["Name"])
		}
	})
}
