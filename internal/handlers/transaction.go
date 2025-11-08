package handlers

import (
	"errors"

	"github.com/nlstn/go-odata/internal/trackchanges"
)

type changeEvent struct {
	entity     interface{}
	changeType trackchanges.ChangeType
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
