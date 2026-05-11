package gormstore

import (
	"context"
	"errors"

	"github.com/nlstn/go-odata/internal/storage"
	"gorm.io/gorm"
)

type gormTx struct {
	tx *gorm.DB
}

func (t *gormTx) DB() *gorm.DB    { return t.tx }
func (t *gormTx) Commit() error   { return t.tx.Commit().Error }
func (t *gormTx) Rollback() error { return t.tx.Rollback().Error }

// Store is the default storage adapter backed by GORM.
type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB(ctx context.Context) *gorm.DB {
	if s.db == nil {
		return nil
	}
	return s.db.WithContext(ctx)
}

func (s *Store) Transaction(ctx context.Context, fn func(tx storage.Tx) error) error {
	return s.DB(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&gormTx{tx: tx})
	})
}

func (s *Store) Begin(ctx context.Context) (storage.Tx, error) {
	tx := s.DB(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &gormTx{tx: tx}, nil
}

func (s *Store) IsNotFound(err error) bool {
	return errors.Is(err, storage.ErrNotFound) || errors.Is(err, gorm.ErrRecordNotFound)
}

func (s *Store) MapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return storage.ErrNotFound
	}
	return err
}

func (s *Store) SupportsGORM() bool { return true }
