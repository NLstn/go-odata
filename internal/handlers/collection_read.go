package handlers

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/skiptoken"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (h *EntityHandler) handleGetCollection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Start tracing span for collection read
	var span trace.Span
	if h.observability != nil {
		tracer := h.observability.Tracer()
		ctx, span = tracer.StartEntityRead(ctx, h.metadata.EntitySetName, "", h.metadata.IsSingleton)
		defer span.End()
		r = r.WithContext(ctx)
	}

	// Check if there's an overwrite handler
	if h.overwrite.hasGetCollection() {
		h.handleGetCollectionOverwrite(w, r)
		return
	}

	// Check if this is a virtual entity without overwrite handler
	if h.metadata.IsVirtual {
		if err := response.WriteError(w, r, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
			"Virtual entities require an overwrite handler for GetCollection operation"); err != nil {
			h.logger.Error("Error writing error response", "error", err)
		}
		return
	}

	pref := preference.ParsePrefer(r)

	h.executeCollectionQuery(w, r, &collectionExecutionContext{
		Metadata:          h.metadata,
		ParseQueryOptions: h.parseCollectionQueryOptions(w, r, pref),
		BeforeRead:        h.beforeReadCollection(r),
		CountFunc:         h.collectionCountFunc(ctx),
		FetchFunc:         h.fetchResultsWithTypeCast(r),
		NextLinkFunc:      h.collectionNextLinkFunc(r),
		AfterRead:         h.afterReadCollection(r),
		WriteResponse:     h.collectionResponseWriter(w, r, pref),
	})
}

// handleGetCollectionOverwrite handles GET collection requests using the overwrite handler
func (h *EntityHandler) handleGetCollectionOverwrite(w http.ResponseWriter, r *http.Request) {
	pref := preference.ParsePrefer(r)

	// Parse and validate query options
	queryOptions, err := query.ParseQueryOptionsWithConfig(query.ParseRawQuery(r.URL.RawQuery), h.metadata, h.getParserConfig())
	if err != nil {
		h.writeInvalidQueryError(w, r, err)
		return
	}

	// Check if geospatial operations are used but not enabled
	if queryOptions.Filter != nil && query.ContainsGeospatialOperations(queryOptions.Filter) {
		if !h.IsGeospatialEnabled() {
			if writeErr := response.WriteError(w, r, http.StatusNotImplemented, "Geospatial features not enabled", "geospatial features are not enabled for this service"); writeErr != nil {
				h.logger.Error("Error writing error response", "error", writeErr)
			}
			return
		}
	}

	// Handle delta token requests - these are not supported with overwrite handlers
	// because delta tokens require change tracking at the data layer
	if queryOptions.DeltaToken != nil {
		h.handleDeltaCollection(w, r, *queryOptions.DeltaToken)
		return
	}

	// Apply max page size preference
	if pref.MaxPageSize != nil {
		queryOptions = h.applyMaxPageSize(queryOptions, *pref.MaxPageSize)
	}

	// Apply default max top if no explicit $top is set
	queryOptions = h.applyDefaultMaxTop(queryOptions)

	if err := applyPolicyFilter(r, h.policy, buildEntityResourceDescriptor(h.metadata, "", nil), queryOptions); err != nil {
		WriteError(w, r, http.StatusForbidden, "Authorization failed", err.Error())
		return
	}
	if err := applyPolicyFiltersToExpand(r, h.policy, h.metadata, queryOptions.Expand); err != nil {
		WriteError(w, r, http.StatusForbidden, "Authorization failed", err.Error())
		return
	}

	// Create overwrite context
	ctx := &OverwriteContext{
		QueryOptions: queryOptions,
		Request:      r,
	}

	// Call the overwrite handler
	result, err := h.overwrite.getCollection(ctx)
	if err != nil {
		h.writeHookError(w, r, err, http.StatusInternalServerError, "Error fetching collection")
		return
	}

	if result == nil {
		result = &CollectionResult{Items: []interface{}{}}
	}

	// Build the response
	if err := h.collectionResponseWriter(w, r, pref)(queryOptions, result.Items, result.Count, nil); err != nil {
		h.logger.Error("Error writing collection response", "error", err)
	}
}

func (h *EntityHandler) parseCollectionQueryOptions(w http.ResponseWriter, r *http.Request, pref *preference.Preference) func() (*query.QueryOptions, error) {
	return func() (*query.QueryOptions, error) {
		queryOptions, err := query.ParseQueryOptionsWithConfig(query.ParseRawQuery(r.URL.RawQuery), h.metadata, h.getParserConfig())
		if err != nil {
			return nil, err
		}

		// Check if geospatial operations are used but not enabled
		if queryOptions.Filter != nil && query.ContainsGeospatialOperations(queryOptions.Filter) {
			if !h.IsGeospatialEnabled() {
				return nil, &GeospatialNotEnabledError{}
			}
		}

		if queryOptions.DeltaToken != nil {
			h.handleDeltaCollection(w, r, *queryOptions.DeltaToken)
			return nil, errRequestHandled
		}

		if err := h.validateSkipToken(queryOptions); err != nil {
			return nil, &collectionRequestError{
				StatusCode: http.StatusBadRequest,
				ErrorCode:  "Invalid $skiptoken",
				Message:    err.Error(),
			}
		}

		if err := h.validateComplexTypeUsage(queryOptions); err != nil {
			return nil, &collectionRequestError{
				StatusCode: http.StatusBadRequest,
				ErrorCode:  "Unsupported query option",
				Message:    err.Error(),
			}
		}

		if pref.MaxPageSize != nil {
			queryOptions = h.applyMaxPageSize(queryOptions, *pref.MaxPageSize)
		}

		// Apply default max top if no explicit $top is set
		queryOptions = h.applyDefaultMaxTop(queryOptions)

		if err := applyPolicyFilter(r, h.policy, buildEntityResourceDescriptor(h.metadata, "", nil), queryOptions); err != nil {
			return nil, &collectionRequestError{
				StatusCode: http.StatusForbidden,
				ErrorCode:  "Authorization failed",
				Message:    err.Error(),
			}
		}
		if err := applyPolicyFiltersToExpand(r, h.policy, h.metadata, queryOptions.Expand); err != nil {
			return nil, &collectionRequestError{
				StatusCode: http.StatusForbidden,
				ErrorCode:  "Authorization failed",
				Message:    err.Error(),
			}
		}

		return queryOptions, nil
	}
}

func (h *EntityHandler) beforeReadCollection(r *http.Request) func(*query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	return func(queryOptions *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
		scopes, err := callBeforeReadCollection(h.metadata, r, queryOptions)
		if err != nil {
			return nil, err
		}

		if typeCast := GetTypeCast(r.Context()); typeCast != "" {
			if scope := h.buildTypeCastScope(typeCast); scope != nil {
				scopes = append(scopes, scope)
			}
		}

		return scopes, nil
	}
}

func (h *EntityHandler) collectionCountFunc(ctx context.Context) func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (*int64, error) {
	return func(queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (*int64, error) {
		if !queryOptions.Count {
			return nil, nil
		}

		count, err := h.countEntities(ctx, queryOptions, scopes)
		if err != nil {
			return nil, err
		}

		return &count, nil
	}
}

func (h *EntityHandler) collectionNextLinkFunc(r *http.Request) func(*query.QueryOptions, interface{}) (*string, interface{}, error) {
	return func(queryOptions *query.QueryOptions, results interface{}) (*string, interface{}, error) {
		nextLink, needsTrim := h.calculateNextLink(queryOptions, results, r)
		if needsTrim && queryOptions.Top != nil {
			results = h.trimResults(results, *queryOptions.Top)
		}
		return nextLink, results, nil
	}
}

func (h *EntityHandler) afterReadCollection(r *http.Request) func(*query.QueryOptions, interface{}) (interface{}, bool, error) {
	return func(queryOptions *query.QueryOptions, results interface{}) (interface{}, bool, error) {
		return callAfterReadCollection(h.metadata, r, queryOptions, results)
	}
}

func (h *EntityHandler) fetchResults(ctx context.Context, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	modifiedOptions := *queryOptions
	if queryOptions.Top != nil {
		topPlusOne := *queryOptions.Top + 1
		modifiedOptions.Top = &topPlusOne
	}

	db := h.db.WithContext(ctx)
	if len(scopes) > 0 {
		db = db.Scopes(scopes...)
	}
	baseDB := db

	if queryOptions.SkipToken != nil {
		db = h.applySkipTokenFilter(db, queryOptions)
	}

	// Get the table name for FTS from metadata (respects custom TableName() methods)
	tableName := h.metadata.TableName

	// Apply query options with FTS support
	db = query.ApplyQueryOptionsWithFTS(db, &modifiedOptions, h.metadata, h.ftsManager, tableName, h.logger)

	// Check if search was applied at database level
	searchAppliedAtDB := false
	if val, ok := db.Get("_fts_search_applied"); ok {
		if applied, ok := val.(bool); ok && applied {
			searchAppliedAtDB = true
		}
	}

	if query.ShouldUseMapResults(queryOptions) {
		var results []map[string]interface{}
		if err := db.Find(&results).Error; err != nil {
			return nil, err
		}
		// If $select is specified, filter out computed properties that aren't selected
		if len(queryOptions.Select) > 0 && queryOptions.Compute != nil {
			computedAliases := make(map[string]bool)
			for _, expr := range queryOptions.Compute.Expressions {
				computedAliases[expr.Alias] = true
			}
			results = query.ApplySelectToMapResults(results, queryOptions.Select, h.metadata, computedAliases)
		}
		return results, nil
	}

	sliceType := reflect.SliceOf(h.metadata.EntityType)
	results := reflect.New(sliceType).Interface()

	if err := db.Find(results).Error; err != nil {
		return nil, err
	}

	if len(queryOptions.Expand) > 0 {
		if err := query.ApplyPerParentExpand(baseDB, results, queryOptions.Expand, h.metadata); err != nil {
			return nil, err
		}
	}

	sliceValue := reflect.ValueOf(results).Elem().Interface()

	// Only apply in-memory search if it wasn't already applied at database level
	if queryOptions.Search != "" && !searchAppliedAtDB {
		sliceValue = query.ApplySearch(sliceValue, queryOptions.Search, h.metadata)
	}

	// Apply compute transformations to expanded entities
	if len(queryOptions.Expand) > 0 {
		sliceValue = query.ApplyExpandComputeToResults(sliceValue, queryOptions.Expand)
	}

	if len(queryOptions.Select) > 0 {
		sliceValue = query.ApplySelect(sliceValue, queryOptions.Select, h.metadata, queryOptions.Expand)
	}

	return sliceValue, nil
}

func (h *EntityHandler) calculateNextLink(queryOptions *query.QueryOptions, sliceValue interface{}, r *http.Request) (*string, bool) {
	if queryOptions.Top == nil {
		return nil, false
	}

	resultCount := reflect.ValueOf(sliceValue).Len()

	if resultCount > *queryOptions.Top {
		nextURL := buildNextLinkWithSkipToken(h.metadata, queryOptions, sliceValue, r)
		if nextURL != nil {
			return nextURL, true
		}

		currentSkip := 0
		if queryOptions.Skip != nil {
			currentSkip = *queryOptions.Skip
		}
		nextSkip := currentSkip + *queryOptions.Top

		fallbackURL := response.BuildNextLink(r, nextSkip)
		return &fallbackURL, true
	}

	return nil, false
}

func (h *EntityHandler) trimResults(sliceValue interface{}, maxLen int) interface{} {
	v := reflect.ValueOf(sliceValue)
	if v.Kind() != reflect.Slice {
		return sliceValue
	}

	if v.Len() <= maxLen {
		return sliceValue
	}

	if v.Len() > 0 && v.Index(0).Kind() == reflect.Map {
		if mapSlice, ok := sliceValue.([]map[string]interface{}); ok {
			return mapSlice[:maxLen]
		}
	}

	return v.Slice(0, maxLen).Interface()
}

func (h *EntityHandler) fetchResultsWithTypeCast(r *http.Request) func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	ctx := r.Context()
	return func(queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
		results, err := h.fetchResults(ctx, queryOptions, scopes)
		if err != nil {
			return nil, err
		}

		if typeCast := GetTypeCast(ctx); typeCast != "" {
			results = h.filterCollectionByType(results, typeCast)
		}

		return results, nil
	}
}

func (h *EntityHandler) filterCollectionByType(results interface{}, typeCast string) interface{} {
	if typeCast == "" {
		return results
	}

	v := reflect.ValueOf(results)
	if v.Kind() != reflect.Slice {
		return results
	}

	filtered := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if h.entityMatchesType(item.Interface(), typeCast) {
			filtered = reflect.Append(filtered, item)
		}
	}

	return filtered.Interface()
}

func (h *EntityHandler) buildTypeCastScope(typeCast string) func(*gorm.DB) *gorm.DB {
	if typeCast == "" {
		return nil
	}

	columnName := h.typeDiscriminatorColumn()
	if columnName == "" {
		return nil
	}

	typeNames := uniqueStrings(h.typeNameCandidates(typeCast))
	if len(typeNames) == 0 {
		typeNames = []string{typeCast}
	}

	values := make([]interface{}, 0, len(typeNames))
	for _, name := range typeNames {
		values = append(values, name)
	}

	if len(values) == 0 {
		return nil
	}

	return func(db *gorm.DB) *gorm.DB {
		return db.Where(clause.IN{Column: clause.Column{Name: columnName}, Values: values})
	}
}

func (h *EntityHandler) applyMaxPageSize(queryOptions *query.QueryOptions, maxPageSize int) *query.QueryOptions {
	if queryOptions.Top == nil || *queryOptions.Top > maxPageSize {
		queryOptions.Top = &maxPageSize
	}
	return queryOptions
}

// applyDefaultMaxTop applies the default max top limit if no explicit $top is set
// Priority: entity-level default > service-level default
func (h *EntityHandler) applyDefaultMaxTop(queryOptions *query.QueryOptions) *query.QueryOptions {
	// If $top is already explicitly set, don't override it
	if queryOptions.Top != nil {
		return queryOptions
	}

	// Try entity-level default first (from metadata)
	if h.metadata != nil && h.metadata.DefaultMaxTop != nil {
		queryOptions.Top = h.metadata.DefaultMaxTop
		return queryOptions
	}

	// Fall back to handler-level default (from service)
	if h.defaultMaxTop != nil {
		queryOptions.Top = h.defaultMaxTop
	}

	return queryOptions
}

func (h *EntityHandler) applySkipTokenFilter(db *gorm.DB, queryOptions *query.QueryOptions) *gorm.DB {
	if queryOptions.SkipToken == nil {
		return db
	}

	token, err := skiptoken.Decode(*queryOptions.SkipToken)
	if err != nil {
		return db
	}

	if len(queryOptions.OrderBy) > 0 {
		orderByProp := queryOptions.OrderBy[0]
		orderByValue, ok := token.OrderByValues[orderByProp.Property]
		if !ok {
			return db
		}

		var keyValue interface{}
		for keyProp := range token.KeyValues {
			keyValue = token.KeyValues[keyProp]
			break
		}

		var orderByColumnName string
		if orderByMetadata := h.metadata.FindProperty(orderByProp.Property); orderByMetadata != nil {
			// Use cached column name from metadata
			orderByColumnName = orderByMetadata.ColumnName
		}
		if orderByColumnName == "" {
			return db
		}

		var keyColumnName string
		for _, keyProp := range h.metadata.KeyProperties {
			// Use cached column name from metadata
			keyColumnName = keyProp.ColumnName
			break
		}

		if orderByProp.Descending {
			db = db.Where(fmt.Sprintf("(%s < ? OR (%s = ? AND %s > ?))",
				orderByColumnName, orderByColumnName, keyColumnName),
				orderByValue, orderByValue, keyValue)
		} else {
			db = db.Where(fmt.Sprintf("(%s > ? OR (%s = ? AND %s > ?))",
				orderByColumnName, orderByColumnName, keyColumnName),
				orderByValue, orderByValue, keyValue)
		}
	} else {
		var keyColumnName string
		var keyValue interface{}
		for _, keyProp := range h.metadata.KeyProperties {
			// Use cached column name from metadata
			keyColumnName = keyProp.ColumnName
			keyValue = token.KeyValues[keyProp.JsonName]
			break
		}

		if keyColumnName != "" && keyValue != nil {
			db = db.Where(fmt.Sprintf("%s > ?", keyColumnName), keyValue)
		}
	}

	return db
}

func (h *EntityHandler) validateSkipToken(queryOptions *query.QueryOptions) error {
	if queryOptions.SkipToken == nil {
		return nil
	}

	_, err := skiptoken.Decode(*queryOptions.SkipToken)
	if err != nil {
		return fmt.Errorf("invalid skiptoken: %w", err)
	}

	return nil
}

func extractAliasesFromApplyTransformation(trans *query.ApplyTransformation, aliases map[string]bool) {
	if trans == nil {
		return
	}

	switch trans.Type {
	case query.ApplyTypeGroupBy:
		if trans.GroupBy != nil {
			aliases["$count"] = true
			for i := range trans.GroupBy.Transform {
				extractAliasesFromApplyTransformation(&trans.GroupBy.Transform[i], aliases)
			}
		}
	case query.ApplyTypeAggregate:
		if trans.Aggregate != nil {
			for _, expr := range trans.Aggregate.Expressions {
				if expr.Alias != "" {
					aliases[expr.Alias] = true
				}
			}
		}
	case query.ApplyTypeCompute:
		if trans.Compute != nil {
			for _, expr := range trans.Compute.Expressions {
				if expr.Alias != "" {
					aliases[expr.Alias] = true
				}
			}
		}
	}
}

func (h *EntityHandler) validateComplexTypeUsage(queryOptions *query.QueryOptions) error {
	computedAliases := make(map[string]bool)
	if queryOptions.Compute != nil {
		for _, expr := range queryOptions.Compute.Expressions {
			computedAliases[expr.Alias] = true
		}
	}

	for i := range queryOptions.Apply {
		extractAliasesFromApplyTransformation(&queryOptions.Apply[i], computedAliases)
	}

	if queryOptions.Filter != nil {
		if err := h.validateFilterForComplexTypes(queryOptions.Filter, false, computedAliases); err != nil {
			return err
		}
	}

	for _, orderBy := range queryOptions.OrderBy {
		if computedAliases[orderBy.Property] {
			continue
		}

		prop, _, err := h.metadata.ResolvePropertyPath(orderBy.Property)
		if err != nil {
			return fmt.Errorf("property path '%s' is not supported", orderBy.Property)
		}
		if prop.IsNavigationProp {
			return fmt.Errorf("ordering by navigation property '%s' is not supported", orderBy.Property)
		}
		if prop.IsComplexType {
			return fmt.Errorf("ordering by complex type property '%s' is not supported", orderBy.Property)
		}
	}

	return nil
}

// validateFilterForComplexTypes recursively validates a filter expression for complex type usage
// The insideLambda parameter indicates if we're validating properties inside a lambda predicate
// The computedAliases parameter contains aliases of computed properties that should be skipped
func (h *EntityHandler) validateFilterForComplexTypes(filter *query.FilterExpression, insideLambda bool, computedAliases map[string]bool) error {
	if filter == nil {
		return nil
	}

	// Skip property validation if we're inside a lambda predicate
	// Properties inside lambda predicates refer to the related entity, not the current entity
	if !insideLambda && filter.Property != "" && !strings.HasPrefix(filter.Property, "_") {
		// Allow $it (current instance reference) - used in isof() per OData v4 spec 5.1.1.11.4
		// $it can appear when isof() is used with a single argument to check entity type
		if filter.Property == "$it" {
			// $it is valid when used with isof operator or when part of a comparison involving isof
			if filter.Operator != query.OpIsOf && filter.Operator != query.OpEqual && filter.Operator != query.OpNotEqual {
				return fmt.Errorf("property path '$it' can only be used with isof() function")
			}
			// No further validation needed for $it
			goto validateChildren
		}

		// Skip validation for computed properties
		if computedAliases[filter.Property] {
			goto validateChildren
		}

		// Allow lambda operators (any/all) on navigation properties - OData v4 spec 5.1.1.10
		if filter.Operator == query.OpAny || filter.Operator == query.OpAll {
			// For lambda operators, the property is the navigation property
			// The predicate is stored in filter.Left
			prop, _, err := h.metadata.ResolvePropertyPath(filter.Property)
			if err != nil {
				return fmt.Errorf("property path '%s' is not supported", filter.Property)
			}
			if !prop.IsNavigationProp {
				return fmt.Errorf("lambda operator '%s' can only be used with navigation properties", filter.Operator)
			}
			goto validateChildren
		}

		// Check if this is a single-entity navigation property path (e.g., "Team/ClubID")
		// Per OData v4 spec 5.1.1.15, single-entity navigation properties support direct property access
		if h.metadata.IsSingleEntityNavigationPath(filter.Property) {
			// This is valid - single-entity navigation property paths are allowed
			goto validateChildren
		}

		// Check if this looks like a navigation property path but wasn't validated above
		// This provides better error messages for invalid navigation paths
		if strings.Contains(filter.Property, "/") {
			segments := strings.Split(filter.Property, "/")
			if len(segments) >= 2 {
				firstSegment := strings.TrimSpace(segments[0])
				navProp := h.metadata.FindNavigationProperty(firstSegment)
				if navProp != nil {
					if navProp.NavigationIsArray {
						return fmt.Errorf("filtering by collection navigation property '%s' requires lambda operators (use any/all)", firstSegment)
					}
					// Multi-level navigation paths are not currently supported
					if len(segments) > 2 {
						return fmt.Errorf("multi-level navigation paths like '%s' are not currently supported (only single-level paths like 'NavProp/Property')", filter.Property)
					}
				}
			}
		}

		prop, _, err := h.metadata.ResolvePropertyPath(filter.Property)
		if err != nil {
			return fmt.Errorf("property path '%s' is not supported", filter.Property)
		}
		if prop.IsNavigationProp {
			// Only collection navigation properties require any/all operators
			if prop.NavigationIsArray {
				return fmt.Errorf("filtering by collection navigation property '%s' is not supported (use any/all operators)", filter.Property)
			}
			// Single-entity navigation properties are not allowed as terminal values
			// (e.g., "Team eq null" is not currently supported, but "Team/ClubID eq 'xyz'" is)
			return fmt.Errorf("filtering by navigation property '%s' requires a property path (e.g., '%s/PropertyName')", filter.Property, filter.Property)
		}
		if prop.IsComplexType {
			return fmt.Errorf("filtering by complex type property '%s' is not supported", filter.Property)
		}
	}

validateChildren:
	isLambda := filter.Operator == query.OpAny || filter.Operator == query.OpAll

	if filter.Left != nil {
		if err := h.validateFilterForComplexTypes(filter.Left, insideLambda || isLambda, computedAliases); err != nil {
			return err
		}
	}

	if filter.Right != nil {
		if err := h.validateFilterForComplexTypes(filter.Right, insideLambda, computedAliases); err != nil {
			return err
		}
	}

	return nil
}

func (h *EntityHandler) getTotalCount(ctx context.Context, queryOptions *query.QueryOptions, w http.ResponseWriter, r *http.Request, scopes []func(*gorm.DB) *gorm.DB) *int64 {
	if !queryOptions.Count {
		return nil
	}

	count, err := h.countEntities(ctx, queryOptions, scopes)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error())
		return nil
	}
	return &count
}
