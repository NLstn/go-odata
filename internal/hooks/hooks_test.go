package hooks_test

import (
	"context"
	"fmt"
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

// PartialEntity demonstrates that entities don't need to implement the full EntityHooks interface.
// In practice, the OData framework uses reflection to check for individual hook methods,
// so you can implement only the hooks you need.
type PartialEntity struct {
	ID   int
	Name string
}

// Only implement BeforeCreate hook - other hooks are optional
func (p PartialEntity) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
	// Custom validation logic
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// TestPartialHookImplementation verifies that entities can implement only some hook methods
func TestPartialHookImplementation(t *testing.T) {
	entity := PartialEntity{
		ID:   1,
		Name: "Test",
	}

	ctx := context.Background()
	req, err := http.NewRequest("POST", "/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Test the implemented hook
	if err := entity.ODataBeforeCreate(ctx, req); err != nil {
		t.Errorf("ODataBeforeCreate failed: %v", err)
	}

	// Verify validation works
	emptyEntity := PartialEntity{ID: 2, Name: ""}
	if err := emptyEntity.ODataBeforeCreate(ctx, req); err == nil {
		t.Error("Expected validation error for empty name, got nil")
	}

	// Note: PartialEntity doesn't implement other hooks like ODataAfterCreate,
	// ODataBeforeUpdate, etc. This is valid - the framework will only call hooks
	// that are actually defined on the entity type.
}
