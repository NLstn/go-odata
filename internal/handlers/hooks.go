package handlers

import (
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/gorm"
)

// callBeforeCreate calls the BeforeCreate hook if it exists on the entity
func (h *EntityHandler) callBeforeCreate(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasBeforeCreate {
		return nil
	}

	return callHook(entity, "BeforeCreate", r)
}

// callAfterCreate calls the AfterCreate hook if it exists on the entity
func (h *EntityHandler) callAfterCreate(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasAfterCreate {
		return nil
	}

	return callHook(entity, "AfterCreate", r)
}

// callBeforeUpdate calls the BeforeUpdate hook if it exists on the entity
func (h *EntityHandler) callBeforeUpdate(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasBeforeUpdate {
		return nil
	}

	return callHook(entity, "BeforeUpdate", r)
}

// callAfterUpdate calls the AfterUpdate hook if it exists on the entity
func (h *EntityHandler) callAfterUpdate(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasAfterUpdate {
		return nil
	}

	return callHook(entity, "AfterUpdate", r)
}

// callBeforeDelete calls the BeforeDelete hook if it exists on the entity
func (h *EntityHandler) callBeforeDelete(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasBeforeDelete {
		return nil
	}

	return callHook(entity, "BeforeDelete", r)
}

// callAfterDelete calls the AfterDelete hook if it exists on the entity
func (h *EntityHandler) callAfterDelete(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasAfterDelete {
		return nil
	}

	return callHook(entity, "AfterDelete", r)
}

// callHook invokes a hook method on an entity using reflection.
// It tries both value and pointer receivers. Hooks receive the request context and
// can access the active transaction via odata.TransactionFromContext when invoked
// from entity and collection write handlers.
func callHook(entity interface{}, methodName string, r *http.Request) error {
	ctx := r.Context()

	// Get the value and type
	entityValue := reflect.ValueOf(entity)

	// If entity is a pointer, get the element
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Try to call the method on the value receiver first
	method := entityValue.MethodByName(methodName)
	if !method.IsValid() {
		// Try pointer receiver
		ptrValue := entityValue.Addr()
		method = ptrValue.MethodByName(methodName)
	}

	if !method.IsValid() {
		// Method not found (shouldn't happen if metadata.Hooks is correct)
		return nil
	}

	// Call the method with context and request
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(r),
	}

	results := method.Call(args)

	// Check if the method returned an error
	if len(results) > 0 {
		if err, ok := results[0].Interface().(error); ok {
			return err
		}
	}

	return nil
}

// callBeforeReadCollection invokes the BeforeReadCollection hook if defined and returns any scopes it produces.
func callBeforeReadCollection(meta *metadata.EntityMetadata, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	if meta == nil || !meta.Hooks.HasBeforeReadCollection {
		return nil, nil
	}

	ctx := r.Context()
	results, ok := invokeReadHook(meta, "BeforeReadCollection", ctx, r, opts)
	if !ok || len(results) == 0 {
		return nil, nil
	}

	var scopes []func(*gorm.DB) *gorm.DB
	if first := results[0]; first.IsValid() && (first.Kind() != reflect.Interface || !first.IsNil()) {
		if s, ok := first.Interface().([]func(*gorm.DB) *gorm.DB); ok {
			scopes = s
		}
	}

	if len(results) > 1 {
		if errVal := results[1]; errVal.IsValid() && !errVal.IsNil() {
			if err, ok := errVal.Interface().(error); ok && err != nil {
				return nil, err
			}
		}
	}

	return scopes, nil
}

// callAfterReadCollection invokes the AfterReadCollection hook if defined and returns an override when provided.
func callAfterReadCollection(meta *metadata.EntityMetadata, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, bool, error) {
	if meta == nil || !meta.Hooks.HasAfterReadCollection {
		return nil, false, nil
	}

	ctx := r.Context()
	callResults, ok := invokeReadHook(meta, "AfterReadCollection", ctx, r, opts, results)
	if !ok || len(callResults) == 0 {
		return nil, false, nil
	}

	overrideProvided := false
	var override interface{}

	if first := callResults[0]; first.IsValid() {
		// Treat typed nils as an explicit override but ignore interface nils.
		if first.Kind() != reflect.Interface || !first.IsNil() {
			override = first.Interface()
			overrideProvided = true
		}
	}

	if len(callResults) > 1 {
		if errVal := callResults[1]; errVal.IsValid() && !errVal.IsNil() {
			if err, ok := errVal.Interface().(error); ok && err != nil {
				return nil, false, err
			}
		}
	}

	return override, overrideProvided, nil
}

// callBeforeReadEntity invokes the BeforeReadEntity hook if defined and returns any scopes it produces.
func callBeforeReadEntity(meta *metadata.EntityMetadata, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	if meta == nil || !meta.Hooks.HasBeforeReadEntity {
		return nil, nil
	}

	ctx := r.Context()
	results, ok := invokeReadHook(meta, "BeforeReadEntity", ctx, r, opts)
	if !ok || len(results) == 0 {
		return nil, nil
	}

	var scopes []func(*gorm.DB) *gorm.DB
	if first := results[0]; first.IsValid() && (first.Kind() != reflect.Interface || !first.IsNil()) {
		if s, ok := first.Interface().([]func(*gorm.DB) *gorm.DB); ok {
			scopes = s
		}
	}

	if len(results) > 1 {
		if errVal := results[1]; errVal.IsValid() && !errVal.IsNil() {
			if err, ok := errVal.Interface().(error); ok && err != nil {
				return nil, err
			}
		}
	}

	return scopes, nil
}

// callAfterReadEntity invokes the AfterReadEntity hook if defined and returns an override when provided.
func callAfterReadEntity(meta *metadata.EntityMetadata, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, bool, error) {
	if meta == nil || !meta.Hooks.HasAfterReadEntity {
		return nil, false, nil
	}

	ctx := r.Context()
	callResults, ok := invokeReadHook(meta, "AfterReadEntity", ctx, r, opts, entity)
	if !ok || len(callResults) == 0 {
		return nil, false, nil
	}

	overrideProvided := false
	var override interface{}

	if first := callResults[0]; first.IsValid() {
		if first.Kind() != reflect.Interface || !first.IsNil() {
			override = first.Interface()
			overrideProvided = true
		}
	}

	if len(callResults) > 1 {
		if errVal := callResults[1]; errVal.IsValid() && !errVal.IsNil() {
			if err, ok := errVal.Interface().(error); ok && err != nil {
				return nil, false, err
			}
		}
	}

	return override, overrideProvided, nil
}

// invokeReadHook instantiates an entity value for the provided metadata and calls the requested hook method.
func invokeReadHook(meta *metadata.EntityMetadata, methodName string, args ...interface{}) ([]reflect.Value, bool) {
	if meta == nil {
		return nil, false
	}

	entityPtr := reflect.New(meta.EntityType)
	entityValue := entityPtr.Elem()

	if method := entityValue.MethodByName(methodName); method.IsValid() {
		return callHookMethod(method, args...), true
	}

	if method := entityPtr.MethodByName(methodName); method.IsValid() {
		return callHookMethod(method, args...), true
	}

	return nil, false
}

// callHookMethod converts arguments to reflect.Values and invokes the method.
func callHookMethod(method reflect.Value, args ...interface{}) []reflect.Value {
	callArgs := make([]reflect.Value, len(args))
	methodType := method.Type()
	for i, arg := range args {
		if arg == nil {
			callArgs[i] = reflect.Zero(methodType.In(i))
			continue
		}
		callArgs[i] = reflect.ValueOf(arg)
	}
	return method.Call(callArgs)
}
