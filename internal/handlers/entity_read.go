package handlers

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/internal/etag"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"gorm.io/gorm"
)

// HandleEntity handles GET, HEAD, DELETE, PATCH, PUT, and OPTIONS requests for individual entities
func (h *EntityHandler) HandleEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetEntity(w, r, entityKey)
	case http.MethodDelete:
		h.handleDeleteEntity(w, r, entityKey)
	case http.MethodPatch:
		h.handlePatchEntity(w, r, entityKey)
	case http.MethodPut:
		h.handlePutEntity(w, r, entityKey)
	case http.MethodOptions:
		h.handleOptionsEntity(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for individual entities", r.Method)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
	}
}

// handleGetEntity handles GET requests for individual entities
func (h *EntityHandler) handleGetEntity(w http.ResponseWriter, r *http.Request, entityKey string) {
	// Parse query options for $expand and $select
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		h.writeInvalidQueryError(w, err)
		return
	}

	// Validate that $top and $skip are not used on individual entities
	// Per OData v4 spec, these query options only apply to collections
	if queryOptions.Top != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$top query option is not applicable to individual entities"); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}
	if queryOptions.Skip != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$skip query option is not applicable to individual entities"); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Invoke BeforeReadEntity hooks to obtain scopes
	scopes, hookErr := callBeforeReadEntity(h.metadata, r, queryOptions)
	if hookErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", hookErr.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Fetch the entity
	result, err := h.fetchEntityByKey(entityKey, queryOptions, scopes)
	if err != nil {
		h.handleFetchError(w, err, entityKey)
		return
	}

	// Check type cast from context - if specified, verify entity matches
	if typeCast := GetTypeCast(r.Context()); typeCast != "" {
		if !h.entityMatchesType(result, typeCast) {
			// Entity exists but doesn't match the type cast
			if writeErr := response.WriteError(w, http.StatusNotFound, "Entity not found",
				fmt.Sprintf("Entity with key '%s' is not of type '%s'", entityKey, typeCast)); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
	}

	// Check If-None-Match header if ETag is configured (before applying select)
	var currentETag string
	if h.metadata.ETagProperty != nil {
		currentETag = etag.Generate(result, h.metadata)
	}

	// Invoke AfterReadEntity hooks to allow mutation or override
	override, hasOverride, afterErr := callAfterReadEntity(h.metadata, r, queryOptions, result)
	if afterErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", afterErr.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}
	if hasOverride {
		result = override
	}

	if h.metadata.ETagProperty != nil {
		ifNoneMatch := r.Header.Get(HeaderIfNoneMatch)
		if currentETag != "" && !etag.NoneMatch(ifNoneMatch, currentETag) {
			w.Header().Set(HeaderETag, currentETag)
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Apply $select if specified (after ETag generation)
	if len(queryOptions.Select) > 0 && !hasOverride {
		result = query.ApplySelectToEntity(result, queryOptions.Select, h.metadata, queryOptions.Expand)
	}

	// Build and write response
	h.writeEntityResponseWithETag(w, r, result, currentETag, http.StatusOK)
}

// HandleEntityRef handles GET requests for entity references (e.g., Products(1)/$ref)
func (h *EntityHandler) HandleEntityRef(w http.ResponseWriter, r *http.Request, entityKey string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for entity references", r.Method)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	// Validate that $expand and $select are not used with $ref
	// According to OData v4 spec, $ref does not support $expand or $select
	queryParams := r.URL.Query()
	if queryParams.Get("$expand") != "" {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$expand is not supported with $ref"); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}
	if queryParams.Get("$select") != "" {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$select is not supported with $ref"); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Invoke BeforeReadEntity hooks for authorization
	refQueryOptions := &query.QueryOptions{}
	refScopes, hookErr := callBeforeReadEntity(h.metadata, r, refQueryOptions)
	if hookErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", hookErr.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Fetch the entity to ensure it exists
	entity := reflect.New(h.metadata.EntityType).Interface()
	db, err := h.buildKeyQuery(h.db, entityKey)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidKey, err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}
	if len(refScopes) > 0 {
		db = db.Scopes(refScopes...)
	}

	if err := db.First(entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if writeErr := response.WriteError(w, http.StatusNotFound, ErrMsgEntityNotFound,
				fmt.Sprintf("Entity with key '%s' not found", entityKey)); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
		} else {
			h.writeDatabaseError(w, err)
		}
		return
	}

	if _, _, afterErr := callAfterReadEntity(h.metadata, r, refQueryOptions, entity); afterErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", afterErr.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Extract key values and build entity ID
	keyValues := response.ExtractEntityKeys(entity, h.metadata.KeyProperties)
	entityID := response.BuildEntityID(h.metadata.EntitySetName, keyValues)

	if err := response.WriteEntityReference(w, r, entityID); err != nil {
		h.logger.Error("Error writing entity reference", "error", err)
	}
}

// HandleCollectionRef handles GET requests for collection references (e.g., Products/$ref)
func (h *EntityHandler) HandleCollectionRef(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			fmt.Sprintf("Method %s is not supported for collection references", r.Method)); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	// Validate that $expand and $select are not used with $ref
	// According to OData v4 spec, $ref only supports $filter, $top, $skip, $orderby, and $count
	queryParams := r.URL.Query()
	if queryParams.Get("$expand") != "" {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$expand is not supported with $ref"); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}
	if queryParams.Get("$select") != "" {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions,
			"$select is not supported with $ref"); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Parse query options (support filtering, ordering, pagination for references)
	queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
	if err != nil {
		if writeErr := response.WriteError(w, http.StatusBadRequest, ErrMsgInvalidQueryOptions, err.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Invoke BeforeReadCollection hooks to obtain scopes
	scopes, hookErr := callBeforeReadCollection(h.metadata, r, queryOptions)
	if hookErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", hookErr.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	}

	// Get the total count if $count=true is specified
	totalCount := h.getTotalCount(queryOptions, w, scopes)
	if totalCount == nil && queryOptions.Count {
		return // Error already written
	}

	// Fetch the results
	results, err := h.fetchResults(queryOptions, scopes)
	if err != nil {
		h.writeDatabaseError(w, err)
		return
	}

	// Calculate next link if pagination is active and trim results if needed
	nextLink, needsTrimming := h.calculateNextLink(queryOptions, results, r)
	if needsTrimming && queryOptions.Top != nil {
		// Trim the results to $top (we fetched $top + 1 to check for more pages)
		results = h.trimResults(results, *queryOptions.Top)
	}

	if override, hasOverride, afterErr := callAfterReadCollection(h.metadata, r, queryOptions, results); afterErr != nil {
		if writeErr := response.WriteError(w, http.StatusForbidden, "Authorization failed", afterErr.Error()); writeErr != nil {
			h.logger.Error("Error writing error response", "error", writeErr)
		}
		return
	} else if hasOverride {
		results = override
	}

	// Build entity IDs for each entity
	var entityIDs []string
	sliceValue := reflect.ValueOf(results)
	if sliceValue.Kind() == reflect.Slice {
		for i := 0; i < sliceValue.Len(); i++ {
			entity := sliceValue.Index(i).Interface()
			keyValues := response.ExtractEntityKeys(entity, h.metadata.KeyProperties)
			entityID := response.BuildEntityID(h.metadata.EntitySetName, keyValues)
			entityIDs = append(entityIDs, entityID)
		}
	}

	if err := response.WriteEntityReferenceCollection(w, r, entityIDs, totalCount, nextLink); err != nil {
		h.logger.Error("Error writing entity reference collection", "error", err)
	}
}
