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
	m.requestDuration, _ = meter.Float64Histogram("odata.request.duration")
	m.requestCount, _ = meter.Int64Counter("odata.request.count")
	m.resultCount, _ = meter.Int64Histogram("odata.result.count")
	m.dbQueryDuration, _ = meter.Float64Histogram("odata.db.query.duration")
	m.batchSize, _ = meter.Int64Histogram("odata.batch.size")
	m.errorCount, _ = meter.Int64Counter("odata.error.count")

	return m
}
