// Package observability provides OpenTelemetry-based instrumentation for the OData service.
//
// It supports distributed tracing, metrics collection, and enhanced structured logging.
//
// All observability features are opt-in. When not configured, no-op implementations
// are used with zero performance overhead.
package observability

import "go.opentelemetry.io/otel/attribute"

// Instrumentation identity constants
const (
	// TracerName is the instrumentation name for tracing.
	TracerName = "github.com/nlstn/go-odata"
	// MeterName is the instrumentation name for metrics.
	MeterName = "github.com/nlstn/go-odata"
)

// OData semantic attribute keys following OpenTelemetry conventions.
const (
	// Entity attributes
	AttrEntitySet   = "odata.entity_set"
	AttrEntityKey   = "odata.entity_key"
	AttrEntityType  = "odata.entity_type"
	AttrOperation   = "odata.operation"
	AttrIsSingleton = "odata.is_singleton"

	// Query option attributes
	AttrQueryFilter  = "odata.query.filter"
	AttrQueryExpand  = "odata.query.expand"
	AttrQuerySelect  = "odata.query.select"
	AttrQueryOrderBy = "odata.query.orderby"
	AttrQueryTop     = "odata.query.top"
	AttrQuerySkip    = "odata.query.skip"
	AttrQueryCount   = "odata.query.count"
	AttrQuerySearch  = "odata.query.search"
	AttrQueryApply   = "odata.query.apply"

	// Result attributes
	AttrResultCount = "odata.result.count"
	AttrHasNextLink = "odata.has_next_link"
	AttrDeltaToken  = "odata.delta_token"

	// Batch attributes
	AttrBatchSize        = "odata.batch.size"
	AttrChangesetSize    = "odata.changeset.size"
	AttrBatchRequestID   = "odata.batch.request_id"
	AttrChangesetSuccess = "odata.changeset.success"

	// Async attributes
	AttrAsyncJobID     = "odata.async.job_id"
	AttrAsyncJobStatus = "odata.async.job_status"

	// Error attributes
	AttrErrorCode    = "odata.error.code"
	AttrErrorMessage = "odata.error.message"

	// Action/Function attributes
	AttrActionName   = "odata.action.name"
	AttrFunctionName = "odata.function.name"
	AttrIsBound      = "odata.is_bound"
)

// Operation types for the odata.operation attribute.
const (
	OpReadCollection = "read_collection"
	OpReadEntity     = "read_entity"
	OpCreate         = "create"
	OpUpdate         = "update"
	OpPatch          = "patch"
	OpDelete         = "delete"
	OpCount          = "count"
	OpBatch          = "batch"
	OpChangeset      = "changeset"
	OpMetadata       = "metadata"
	OpServiceDoc     = "service_document"
	OpAction         = "action"
	OpFunction       = "function"
	OpDelta          = "delta"
	OpRef            = "ref"
)

// Log field keys for structured logging with trace context.
const (
	LogFieldEntitySet   = "odata.entity_set"
	LogFieldEntityKey   = "odata.entity_key"
	LogFieldOperation   = "odata.operation"
	LogFieldTraceID     = "trace_id"
	LogFieldSpanID      = "span_id"
	LogFieldRequestID   = "request_id"
	LogFieldDuration    = "duration_ms"
	LogFieldResultCount = "result_count"
	LogFieldError       = "error"
)

// EntitySetAttr creates an attribute for the entity set name.
func EntitySetAttr(name string) attribute.KeyValue {
	return attribute.String(AttrEntitySet, name)
}

// EntityKeyAttr creates an attribute for the entity key.
func EntityKeyAttr(key string) attribute.KeyValue {
	return attribute.String(AttrEntityKey, key)
}

// OperationAttr creates an attribute for the operation type.
func OperationAttr(op string) attribute.KeyValue {
	return attribute.String(AttrOperation, op)
}

// ResultCountAttr creates an attribute for the result count.
func ResultCountAttr(count int64) attribute.KeyValue {
	return attribute.Int64(AttrResultCount, count)
}

// QueryFilterAttr creates an attribute for the $filter expression.
func QueryFilterAttr(filter string) attribute.KeyValue {
	return attribute.String(AttrQueryFilter, filter)
}

// QueryExpandAttr creates an attribute for the $expand expression.
func QueryExpandAttr(expand string) attribute.KeyValue {
	return attribute.String(AttrQueryExpand, expand)
}

// QuerySelectAttr creates an attribute for the $select expression.
func QuerySelectAttr(selectExpr string) attribute.KeyValue {
	return attribute.String(AttrQuerySelect, selectExpr)
}

// QueryOrderByAttr creates an attribute for the $orderby expression.
func QueryOrderByAttr(orderby string) attribute.KeyValue {
	return attribute.String(AttrQueryOrderBy, orderby)
}

// QueryTopAttr creates an attribute for the $top value.
func QueryTopAttr(top int) attribute.KeyValue {
	return attribute.Int(AttrQueryTop, top)
}

// QuerySkipAttr creates an attribute for the $skip value.
func QuerySkipAttr(skip int) attribute.KeyValue {
	return attribute.Int(AttrQuerySkip, skip)
}

// BatchSizeAttr creates an attribute for the batch size.
func BatchSizeAttr(size int) attribute.KeyValue {
	return attribute.Int(AttrBatchSize, size)
}

// ChangesetSizeAttr creates an attribute for the changeset size.
func ChangesetSizeAttr(size int) attribute.KeyValue {
	return attribute.Int(AttrChangesetSize, size)
}

// AsyncJobIDAttr creates an attribute for the async job ID.
func AsyncJobIDAttr(jobID string) attribute.KeyValue {
	return attribute.String(AttrAsyncJobID, jobID)
}

// ErrorCodeAttr creates an attribute for the error code.
func ErrorCodeAttr(code string) attribute.KeyValue {
	return attribute.String(AttrErrorCode, code)
}
