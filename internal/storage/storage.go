package storage

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var (
	// ErrNotFound indicates a requested record does not exist.
	ErrNotFound = errors.New("storage: not found")
)

// Query describes parsed OData query intent passed to a storage backend.
// This is intentionally minimal in the first decoupling phase.
type Query struct {
	Options interface{}
}

// Tx represents an active storage transaction.
type Tx interface {
	DB() *gorm.DB
	Commit() error
	Rollback() error
}

// Store provides the storage contract consumed by runtime handlers.
type Store interface {
	DB(ctx context.Context) *gorm.DB
	Transaction(ctx context.Context, fn func(tx Tx) error) error
	Begin(ctx context.Context) (Tx, error)
	IsNotFound(err error) bool
	MapError(err error) error
	SupportsGORM() bool
}
