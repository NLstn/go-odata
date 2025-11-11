package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
)

type changeEvent struct {
	entity     interface{}
	changeType trackchanges.ChangeType
}

type pendingChangeEvent struct {
	handler *EntityHandler
	event   changeEvent
}

// transactionHandledError indicates the transaction already wrote an HTTP response
// and should simply be rolled back without additional error handling.
type transactionHandledError struct {
	err error
}

func (e *transactionHandledError) Error() string {
	if e == nil {
		return "transaction handled"
	}
	if e.err == nil {
		return "transaction handled"
	}
	return e.err.Error()
}

// newTransactionHandledError wraps an error indicating the response has been handled.
func newTransactionHandledError(err error) error {
	return &transactionHandledError{err: err}
}

// isTransactionHandled reports whether the error indicates the HTTP response was handled.
func isTransactionHandled(err error) bool {
	if err == nil {
		return false
	}
	var target *transactionHandledError
	return errors.As(err, &target)
}

func (h *EntityHandler) runInTransaction(ctx context.Context, r *http.Request, fn func(tx *gorm.DB, hookReq *http.Request) error) error {
	if ctxTx, ok := TransactionFromContext(ctx); ok && ctxTx != nil {
		tx := ctxTx.WithContext(ctx)
		return fn(tx, requestWithTransaction(r, tx))
	}

	return h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx, requestWithTransaction(r, tx))
	})
}

func flushPendingChangeEvents(events []pendingChangeEvent) {
	for _, evt := range events {
		if evt.handler == nil {
			continue
		}
		evt.handler.recordChange(evt.event.entity, evt.event.changeType)
	}
}
