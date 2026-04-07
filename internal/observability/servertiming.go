package observability

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

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

// serverTimingKey is the context key for the server-timing accumulator.
type serverTimingKey struct{}

// metricEntry stores a single metric's description and duration.
type metricEntry struct {
	desc     string
	duration time.Duration
}

// serverTimingAccumulator aggregates timing metrics for a request.
type serverTimingAccumulator struct {
	mu      sync.Mutex
	metrics map[string]metricEntry
}

// add adds or accumulates a duration for a named metric.
func (a *serverTimingAccumulator) add(name, desc string, d time.Duration) {
	if a == nil {
		return
	}
	a.mu.Lock()
	if a.metrics == nil {
		a.metrics = make(map[string]metricEntry)
	}
	e, ok := a.metrics[name]
	if ok {
		e.duration += d
		if e.desc == "" {
			e.desc = desc
		}
	} else {
		e = metricEntry{desc: desc, duration: d}
	}
	a.metrics[name] = e
	a.mu.Unlock()
}

// set sets the duration for a named metric (overwrite).
func (a *serverTimingAccumulator) set(name, desc string, d time.Duration) {
	if a == nil {
		return
	}
	a.mu.Lock()
	if a.metrics == nil {
		a.metrics = make(map[string]metricEntry)
	}
	a.metrics[name] = metricEntry{desc: desc, duration: d}
	a.mu.Unlock()
}

// snapshot returns a copy of metrics for safe iteration.
func (a *serverTimingAccumulator) snapshot() map[string]metricEntry {
	out := make(map[string]metricEntry)
	if a == nil {
		return out
	}
	a.mu.Lock()
	for k, v := range a.metrics {
		out[k] = v
	}
	a.mu.Unlock()
	return out
}

// WithServerTimingAccumulator returns a new context with an empty server-timing accumulator.
func WithServerTimingAccumulator(ctx context.Context) context.Context {
	return context.WithValue(ctx, serverTimingKey{}, &serverTimingAccumulator{})
}

// serverTimingAccumulatorFromContext retrieves the timing accumulator from the context.
func serverTimingAccumulatorFromContext(ctx context.Context) *serverTimingAccumulator {
	val := ctx.Value(serverTimingKey{})
	if val == nil {
		return nil
	}
	acc, ok := val.(*serverTimingAccumulator)
	if !ok {
		return nil
	}
	return acc
}

// ServerTimingMetric is a handle for a running timing metric.
type ServerTimingMetric struct {
	name    string
	desc    string
	start   time.Time
	ctx     context.Context
	stopped bool
}

// Stop finishes the timing metric and records it into the context accumulator.
func (m *ServerTimingMetric) Stop() {
	if m == nil || m.stopped {
		return
	}
	m.stopped = true
	if m.ctx == nil {
		return
	}
	d := time.Since(m.start)
	if acc := serverTimingAccumulatorFromContext(m.ctx); acc != nil {
		acc.add(m.name, m.desc, d)
	}
}

// StartServerTiming starts a server-timing metric with the given name.
// Returns a metric that should be stopped when the timed operation completes.
// If server timing is not enabled or the context doesn't contain timing info, returns a no-op metric.
func StartServerTiming(ctx context.Context, name string) *ServerTimingMetric {
	return StartServerTimingWithDesc(ctx, name, "")
}

// StartServerTimingWithDesc starts a server-timing metric with the given name and description.
func StartServerTimingWithDesc(ctx context.Context, name, description string) *ServerTimingMetric {
	if serverTimingAccumulatorFromContext(ctx) == nil {
		return &ServerTimingMetric{ctx: ctx, stopped: true}
	}
	return &ServerTimingMetric{name: name, desc: description, start: time.Now(), ctx: ctx}
}

// SetServerTimingMetricDuration sets (overwrites) the duration for a named metric in the context.
func SetServerTimingMetricDuration(ctx context.Context, name string, d time.Duration, description string) {
	if acc := serverTimingAccumulatorFromContext(ctx); acc != nil {
		acc.set(name, description, d)
	}
}

// ServerTimingHeader builds the Server-Timing header value from the accumulator in the context.
// It returns an empty string if there is no accumulator.
func ServerTimingHeader(ctx context.Context) string {
	acc := serverTimingAccumulatorFromContext(ctx)
	if acc == nil {
		return ""
	}
	metrics := acc.snapshot()
	if len(metrics) == 0 {
		return ""
	}

	// Sort keys for deterministic header order
	keys := make([]string, 0, len(metrics))
	for k := range metrics {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		e := metrics[k]
		// dur in milliseconds with 3 decimal places
		ms := float64(e.duration) / float64(time.Millisecond)
		if e.desc != "" {
			parts = append(parts, fmt.Sprintf("%s;desc=\"%s\";dur=%.3f", sanitizeName(k), sanitizeDesc(e.desc), ms))
		} else {
			parts = append(parts, fmt.Sprintf("%s;dur=%.3f", sanitizeName(k), ms))
		}
	}
	return strings.Join(parts, ", ")
}

func sanitizeName(n string) string {
	n = strings.ReplaceAll(n, " ", "_")
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1 // strip control chars
		}
		return r
	}, n)
}

func sanitizeDesc(d string) string {
	d = strings.ReplaceAll(d, `"`, "'")
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, d)
}