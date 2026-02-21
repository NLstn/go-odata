package handlers

import (
	"errors"
	"net/http"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/preference"
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

// collectionExecutionContext provides the data and optional override hooks required to
// execute a collection query pipeline.
//
// For the standard entity collection path, set W, R, and Pref; the executor will call
// EntityHandler methods directly, avoiding closure allocations on every request.
//
// For specialised paths (e.g. navigation properties, tests), set the individual function
// fields as overrides. When a function field is non-nil it takes precedence over the
// data-field fallback.
type collectionExecutionContext struct {
	Metadata *metadata.EntityMetadata

	// Data fields used by the standard entity-collection path.
	// When W and R are set, any nil function field falls back to the corresponding
	// EntityHandler method, eliminating per-request closure allocations.
	W    http.ResponseWriter
	R    *http.Request
	Pref *preference.Preference

	// Optional function overrides. Non-nil values take precedence over the data-field
	// fallback. Used by navigation properties, tests, and other specialised callers.
	ParseQueryOptions func() (*query.QueryOptions, error)
	BeforeRead        func(*query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)
	CountFunc         func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (*int64, error)
	FetchFunc         func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error)
	NextLinkFunc      func(*query.QueryOptions, interface{}) (*string, interface{}, error)
	AfterRead         func(*query.QueryOptions, interface{}) (interface{}, bool, error)
	WriteResponse     func(*query.QueryOptions, interface{}, *int64, *string) error
}

func (h *EntityHandler) executeCollectionQuery(w http.ResponseWriter, r *http.Request, ctx *collectionExecutionContext) {
	if ctx == nil {
		h.logger.Error("executeCollectionQuery: nil context - this is a programming error")
		if err := response.WriteError(w, r, http.StatusInternalServerError, "Internal error", "executeCollectionQuery context is nil"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	// Require either function overrides for the three required steps, or data fields (W+R)
	// that let the executor fall back to EntityHandler methods without closures.
	hasDataFields := ctx.W != nil && ctx.R != nil
	if (ctx.ParseQueryOptions == nil || ctx.FetchFunc == nil || ctx.WriteResponse == nil) && !hasDataFields {
		h.logger.Error("executeCollectionQuery: missing required callbacks - this is a programming error")
		if err := response.WriteError(w, r, http.StatusInternalServerError, "Internal error", "executeCollectionQuery requires ParseQueryOptions, FetchFunc, and WriteResponse callbacks"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	var queryOptions *query.QueryOptions
	var err error
	if ctx.ParseQueryOptions != nil {
		queryOptions, err = ctx.ParseQueryOptions()
	} else {
		queryOptions, err = h.parseCollectionQueryOptionsImpl(ctx.W, ctx.R, ctx.Pref)
	}
	if !h.handleCollectionError(w, r, err, http.StatusBadRequest, ErrMsgInvalidQueryOptions) {
		return
	}

	var scopes []func(*gorm.DB) *gorm.DB
	if ctx.BeforeRead != nil {
		scopes, err = ctx.BeforeRead(queryOptions)
		if !h.handleCollectionError(w, r, err, http.StatusForbidden, "Authorization failed") {
			return
		}
	} else if hasDataFields {
		scopes, err = h.beforeReadCollectionImpl(ctx.R, queryOptions)
		if !h.handleCollectionError(w, r, err, http.StatusForbidden, "Authorization failed") {
			return
		}
	}

	var totalCount *int64
	if ctx.CountFunc != nil {
		totalCount, err = ctx.CountFunc(queryOptions, scopes)
		if !h.handleCollectionError(w, r, err, http.StatusInternalServerError, ErrMsgDatabaseError) {
			return
		}
	} else if hasDataFields {
		totalCount, err = h.collectionCountFuncImpl(ctx.R.Context(), queryOptions, scopes)
		if !h.handleCollectionError(w, r, err, http.StatusInternalServerError, ErrMsgDatabaseError) {
			return
		}
	}

	var results interface{}
	if ctx.FetchFunc != nil {
		results, err = ctx.FetchFunc(queryOptions, scopes)
	} else {
		results, err = h.fetchResultsWithTypeCastImpl(ctx.R, queryOptions, scopes)
	}
	if !h.handleCollectionError(w, r, err, http.StatusInternalServerError, ErrMsgDatabaseError) {
		return
	}

	var nextLink *string
	if ctx.NextLinkFunc != nil {
		nextLink, results, err = ctx.NextLinkFunc(queryOptions, results)
		if !h.handleCollectionError(w, r, err, http.StatusInternalServerError, ErrMsgInternalError) {
			return
		}
	} else if hasDataFields {
		nextLink, results = h.collectionNextLinkFuncImpl(ctx.R, queryOptions, results)
	}

	if ctx.AfterRead != nil {
		if override, hasOverride, hookErr := ctx.AfterRead(queryOptions, results); !h.handleCollectionError(w, r, hookErr, http.StatusForbidden, "Authorization failed") {
			return
		} else if hasOverride {
			results = override
		}
	} else if hasDataFields {
		if override, hasOverride, hookErr := h.afterReadCollectionImpl(ctx.R, queryOptions, results); !h.handleCollectionError(w, r, hookErr, http.StatusForbidden, "Authorization failed") {
			return
		} else if hasOverride {
			results = override
		}
	}

	if ctx.WriteResponse != nil {
		h.handleCollectionError(w, r, ctx.WriteResponse(queryOptions, results, totalCount, nextLink), http.StatusInternalServerError, ErrMsgInternalError)
	} else {
		h.handleCollectionError(w, r, h.collectionResponseWriterImpl(ctx.W, ctx.R, ctx.Pref, queryOptions, results, totalCount, nextLink), http.StatusInternalServerError, ErrMsgInternalError)
	}
}

func (h *EntityHandler) handleCollectionError(w http.ResponseWriter, r *http.Request, err error, defaultStatus int, defaultCode string) bool {
	if err == nil {
		return true
	}

	if errors.Is(err, errRequestHandled) {
		return false
	}

	// Check for GeospatialNotEnabledError
	if IsGeospatialNotEnabledError(err) {
		if writeErr := response.WriteError(w, r, http.StatusNotImplemented, "Geospatial features not enabled", err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return false
	}

	// Check for HookError first (public API error type)
	if isHookErr, status, message, details := extractHookErrorDetails(err, defaultStatus, defaultCode); isHookErr {
		if writeErr := response.WriteError(w, r, status, message, details); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
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

		if writeErr := response.WriteError(w, r, status, code, reqErr.Message); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return false
	}

	if writeErr := response.WriteError(w, r, defaultStatus, defaultCode, err.Error()); writeErr != nil {
		h.logger.Error("Error writing error response", "error", writeErr)
	}
	return false
}
