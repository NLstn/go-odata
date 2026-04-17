package odata

import (
	"net/http"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/query"
)

// QueryOptions represents parsed OData query options from an HTTP request.
//
// QueryOptions contains all standard OData v4.01 query options:
//   - Filter: $filter expression for filtering entities
//   - Select: $select properties to include in response
//   - Expand: $expand navigation properties to include
//   - OrderBy: $orderby sorting specification
//   - Top: $top maximum number of entities to return
//   - Skip: $skip number of entities to skip
//   - SkipToken: $skiptoken for server-driven paging
//   - DeltaToken: $deltatoken for change tracking
//   - Count: $count whether to include total count
//   - Apply: $apply data aggregation transformations
//   - Search: $search full-text search query
//   - Compute: $compute computed properties
//   - Index: $index whether to add @odata.index annotations (OData v4.01)
//   - SchemaVersion: $schemaversion for metadata versioning (OData v4.01)
//
// The Index field, when true, adds @odata.index annotations to each item in a
// collection response, indicating the item's position in the original result set
// before any Top or Skip operations.
//
// The SchemaVersion field allows clients to request a specific version of the
// service's metadata schema, enabling metadata versioning scenarios.
//
// Example accessing query options in an overwrite handler:
//
//	func(ctx *OverwriteContext) (*CollectionResult, error) {
//	    if ctx.QueryOptions.Top != nil {
//	        log.Printf("Client requested top %d items", *ctx.QueryOptions.Top)
//	    }
//	    if ctx.QueryOptions.Index {
//	        log.Println("Client requested index annotations")
//	    }
//	    if ctx.QueryOptions.SchemaVersion != nil {
//	        log.Printf("Client requested schema version: %s", *ctx.QueryOptions.SchemaVersion)
//	    }
//	    // ... fetch and return data
//	}
type QueryOptions = query.QueryOptions

// FilterExpression re-exports the parsed $filter expression type for external consumers.
type FilterExpression = query.FilterExpression

// OrderByItem re-exports the parsed $orderby item type for external consumers.
type OrderByItem = query.OrderByItem

// ExpandOption re-exports the parsed $expand option type for external consumers.
type ExpandOption = query.ExpandOption

// FilterOperator re-exports supported filter operators for external consumers.
type FilterOperator = query.FilterOperator

// LogicalOperator re-exports supported logical operators for external consumers.
type LogicalOperator = query.LogicalOperator

// ApplyTransformation re-exports the apply transformation type for external consumers.
type ApplyTransformation = query.ApplyTransformation

// ApplyTransformationType re-exports the apply transformation type enumeration for external consumers.
type ApplyTransformationType = query.ApplyTransformationType

// GroupByTransformation re-exports the groupby transformation type for external consumers.
type GroupByTransformation = query.GroupByTransformation

// AggregateTransformation re-exports the aggregate transformation type for external consumers.
type AggregateTransformation = query.AggregateTransformation

// AggregateExpression re-exports the aggregate expression type for external consumers.
type AggregateExpression = query.AggregateExpression

// AggregationMethod re-exports the aggregation method type for external consumers.
type AggregationMethod = query.AggregationMethod

// ComputeTransformation re-exports the compute transformation type for external consumers.
type ComputeTransformation = query.ComputeTransformation

// ComputeExpression re-exports the compute expression type for external consumers.
type ComputeExpression = query.ComputeExpression

// NestTransformation re-exports the nest transformation type for external consumers.
type NestTransformation = query.NestTransformation

// FromTransformation re-exports the from transformation type for external consumers.
type FromTransformation = query.FromTransformation

// ParserConfig re-exports the parser configuration type for external consumers.
type ParserConfig = query.ParserConfig

// Filter operator constants
const (
	// Comparison operators
	OpEqual              FilterOperator = query.OpEqual
	OpNotEqual           FilterOperator = query.OpNotEqual
	OpGreaterThan        FilterOperator = query.OpGreaterThan
	OpGreaterThanOrEqual FilterOperator = query.OpGreaterThanOrEqual
	OpLessThan           FilterOperator = query.OpLessThan
	OpLessThanOrEqual    FilterOperator = query.OpLessThanOrEqual
	OpIn                 FilterOperator = query.OpIn

	// String functions
	OpContains   FilterOperator = query.OpContains
	OpStartsWith FilterOperator = query.OpStartsWith
	OpEndsWith   FilterOperator = query.OpEndsWith
	OpToLower    FilterOperator = query.OpToLower
	OpToUpper    FilterOperator = query.OpToUpper
	OpTrim       FilterOperator = query.OpTrim
	OpLength     FilterOperator = query.OpLength
	OpIndexOf    FilterOperator = query.OpIndexOf
	OpSubstring  FilterOperator = query.OpSubstring
	OpConcat     FilterOperator = query.OpConcat

	// OData v4.01 string functions
	OpMatchesPattern FilterOperator = query.OpMatchesPattern

	// Enum operator
	OpHas FilterOperator = query.OpHas

	// Arithmetic operators
	OpAdd   FilterOperator = query.OpAdd
	OpSub   FilterOperator = query.OpSub
	OpMul   FilterOperator = query.OpMul
	OpDiv   FilterOperator = query.OpDiv
	OpDivBy FilterOperator = query.OpDivBy
	OpMod   FilterOperator = query.OpMod

	// Math functions
	OpCeiling FilterOperator = query.OpCeiling
	OpFloor   FilterOperator = query.OpFloor
	OpRound   FilterOperator = query.OpRound

	// Date functions
	OpYear               FilterOperator = query.OpYear
	OpMonth              FilterOperator = query.OpMonth
	OpDay                FilterOperator = query.OpDay
	OpHour               FilterOperator = query.OpHour
	OpMinute             FilterOperator = query.OpMinute
	OpSecond             FilterOperator = query.OpSecond
	OpDate               FilterOperator = query.OpDate
	OpTime               FilterOperator = query.OpTime
	OpNow                FilterOperator = query.OpNow
	OpFractionalSeconds  FilterOperator = query.OpFractionalSeconds
	OpTotalOffsetMinutes FilterOperator = query.OpTotalOffsetMinutes
	OpTotalSeconds       FilterOperator = query.OpTotalSeconds
	OpMinDatetime        FilterOperator = query.OpMinDatetime
	OpMaxDatetime        FilterOperator = query.OpMaxDatetime

	// Lambda operators
	OpAny FilterOperator = query.OpAny
	OpAll FilterOperator = query.OpAll

	// Type conversion functions
	OpCast FilterOperator = query.OpCast
	OpIsOf FilterOperator = query.OpIsOf

	// Geospatial functions
	OpGeoDistance   FilterOperator = query.OpGeoDistance
	OpGeoLength     FilterOperator = query.OpGeoLength
	OpGeoIntersects FilterOperator = query.OpGeoIntersects
)

// Logical operator constants
const (
	LogicalAnd LogicalOperator = query.LogicalAnd
	LogicalOr  LogicalOperator = query.LogicalOr
)

// Apply transformation type constants
const (
	ApplyTypeGroupBy   ApplyTransformationType = query.ApplyTypeGroupBy
	ApplyTypeAggregate ApplyTransformationType = query.ApplyTypeAggregate
	ApplyTypeFilter    ApplyTransformationType = query.ApplyTypeFilter
	ApplyTypeCompute   ApplyTransformationType = query.ApplyTypeCompute
	ApplyTypeJoin      ApplyTransformationType = query.ApplyTypeJoin
	ApplyTypeOuterJoin ApplyTransformationType = query.ApplyTypeOuterJoin
	ApplyTypeNest      ApplyTransformationType = query.ApplyTypeNest
	ApplyTypeFrom      ApplyTransformationType = query.ApplyTypeFrom
)

// Aggregation method constants
const (
	AggregationSum           AggregationMethod = query.AggregationSum
	AggregationAvg           AggregationMethod = query.AggregationAvg
	AggregationMin           AggregationMethod = query.AggregationMin
	AggregationMax           AggregationMethod = query.AggregationMax
	AggregationCount         AggregationMethod = query.AggregationCount
	AggregationCountDistinct AggregationMethod = query.AggregationCountDistinct
)

// ParseFilter parses an OData $filter string into a FilterExpression.
// This function performs parsing without metadata validation, allowing it to be used
// for building filter expressions programmatically without requiring entity metadata.
//
// For filters with metadata validation, use the internal parsing through ParseQueryOptions.
//
// Example:
//
//	filter, err := odata.ParseFilter("Name eq 'John'")
//	if err != nil {
//	    log.Fatal(err)
//	}
func ParseFilter(filterStr string) (*FilterExpression, error) {
	return query.ParseFilterWithoutMetadata(filterStr)
}

// GetQueryOptionsFromRequest retrieves the parsed OData query options from the
// HTTP request context. Call this inside an ActionHandler or FunctionHandler to
// access system query options such as $filter, $orderby, $top, and $skip that
// were included in the request URL.
//
// The framework automatically parses the query options before invoking the
// handler and stores them in the request context. Invalid OData system query
// options (e.g. malformed $top) cause the framework to return 400 Bad Request
// before the handler is called, so when this function is called from within a
// handler the result is always a valid, non-nil *QueryOptions.
//
// Returns nil only if called outside of a framework-managed handler invocation
// (e.g. in tests that construct a bare *http.Request without going through the
// operations handler).
//
// Example:
//
//	func myFunctionHandler(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
//	    opts := odata.GetQueryOptionsFromRequest(r)
//	    results, err := fetchResults()
//	    if err != nil {
//	        return nil, err
//	    }
//	    return odata.ApplyQueryOptionsToSlice(results, opts, nil)
//	}
func GetQueryOptionsFromRequest(r *http.Request) *QueryOptions {
	return actions.QueryOptionsFromRequest(r)
}

// NavigationBindingContext contains context about the parent navigation path
// when a bound action or function is invoked through navigation composition
// (e.g. /Categories(1)/Products/GetAveragePrice()).
//
// Use GetNavigationBindingContextFromRequest inside an ActionHandler or
// FunctionHandler to access this context.
type NavigationBindingContext = actions.NavigationBindingContext

// GetNavigationBindingContextFromRequest retrieves the parent navigation
// context from the HTTP request. Call this inside an ActionHandler or
// FunctionHandler to determine the parent entity set, parent key, and
// navigation property when the operation was invoked via navigation
// composition.
//
// Returns nil when the operation was not invoked through navigation
// composition (e.g. a direct call such as /Products/GetAveragePrice()).
//
// Example:
//
//	func myFunctionHandler(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
//	    navCtx := odata.GetNavigationBindingContextFromRequest(r)
//	    if navCtx != nil {
//	        // invoked via navigation: e.g. /Categories(1)/Products/GetAveragePrice()
//	        log.Printf("parent entity set: %s, key: %s, nav: %s",
//	            navCtx.ParentEntitySet, navCtx.ParentKey, navCtx.NavigationProperty)
//	    }
//	    // ... implement function logic
//	}
func GetNavigationBindingContextFromRequest(r *http.Request) *NavigationBindingContext {
	return actions.NavigationBindingContextFromRequest(r)
}
