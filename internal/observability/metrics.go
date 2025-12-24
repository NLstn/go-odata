package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the OData-specific metric instruments.
type Metrics struct {
	requestDuration metric.Float64Histogram
	requestCount    metric.Int64Counter
	resultCount     metric.Int64Histogram
	dbQueryDuration metric.Float64Histogram
	batchSize       metric.Int64Histogram
	errorCount      metric.Int64Counter
}

// NewMetrics creates a new Metrics instance with the given MeterProvider.
func NewMetrics(mp metric.MeterProvider) *Metrics {
	meter := mp.Meter(MeterName)
	m := &Metrics{}

	// Note: errors from meter instrument creation are unlikely in practice
	// and would only occur with invalid parameters. We use explicit checks
	// to satisfy the linter while continuing with partial metrics on error.
	var err error

	m.requestDuration, err = meter.Float64Histogram(
		"odata.request.duration",
		metric.WithDescription("Duration of OData requests in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		m.requestDuration, _ = meter.Float64Histogram("odata.request.duration")
	}

	m.requestCount, err = meter.Int64Counter(
		"odata.request.count",
		metric.WithDescription("Total number of OData requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		m.requestCount, _ = meter.Int64Counter("odata.request.count")
	}

	m.resultCount, err = meter.Int64Histogram(
		"odata.result.count",
		metric.WithDescription("Number of entities returned in collection queries"),
		metric.WithUnit("{entity}"),
	)
	if err != nil {
		m.resultCount, _ = meter.Int64Histogram("odata.result.count")
	}

	m.dbQueryDuration, err = meter.Float64Histogram(
		"odata.db.query.duration",
		metric.WithDescription("Duration of database queries in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		m.dbQueryDuration, _ = meter.Float64Histogram("odata.db.query.duration")
	}

	m.batchSize, err = meter.Int64Histogram(
		"odata.batch.size",
		metric.WithDescription("Number of requests in a batch operation"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		m.batchSize, _ = meter.Int64Histogram("odata.batch.size")
	}

	m.errorCount, err = meter.Int64Counter(
		"odata.error.count",
		metric.WithDescription("Total number of OData errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		m.errorCount, _ = meter.Int64Counter("odata.error.count")
	}

	return m
}

// RecordRequest records metrics for a completed request.
func (m *Metrics) RecordRequest(ctx context.Context, entitySet, operation string, statusCode int, duration time.Duration) {
	attrs := metric.WithAttributes(
		EntitySetAttr(entitySet),
		OperationAttr(operation),
		attribute.Int("http.status_code", statusCode),
	)
	m.requestDuration.Record(ctx, float64(duration.Milliseconds()), attrs)
	m.requestCount.Add(ctx, 1, attrs)
}

// RecordResultCount records the number of entities returned in a collection query.
func (m *Metrics) RecordResultCount(ctx context.Context, entitySet string, count int64) {
	attrs := metric.WithAttributes(EntitySetAttr(entitySet))
	m.resultCount.Record(ctx, count, attrs)
}

// RecordDBQuery records metrics for a database query.
func (m *Metrics) RecordDBQuery(ctx context.Context, operation string, duration time.Duration) {
	attrs := metric.WithAttributes(attribute.String("db.operation", operation))
	m.dbQueryDuration.Record(ctx, float64(duration.Milliseconds()), attrs)
}

// RecordBatchSize records the size of a batch request.
func (m *Metrics) RecordBatchSize(ctx context.Context, size int) {
	m.batchSize.Record(ctx, int64(size))
}

// RecordError records an error occurrence.
func (m *Metrics) RecordError(ctx context.Context, entitySet, operation, errorType string) {
	attrs := metric.WithAttributes(
		EntitySetAttr(entitySet),
		OperationAttr(operation),
		attribute.String("error.type", errorType),
	)
	m.errorCount.Add(ctx, 1, attrs)
}

// RecordRequestStart records the start of a request (for tracking active requests).
func (m *Metrics) RecordRequestStart(ctx context.Context) {
	// No-op for now - could be used with an UpDownCounter for active requests
}

// RecordRequestEnd records the end of a request.
func (m *Metrics) RecordRequestEnd(ctx context.Context, duration time.Duration, statusCode int) {
	// General request metrics (without entity set context)
	attrs := metric.WithAttributes(
		attribute.Int("http.status_code", statusCode),
	)
	m.requestDuration.Record(ctx, float64(duration.Milliseconds()), attrs)
}

// RecordRequestDuration records duration for a specific entity operation.
func (m *Metrics) RecordRequestDuration(ctx context.Context, duration time.Duration, entitySet, operation string, statusCode int) {
	attrs := metric.WithAttributes(
		EntitySetAttr(entitySet),
		OperationAttr(operation),
		attribute.Int("http.status_code", statusCode),
	)
	m.requestDuration.Record(ctx, float64(duration.Milliseconds()), attrs)
	m.requestCount.Add(ctx, 1, attrs)
}
