package gormstore

import (
	"context"
	"errors"
	"testing"

	"github.com/nlstn/go-odata/internal/storage"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type txTestEntity struct {
	ID   int `gorm:"primaryKey"`
	Name string
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&txTestEntity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return New(db)
}

func TestStoreTransactionCommitAndRollback(t *testing.T) {
	ctx := context.Background()

	t.Run("commit", func(t *testing.T) {
		s := newTestStore(t)
		if err := s.Transaction(ctx, func(tx storage.Tx) error {
			return tx.DB().Create(&txTestEntity{ID: 1, Name: "ok"}).Error
		}); err != nil {
			t.Fatalf("transaction: %v", err)
		}
		var count int64
		if err := s.DB(ctx).Model(&txTestEntity{}).Where("id = ?", 1).Count(&count).Error; err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 row, got %d", count)
		}
	})

	t.Run("rollback", func(t *testing.T) {
		s := newTestStore(t)
		_ = s.Transaction(ctx, func(tx storage.Tx) error {
			if err := tx.DB().Create(&txTestEntity{ID: 2, Name: "rollback"}).Error; err != nil {
				return err
			}
			return errors.New("force rollback")
		})
		var count int64
		if err := s.DB(ctx).Model(&txTestEntity{}).Where("id = ?", 2).Count(&count).Error; err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 rows after rollback, got %d", count)
		}
	})
}

func TestStoreMapErrorNotFound(t *testing.T) {
	s := New(nil)
	mapped := s.MapError(gorm.ErrRecordNotFound)
	if !errors.Is(mapped, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound mapping, got %v", mapped)
	}
}
