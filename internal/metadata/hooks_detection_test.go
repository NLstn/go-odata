package metadata_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/gorm"
)

type testReadHooksEntity struct {
	ID int `json:"id" odata:"key"`
}

func (testReadHooksEntity) BeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	return nil, nil
}

func (testReadHooksEntity) AfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error) {
	return results, nil
}

func (*testReadHooksEntity) BeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	return nil, nil
}

func (*testReadHooksEntity) AfterReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, error) {
	return entity, nil
}

func TestDetectReadHooks(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(testReadHooksEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity(testReadHooksEntity) returned error: %v", err)
	}

	if !meta.Hooks.HasBeforeReadCollection {
		t.Errorf("expected HasBeforeReadCollection to be true")
	}
	if !meta.Hooks.HasAfterReadCollection {
		t.Errorf("expected HasAfterReadCollection to be true")
	}
	if !meta.Hooks.HasBeforeReadEntity {
		t.Errorf("expected HasBeforeReadEntity to be true")
	}
	if !meta.Hooks.HasAfterReadEntity {
		t.Errorf("expected HasAfterReadEntity to be true")
	}
}
