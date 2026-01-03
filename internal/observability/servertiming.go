package observability

import (
	"context"

	servertiming "github.com/mitchellh/go-server-timing"
)

// ServerTimingMetric wraps the server-timing library's Metric type.
type ServerTimingMetric struct {
	metric *servertiming.Metric
}

// Stop stops the timing metric.
func (m *ServerTimingMetric) Stop() {
	if m != nil && m.metric != nil {
		m.metric.Stop()
	}
}

// StartServerTiming starts a server-timing metric with the given name.
// Returns a metric that should be stopped when the timed operation completes.
// If server timing is not enabled or the context doesn't contain timing info, returns a no-op metric.
func StartServerTiming(ctx context.Context, name string) *ServerTimingMetric {
	timing := servertiming.FromContext(ctx)
	if timing == nil {
		return &ServerTimingMetric{}
	}

	return &ServerTimingMetric{
		metric: timing.NewMetric(name).Start(),
	}
}

// StartServerTimingWithDesc starts a server-timing metric with the given name and description.
// Returns a metric that should be stopped when the timed operation completes.
// If server timing is not enabled or the context doesn't contain timing info, returns a no-op metric.
func StartServerTimingWithDesc(ctx context.Context, name, description string) *ServerTimingMetric {
	timing := servertiming.FromContext(ctx)
	if timing == nil {
		return &ServerTimingMetric{}
	}

	return &ServerTimingMetric{
		metric: timing.NewMetric(name).WithDesc(description).Start(),
	}
}
