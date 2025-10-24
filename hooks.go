package odata

import (
	"github.com/nlstn/go-odata/internal/hooks"
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
//
// This interface is re-exported from internal/hooks for backwards compatibility.
type EntityHooks = hooks.EntityHooks
