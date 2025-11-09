package odata

import (
	"context"

	"github.com/nlstn/go-odata/internal/handlers"
	"gorm.io/gorm"
)

// TransactionFromContext returns the active *gorm.DB transaction stored for hook execution.
// Hooks invoked by the entity and collection write handlers can opt into the shared
// transaction by calling this helper with the context they receive.
func TransactionFromContext(ctx context.Context) (*gorm.DB, bool) {
	return handlers.TransactionFromContext(ctx)
}
