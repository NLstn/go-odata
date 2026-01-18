package odata_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Club entity with auto fields for testing
type Club struct {
	ID        string    `json:"id" gorm:"primarykey" odata:"key"`
	Name      string    `json:"name" odata:"required"`
	CreatedAt time.Time `json:"created_at" odata:"auto"`
	CreatedBy string    `json:"created_by" odata:"auto"`
	UpdatedAt time.Time `json:"updated_at" odata:"auto"`
	UpdatedBy string    `json:"updated_by" odata:"auto"`
}

// ODataBeforeCreate hook sets auto fields from context
func (c *Club) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
	userID := "test-user"
	now := time.Now()
	c.CreatedAt = now
	c.CreatedBy = userID
	c.UpdatedAt = now
	c.UpdatedBy = userID
	return nil
}

// ODataBeforeUpdate hook sets auto fields from context
func (c *Club) ODataBeforeUpdate(ctx context.Context, r *http.Request) error {
	userID := "test-user"
	now := time.Now()
	c.UpdatedAt = now
	c.UpdatedBy = userID
	return nil
}

func setupAutoFieldsTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Club{}); err != nil {
		t.Fatalf("Failed to auto-migrate: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }

	if err := service.RegisterEntity(&Club{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestAutoFields_POST_RejectClientProvidedValues(t *testing.T) {
	service, _ := setupAutoFieldsTestService(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		errorContains  string
	}{
		{
			name: "Reject created_at field",
			requestBody: map[string]interface{}{
				"id":         "club1",
				"name":       "Test Club",
				"created_at": "2024-01-01T00:00:00Z",
			},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "created_at",
		},
		{
			name: "Reject created_by field",
			requestBody: map[string]interface{}{
				"id":         "club2",
				"name":       "Test Club",
				"created_by": "hacker",
			},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "created_by",
		},
		{
			name: "Reject multiple auto fields",
			requestBody: map[string]interface{}{
				"id":         "club3",
				"name":       "Test Club",
				"created_at": "2024-01-01T00:00:00Z",
				"updated_by": "hacker",
			},
			expectedStatus: http.StatusBadRequest,
			errorContains:  "auto",
		},
		{
			name: "Accept request without auto fields",
			requestBody: map[string]interface{}{
				"id":   "club4",
				"name": "Valid Club",
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/Clubs", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.errorContains != "" && !strings.Contains(w.Body.String(), tt.errorContains) {
				t.Errorf("Expected error to contain %q, got: %s", tt.errorContains, w.Body.String())
			}
		})
	}
}

func TestAutoFields_POST_HooksSetValues(t *testing.T) {
	service, db := setupAutoFieldsTestService(t)

	requestBody := map[string]interface{}{
		"id":   "club1",
		"name": "Test Club",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/Clubs?$select=id,name,created_at,created_by,updated_at,updated_by", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	// Verify the entity was created with auto fields set
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["created_by"] == nil || response["created_by"] == "" {
		t.Error("Expected created_by to be set by hook")
	}

	if response["created_at"] == nil || response["created_at"] == "" {
		t.Error("Expected created_at to be set by hook")
	}

	if response["updated_by"] == nil || response["updated_by"] == "" {
		t.Error("Expected updated_by to be set by hook")
	}

	if response["updated_at"] == nil || response["updated_at"] == "" {
		t.Error("Expected updated_at to be set by hook")
	}

	// Verify in database
	var dbClub Club
	db.First(&dbClub, "id = ?", "club1")
	if dbClub.CreatedBy != "test-user" {
		t.Errorf("Expected created_by to be 'test-user', got %q", dbClub.CreatedBy)
	}
	if dbClub.CreatedAt.IsZero() {
		t.Error("Expected created_at to be set")
	}
}
