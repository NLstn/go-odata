package hooks_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nlstn/go-odata/internal/hooks"
)

// MockEntity implements the EntityHooks interface for testing
type MockEntity struct {
	ID   int
	Name string
}

func (m MockEntity) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
	return nil
}

func (m MockEntity) ODataAfterCreate(ctx context.Context, r *http.Request) error {
	return nil
}

func (m MockEntity) ODataBeforeUpdate(ctx context.Context, r *http.Request) error {
	return nil
}

func (m MockEntity) ODataAfterUpdate(ctx context.Context, r *http.Request) error {
	return nil
}

func (m MockEntity) ODataBeforeDelete(ctx context.Context, r *http.Request) error {
	return nil
}

func (m MockEntity) ODataAfterDelete(ctx context.Context, r *http.Request) error {
	return nil
}

// TestEntityHooksInterface verifies that the EntityHooks interface can be implemented
func TestEntityHooksInterface(t *testing.T) {
	var _ hooks.EntityHooks = MockEntity{}

	entity := MockEntity{
		ID:   1,
		Name: "Test",
	}

	ctx := context.Background()
	req, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Test all hook methods
	if err := entity.ODataBeforeCreate(ctx, req); err != nil {
		t.Errorf("ODataBeforeCreate failed: %v", err)
	}

	if err := entity.ODataAfterCreate(ctx, req); err != nil {
		t.Errorf("ODataAfterCreate failed: %v", err)
	}

	if err := entity.ODataBeforeUpdate(ctx, req); err != nil {
		t.Errorf("ODataBeforeUpdate failed: %v", err)
	}

	if err := entity.ODataAfterUpdate(ctx, req); err != nil {
		t.Errorf("ODataAfterUpdate failed: %v", err)
	}

	if err := entity.ODataBeforeDelete(ctx, req); err != nil {
		t.Errorf("ODataBeforeDelete failed: %v", err)
	}

	if err := entity.ODataAfterDelete(ctx, req); err != nil {
		t.Errorf("ODataAfterDelete failed: %v", err)
	}
}

// TestPartialEntityHooksImplementation verifies that entities can implement only some hooks
func TestPartialEntityHooksImplementation(t *testing.T) {
	type PartialEntity struct {
		ID int
	}

	// Implement only some hooks
	var beforeCreate = func(p PartialEntity) func(context.Context, *http.Request) error {
		return func(ctx context.Context, r *http.Request) error {
			return nil
		}
	}

	entity := PartialEntity{ID: 1}
	fn := beforeCreate(entity)

	ctx := context.Background()
	req, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if err := fn(ctx, req); err != nil {
		t.Errorf("Hook function failed: %v", err)
	}
}
