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

func (testReadHooksEntity) ODataBeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	return nil, nil
}

func (testReadHooksEntity) ODataAfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error) {
	return results, nil
}

func (*testReadHooksEntity) ODataBeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	return nil, nil
}

func (*testReadHooksEntity) ODataAfterReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, error) {
	return entity, nil
}

func TestDetectReadHooks(t *testing.T) {
	meta, err := metadata.AnalyzeEntity(testReadHooksEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity(testReadHooksEntity) returned error: %v", err)
	}

	if !meta.Hooks.HasODataBeforeReadCollection {
		t.Errorf("expected HasODataBeforeReadCollection to be true")
	}
	if !meta.Hooks.HasODataAfterReadCollection {
		t.Errorf("expected HasODataAfterReadCollection to be true")
	}
	if !meta.Hooks.HasODataBeforeReadEntity {
		t.Errorf("expected HasODataBeforeReadEntity to be true")
	}
	if !meta.Hooks.HasODataAfterReadEntity {
		t.Errorf("expected HasODataAfterReadEntity to be true")
	}
}
