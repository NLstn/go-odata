package handlers

import (
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
)

// callBeforeCreate calls the ODataBeforeCreate hook if it exists on the entity
func (h *EntityHandler) callBeforeCreate(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasODataBeforeCreate {
		return nil
	}

	return callHook(entity, "ODataBeforeCreate", r)
}

// callAfterCreate calls the ODataAfterCreate hook if it exists on the entity
func (h *EntityHandler) callAfterCreate(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasODataAfterCreate {
		return nil
	}

	return callHook(entity, "ODataAfterCreate", r)
}

// callBeforeUpdate calls the ODataBeforeUpdate hook if it exists on the entity
func (h *EntityHandler) callBeforeUpdate(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasODataBeforeUpdate {
		return nil
	}

	return callHook(entity, "ODataBeforeUpdate", r)
}

// callAfterUpdate calls the ODataAfterUpdate hook if it exists on the entity
func (h *EntityHandler) callAfterUpdate(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasODataAfterUpdate {
		return nil
	}

	return callHook(entity, "ODataAfterUpdate", r)
}

// callBeforeDelete calls the ODataBeforeDelete hook if it exists on the entity
func (h *EntityHandler) callBeforeDelete(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasODataBeforeDelete {
		return nil
	}

	return callHook(entity, "ODataBeforeDelete", r)
}

// callAfterDelete calls the ODataAfterDelete hook if it exists on the entity
func (h *EntityHandler) callAfterDelete(entity interface{}, r *http.Request) error {
	if !h.metadata.Hooks.HasODataAfterDelete {
		return nil
	}

	return callHook(entity, "ODataAfterDelete", r)
}

// callHook invokes a hook method on an entity using reflection.
// It tries both value and pointer receivers. Hooks invoked from entity and collection
// write handlers receive a request context that carries the active transaction
// (see TransactionFromContext in this package).
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

// callBeforeReadCollection invokes the ODataBeforeReadCollectionGeneric hook if defined.
func callBeforeReadCollection(meta *metadata.EntityMetadata, r *http.Request, opts *query.QueryOptions) error {
	if meta == nil {
		return nil
	}
	return invokeGenericBeforeReadHook(meta, "ODataBeforeReadCollectionGeneric", r, opts)
}

// callAfterReadCollection invokes the ODataAfterReadCollectionGeneric hook if defined and returns an override when provided.
func callAfterReadCollection(meta *metadata.EntityMetadata, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, bool, error) {
	if meta == nil {
		return nil, false, nil
	}
	override, hasOverride, err := invokeGenericAfterReadHook(meta, "ODataAfterReadCollectionGeneric", r, opts, results)
	return override, hasOverride, err
}

// callBeforeReadEntity invokes the ODataBeforeReadEntityGeneric hook if defined.
func callBeforeReadEntity(meta *metadata.EntityMetadata, r *http.Request, opts *query.QueryOptions) error {
	if meta == nil {
		return nil
	}
	return invokeGenericBeforeReadHook(meta, "ODataBeforeReadEntityGeneric", r, opts)
}

// callAfterReadEntity invokes the ODataAfterReadEntityGeneric hook if defined and returns an override when provided.
func callAfterReadEntity(meta *metadata.EntityMetadata, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, bool, error) {
	if meta == nil {
		return nil, false, nil
	}
	override, hasOverride, err := invokeGenericAfterReadHook(meta, "ODataAfterReadEntityGeneric", r, opts, entity)
	return override, hasOverride, err
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

func invokeGenericBeforeReadHook(meta *metadata.EntityMetadata, methodName string, r *http.Request, opts *query.QueryOptions) error {
	ctx := r.Context()
	results, ok := invokeReadHook(meta, methodName, ctx, r, opts)
	if !ok {
		return nil
	}
	if len(results) > 0 {
		if errVal := results[0]; errVal.IsValid() && !errVal.IsNil() {
			if err, ok := errVal.Interface().(error); ok && err != nil {
				return err
			}
		}
	}
	return nil
}

func invokeGenericAfterReadHook(meta *metadata.EntityMetadata, methodName string, r *http.Request, opts *query.QueryOptions, payload interface{}) (interface{}, bool, error) {
	ctx := r.Context()
	results, ok := invokeReadHook(meta, methodName, ctx, r, opts, payload)
	if !ok {
		return nil, false, nil
	}

	overrideProvided := false
	var override interface{}
	if len(results) > 0 {
		first := results[0]
		if first.IsValid() && (first.Kind() != reflect.Interface || !first.IsNil()) {
			override = first.Interface()
			overrideProvided = true
		}
	}
	if len(results) > 1 {
		if errVal := results[1]; errVal.IsValid() && !errVal.IsNil() {
			if err, ok := errVal.Interface().(error); ok && err != nil {
				return nil, false, err
			}
		}
	}
	return override, overrideProvided, nil
}
