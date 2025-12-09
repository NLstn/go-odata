package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type transactionHookEntity struct {
	ObservedTx *gorm.DB
	ObservedOK bool
}

func (e *transactionHookEntity) ODataBeforeCreate(ctx context.Context, _ *http.Request) error {
	e.ObservedTx, e.ObservedOK = TransactionFromContext(ctx)
	return nil
}

func TestCallHookIncludesTransactionInContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	req := httptest.NewRequest("POST", "/entities", nil)
	entity := &transactionHookEntity{}

	if err := db.Transaction(func(tx *gorm.DB) error {
		hookReq := requestWithTransaction(req, tx)
		if err := callHook(entity, "ODataBeforeCreate", hookReq); err != nil {
			return err
		}
		if !entity.ObservedOK {
			t.Fatalf("transaction was not available to hook")
		}
		if entity.ObservedTx != tx {
			t.Fatalf("hook received unexpected transaction pointer")
		}
		return nil
	}); err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
}
