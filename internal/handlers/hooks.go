package handlers

import (
	"net/http"
	"reflect"
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

// callHook invokes a hook method on an entity using reflection
// It tries both value and pointer receivers
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
