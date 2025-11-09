package handlers

import (
	"context"
	"net/http"

	"gorm.io/gorm"
)

// Context keys for request-scoped values
type contextKey string

const (
	typeCastKey      contextKey = "odata_type_cast"
	transactionDBKey contextKey = "odata_transaction_db"
)

// WithTypeCast adds a type cast filter to the request context
func WithTypeCast(ctx context.Context, typeName string) context.Context {
	return context.WithValue(ctx, typeCastKey, typeName)
}

// GetTypeCast retrieves the type cast filter from the request context
// Returns empty string if no type cast is present
func GetTypeCast(ctx context.Context) string {
	if typeCast, ok := ctx.Value(typeCastKey).(string); ok {
		return typeCast
	}
	return ""
}

// withTransaction attaches the active transaction to the context for hook consumption.
func withTransaction(ctx context.Context, tx *gorm.DB) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, transactionDBKey, tx)
}

// requestWithTransaction returns a shallow copy of the request whose context includes the transaction.
func requestWithTransaction(r *http.Request, tx *gorm.DB) *http.Request {
	if r == nil {
		return nil
	}
	return r.WithContext(withTransaction(r.Context(), tx))
}

// TransactionFromContext retrieves the active transaction stored for hook execution.
func TransactionFromContext(ctx context.Context) (*gorm.DB, bool) {
	if ctx == nil {
		return nil, false
	}
	tx, ok := ctx.Value(transactionDBKey).(*gorm.DB)
	if !ok || tx == nil {
		return nil, false
	}
	return tx, true
}
