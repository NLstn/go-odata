package observability

import (
	"context"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer wraps an OpenTelemetry tracer with OData-specific span creation methods.
type Tracer struct {
	tracer      trace.Tracer
	serviceName string
}

// NewTracer creates a new Tracer using the given TracerProvider.
func NewTracer(tp trace.TracerProvider, serviceName string) *Tracer {
	return &Tracer{
		tracer:      tp.Tracer(TracerName),
		serviceName: serviceName,
	}
}

// StartSpan starts a new span with the given name and attributes.
func (t *Tracer) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
	return ctx, span
}

// StartEntityRead starts a span for reading an entity or collection.
func (t *Tracer) StartEntityRead(ctx context.Context, entitySet string, key string, isSingleton bool) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		EntitySetAttr(entitySet),
		attribute.Bool(AttrIsSingleton, isSingleton),
	}
	if key != "" {
		attrs = append(attrs, EntityKeyAttr(key), OperationAttr(OpReadEntity))
	} else {
		attrs = append(attrs, OperationAttr(OpReadCollection))
	}
	return t.tracer.Start(ctx, "odata.read", trace.WithAttributes(attrs...))
}

// StartEntityCreate starts a span for creating an entity.
func (t *Tracer) StartEntityCreate(ctx context.Context, entitySet string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "odata.create", trace.WithAttributes(
		EntitySetAttr(entitySet),
		OperationAttr(OpCreate),
	))
}

// StartEntityUpdate starts a span for updating an entity.
func (t *Tracer) StartEntityUpdate(ctx context.Context, entitySet string, key string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "odata.update", trace.WithAttributes(
		EntitySetAttr(entitySet),
		EntityKeyAttr(key),
		OperationAttr(OpUpdate),
	))
}

// StartEntityPatch starts a span for patching an entity.
func (t *Tracer) StartEntityPatch(ctx context.Context, entitySet string, key string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "odata.patch", trace.WithAttributes(
		EntitySetAttr(entitySet),
		EntityKeyAttr(key),
		OperationAttr(OpPatch),
	))
}

// StartEntityDelete starts a span for deleting an entity.
func (t *Tracer) StartEntityDelete(ctx context.Context, entitySet string, key string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "odata.delete", trace.WithAttributes(
		EntitySetAttr(entitySet),
		EntityKeyAttr(key),
		OperationAttr(OpDelete),
	))
}

// StartBatch starts a span for a batch operation.
func (t *Tracer) StartBatch(ctx context.Context, requestCount int) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "odata.batch", trace.WithAttributes(
		OperationAttr(OpBatch),
		BatchSizeAttr(requestCount),
	))
}

// StartRequest starts a span for an HTTP request.
func (t *Tracer) StartRequest(ctx context.Context, r *http.Request) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "odata.request", trace.WithAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
		attribute.String("http.route", r.URL.Path),
	))
}

// SetHTTPStatus sets the HTTP status code on the current span.
func (t *Tracer) SetHTTPStatus(ctx context.Context, statusCode int) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int("http.status_code", statusCode))
	if statusCode >= 400 {
		span.SetStatus(codes.Error, http.StatusText(statusCode))
	}
}

// StartChangeset starts a span for a changeset within a batch.
func (t *Tracer) StartChangeset(ctx context.Context, changesetID string, requestCount int) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "odata.changeset", trace.WithAttributes(
		OperationAttr(OpChangeset),
		attribute.String(AttrBatchRequestID, changesetID),
		ChangesetSizeAttr(requestCount),
	))
}

// StartAction starts a span for an action invocation.
func (t *Tracer) StartAction(ctx context.Context, actionName string, entitySet string, isBound bool) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		OperationAttr(OpAction),
		attribute.String(AttrActionName, actionName),
		attribute.Bool(AttrIsBound, isBound),
	}
	if entitySet != "" {
		attrs = append(attrs, EntitySetAttr(entitySet))
	}
	return t.tracer.Start(ctx, "odata.action", trace.WithAttributes(attrs...))
}

// StartFunction starts a span for a function invocation.
func (t *Tracer) StartFunction(ctx context.Context, functionName string, entitySet string, isBound bool) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		OperationAttr(OpFunction),
		attribute.String(AttrFunctionName, functionName),
		attribute.Bool(AttrIsBound, isBound),
	}
	if entitySet != "" {
		attrs = append(attrs, EntitySetAttr(entitySet))
	}
	return t.tracer.Start(ctx, "odata.function", trace.WithAttributes(attrs...))
}

// StartDBQuery starts a span for a database query.
func (t *Tracer) StartDBQuery(ctx context.Context, operation string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "db.query", trace.WithAttributes(
		attribute.String("db.operation", operation),
	))
}

// RecordError records an error on the span.
func (t *Tracer) RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// AddQueryOptions adds query option attributes to a span (if enabled).
func (t *Tracer) AddQueryOptions(span trace.Span, filter, expand, selectOpt, orderby string, top, skip int) {
	var attrs []attribute.KeyValue
	if filter != "" {
		attrs = append(attrs, QueryFilterAttr(filter))
	}
	if expand != "" {
		attrs = append(attrs, QueryExpandAttr(expand))
	}
	if selectOpt != "" {
		attrs = append(attrs, QuerySelectAttr(selectOpt))
	}
	if orderby != "" {
		attrs = append(attrs, QueryOrderByAttr(orderby))
	}
	if top > 0 {
		attrs = append(attrs, QueryTopAttr(top))
	}
	if skip > 0 {
		attrs = append(attrs, QuerySkipAttr(skip))
	}
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}

// LoggerWithTrace returns a logger enriched with trace context.
func LoggerWithTrace(ctx context.Context, logger *slog.Logger) *slog.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return logger
	}
	return logger.With(
		slog.String(LogFieldTraceID, span.SpanContext().TraceID().String()),
		slog.String(LogFieldSpanID, span.SpanContext().SpanID().String()),
	)
}
