package observability

import (
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// NewNoopTracer creates a tracer that does nothing.
func NewNoopTracer() *Tracer {
	return &Tracer{
		tracer:      tracenoop.NewTracerProvider().Tracer(""),
		serviceName: "",
	}
}

// NewNoopMetrics creates metrics that do nothing.
func NewNoopMetrics() *Metrics {
	meter := noop.NewMeterProvider().Meter("")
	m := &Metrics{}

	// Note: noop meter never returns errors, but we must check them to satisfy the linter.
	m.requestDuration, _ = meter.Float64Histogram("odata.request.duration")   //nolint:errcheck
	m.requestCount, _ = meter.Int64Counter("odata.request.count")             //nolint:errcheck
	m.resultCount, _ = meter.Int64Histogram("odata.result.count")             //nolint:errcheck
	m.dbQueryDuration, _ = meter.Float64Histogram("odata.db.query.duration") //nolint:errcheck
	m.batchSize, _ = meter.Int64Histogram("odata.batch.size")                 //nolint:errcheck
	m.errorCount, _ = meter.Int64Counter("odata.error.count")                 //nolint:errcheck

	return m
}

