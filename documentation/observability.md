# Observability

go-odata provides comprehensive observability support using [OpenTelemetry](https://opentelemetry.io/), the industry-standard open-source framework for distributed tracing, metrics, and logging. This allows you to monitor your OData service in production, debug performance issues, and gain insights into request patterns.

## Features

- **Distributed Tracing**: Track requests across your entire system with spans for each OData operation
- **Metrics Collection**: Gather request durations, counts, error rates, and more
- **Structured Logging**: Enhanced logging with trace context propagation
- **GORM Integration**: Optional detailed database query tracing
- **Zero Overhead**: All features are opt-in; disabled by default with no performance impact

## Quick Start

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/nlstn/go-odata"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func main() {
    ctx := context.Background()

    // Set up OTLP trace exporter (e.g., to Jaeger, Tempo, etc.)
    traceExporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint("localhost:4318"),
        otlptracehttp.WithInsecure(),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create trace provider
    tracerProvider := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(traceExporter),
    )
    defer tracerProvider.Shutdown(ctx)

    // Set up Prometheus metrics exporter
    prometheusExporter, err := prometheus.New()
    if err != nil {
        log.Fatal(err)
    }
    meterProvider := metric.NewMeterProvider(
        metric.WithReader(prometheusExporter),
    )

    // Create database connection
    db, err := gorm.Open(sqlite.Open("odata.db"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }

    // Create OData service
    service := odata.NewService(db)

    // Configure observability
    if err := service.SetObservability(odata.ObservabilityConfig{
        TracerProvider: tracerProvider,
        MeterProvider:  meterProvider,
        ServiceName:    "my-odata-api",
        ServiceVersion: "1.0.0",
    }); err != nil {
        log.Fatal(err)
    }

    // Register entities
    service.RegisterEntity(&Product{})

    // Start server
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", service))
}
```

## Configuration

### ObservabilityConfig

| Field | Type | Description |
|-------|------|-------------|
| `TracerProvider` | `trace.TracerProvider` | OpenTelemetry tracer provider. If nil, tracing is disabled. |
| `MeterProvider` | `metric.MeterProvider` | OpenTelemetry meter provider. If nil, metrics are disabled. |
| `ServiceName` | `string` | Service name for telemetry. Defaults to "odata-service". |
| `ServiceVersion` | `string` | Service version for telemetry attributes. |
| `EnableDetailedDBTracing` | `bool` | Enable per-query database spans. Can be verbose. |

### Minimal Configuration (Tracing Only)

```go
service.SetObservability(odata.ObservabilityConfig{
    TracerProvider: tracerProvider,
    ServiceName:    "my-api",
})
```

### Full Configuration

```go
service.SetObservability(odata.ObservabilityConfig{
    TracerProvider:           tracerProvider,
    MeterProvider:            meterProvider,
    ServiceName:              "my-odata-api",
    ServiceVersion:           "1.0.0",
    EnableDetailedDBTracing:  true,  // Enables per-query DB spans
})
```

## Tracing

### Span Hierarchy

When a request is processed, spans are created in a hierarchical structure:

```
odata.request                          # Root span for HTTP request
├── odata.read                         # Read entity or collection
│   └── db.query                       # Database query (if detailed DB tracing enabled)
├── odata.create                       # Create entity
├── odata.update                       # Update entity
├── odata.patch                        # Patch entity
├── odata.delete                       # Delete entity
└── odata.batch                        # Batch operation
    └── odata.changeset                # Changeset within batch
```

### Span Attributes

Each span includes relevant attributes following OpenTelemetry semantic conventions:

| Attribute | Description | Example |
|-----------|-------------|---------|
| `odata.entity_set` | Entity set name | `Products` |
| `odata.entity_key` | Entity key (for single entity operations) | `1` |
| `odata.operation` | Operation type | `read_collection`, `create`, `update`, `delete` |
| `odata.query.filter` | $filter expression | `Price gt 100` |
| `odata.query.expand` | $expand expression | `Category,Supplier` |
| `odata.query.select` | $select expression | `Name,Price` |
| `odata.query.top` | $top value | `10` |
| `odata.query.skip` | $skip value | `20` |
| `odata.result.count` | Number of results returned | `10` |
| `http.method` | HTTP method | `GET` |
| `http.status_code` | Response status code | `200` |
| `odata.batch.size` | Number of operations in batch | `5` |

### Operation Types

| Operation | Description |
|-----------|-------------|
| `read_collection` | GET on entity collection |
| `read_entity` | GET on single entity |
| `create` | POST to create entity |
| `update` | PUT to replace entity |
| `patch` | PATCH to update entity |
| `delete` | DELETE entity |
| `count` | $count operation |
| `batch` | $batch request |
| `changeset` | Changeset within batch |
| `metadata` | $metadata request |
| `service_document` | Service document request |
| `action` | Action invocation |
| `function` | Function invocation |

## Metrics

### Available Metrics

| Metric | Type | Unit | Description |
|--------|------|------|-------------|
| `odata.request.duration` | Histogram | ms | Request duration by entity set and operation |
| `odata.request.count` | Counter | 1 | Total requests by entity set, operation, and status |
| `odata.result.count` | Histogram | 1 | Number of entities returned per request |
| `odata.db.query.duration` | Histogram | ms | Database query duration (when detailed DB tracing is enabled) |
| `odata.batch.size` | Histogram | 1 | Number of operations per batch |
| `odata.error.count` | Counter | 1 | Errors by type and entity set |

### Prometheus Integration

Expose metrics for Prometheus scraping:

```go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
)

// Create Prometheus exporter
exporter, err := prometheus.New()
if err != nil {
    log.Fatal(err)
}
meterProvider := metric.NewMeterProvider(
    metric.WithReader(exporter),
)

// Configure OData service
service.SetObservability(odata.ObservabilityConfig{
    MeterProvider: meterProvider,
})

// Expose /metrics endpoint
http.Handle("/metrics", promhttp.Handler())
http.Handle("/", service)
```

## Database Tracing

When `EnableDetailedDBTracing` is set to `true`, GORM callbacks are registered to trace database operations:

```go
service.SetObservability(odata.ObservabilityConfig{
    TracerProvider:          tracerProvider,
    EnableDetailedDBTracing: true,
})
```

This adds spans for:
- SELECT queries
- INSERT operations
- UPDATE operations
- DELETE operations

Each database span includes:
- `db.system`: Database system ("gorm")
- `db.sql.table`: Table name (when available)
- `db.rows_affected`: Number of rows affected

**Note**: Detailed DB tracing can generate significant trace data. Use it judiciously in production.

## Logging Integration

Structured logging is automatically enhanced with trace context when observability is configured:

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

service.SetLogger(logger)
service.SetObservability(odata.ObservabilityConfig{
    TracerProvider: tracerProvider,
})
```

Log entries will include `trace_id` and `span_id` fields, allowing correlation between logs and traces:

```json
{
    "time": "2024-01-15T10:30:00Z",
    "level": "INFO",
    "msg": "Entity operation completed",
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
    "span_id": "00f067aa0ba902b7",
    "odata.entity_set": "Products",
    "odata.operation": "read_collection"
}
```

## Common Integrations

### Jaeger

```go
import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

exporter, _ := otlptracehttp.New(ctx,
    otlptracehttp.WithEndpoint("jaeger:4318"),
    otlptracehttp.WithInsecure(),
)
tracerProvider := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
)
```

### Grafana Tempo

```go
import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

exporter, _ := otlptracehttp.New(ctx,
    otlptracehttp.WithEndpoint("tempo:4318"),
    otlptracehttp.WithInsecure(),
)
tracerProvider := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
)
```

### Datadog

```go
import "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentelemetry"

provider := opentelemetry.NewTracerProvider()
```

### AWS X-Ray

```go
import (
    "go.opentelemetry.io/contrib/propagators/aws/xray"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

exporter, _ := otlptracegrpc.New(ctx)
tracerProvider := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
    sdktrace.WithIDGenerator(xray.NewIDGenerator()),
)
```

## Best Practices

### 1. Use Sampling in Production

For high-traffic services, use sampling to reduce trace volume:

```go
tracerProvider := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
    sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)), // Sample 10%
)
```

### 2. Set Meaningful Service Names

Use descriptive service names that identify the service in your infrastructure:

```go
service.SetObservability(odata.ObservabilityConfig{
    ServiceName:    "inventory-api-v2",
    ServiceVersion: "2.1.0",
})
```

### 3. Enable Detailed Tracing Selectively

Enable detailed DB tracing only when debugging specific issues:

```go
// In development or when debugging
service.SetObservability(odata.ObservabilityConfig{
    EnableDetailedDBTracing:  os.Getenv("ENABLE_DB_TRACING") == "true",
})
```

### 4. Correlate with External Services

When calling external services from hooks, propagate the trace context:

```go
func (p *Product) ODataAfterCreate(ctx context.Context, r *http.Request) error {
    // Create HTTP client with trace propagation
    client := &http.Client{
        Transport: otelhttp.NewTransport(http.DefaultTransport),
    }
    
    req, _ := http.NewRequestWithContext(ctx, "POST", "https://inventory.example.com/notify", nil)
    _, err := client.Do(req)
    return err
}
```

### 5. Add Custom Spans

For complex business logic, add custom spans:

```go
import "go.opentelemetry.io/otel"

func (p *Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
    tracer := otel.Tracer("my-app")
    ctx, span := tracer.Start(ctx, "validate-inventory")
    defer span.End()
    
    // Validation logic...
    return nil
}
```

## Troubleshooting

### No Traces Appearing

1. Verify the tracer provider is correctly configured
2. Check that the exporter endpoint is reachable
3. Ensure the service name is set
4. Check for errors in trace exporter initialization

### High Cardinality Metrics

If metrics have too many unique label combinations:
- Reduce the number of custom attributes
- Use appropriate bucket boundaries for histograms
- Consider using exemplars instead of high-cardinality labels

### Performance Impact

If observability causes performance degradation:
- Disable detailed DB tracing in production
- Increase sampling rate
- Use async exporters (default with batching)
- Review span creation frequency

## API Reference

### Types

```go
// ObservabilityConfig configures observability features for the service.
type ObservabilityConfig struct {
    TracerProvider           trace.TracerProvider
    MeterProvider            metric.MeterProvider
    ServiceName              string
    ServiceVersion           string
    EnableDetailedDBTracing  bool
}
```

### Methods

```go
// SetObservability configures OpenTelemetry-based observability.
func (s *Service) SetObservability(cfg ObservabilityConfig) error

// Observability returns the current observability configuration.
func (s *Service) Observability() *observability.Config
```

## See Also

- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/instrumentation/go/)
- [OData v4 Specification](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html)
- [Prometheus Go Client](https://prometheus.io/docs/guides/go-application/)
- [Jaeger](https://www.jaegertracing.io/)
