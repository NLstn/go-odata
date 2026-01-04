package observability

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the observability configuration for the OData service.
type Config struct {
	// TracerProvider is the OpenTelemetry tracer provider.
	// If nil, tracing is disabled.
	TracerProvider trace.TracerProvider

	// MeterProvider is the OpenTelemetry meter provider.
	// If nil, metrics collection is disabled.
	MeterProvider metric.MeterProvider

	// ServiceName is used to identify this service in traces and metrics.
	ServiceName string

	// ServiceVersion is the version of this service.
	ServiceVersion string

	// EnableDetailedDBTracing enables tracing for individual database queries.
	// This adds overhead but provides detailed insight into query performance.
	EnableDetailedDBTracing bool

	// EnableQueryOptionTracing is reserved for future implementation.
	// When implemented, it will add query options ($filter, $select, etc.) as span attributes.
	EnableQueryOptionTracing bool

	// EnableServerTiming enables the Server-Timing HTTP response header.
	// When enabled, timing metrics are added to responses for debugging in browser dev tools.
	EnableServerTiming bool

	// tracer is the configured tracer instance.
	tracer *Tracer

	// metrics is the configured metrics instance.
	metrics *Metrics
}

// Option is a functional option for configuring observability.
type Option func(*Config)

// WithTracerProvider sets the tracer provider.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(c *Config) {
		c.TracerProvider = tp
	}
}

// WithMeterProvider sets the meter provider.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return func(c *Config) {
		c.MeterProvider = mp
	}
}

// WithServiceName sets the service name for identification.
func WithServiceName(name string) Option {
	return func(c *Config) {
		c.ServiceName = name
	}
}

// WithDetailedDBTracing enables detailed database query tracing.
func WithDetailedDBTracing() Option {
	return func(c *Config) {
		c.EnableDetailedDBTracing = true
	}
}

// WithQueryOptionTracing enables query option attributes on spans.
func WithQueryOptionTracing() Option {
	return func(c *Config) {
		c.EnableQueryOptionTracing = true
	}
}

// WithServerTiming enables the Server-Timing HTTP response header.
func WithServerTiming() Option {
	return func(c *Config) {
		c.EnableServerTiming = true
	}
}

// WithServiceVersion sets the service version for identification.
func WithServiceVersion(version string) Option {
	return func(c *Config) {
		c.ServiceVersion = version
	}
}

// WithLogger sets a logger for observability debug information.
// Note: This is currently unused but reserved for future use.
func WithLogger(_ interface{}) Option {
	return func(_ *Config) {
		// Reserved for future use
	}
}

// NewConfig creates a new observability configuration with the given options.
func NewConfig(opts ...Option) *Config {
	cfg := &Config{
		ServiceName: "odata-service",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// Initialize sets up the tracer and metrics based on configuration.
// This should be called after all options are set.
func (c *Config) Initialize() error {
	if c.TracerProvider != nil {
		c.tracer = NewTracer(c.TracerProvider, c.ServiceName)
	} else {
		c.tracer = NewNoopTracer()
	}

	if c.MeterProvider != nil {
		c.metrics = NewMetrics(c.MeterProvider)
	} else {
		c.metrics = NewNoopMetrics()
	}
	return nil
}

// Tracer returns the configured tracer, or a no-op tracer if not configured.
func (c *Config) Tracer() *Tracer {
	if c == nil || c.tracer == nil {
		return NewNoopTracer()
	}
	return c.tracer
}

// Metrics returns the configured metrics, or a no-op metrics if not configured.
func (c *Config) Metrics() *Metrics {
	if c == nil || c.metrics == nil {
		return NewNoopMetrics()
	}
	return c.metrics
}

// IsEnabled returns true if any observability features are configured.
func (c *Config) IsEnabled() bool {
	return c != nil && (c.TracerProvider != nil || c.MeterProvider != nil)
}

// ServerTimingEnabled returns true if Server-Timing header is enabled.
func (c *Config) ServerTimingEnabled() bool {
	return c != nil && c.EnableServerTiming
}
