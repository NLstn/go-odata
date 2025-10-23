package odata

import (
	"context"
	"net/http"
)

// EntityHooks defines the interface for entity lifecycle hooks that users can implement
// on their entity types to add custom business logic.
//
// All hook methods are optional. If a hook method exists on an entity type, it will be
// automatically detected and called at the appropriate time in the request lifecycle.
//
// Hook methods should be defined as methods on the entity type (not pointer receivers
// unless the entity is always passed by pointer).
//
// Example:
//
//	type Product struct {
//	    ID    uint    `json:"ID" odata:"key"`
//	    Name  string  `json:"Name"`
//	    Price float64 `json:"Price"`
//	}
//
//	// BeforeCreate is called before creating a new Product
//	func (p Product) BeforeCreate(ctx context.Context, r *http.Request) error {
//	    // Check if user is admin
//	    if !isAdmin(r) {
//	        return fmt.Errorf("only admins can create products")
//	    }
//	    return nil
//	}
type EntityHooks interface {
	// BeforeCreate is called before a new entity is created via POST.
	// Return an error to prevent the creation and return that error to the client.
	BeforeCreate(ctx context.Context, r *http.Request) error

	// AfterCreate is called after a new entity has been successfully created.
	// Any error returned will be logged but won't affect the response to the client.
	AfterCreate(ctx context.Context, r *http.Request) error

	// BeforeUpdate is called before an entity is updated via PATCH or PUT.
	// Return an error to prevent the update and return that error to the client.
	BeforeUpdate(ctx context.Context, r *http.Request) error

	// AfterUpdate is called after an entity has been successfully updated.
	// Any error returned will be logged but won't affect the response to the client.
	AfterUpdate(ctx context.Context, r *http.Request) error

	// BeforeDelete is called before an entity is deleted via DELETE.
	// Return an error to prevent the deletion and return that error to the client.
	BeforeDelete(ctx context.Context, r *http.Request) error

	// AfterDelete is called after an entity has been successfully deleted.
	// Any error returned will be logged but won't affect the response to the client.
	AfterDelete(ctx context.Context, r *http.Request) error
}
