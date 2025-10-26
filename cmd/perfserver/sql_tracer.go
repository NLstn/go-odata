package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm/logger"
)

// SQLTracer is a custom GORM logger that tracks query performance and patterns
// to help identify optimization opportunities
type SQLTracer struct {
	logger.Config
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
	mu                                  sync.Mutex
	queries                             map[string]*QueryStats
	slowThreshold                       time.Duration
	enableExplain                       bool
}

// QueryStats tracks statistics for a specific query pattern
type QueryStats struct {
	Pattern       string
	Count         int
	TotalDuration time.Duration
	MaxDuration   time.Duration
	MinDuration   time.Duration
	FirstSeen     time.Time
	LastSeen      time.Time
	Example       string // Store one example of the full query
}

// NewSQLTracer creates a new SQL tracer with optimization insights
func NewSQLTracer(slowThreshold time.Duration, enableExplain bool) *SQLTracer {
	return &SQLTracer{
		Config: logger.Config{
			SlowThreshold:             slowThreshold,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		},
		infoStr:       "[INFO] ",
		warnStr:       "[WARN] ",
		errStr:        "[ERROR] ",
		traceStr:      "[SQL] ",
		traceErrStr:   "[SQL ERROR] ",
		traceWarnStr:  "[SQL SLOW] ",
		queries:       make(map[string]*QueryStats),
		slowThreshold: slowThreshold,
		enableExplain: enableExplain,
	}
}

// LogMode sets the log level
func (l *SQLTracer) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := &SQLTracer{
		Config:        l.Config,
		infoStr:       l.infoStr,
		warnStr:       l.warnStr,
		errStr:        l.errStr,
		traceStr:      l.traceStr,
		traceErrStr:   l.traceErrStr,
		traceWarnStr:  l.traceWarnStr,
		queries:       l.queries, // Share the same query map
		slowThreshold: l.slowThreshold,
		enableExplain: l.enableExplain,
	}
	newLogger.LogLevel = level
	return newLogger
}

// Info prints info messages
func (l *SQLTracer) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		fmt.Printf(l.infoStr+msg+"\n", data...)
	}
}

// Warn prints warning messages
func (l *SQLTracer) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		fmt.Printf(l.warnStr+msg+"\n", data...)
	}
}

// Error prints error messages
func (l *SQLTracer) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		fmt.Printf(l.errStr+msg+"\n", data...)
	}
}

// Trace logs SQL queries with performance tracking
func (l *SQLTracer) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// Normalize the query to identify patterns (replace values with placeholders)
	pattern := l.normalizeQuery(sql)

	// Track statistics
	l.mu.Lock()
	stats, exists := l.queries[pattern]
	if !exists {
		stats = &QueryStats{
			Pattern:     pattern,
			Count:       0,
			FirstSeen:   begin,
			MinDuration: elapsed,
			Example:     sql,
		}
		l.queries[pattern] = stats
	}

	stats.Count++
	stats.TotalDuration += elapsed
	stats.LastSeen = time.Now()
	if elapsed > stats.MaxDuration {
		stats.MaxDuration = elapsed
	}
	if elapsed < stats.MinDuration {
		stats.MinDuration = elapsed
	}
	l.mu.Unlock()

	// Log the query
	switch {
	case err != nil && l.LogLevel >= logger.Error:
		fmt.Printf("%s[%.3fms] [rows:%d] %s | ERROR: %v\n",
			l.traceErrStr, float64(elapsed.Nanoseconds())/1e6, rows, sql, err)
	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.LogLevel >= logger.Warn:
		fmt.Printf("%s[%.3fms] [rows:%d] %s ⚠️  SLOW QUERY (>%.0fms)\n",
			l.traceWarnStr, float64(elapsed.Nanoseconds())/1e6, rows, sql, float64(l.slowThreshold.Milliseconds()))
	case l.LogLevel >= logger.Info:
		fmt.Printf("%s[%.3fms] [rows:%d] %s\n",
			l.traceStr, float64(elapsed.Nanoseconds())/1e6, rows, sql)
	}
}

// normalizeQuery converts a query to a pattern by replacing literal values
func (l *SQLTracer) normalizeQuery(sql string) string {
	// Remove extra whitespace
	pattern := regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(sql), " ")

	// Replace string literals
	pattern = regexp.MustCompile(`'[^']*'`).ReplaceAllString(pattern, "?")

	// Replace numeric literals (simple approach - replace all standalone numbers)
	// This will also replace LIMIT/OFFSET values, but that's okay for pattern matching
	pattern = regexp.MustCompile(`\b\d+\b`).ReplaceAllString(pattern, "?")

	// Replace IN clauses with multiple values
	pattern = regexp.MustCompile(`IN\s*\([^)]+\)`).ReplaceAllString(pattern, "IN (?)")

	return pattern
}

// PrintSummary outputs a comprehensive analysis of query patterns
func (l *SQLTracer) PrintSummary() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.queries) == 0 {
		fmt.Println("\n📊 SQL Query Analysis: No queries tracked")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 SQL QUERY OPTIMIZATION ANALYSIS")
	fmt.Println(strings.Repeat("=", 80))

	// Calculate totals
	totalQueries := 0
	totalDuration := time.Duration(0)
	for _, stats := range l.queries {
		totalQueries += stats.Count
		totalDuration += stats.TotalDuration
	}

	fmt.Printf("\n📈 Overall Statistics:\n")
	fmt.Printf("  Total Queries Executed: %d\n", totalQueries)
	fmt.Printf("  Unique Query Patterns:  %d\n", len(l.queries))
	fmt.Printf("  Total SQL Time:         %.3fms\n", float64(totalDuration.Nanoseconds())/1e6)
	fmt.Printf("  Average Query Time:     %.3fms\n", float64(totalDuration.Nanoseconds())/1e6/float64(totalQueries))

	// Sort queries by different metrics
	byTotalTime := l.getSortedStats(func(a, b *QueryStats) bool {
		return a.TotalDuration > b.TotalDuration
	})

	byCount := l.getSortedStats(func(a, b *QueryStats) bool {
		return a.Count > b.Count
	})

	bySlowest := l.getSortedStats(func(a, b *QueryStats) bool {
		return a.MaxDuration > b.MaxDuration
	})

	// Print top queries by total time
	fmt.Println("\n🔥 Top Queries by Total Time (Target for Optimization):")
	fmt.Println(strings.Repeat("-", 80))
	l.printTopStats(byTotalTime, 10)

	// Print queries with high execution count (potential N+1)
	fmt.Println("\n🔁 Queries with High Execution Count (Potential N+1 Problems):")
	fmt.Println(strings.Repeat("-", 80))
	nPlusOneQueries := 0
	for _, stats := range byCount {
		if stats.Count > 10 { // Threshold for N+1 detection
			nPlusOneQueries++
		}
	}
	if nPlusOneQueries > 0 {
		l.printTopStats(byCount[:min(nPlusOneQueries, 10)], 10)
		if nPlusOneQueries > 10 {
			fmt.Printf("\n  ... and %d more queries with >10 executions\n", nPlusOneQueries-10)
		}
	} else {
		fmt.Println("  ✓ No N+1 query problems detected (all queries executed ≤10 times)")
	}

	// Print slowest individual queries
	fmt.Println("\n🐌 Slowest Individual Query Executions:")
	fmt.Println(strings.Repeat("-", 80))
	l.printTopStats(bySlowest, 5)

	// Print optimization recommendations
	fmt.Println("\n💡 Optimization Recommendations:")
	fmt.Println(strings.Repeat("-", 80))
	recommendations := l.generateRecommendations(byTotalTime, byCount)
	if len(recommendations) == 0 {
		fmt.Println("  ✓ No major optimization opportunities detected")
	} else {
		for i, rec := range recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

// getSortedStats returns query stats sorted by the given comparison function
func (l *SQLTracer) getSortedStats(less func(a, b *QueryStats) bool) []*QueryStats {
	stats := make([]*QueryStats, 0, len(l.queries))
	for _, s := range l.queries {
		stats = append(stats, s)
	}
	sort.Slice(stats, func(i, j int) bool {
		return less(stats[i], stats[j])
	})
	return stats
}

// printTopStats prints the top N query statistics
func (l *SQLTracer) printTopStats(stats []*QueryStats, limit int) {
	count := min(len(stats), limit)
	for i := 0; i < count; i++ {
		s := stats[i]
		avgDuration := float64(s.TotalDuration.Nanoseconds()) / float64(s.Count) / 1e6

		fmt.Printf("\n  #%d: Executed %d times | Total: %.3fms | Avg: %.3fms | Max: %.3fms\n",
			i+1, s.Count,
			float64(s.TotalDuration.Nanoseconds())/1e6,
			avgDuration,
			float64(s.MaxDuration.Nanoseconds())/1e6)

		// Print pattern (truncate if too long)
		pattern := s.Pattern
		if len(pattern) > 150 {
			pattern = pattern[:147] + "..."
		}
		fmt.Printf("      %s\n", pattern)

		// Show example query for context
		if s.Example != s.Pattern && len(s.Example) < 200 {
			fmt.Printf("      Example: %s\n", s.Example)
		}
	}
}

// generateRecommendations creates optimization suggestions based on query patterns
func (l *SQLTracer) generateRecommendations(byTotalTime, byCount []*QueryStats) []string {
	var recommendations []string

	// Check for N+1 queries
	for _, stats := range byCount {
		if stats.Count > 50 {
			recommendations = append(recommendations,
				fmt.Sprintf("⚠️  N+1 Query Detected: Query executed %d times. Consider using eager loading or batch queries.", stats.Count))
			break // Only report the worst one
		}
	}

	// Check for slow queries
	for _, stats := range byTotalTime[:min(3, len(byTotalTime))] {
		avgTime := float64(stats.TotalDuration.Nanoseconds()) / float64(stats.Count) / 1e6
		if avgTime > 50 {
			// Try to identify the table
			table := l.extractTableName(stats.Pattern)
			if table != "" {
				recommendations = append(recommendations,
					fmt.Sprintf("🐌 Slow Query on '%s': Avg %.1fms. Consider adding indexes or optimizing WHERE clauses.", table, avgTime))
			} else {
				recommendations = append(recommendations,
					fmt.Sprintf("🐌 Slow Query: Avg %.1fms. Review query plan and consider adding indexes.", avgTime))
			}
		}
	}

	// Check for SELECT * queries
	for _, stats := range l.queries {
		if strings.Contains(stats.Pattern, "SELECT *") && stats.Count > 10 {
			recommendations = append(recommendations,
				fmt.Sprintf("📋 SELECT * Detected: Query executed %d times. Consider selecting only needed columns.", stats.Count))
			break // Only report once
		}
	}

	return recommendations
}

// extractTableName attempts to extract the main table name from a query
func (l *SQLTracer) extractTableName(query string) string {
	// Try to extract FROM clause
	fromRe := regexp.MustCompile(`FROM\s+["']?(\w+)["']?`)
	if matches := fromRe.FindStringSubmatch(query); len(matches) > 1 {
		return matches[1]
	}

	// Try to extract UPDATE clause
	updateRe := regexp.MustCompile(`UPDATE\s+["']?(\w+)["']?`)
	if matches := updateRe.FindStringSubmatch(query); len(matches) > 1 {
		return matches[1]
	}

	// Try to extract INSERT INTO clause
	insertRe := regexp.MustCompile(`INSERT INTO\s+["']?(\w+)["']?`)
	if matches := insertRe.FindStringSubmatch(query); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// ExportToFile writes detailed query statistics to a file
func (l *SQLTracer) ExportToFile(filename string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if _, err = fmt.Fprintf(f, "SQL Query Analysis Report\n"); err != nil {
		return err
	}
	if _, err = fmt.Fprintf(f, "Generated: %s\n\n", time.Now().Format(time.RFC3339)); err != nil {
		return err
	}

	byTotalTime := l.getSortedStats(func(a, b *QueryStats) bool {
		return a.TotalDuration > b.TotalDuration
	})

	for i, stats := range byTotalTime {
		avgDuration := float64(stats.TotalDuration.Nanoseconds()) / float64(stats.Count) / 1e6
		if _, err = fmt.Fprintf(f, "Query #%d:\n", i+1); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(f, "  Execution Count: %d\n", stats.Count); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(f, "  Total Time: %.3fms\n", float64(stats.TotalDuration.Nanoseconds())/1e6); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(f, "  Average Time: %.3fms\n", avgDuration); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(f, "  Min Time: %.3fms\n", float64(stats.MinDuration.Nanoseconds())/1e6); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(f, "  Max Time: %.3fms\n", float64(stats.MaxDuration.Nanoseconds())/1e6); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(f, "  Pattern: %s\n", stats.Pattern); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(f, "  Example: %s\n\n", stats.Example); err != nil {
			return err
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// WriteQueryLogHeader writes a header for the query log
func WriteQueryLogHeader() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🔍 SQL QUERY TRACE LOG")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Format: [SQL] [duration_ms] [rows_affected] query")
	fmt.Println("Slow queries (>100ms) will be marked with ⚠️")
	fmt.Println(strings.Repeat("=", 80) + "\n")
}
