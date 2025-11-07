package handlers

import (
	"errors"
	"net/http"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// errRequestHandled is used to signal that the request has already been handled
// and no further processing should occur.
var errRequestHandled = errors.New("request already handled")

// collectionRequestError represents an error that should be returned to the client
// with a specific HTTP status code and error message.
type collectionRequestError struct {
	StatusCode int
	ErrorCode  string
	Message    string
}

func (e *collectionRequestError) Error() string {
	return e.Message
}

// collectionExecutionContext provides the hooks required to execute a collection
// query pipeline. Implementations can customize individual phases such as parsing
// query options, running hooks, fetching data, computing next links, and writing
// the final response while sharing the common orchestration logic.
type collectionExecutionContext struct {
	Metadata *metadata.EntityMetadata

	ParseQueryOptions func() (*query.QueryOptions, error)
	BeforeRead        func(*query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)
	CountFunc         func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (*int64, error)
	FetchFunc         func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error)
	NextLinkFunc      func(*query.QueryOptions, interface{}) (*string, interface{}, error)
	AfterRead         func(*query.QueryOptions, interface{}) (interface{}, bool, error)
	WriteResponse     func(*query.QueryOptions, interface{}, *int64, *string) error
}

func (h *EntityHandler) executeCollectionQuery(w http.ResponseWriter, ctx *collectionExecutionContext) {
	if ctx == nil || ctx.ParseQueryOptions == nil || ctx.FetchFunc == nil || ctx.WriteResponse == nil {
		panic("executeCollectionQuery requires ParseQueryOptions, FetchFunc, and WriteResponse callbacks")
	}

	queryOptions, err := ctx.ParseQueryOptions()
	if !h.handleCollectionError(w, err, http.StatusBadRequest, ErrMsgInvalidQueryOptions) {
		return
	}

	var scopes []func(*gorm.DB) *gorm.DB
	if ctx.BeforeRead != nil {
		scopes, err = ctx.BeforeRead(queryOptions)
		if !h.handleCollectionError(w, err, http.StatusForbidden, "Authorization failed") {
			return
		}
	}

	var totalCount *int64
	if ctx.CountFunc != nil {
		totalCount, err = ctx.CountFunc(queryOptions, scopes)
		if !h.handleCollectionError(w, err, http.StatusInternalServerError, ErrMsgDatabaseError) {
			return
		}
	}

	results, err := ctx.FetchFunc(queryOptions, scopes)
	if !h.handleCollectionError(w, err, http.StatusInternalServerError, ErrMsgDatabaseError) {
		return
	}

	var nextLink *string
	if ctx.NextLinkFunc != nil {
		nextLink, results, err = ctx.NextLinkFunc(queryOptions, results)
		if !h.handleCollectionError(w, err, http.StatusInternalServerError, ErrMsgInternalError) {
			return
		}
	}

	if ctx.AfterRead != nil {
		if override, hasOverride, hookErr := ctx.AfterRead(queryOptions, results); !h.handleCollectionError(w, hookErr, http.StatusForbidden, "Authorization failed") {
			return
		} else if hasOverride {
			results = override
		}
	}

	h.handleCollectionError(w, ctx.WriteResponse(queryOptions, results, totalCount, nextLink), http.StatusInternalServerError, ErrMsgInternalError)
}

func (h *EntityHandler) handleCollectionError(w http.ResponseWriter, err error, defaultStatus int, defaultCode string) bool {
	if err == nil {
		return true
	}

	if errors.Is(err, errRequestHandled) {
		return false
	}

	var reqErr *collectionRequestError
	if errors.As(err, &reqErr) {
		status := reqErr.StatusCode
		if status == 0 {
			status = defaultStatus
		}

		code := reqErr.ErrorCode
		if code == "" {
			code = defaultCode
		}

		if writeErr := response.WriteError(w, status, code, reqErr.Message); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return false
	}

	if writeErr := response.WriteError(w, defaultStatus, defaultCode, err.Error()); writeErr != nil {
		h.logger.Error("Error writing error response", "error", writeErr)
	}
	return false
}
