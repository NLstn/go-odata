package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// HookedTestEntity is a test entity with hooks
type HookedTestEntity struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name"`
}

// ODataBeforeCreate is called before creating the entity
func (e *HookedTestEntity) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
	if e.Name == "error" {
		return errors.New("before create error")
	}
	e.Name = "beforeCreate:" + e.Name
	return nil
}

// ODataAfterCreate is called after creating the entity
func (e *HookedTestEntity) ODataAfterCreate(ctx context.Context, r *http.Request) error {
	if e.Name == "beforeCreate:afterError" {
		return errors.New("after create error")
	}
	return nil
}

// ODataBeforeUpdate is called before updating the entity
func (e *HookedTestEntity) ODataBeforeUpdate(ctx context.Context, r *http.Request) error {
	if e.Name == "updateError" {
		return errors.New("before update error")
	}
	return nil
}

// ODataAfterUpdate is called after updating the entity
func (e *HookedTestEntity) ODataAfterUpdate(ctx context.Context, r *http.Request) error {
	return nil
}

// ODataBeforeDelete is called before deleting the entity
func (e *HookedTestEntity) ODataBeforeDelete(ctx context.Context, r *http.Request) error {
	if e.Name == "deleteError" {
		return errors.New("before delete error")
	}
	return nil
}

// ODataAfterDelete is called after deleting the entity
func (e *HookedTestEntity) ODataAfterDelete(ctx context.Context, r *http.Request) error {
	return nil
}

func createHandlerWithHook(hookName string, enabled bool) *EntityHandler {
	meta := &metadata.EntityMetadata{}
	switch hookName {
	case "BeforeCreate":
		meta.Hooks.HasODataBeforeCreate = enabled
	case "AfterCreate":
		meta.Hooks.HasODataAfterCreate = enabled
	case "BeforeUpdate":
		meta.Hooks.HasODataBeforeUpdate = enabled
	case "AfterUpdate":
		meta.Hooks.HasODataAfterUpdate = enabled
	case "BeforeDelete":
		meta.Hooks.HasODataBeforeDelete = enabled
	case "AfterDelete":
		meta.Hooks.HasODataAfterDelete = enabled
	}
	return &EntityHandler{metadata: meta}
}

func TestCallBeforeCreate(t *testing.T) {
	tests := []struct {
		name       string
		entity     interface{}
		hasHook    bool
		wantErr    bool
		wantName   string
	}{
		{
			name:       "With hook that succeeds",
			entity:     &HookedTestEntity{Name: "test"},
			hasHook:    true,
			wantErr:    false,
			wantName:   "beforeCreate:test",
		},
		{
			name:       "With hook that fails",
			entity:     &HookedTestEntity{Name: "error"},
			hasHook:    true,
			wantErr:    true,
			wantName:   "error",
		},
		{
			name:       "Without hook",
			entity:     &HookedTestEntity{Name: "test"},
			hasHook:    false,
			wantErr:    false,
			wantName:   "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createHandlerWithHook("BeforeCreate", tt.hasHook)
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			err := handler.callBeforeCreate(tt.entity, req)

			if (err != nil) != tt.wantErr {
				t.Errorf("callBeforeCreate() error = %v, wantErr %v", err, tt.wantErr)
			}

			entity := tt.entity.(*HookedTestEntity)
			if entity.Name != tt.wantName {
				t.Errorf("entity.Name = %v, want %v", entity.Name, tt.wantName)
			}
		})
	}
}

func TestCallAfterCreate(t *testing.T) {
	tests := []struct {
		name    string
		entity  interface{}
		hasHook bool
		wantErr bool
	}{
		{
			name:    "With hook that succeeds",
			entity:  &HookedTestEntity{Name: "test"},
			hasHook: true,
			wantErr: false,
		},
		{
			name:    "With hook that fails",
			entity:  &HookedTestEntity{Name: "beforeCreate:afterError"},
			hasHook: true,
			wantErr: true,
		},
		{
			name:    "Without hook",
			entity:  &HookedTestEntity{Name: "test"},
			hasHook: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createHandlerWithHook("AfterCreate", tt.hasHook)
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			err := handler.callAfterCreate(tt.entity, req)

			if (err != nil) != tt.wantErr {
				t.Errorf("callAfterCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCallBeforeUpdate(t *testing.T) {
	tests := []struct {
		name    string
		entity  interface{}
		hasHook bool
		wantErr bool
	}{
		{
			name:    "With hook that succeeds",
			entity:  &HookedTestEntity{Name: "test"},
			hasHook: true,
			wantErr: false,
		},
		{
			name:    "With hook that fails",
			entity:  &HookedTestEntity{Name: "updateError"},
			hasHook: true,
			wantErr: true,
		},
		{
			name:    "Without hook",
			entity:  &HookedTestEntity{Name: "test"},
			hasHook: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createHandlerWithHook("BeforeUpdate", tt.hasHook)
			req := httptest.NewRequest(http.MethodPatch, "/test", nil)
			err := handler.callBeforeUpdate(tt.entity, req)

			if (err != nil) != tt.wantErr {
				t.Errorf("callBeforeUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCallAfterUpdate(t *testing.T) {
	tests := []struct {
		name    string
		entity  interface{}
		hasHook bool
		wantErr bool
	}{
		{
			name:    "With hook that succeeds",
			entity:  &HookedTestEntity{Name: "test"},
			hasHook: true,
			wantErr: false,
		},
		{
			name:    "Without hook",
			entity:  &HookedTestEntity{Name: "test"},
			hasHook: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createHandlerWithHook("AfterUpdate", tt.hasHook)
			req := httptest.NewRequest(http.MethodPatch, "/test", nil)
			err := handler.callAfterUpdate(tt.entity, req)

			if (err != nil) != tt.wantErr {
				t.Errorf("callAfterUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCallBeforeDelete(t *testing.T) {
	tests := []struct {
		name    string
		entity  interface{}
		hasHook bool
		wantErr bool
	}{
		{
			name:    "With hook that succeeds",
			entity:  &HookedTestEntity{Name: "test"},
			hasHook: true,
			wantErr: false,
		},
		{
			name:    "With hook that fails",
			entity:  &HookedTestEntity{Name: "deleteError"},
			hasHook: true,
			wantErr: true,
		},
		{
			name:    "Without hook",
			entity:  &HookedTestEntity{Name: "test"},
			hasHook: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createHandlerWithHook("BeforeDelete", tt.hasHook)
			req := httptest.NewRequest(http.MethodDelete, "/test", nil)
			err := handler.callBeforeDelete(tt.entity, req)

			if (err != nil) != tt.wantErr {
				t.Errorf("callBeforeDelete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCallAfterDelete(t *testing.T) {
	tests := []struct {
		name    string
		entity  interface{}
		hasHook bool
		wantErr bool
	}{
		{
			name:    "With hook that succeeds",
			entity:  &HookedTestEntity{Name: "test"},
			hasHook: true,
			wantErr: false,
		},
		{
			name:    "Without hook",
			entity:  &HookedTestEntity{Name: "test"},
			hasHook: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := createHandlerWithHook("AfterDelete", tt.hasHook)
			req := httptest.NewRequest(http.MethodDelete, "/test", nil)
			err := handler.callAfterDelete(tt.entity, req)

			if (err != nil) != tt.wantErr {
				t.Errorf("callAfterDelete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCallHook_MethodNotFound(t *testing.T) {
	entity := &HookedTestEntity{Name: "test"}
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Call a hook method that doesn't exist
	err := callHook(entity, "NonExistentMethod", req)

	if err != nil {
		t.Errorf("callHook() for non-existent method should return nil, got %v", err)
	}
}

func TestCallHookMethod(t *testing.T) {
	entity := &HookedTestEntity{Name: "test"}

	// Get the method
	method := reflect.ValueOf(entity).MethodByName("ODataBeforeCreate")
	if !method.IsValid() {
		t.Fatal("Method should be valid")
	}

	ctx := context.Background()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	results := callHookMethod(method, ctx, req)

	if len(results) != 1 {
		t.Errorf("callHookMethod() should return 1 result, got %d", len(results))
	}

	// Check that the result is nil (no error)
	if !results[0].IsNil() {
		t.Errorf("callHookMethod() should return nil error, got %v", results[0].Interface())
	}
}

func TestCallHookMethod_WithNilArg(t *testing.T) {
	entity := &HookedTestEntity{Name: "test"}

	// Get the method
	method := reflect.ValueOf(entity).MethodByName("ODataBeforeCreate")
	if !method.IsValid() {
		t.Fatal("Method should be valid")
	}

	// Call with nil args
	results := callHookMethod(method, nil, nil)

	if len(results) != 1 {
		t.Errorf("callHookMethod() should return 1 result, got %d", len(results))
	}
}
