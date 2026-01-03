package observability

import (
	"context"
	"sync"
	"time"

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

// dbTimeAccumulatorKey is the context key for the database time accumulator.
type dbTimeAccumulatorKey struct{}

// DBTimeAccumulator tracks total database time during a request.
// It is safe for concurrent use.
type DBTimeAccumulator struct {
	mu       sync.Mutex
	duration time.Duration
}

// Add adds a duration to the accumulator.
func (a *DBTimeAccumulator) Add(d time.Duration) {
	a.mu.Lock()
	a.duration += d
	a.mu.Unlock()
}

// Duration returns the total accumulated duration.
func (a *DBTimeAccumulator) Duration() time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.duration
}

// WithDBTimeAccumulator returns a new context with a database time accumulator.
func WithDBTimeAccumulator(ctx context.Context) context.Context {
	return context.WithValue(ctx, dbTimeAccumulatorKey{}, &DBTimeAccumulator{})
}

// DBTimeAccumulatorFromContext retrieves the database time accumulator from the context.
// Returns nil if no accumulator is present.
func DBTimeAccumulatorFromContext(ctx context.Context) *DBTimeAccumulator {
	val := ctx.Value(dbTimeAccumulatorKey{})
	if val == nil {
		return nil
	}
	acc, ok := val.(*DBTimeAccumulator)
	if !ok {
		return nil
	}
	return acc
}

// AddDBTime adds a database operation duration to the accumulator in the context.
// This is a no-op if the context does not contain an accumulator.
func AddDBTime(ctx context.Context, d time.Duration) {
	if acc := DBTimeAccumulatorFromContext(ctx); acc != nil {
		acc.Add(d)
	}
}
