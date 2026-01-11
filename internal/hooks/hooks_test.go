package hooks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockEntity is a test entity that implements EntityHooks interface methods.
type mockEntity struct {
	ID                   int
	Name                 string
	beforeCreateCalled   bool
	afterCreateCalled    bool
	beforeUpdateCalled   bool
	afterUpdateCalled    bool
	beforeDeleteCalled   bool
	afterDeleteCalled    bool
	shouldReturnError    bool
	shouldReturnErrorMsg string
}

// ODataBeforeCreate tests hook invocation.
func (m *mockEntity) ODataBeforeCreate(_ context.Context, _ *http.Request) error {
	m.beforeCreateCalled = true
	if m.shouldReturnError {
		return &mockHookError{msg: m.shouldReturnErrorMsg}
	}
	return nil
}

// ODataAfterCreate tests hook invocation.
func (m *mockEntity) ODataAfterCreate(_ context.Context, _ *http.Request) error {
	m.afterCreateCalled = true
	if m.shouldReturnError {
		return &mockHookError{msg: m.shouldReturnErrorMsg}
	}
	return nil
}

// ODataBeforeUpdate tests hook invocation.
func (m *mockEntity) ODataBeforeUpdate(_ context.Context, _ *http.Request) error {
	m.beforeUpdateCalled = true
	if m.shouldReturnError {
		return &mockHookError{msg: m.shouldReturnErrorMsg}
	}
	return nil
}

// ODataAfterUpdate tests hook invocation.
func (m *mockEntity) ODataAfterUpdate(_ context.Context, _ *http.Request) error {
	m.afterUpdateCalled = true
	if m.shouldReturnError {
		return &mockHookError{msg: m.shouldReturnErrorMsg}
	}
	return nil
}

// ODataBeforeDelete tests hook invocation.
func (m *mockEntity) ODataBeforeDelete(_ context.Context, _ *http.Request) error {
	m.beforeDeleteCalled = true
	if m.shouldReturnError {
		return &mockHookError{msg: m.shouldReturnErrorMsg}
	}
	return nil
}

// ODataAfterDelete tests hook invocation.
func (m *mockEntity) ODataAfterDelete(_ context.Context, _ *http.Request) error {
	m.afterDeleteCalled = true
	if m.shouldReturnError {
		return &mockHookError{msg: m.shouldReturnErrorMsg}
	}
	return nil
}

// mockHookError is a simple error type for testing.
type mockHookError struct {
	msg string
}

func (e *mockHookError) Error() string {
	return e.msg
}

// entityWithoutHooks is a test entity that doesn't implement hooks.
type entityWithoutHooks struct {
	ID   int
	Name string
}

func TestEntityHooksInterface(t *testing.T) {
	// Test that mockEntity implements EntityHooks interface
	var _ EntityHooks = &mockEntity{}

	t.Run("BeforeCreate hook called", func(t *testing.T) {
		entity := &mockEntity{}
		req := httptest.NewRequest(http.MethodPost, "/entities", nil)
		ctx := context.Background()

		err := entity.ODataBeforeCreate(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !entity.beforeCreateCalled {
			t.Error("Expected beforeCreateCalled to be true")
		}
	})

	t.Run("BeforeCreate hook returns error", func(t *testing.T) {
		entity := &mockEntity{
			shouldReturnError:    true,
			shouldReturnErrorMsg: "creation not allowed",
		}
		req := httptest.NewRequest(http.MethodPost, "/entities", nil)
		ctx := context.Background()

		err := entity.ODataBeforeCreate(ctx, req)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if err.Error() != "creation not allowed" {
			t.Errorf("Expected error message 'creation not allowed', got %s", err.Error())
		}
	})

	t.Run("AfterCreate hook called", func(t *testing.T) {
		entity := &mockEntity{}
		req := httptest.NewRequest(http.MethodPost, "/entities", nil)
		ctx := context.Background()

		err := entity.ODataAfterCreate(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !entity.afterCreateCalled {
			t.Error("Expected afterCreateCalled to be true")
		}
	})

	t.Run("BeforeUpdate hook called", func(t *testing.T) {
		entity := &mockEntity{}
		req := httptest.NewRequest(http.MethodPatch, "/entities(1)", nil)
		ctx := context.Background()

		err := entity.ODataBeforeUpdate(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !entity.beforeUpdateCalled {
			t.Error("Expected beforeUpdateCalled to be true")
		}
	})

	t.Run("AfterUpdate hook called", func(t *testing.T) {
		entity := &mockEntity{}
		req := httptest.NewRequest(http.MethodPatch, "/entities(1)", nil)
		ctx := context.Background()

		err := entity.ODataAfterUpdate(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !entity.afterUpdateCalled {
			t.Error("Expected afterUpdateCalled to be true")
		}
	})

	t.Run("BeforeDelete hook called", func(t *testing.T) {
		entity := &mockEntity{}
		req := httptest.NewRequest(http.MethodDelete, "/entities(1)", nil)
		ctx := context.Background()

		err := entity.ODataBeforeDelete(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !entity.beforeDeleteCalled {
			t.Error("Expected beforeDeleteCalled to be true")
		}
	})

	t.Run("AfterDelete hook called", func(t *testing.T) {
		entity := &mockEntity{}
		req := httptest.NewRequest(http.MethodDelete, "/entities(1)", nil)
		ctx := context.Background()

		err := entity.ODataAfterDelete(ctx, req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !entity.afterDeleteCalled {
			t.Error("Expected afterDeleteCalled to be true")
		}
	})
}

func TestEntityHooksLifecycle(t *testing.T) {
	t.Run("Full lifecycle hooks can be called in sequence", func(t *testing.T) {
		entity := &mockEntity{}
		req := httptest.NewRequest(http.MethodPost, "/entities", nil)
		ctx := context.Background()

		// Simulate create lifecycle
		if err := entity.ODataBeforeCreate(ctx, req); err != nil {
			t.Fatalf("BeforeCreate failed: %v", err)
		}
		if err := entity.ODataAfterCreate(ctx, req); err != nil {
			t.Fatalf("AfterCreate failed: %v", err)
		}

		// Simulate update lifecycle
		req = httptest.NewRequest(http.MethodPatch, "/entities(1)", nil)
		if err := entity.ODataBeforeUpdate(ctx, req); err != nil {
			t.Fatalf("BeforeUpdate failed: %v", err)
		}
		if err := entity.ODataAfterUpdate(ctx, req); err != nil {
			t.Fatalf("AfterUpdate failed: %v", err)
		}

		// Simulate delete lifecycle
		req = httptest.NewRequest(http.MethodDelete, "/entities(1)", nil)
		if err := entity.ODataBeforeDelete(ctx, req); err != nil {
			t.Fatalf("BeforeDelete failed: %v", err)
		}
		if err := entity.ODataAfterDelete(ctx, req); err != nil {
			t.Fatalf("AfterDelete failed: %v", err)
		}

		// Verify all hooks were called
		if !entity.beforeCreateCalled || !entity.afterCreateCalled ||
			!entity.beforeUpdateCalled || !entity.afterUpdateCalled ||
			!entity.beforeDeleteCalled || !entity.afterDeleteCalled {
			t.Error("Not all hooks were called during lifecycle")
		}
	})
}

func TestEntityWithoutHooks(t *testing.T) {
	// This test verifies that entities without hooks don't need to implement the interface
	// The interface is optional - entities can implement any subset of the hook methods
	entity := &entityWithoutHooks{ID: 1, Name: "Test"}

	// Verify the entity exists but doesn't implement EntityHooks
	// This is a compile-time check - if it compiles, the test passes
	if entity.ID != 1 {
		t.Error("Entity should have ID 1")
	}
	if entity.Name != "Test" {
		t.Error("Entity should have Name 'Test'")
	}
}

func TestHookWithNilContext(t *testing.T) {
	entity := &mockEntity{}
	req := httptest.NewRequest(http.MethodPost, "/entities", nil)

	// Test that hooks can be called with nil context (though not recommended)
	// This should not panic
	err := entity.ODataBeforeCreate(nil, req) //nolint:staticcheck // Testing nil context handling
	if err != nil {
		t.Errorf("Expected no error with nil context, got %v", err)
	}
}

func TestHookWithCancelledContext(t *testing.T) {
	entity := &mockEntity{}
	req := httptest.NewRequest(http.MethodPost, "/entities", nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Hooks should still be callable with cancelled context
	// (though the hook implementation might choose to check for cancellation)
	err := entity.ODataBeforeCreate(ctx, req)
	if err != nil {
		t.Errorf("Expected no error with cancelled context, got %v", err)
	}
}
