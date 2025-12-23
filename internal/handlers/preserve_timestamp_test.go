package handlers

import (
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPreserveTimestampFields(t *testing.T) {
	type testEntity struct {
		ID             int        `json:"id" gorm:"primarykey" odata:"key"`
		Name           string     `json:"name"`
		CreatedAt      time.Time  `json:"createdAt"`
		UpdatedAt      time.Time  `json:"updatedAt"`
		LastAccessedAt *time.Time `json:"lastAccessedAt"`
		DeletedAt      *time.Time `json:"deletedAt"`
		OptionalDate   *time.Time `json:"optionalDate"`
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(testEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	handler := NewEntityHandler(db, entityMeta, nil)

	// Create source entity with valid timestamps
	now := time.Now()
	lastAccessed := time.Now().Add(-1 * time.Hour)
	deletedTime := time.Now().Add(-2 * time.Hour)
	source := &testEntity{
		ID:             1,
		Name:           "Original",
		CreatedAt:      now,
		UpdatedAt:      now,
		LastAccessedAt: &lastAccessed,
		DeletedAt:      &deletedTime,
		OptionalDate:   nil,
	}

	t.Run("Preserve non-pointer time.Time field when destination is zero", func(t *testing.T) {
		dest := &testEntity{
			ID:        1,
			Name:      "Updated",
			CreatedAt: time.Time{}, // Zero value
			UpdatedAt: now,
		}

		handler.preserveTimestampFields(source, dest)

		if dest.CreatedAt.IsZero() {
			t.Error("CreatedAt should have been preserved from source")
		}
		if !dest.CreatedAt.Equal(source.CreatedAt) {
			t.Errorf("CreatedAt = %v, want %v", dest.CreatedAt, source.CreatedAt)
		}
	})

	t.Run("Do not overwrite non-zero time.Time field", func(t *testing.T) {
		dest := &testEntity{
			ID:        1,
			Name:      "Updated",
			CreatedAt: now.Add(1 * time.Hour), // Non-zero value
		}

		handler.preserveTimestampFields(source, dest)

		if dest.CreatedAt.Equal(source.CreatedAt) {
			t.Error("CreatedAt should not have been overwritten")
		}
	})

	t.Run("Preserve *time.Time field when destination is nil", func(t *testing.T) {
		dest := &testEntity{
			ID:             1,
			Name:           "Updated",
			LastAccessedAt: nil, // Nil pointer
		}

		handler.preserveTimestampFields(source, dest)

		if dest.LastAccessedAt == nil {
			t.Error("LastAccessedAt should have been preserved from source")
		}
		if dest.LastAccessedAt != nil && !dest.LastAccessedAt.Equal(*source.LastAccessedAt) {
			t.Errorf("LastAccessedAt = %v, want %v", *dest.LastAccessedAt, *source.LastAccessedAt)
		}
	})

	t.Run("Preserve *time.Time field when destination points to zero time", func(t *testing.T) {
		zeroTime := time.Time{}
		dest := &testEntity{
			ID:        1,
			Name:      "Updated",
			DeletedAt: &zeroTime, // Points to zero time
		}

		handler.preserveTimestampFields(source, dest)

		if dest.DeletedAt == nil {
			t.Error("DeletedAt should have been preserved from source")
		}
		if dest.DeletedAt != nil && !dest.DeletedAt.Equal(*source.DeletedAt) {
			t.Errorf("DeletedAt = %v, want %v", *dest.DeletedAt, *source.DeletedAt)
		}
	})

	t.Run("Do not overwrite non-nil, non-zero *time.Time field", func(t *testing.T) {
		differentTime := now.Add(3 * time.Hour)
		dest := &testEntity{
			ID:             1,
			Name:           "Updated",
			LastAccessedAt: &differentTime, // Non-nil, non-zero
		}

		handler.preserveTimestampFields(source, dest)

		if dest.LastAccessedAt == nil || !dest.LastAccessedAt.Equal(differentTime) {
			t.Error("LastAccessedAt should not have been overwritten")
		}
	})

	t.Run("Do not overwrite when source *time.Time is nil", func(t *testing.T) {
		dest := &testEntity{
			ID:           1,
			Name:         "Updated",
			OptionalDate: nil,
		}

		handler.preserveTimestampFields(source, dest)

		// Since source.OptionalDate is nil, destination should remain nil
		if dest.OptionalDate != nil {
			t.Error("OptionalDate should remain nil when source is nil")
		}
	})

	t.Run("Do not overwrite when source *time.Time points to zero time", func(t *testing.T) {
		zeroTime := time.Time{}
		sourceCopy := *source
		sourceCopy.OptionalDate = &zeroTime

		dest := &testEntity{
			ID:           1,
			Name:         "Updated",
			OptionalDate: nil,
		}

		handler.preserveTimestampFields(&sourceCopy, dest)

		// Since source.OptionalDate points to zero time, destination should remain nil
		if dest.OptionalDate != nil {
			t.Error("OptionalDate should remain nil when source points to zero time")
		}
	})
}
