package observability

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPMiddleware returns an HTTP middleware that instruments requests with tracing.
// It uses otelhttp for automatic span propagation and HTTP semantic attributes.
func HTTPMiddleware(cfg *Config) func(http.Handler) http.Handler {
	if cfg == nil || cfg.TracerProvider == nil {
		// Return a passthrough middleware if tracing is not configured
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "odata.http",
			otelhttp.WithTracerProvider(cfg.TracerProvider),
			otelhttp.WithMeterProvider(cfg.MeterProvider),
		)
	}
}
