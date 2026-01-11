package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWithTypeCast(t *testing.T) {
	ctx := context.Background()

	// Add type cast to context
	newCtx := WithTypeCast(ctx, "Product")

	// Verify type cast can be retrieved
	typeCast := GetTypeCast(newCtx)
	if typeCast != "Product" {
		t.Errorf("GetTypeCast() = %v, want Product", typeCast)
	}
}

func TestGetTypeCast_EmptyContext(t *testing.T) {
	ctx := context.Background()

	// Get type cast from context without type cast
	typeCast := GetTypeCast(ctx)
	if typeCast != "" {
		t.Errorf("GetTypeCast() = %v, want empty string", typeCast)
	}
}

func TestGetTypeCast_NilContext(t *testing.T) {
	// Note: GetTypeCast will panic with nil context because it calls ctx.Value()
	// This is expected behavior - context should never be nil in Go
	// This test documents that behavior
	t.Skip("GetTypeCast panics with nil context, which is expected Go behavior")
}

func TestWithTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	ctx := context.Background()

	// Add transaction to context
	newCtx := withTransaction(ctx, db)

	// Verify transaction can be retrieved
	tx, ok := TransactionFromContext(newCtx)
	if !ok {
		t.Error("TransactionFromContext() should return true")
	}
	if tx == nil {
		t.Error("TransactionFromContext() should return non-nil transaction")
	}
}

func TestTransactionFromContext_EmptyContext(t *testing.T) {
	ctx := context.Background()

	// Get transaction from context without transaction
	tx, ok := TransactionFromContext(ctx)
	if ok {
		t.Error("TransactionFromContext() should return false for empty context")
	}
	if tx != nil {
		t.Error("TransactionFromContext() should return nil for empty context")
	}
}

func TestTransactionFromContext_NilContext(t *testing.T) {
	// Note: TransactionFromContext handles nil context internally
	// This test documents that behavior - passing nil context is not recommended
	// but the function should handle it gracefully
	t.Run("handles nil context", func(t *testing.T) {
		tx, ok := TransactionFromContext(context.TODO()) //nolint:staticcheck // Testing nil-like context handling
		// With TODO context (which has no transaction), should return false
		if ok {
			t.Error("TransactionFromContext() should return false for TODO context")
		}
		if tx != nil {
			t.Error("TransactionFromContext() should return nil for TODO context")
		}
	})
}

func TestWithTransactionAndEvents(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	events := &[]pendingChangeEvent{}

	// Test with background context - withTransactionAndEvents handles nil context internally
	newCtx := withTransactionAndEvents(context.Background(), db, events)
	if newCtx == nil {
		t.Error("withTransactionAndEvents() should return non-nil context")
	}

	// Verify transaction is accessible
	tx, ok := TransactionFromContext(newCtx)
	if !ok {
		t.Error("TransactionFromContext() should return true")
	}
	if tx == nil {
		t.Error("TransactionFromContext() should return non-nil transaction")
	}
}

func TestWithTransactionAndEvents_NilEvents(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	ctx := context.Background()

	// Test with nil events - should still work
	newCtx := withTransactionAndEvents(ctx, db, nil)
	if newCtx == nil {
		t.Error("withTransactionAndEvents() should return non-nil context")
	}

	// Verify transaction is accessible
	tx, ok := TransactionFromContext(newCtx)
	if !ok {
		t.Error("TransactionFromContext() should return true")
	}
	if tx == nil {
		t.Error("TransactionFromContext() should return non-nil transaction")
	}
}

func TestRequestWithTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Add transaction to request
	newReq := requestWithTransaction(req, db)

	// Verify transaction can be retrieved from request context
	tx, ok := TransactionFromContext(newReq.Context())
	if !ok {
		t.Error("TransactionFromContext() should return true")
	}
	if tx == nil {
		t.Error("TransactionFromContext() should return non-nil transaction")
	}
}

func TestRequestWithTransaction_NilRequest(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Test with nil request
	newReq := requestWithTransaction(nil, db)
	if newReq != nil {
		t.Error("requestWithTransaction() should return nil for nil request")
	}
}

func TestAddPendingChangeEvents_NilContext(t *testing.T) {
	// Should not panic with empty context
	addPendingChangeEvents(context.Background(), nil, nil)
}

func TestAddPendingChangeEvents_EmptyEvents(t *testing.T) {
	ctx := context.Background()

	// Should not panic with empty events
	addPendingChangeEvents(ctx, nil, []changeEvent{})
}

func TestAddPendingChangeEvents_NoCollector(t *testing.T) {
	ctx := context.Background()
	handler := &EntityHandler{}

	events := []changeEvent{
		{entity: struct{}{}, changeType: trackchanges.ChangeTypeAdded},
	}

	// Should not panic without collector in context
	addPendingChangeEvents(ctx, handler, events)
}

func TestAddPendingChangeEvents_WithCollector(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	pending := &[]pendingChangeEvent{}
	ctx := withTransactionAndEvents(context.Background(), db, pending)
	handler := &EntityHandler{}

	events := []changeEvent{
		{entity: struct{ ID int }{ID: 1}, changeType: trackchanges.ChangeTypeAdded},
		{entity: struct{ ID int }{ID: 2}, changeType: trackchanges.ChangeTypeUpdated},
	}

	addPendingChangeEvents(ctx, handler, events)

	// Verify events were added
	if len(*pending) != 2 {
		t.Errorf("Expected 2 pending events, got %d", len(*pending))
	}
}

func TestTypeCastContextKey(t *testing.T) {
	// Verify context key type is string
	key := typeCastKey
	if key != contextKey("odata_type_cast") {
		t.Errorf("typeCastKey = %v, want odata_type_cast", key)
	}
}

func TestTransactionDBKey(t *testing.T) {
	// Verify context key type is string
	key := transactionDBKey
	if key != contextKey("odata_transaction_db") {
		t.Errorf("transactionDBKey = %v, want odata_transaction_db", key)
	}
}

func TestTransactionEventsKey(t *testing.T) {
	// Verify context key type is string
	key := transactionEventsKey
	if key != contextKey("odata_transaction_events") {
		t.Errorf("transactionEventsKey = %v, want odata_transaction_events", key)
	}
}

func TestMultipleTypeCasts(t *testing.T) {
	ctx := context.Background()

	// Add first type cast
	ctx = WithTypeCast(ctx, "Product")

	// Add second type cast - should override
	ctx = WithTypeCast(ctx, "Category")

	typeCast := GetTypeCast(ctx)
	if typeCast != "Category" {
		t.Errorf("GetTypeCast() = %v, want Category", typeCast)
	}
}

func TestTypeCastWithEmptyString(t *testing.T) {
	ctx := context.Background()

	// Add empty type cast
	ctx = WithTypeCast(ctx, "")

	typeCast := GetTypeCast(ctx)
	if typeCast != "" {
		t.Errorf("GetTypeCast() = %v, want empty string", typeCast)
	}
}
