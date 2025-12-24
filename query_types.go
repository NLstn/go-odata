package odata

import "github.com/nlstn/go-odata/internal/query"

// QueryOptions re-exports the parsed OData query options type for external consumers.
type QueryOptions = query.QueryOptions

// FilterExpression re-exports the parsed $filter expression type for external consumers.
type FilterExpression = query.FilterExpression

// OrderByItem re-exports the parsed $orderby item type for external consumers.
type OrderByItem = query.OrderByItem

// FilterOperator re-exports supported filter operators for external consumers.
type FilterOperator = query.FilterOperator

// LogicalOperator re-exports supported logical operators for external consumers.
type LogicalOperator = query.LogicalOperator
