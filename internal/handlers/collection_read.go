package handlers

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/response"
	"github.com/nlstn/go-odata/internal/skiptoken"
	"gorm.io/gorm"
)

func (h *EntityHandler) handleGetCollection(w http.ResponseWriter, r *http.Request) {
	pref := preference.ParsePrefer(r)

	h.executeCollectionQuery(w, &collectionExecutionContext{
		Metadata:          h.metadata,
		ParseQueryOptions: h.parseCollectionQueryOptions(w, r, pref),
		BeforeRead:        h.beforeReadCollection(r),
		CountFunc:         h.collectionCountFunc(),
		FetchFunc:         h.fetchResults,
		NextLinkFunc:      h.collectionNextLinkFunc(r),
		AfterRead:         h.afterReadCollection(r),
		WriteResponse:     h.collectionResponseWriter(w, r, pref),
	})
}

func (h *EntityHandler) parseCollectionQueryOptions(w http.ResponseWriter, r *http.Request, pref *preference.Preference) func() (*query.QueryOptions, error) {
	return func() (*query.QueryOptions, error) {
		queryOptions, err := query.ParseQueryOptions(r.URL.Query(), h.metadata)
		if err != nil {
			return nil, err
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
			typeParts := strings.Split(typeCast, ".")
			simpleTypeName := typeCast
			if len(typeParts) > 0 {
				simpleTypeName = typeParts[len(typeParts)-1]
			}

			typeCastScope := func(db *gorm.DB) *gorm.DB {
				return db.Where("product_type = ? OR product_type = ?", typeCast, simpleTypeName)
			}
			scopes = append(scopes, typeCastScope)
		}

		return scopes, nil
	}
}

func (h *EntityHandler) collectionCountFunc() func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (*int64, error) {
	return func(queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (*int64, error) {
		if !queryOptions.Count {
			return nil, nil
		}

		countDB := h.db.Model(reflect.New(h.metadata.EntityType).Interface())
		if len(scopes) > 0 {
			countDB = countDB.Scopes(scopes...)
		}

		if queryOptions.Filter != nil {
			countDB = query.ApplyFilterOnly(countDB, queryOptions.Filter, h.metadata)
		}

		var count int64
		if err := countDB.Count(&count).Error; err != nil {
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

func (h *EntityHandler) fetchResults(queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
	modifiedOptions := *queryOptions
	if queryOptions.Top != nil {
		topPlusOne := *queryOptions.Top + 1
		modifiedOptions.Top = &topPlusOne
	}

	db := h.db
	if len(scopes) > 0 {
		db = db.Scopes(scopes...)
	}

	if queryOptions.SkipToken != nil {
		db = h.applySkipTokenFilter(db, queryOptions)
	}

	db = query.ApplyQueryOptions(db, &modifiedOptions, h.metadata)

	if query.ShouldUseMapResults(queryOptions) {
		var results []map[string]interface{}
		if err := db.Find(&results).Error; err != nil {
			return nil, err
		}
		return results, nil
	}

	sliceType := reflect.SliceOf(h.metadata.EntityType)
	results := reflect.New(sliceType).Interface()

	if err := db.Find(results).Error; err != nil {
		return nil, err
	}

	sliceValue := reflect.ValueOf(results).Elem().Interface()

	if queryOptions.Search != "" {
		sliceValue = query.ApplySearch(sliceValue, queryOptions.Search, h.metadata)
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
		nextURL := h.buildNextLinkWithSkipToken(queryOptions, sliceValue, r)
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

func (h *EntityHandler) applyMaxPageSize(queryOptions *query.QueryOptions, maxPageSize int) *query.QueryOptions {
	if queryOptions.Top == nil || *queryOptions.Top > maxPageSize {
		queryOptions.Top = &maxPageSize
	}
	return queryOptions
}

func (h *EntityHandler) buildNextLinkWithSkipToken(queryOptions *query.QueryOptions, sliceValue interface{}, r *http.Request) *string {
	v := reflect.ValueOf(sliceValue)
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return nil
	}

	lastIndex := *queryOptions.Top - 1
	if lastIndex < 0 || lastIndex >= v.Len() {
		return nil
	}

	lastEntity := v.Index(lastIndex).Interface()

	keyProps := make([]string, len(h.metadata.KeyProperties))
	for i, kp := range h.metadata.KeyProperties {
		keyProps[i] = kp.JsonName
	}

	orderByProps := make([]string, len(queryOptions.OrderBy))
	for i, ob := range queryOptions.OrderBy {
		orderByProps[i] = ob.Property
		if ob.Descending {
			orderByProps[i] += " desc"
		}
	}

	token, err := skiptoken.ExtractFromEntity(lastEntity, keyProps, orderByProps)
	if err != nil {
		return nil
	}

	encoded, err := skiptoken.Encode(token)
	if err != nil {
		return nil
	}

	nextURL := response.BuildNextLinkWithSkipToken(r, encoded)
	return &nextURL
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
			orderByColumnName = toSnakeCase(orderByMetadata.Name)
		}

		if orderByColumnName == "" {
			return db
		}

		var keyColumnName string
		for _, keyProp := range h.metadata.KeyProperties {
			keyColumnName = toSnakeCase(keyProp.Name)
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
			keyColumnName = toSnakeCase(keyProp.Name)
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

		prop, _, err := h.metadata.ResolvePropertyPath(filter.Property)
		if err != nil {
			return fmt.Errorf("property path '%s' is not supported", filter.Property)
		}
		if prop.IsNavigationProp {
			return fmt.Errorf("filtering by navigation property '%s' is not supported (use any/all operators)", filter.Property)
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

func (h *EntityHandler) getTotalCount(queryOptions *query.QueryOptions, w http.ResponseWriter, scopes []func(*gorm.DB) *gorm.DB) *int64 {
	if !queryOptions.Count {
		return nil
	}

	var count int64
	countDB := h.db.Model(reflect.New(h.metadata.EntityType).Interface())
	if len(scopes) > 0 {
		countDB = countDB.Scopes(scopes...)
	}

	if queryOptions.Filter != nil {
		countDB = query.ApplyFilterOnly(countDB, queryOptions.Filter, h.metadata)
	}

	if err := countDB.Count(&count).Error; err != nil {
		WriteError(w, http.StatusInternalServerError, ErrMsgDatabaseError, err.Error())
		return nil
	}
	return &count
}
