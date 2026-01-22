package odata

import "github.com/nlstn/go-odata/internal/query"

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

	// Enum operator
	OpHas FilterOperator = query.OpHas

	// Arithmetic operators
	OpAdd FilterOperator = query.OpAdd
	OpSub FilterOperator = query.OpSub
	OpMul FilterOperator = query.OpMul
	OpDiv FilterOperator = query.OpDiv
	OpMod FilterOperator = query.OpMod

	// Math functions
	OpCeiling FilterOperator = query.OpCeiling
	OpFloor   FilterOperator = query.OpFloor
	OpRound   FilterOperator = query.OpRound

	// Date functions
	OpYear   FilterOperator = query.OpYear
	OpMonth  FilterOperator = query.OpMonth
	OpDay    FilterOperator = query.OpDay
	OpHour   FilterOperator = query.OpHour
	OpMinute FilterOperator = query.OpMinute
	OpSecond FilterOperator = query.OpSecond
	OpDate   FilterOperator = query.OpDate
	OpTime   FilterOperator = query.OpTime
	OpNow    FilterOperator = query.OpNow

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
