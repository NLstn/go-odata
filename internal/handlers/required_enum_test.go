package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// IssueStatus represents the status of an issue, starting at 0
type IssueStatus int64

const (
	IssueStatusNew        IssueStatus = 0
	IssueStatusInProgress IssueStatus = 1
	IssueStatusPending    IssueStatus = 2
	IssueStatusResolved   IssueStatus = 3
	IssueStatusClosed     IssueStatus = 4
)

// IssuePriority represents priority levels, starting at 0
type IssuePriority int64

const (
	IssuePriorityLow      IssuePriority = 0
	IssuePriorityMedium   IssuePriority = 1
	IssuePriorityHigh     IssuePriority = 2
	IssuePriorityCritical IssuePriority = 3
)

// Issue entity with required enum fields
type Issue struct {
	ID       uint          `json:"ID" gorm:"primaryKey" odata:"key"`
	Title    string        `json:"Title" odata:"required"`
	Status   IssueStatus   `json:"Status" odata:"required,enum=IssueStatus"`
	Priority IssuePriority `json:"Priority" odata:"required,enum=IssuePriority"`
}

func setupIssueTestHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	t.Helper()

	// Register enum types
	if err := metadata.RegisterEnumMembers(reflect.TypeOf(IssueStatus(0)), []metadata.EnumMember{
		{Name: "New", Value: 0},
		{Name: "InProgress", Value: 1},
		{Name: "Pending", Value: 2},
		{Name: "Resolved", Value: 3},
		{Name: "Closed", Value: 4},
	}); err != nil {
		t.Fatalf("Failed to register IssueStatus enum: %v", err)
	}

	if err := metadata.RegisterEnumMembers(reflect.TypeOf(IssuePriority(0)), []metadata.EnumMember{
		{Name: "Low", Value: 0},
		{Name: "Medium", Value: 1},
		{Name: "High", Value: 2},
		{Name: "Critical", Value: 3},
	}); err != nil {
		t.Fatalf("Failed to register IssuePriority enum: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&Issue{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(Issue{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)
	return handler, db
}

// TestHandlePostEntity_RequiredEnumZeroValue tests that enum fields with value 0 are accepted
// when they are required fields. This is a regression test for the issue where IsZero()
// incorrectly rejected enum value 0 as missing.
func TestHandlePostEntity_RequiredEnumZeroValue(t *testing.T) {
	handler, db := setupIssueTestHandler(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		wantStatusCode int
		wantError      bool
		description    string
	}{
		{
			name: "Status=0, Priority=1 - should succeed",
			requestBody: map[string]interface{}{
				"Title":    "Test Issue",
				"Status":   0, // IssueStatusNew - zero value but valid
				"Priority": 1, // IssuePriorityMedium
			},
			wantStatusCode: http.StatusCreated,
			wantError:      false,
			description:    "Enum with zero value should be accepted as valid",
		},
		{
			name: "Status=1, Priority=0 - should succeed",
			requestBody: map[string]interface{}{
				"Title":    "Test Issue 2",
				"Status":   1, // IssueStatusInProgress
				"Priority": 0, // IssuePriorityLow - zero value but valid
			},
			wantStatusCode: http.StatusCreated,
			wantError:      false,
			description:    "Another enum with zero value should be accepted",
		},
		{
			name: "Both Status and Priority as 0 - should succeed",
			requestBody: map[string]interface{}{
				"Title":    "Test Issue 3",
				"Status":   0, // IssueStatusNew - zero value
				"Priority": 0, // IssuePriorityLow - zero value
			},
			wantStatusCode: http.StatusCreated,
			wantError:      false,
			description:    "Multiple enum fields with zero values should be accepted",
		},
		{
			name: "Missing Status - should fail",
			requestBody: map[string]interface{}{
				"Title":    "Test Issue 4",
				"Priority": 1,
				// Status is missing
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
			description:    "Missing required enum field should be rejected",
		},
		{
			name: "Missing Priority - should fail",
			requestBody: map[string]interface{}{
				"Title":  "Test Issue 5",
				"Status": 1,
				// Priority is missing
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
			description:    "Missing required enum field should be rejected",
		},
		{
			name: "Missing Title - should fail",
			requestBody: map[string]interface{}{
				"Status":   0,
				"Priority": 0,
				// Title is missing
			},
			wantStatusCode: http.StatusBadRequest,
			wantError:      true,
			description:    "Missing required string field should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear the database before each test
			db.Exec("DELETE FROM issues")

			bodyBytes, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/Issues", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			handler.handlePostEntity(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("%s: Status = %v, want %v", tt.description, w.Code, tt.wantStatusCode)
				t.Logf("Response body: %s", w.Body.String())
			}

			if tt.wantError {
				var errorResponse map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
					t.Errorf("Failed to decode error response: %v", err)
				}
				if _, ok := errorResponse["error"]; !ok {
					t.Errorf("Expected error response, got: %v", errorResponse)
				}
			} else {
				// Verify the entity was created with correct values
				var created Issue
				if err := db.First(&created).Error; err != nil {
					t.Errorf("Failed to retrieve created entity: %v", err)
				} else {
					// Check that the values match what was sent
					if created.Title != tt.requestBody["Title"] {
						t.Errorf("Title = %v, want %v", created.Title, tt.requestBody["Title"])
					}
					if int64(created.Status) != int64(tt.requestBody["Status"].(int)) {
						t.Errorf("Status = %v, want %v", created.Status, tt.requestBody["Status"])
					}
					if int64(created.Priority) != int64(tt.requestBody["Priority"].(int)) {
						t.Errorf("Priority = %v, want %v", created.Priority, tt.requestBody["Priority"])
					}
				}
			}
		})
	}
}

// TestValidateRequiredProperties_EnumZeroValue tests the validateRequiredProperties function directly
func TestValidateRequiredProperties_EnumZeroValue(t *testing.T) {
	handler, _ := setupIssueTestHandler(t)

	tests := []struct {
		name        string
		entity      *Issue
		requestData map[string]interface{}
		wantErr     bool
		errMsg      string
	}{
		{
			name: "All required fields provided with Status=0",
			entity: &Issue{
				Title:    "Test",
				Status:   0, // Zero value enum
				Priority: 1,
			},
			requestData: map[string]interface{}{
				"Title":    "Test",
				"Status":   0,
				"Priority": 1,
			},
			wantErr: false,
		},
		{
			name: "All required fields provided with Priority=0",
			entity: &Issue{
				Title:    "Test",
				Status:   1,
				Priority: 0, // Zero value enum
			},
			requestData: map[string]interface{}{
				"Title":    "Test",
				"Status":   1,
				"Priority": 0,
			},
			wantErr: false,
		},
		{
			name: "All required fields provided with both enums=0",
			entity: &Issue{
				Title:    "Test",
				Status:   0,
				Priority: 0,
			},
			requestData: map[string]interface{}{
				"Title":    "Test",
				"Status":   0,
				"Priority": 0,
			},
			wantErr: false,
		},
		{
			name: "Missing Title",
			entity: &Issue{
				Status:   0,
				Priority: 0,
			},
			requestData: map[string]interface{}{
				"Status":   0,
				"Priority": 0,
			},
			wantErr: true,
			errMsg:  "Title",
		},
		{
			name: "Missing Status",
			entity: &Issue{
				Title:    "Test",
				Priority: 0,
			},
			requestData: map[string]interface{}{
				"Title":    "Test",
				"Priority": 0,
			},
			wantErr: true,
			errMsg:  "Status",
		},
		{
			name: "Missing Priority",
			entity: &Issue{
				Title:  "Test",
				Status: 0,
			},
			requestData: map[string]interface{}{
				"Title":  "Test",
				"Status": 0,
			},
			wantErr: true,
			errMsg:  "Priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateRequiredProperties(tt.requestData)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequiredProperties() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Error message = %v, should contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
