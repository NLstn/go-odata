# Performance Analysis Guide

This guide explains how to analyze performance bottlenecks in the go-odata library using the perfserver.

## Table of Contents

1. [Running Performance Tests](#running-performance-tests)
2. [Analyzing HTTP Metrics](#analyzing-http-metrics)
3. [CPU Profiling](#cpu-profiling)
4. [SQL Query Analysis](#sql-query-analysis)
5. [Combined Analysis](#combined-analysis)
6. [Common Bottlenecks](#common-bottlenecks)

---

## Running Performance Tests

### Basic Load Test (HTTP Metrics Only)

```bash
cd cmd/perfserver
./run_load_tests.sh --wrk -d 30s
```

This provides:
- Requests per second
- Latency distribution (50th, 75th, 90th, 99th percentile)
- Throughput (MB/sec)
- Error rates

**Use when:** You want to understand overall system capacity and response times.

### Load Test with CPU Profiling

```bash
./run_load_tests.sh --wrk -d 30s --cpu-profile
```

This generates `load-test-results/cpu.prof` showing:
- Where CPU time is spent
- Hot functions and call paths
- Memory allocation patterns

**Use when:** You need to identify CPU-intensive code paths.

### Load Test with SQL Tracing

```bash
./run_load_tests.sh --wrk -d 30s --sql-trace
```

This generates `load-test-results/sql-trace.txt` showing:
- All SQL queries executed
- Query execution times
- Slow queries (>100ms)
- N+1 query problems
- Optimization recommendations

**Use when:** You suspect database queries are the bottleneck.

### Full Profiling (CPU + SQL)

```bash
./run_load_tests.sh --wrk -d 30s --cpu-profile --sql-trace
```

**Use when:** You want comprehensive performance insights.

---

## Analyzing HTTP Metrics

After running tests, examine the results in `load-test-results/`:

### Key Files

- `summary.txt` - Test execution summary
- `wrk_*.txt` - Individual test results

### Interpreting Results

```
Running 30s test @ http://localhost:9091/Products
  10 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     2.00ms    1.99ms  23.85ms   85.10%
    Req/Sec     6.27k   767.00    41.01k    93.07%
  Latency Distribution
     50%    1.42ms    ← Half of requests complete in 1.42ms
     75%    2.97ms    ← 75% complete in under 2.97ms
     90%    4.75ms    ← 90% complete in under 4.75ms
     99%    8.59ms    ← 99% complete in under 8.59ms (tail latency)
  1,871,303 requests in 30.10s, 488.98MB read
Requests/sec: 62,169.18    ← Throughput
Transfer/sec:     16.25MB
```

### Performance Benchmarks

**Excellent:** 
- Latency p50 < 2ms
- Latency p99 < 10ms
- Throughput > 50k req/sec

**Good:**
- Latency p50 < 5ms
- Latency p99 < 20ms
- Throughput > 20k req/sec

**Needs Investigation:**
- Latency p50 > 10ms
- Latency p99 > 50ms
- Throughput < 10k req/sec

### Comparing Endpoints

Compare throughput across different query types:

```bash
# Extract requests/sec from all tests
grep "Requests/sec:" load-test-results/wrk_*.txt | sort -t: -k2 -n
```

**Look for:**
- Disproportionately slow endpoints (>50% slower than similar queries)
- High standard deviation (inconsistent performance)
- Error responses (Non-2xx or 3xx responses)

---

## CPU Profiling

### Analyzing CPU Profiles

After running with `--cpu-profile`, analyze the profile:

#### Interactive Analysis

```bash
go tool pprof load-test-results/cpu.prof
```

**Useful commands in pprof:**

```
(pprof) top10              # Top 10 functions by CPU time
(pprof) top10 -cum         # Top 10 by cumulative time
(pprof) list functionName  # Show source code for function
(pprof) web                # Generate visual call graph (requires graphviz)
(pprof) peek functionName  # Show callers and callees
```

#### Web UI (Recommended)

```bash
go tool pprof -http=:8080 load-test-results/cpu.prof
```

Open `http://localhost:8080` in browser. Features:
- **Flame graph** - Visual representation of call stacks
- **Graph view** - Call graph with percentages
- **Source view** - Line-by-line CPU usage
- **Top functions** - Sortable table

### What to Look For

1. **Hot Functions** - Functions using >5% CPU
2. **Unexpected calls** - Functions that shouldn't be in hot path
3. **Reflection overhead** - Excessive `reflect.*` calls
4. **JSON marshaling** - Check if `encoding/json` is dominating
5. **String operations** - Look for `strings.*`, `fmt.*`
6. **Memory allocations** - Functions with high `runtime.mallocgc`

### Example Analysis

```
(pprof) top10 -cum
      flat  flat%   sum%        cum   cum%
     0.05s  0.15%  0.15%     28.50s 86.14%  net/http.(*conn).serve
     0.12s  0.36%  0.51%     27.30s 82.51%  github.com/nlstn/go-odata.(*Service).ServeHTTP
     1.20s  3.63%  4.14%     22.10s 66.77%  github.com/nlstn/go-odata/internal/handlers.HandleEntitySet
     0.80s  2.42%  6.56%     18.50s 55.92%  github.com/nlstn/go-odata/internal/query.ApplyQueryOptions
     2.10s  6.35% 12.91%     12.40s 37.48%  gorm.io/gorm.(*DB).Find
```

**Interpretation:**
- HTTP serving is efficient (0.15% flat)
- Most time in OData request handling (86% cumulative)
- Query processing takes significant time (55%)
- GORM database operations are 37% of total

### Common CPU Bottlenecks

| Function Pattern | Likely Cause | Solution |
|-----------------|--------------|----------|
| `encoding/json.Marshal` | JSON serialization | Use streaming encoder, reduce payload |
| `reflect.Value.*` | Reflection overhead | Cache type information |
| `runtime.mallocgc` | Memory allocations | Reuse objects, reduce allocations |
| `strings.Builder.String` | String concatenation | Use pre-allocated buffers |
| `gorm.io/gorm.*` | Database operations | Add indexes, optimize queries |

---

## SQL Query Analysis

### Understanding the SQL Trace

The SQL trace file (`sql-trace.txt`) contains:

1. **Query Log** - All queries with timestamps
2. **Summary Statistics** - Per-query-pattern analysis
3. **Slow Queries** - Queries exceeding threshold (100ms)
4. **Recommendations** - Suggested optimizations

### Example SQL Trace Output

```
=== SQL QUERY TRACE ===
[2025-10-26 21:15:32.123] [2.3ms] SELECT * FROM "products" WHERE price > 100 ORDER BY price DESC LIMIT 20

...

=== SUMMARY ===
Total queries: 125,430
Unique patterns: 15
Total time: 287.5s
Average time: 2.29ms

Top 10 slowest query patterns:
1. SELECT * FROM products WHERE category_id = ? [avg: 5.2ms, count: 50,000]
2. SELECT * FROM product_descriptions WHERE product_id IN (...) [avg: 8.1ms, count: 25,000]

=== SLOW QUERIES (>100ms) ===
[SLOW] [245ms] SELECT * FROM products WHERE name LIKE '%search%' ORDER BY created_at

=== RECOMMENDATIONS ===
⚠️  N+1 Query Detected: product_descriptions queries (50,000 times)
   → Consider using eager loading with JOIN or preload
   
⚠️  Missing Index: products.category_id used in 50,000 queries
   → CREATE INDEX idx_products_category_id ON products(category_id)
```

### Key Metrics to Check

1. **Total Query Count**
   - High count relative to HTTP requests → N+1 problem
   - Should be ~1-3 queries per HTTP request

2. **Average Query Time**
   - < 1ms: Excellent (in-memory, indexed)
   - 1-5ms: Good (indexed lookups)
   - 5-10ms: Acceptable (small scans)
   - \> 10ms: Investigate (missing index, large scan)

3. **Slow Queries**
   - Any query > 100ms needs optimization
   - Look for missing indexes, full table scans

4. **Query Patterns**
   - Repeated identical queries → missing cache
   - Many queries for same entity → N+1 problem
   - Complex JOINs → consider denormalization

### Optimizing SQL Performance

#### Missing Indexes

```sql
-- Example: If trace shows slow lookups on category_id
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_descriptions_product_id ON product_descriptions(product_id);
```

#### N+1 Query Problems

**Before (N+1):**
```
SELECT * FROM products         -- 1 query
SELECT * FROM categories WHERE id = 1  -- N queries
SELECT * FROM categories WHERE id = 2
...
```

**After (Eager Loading):**
```
SELECT * FROM products
SELECT * FROM categories WHERE id IN (1,2,3,...)  -- 1 query
```

**In GORM:**
```go
db.Preload("Category").Find(&products)
```

#### Query Optimization

- Use `SELECT field1, field2` instead of `SELECT *`
- Add `LIMIT` to unbounded queries
- Use composite indexes for multi-column filters
- Avoid `LIKE '%term%'` (can't use index)

---

## Combined Analysis

### Workflow for Finding Bottlenecks

1. **Run basic load test** - Identify slow endpoints
   ```bash
   ./run_load_tests.sh --wrk -d 30s
   ```

2. **Check HTTP metrics** - Which queries are slow?
   ```bash
   grep "Requests/sec:" load-test-results/wrk_*.txt | sort -t: -k2 -n
   ```

3. **Run with SQL tracing** - Is database the bottleneck?
   ```bash
   ./run_load_tests.sh --wrk -d 30s --sql-trace
   cat load-test-results/sql-trace.txt
   ```

4. **If SQL is fast, run CPU profiling** - Where is CPU time spent?
   ```bash
   ./run_load_tests.sh --wrk -d 30s --cpu-profile
   go tool pprof -http=:8080 load-test-results/cpu.prof
   ```

5. **Optimize and re-test** - Verify improvements
   ```bash
   ./run_load_tests.sh --wrk -d 30s
   ```

### Example Investigation

**Problem:** `/Products?$filter=Price gt 100` is slow (20k req/sec vs 60k for simple queries)

**Step 1:** Check SQL trace
```
Query: SELECT * FROM products WHERE price > 100
Count: 50,000
Avg time: 0.8ms  ← SQL is fast!
```

**Step 2:** Check CPU profile
```
(pprof) top10
  2.1s  12% json.Marshal
  1.8s  10% reflect.Value.Field
  1.2s   7% query.ApplyFilter
```

**Conclusion:** JSON serialization is bottleneck, not database

**Solution:** Reduce response size, implement field selection

---

## Common Bottlenecks

### 1. JSON Serialization

**Symptoms:**
- `encoding/json.Marshal` high in CPU profile
- Response size correlates with latency

**Solutions:**
- Implement `$select` to reduce fields
- Use `$top` to limit results
- Consider alternative serializers (jsoniter)

### 2. N+1 Queries

**Symptoms:**
- Query count >> request count
- Many similar queries in SQL trace
- High cumulative database time

**Solutions:**
- Use `Preload()` for relationships
- Implement eager loading in `$expand`
- Batch queries where possible

### 3. Missing Database Indexes

**Symptoms:**
- Slow query times in SQL trace
- Full table scans
- Performance degrades with data size

**Solutions:**
- Add indexes on filter columns
- Add indexes on foreign keys
- Use composite indexes for multi-column filters

### 4. Reflection Overhead

**Symptoms:**
- High `reflect.*` in CPU profile
- Slowness in metadata operations

**Solutions:**
- Cache type information
- Use code generation
- Minimize runtime reflection

### 5. Memory Allocations

**Symptoms:**
- High `runtime.mallocgc` in CPU profile
- GC pressure

**Solutions:**
- Reuse objects/buffers
- Use sync.Pool for temporary objects
- Reduce string concatenation

### 6. Unbounded Queries

**Symptoms:**
- Wide variance in response times
- Memory spikes
- Timeout errors

**Solutions:**
- Implement default `$top` limit
- Add max page size enforcement
- Return error for unbounded queries

---

## Performance Testing Best Practices

### Test Configuration

- **Short tests (10-30s):** Quick feedback during development
- **Medium tests (1-5min):** Detect warm-up effects, cache behavior
- **Long tests (10-30min):** Identify memory leaks, degradation

### Load Patterns

- **Low concurrency (10 connections):** Identify per-request overhead
- **Medium concurrency (100 connections):** Typical web load
- **High concurrency (500+ connections):** Stress testing

### Database Considerations

- **SQLite in-memory:** Fast, but single-threaded, no network overhead
- **SQLite file:** Adds disk I/O, more realistic
- **PostgreSQL:** Production-like, parallel queries, connection pooling

### Metrics to Track

1. **Throughput** - Requests/sec
2. **Latency** - p50, p95, p99
3. **Error rate** - Non-2xx responses
4. **CPU usage** - System-wide and per-core
5. **Memory usage** - Heap size, GC frequency
6. **Database metrics** - Query time, connection pool

---

## Tools Reference

### Load Testing Tools

- **wrk** - Fast, scriptable HTTP benchmarking (recommended)
- **vegeta** - Constant rate load testing
- **k6** - Programmable load testing

### Profiling Tools

- **pprof** - CPU and memory profiling
- **trace** - Detailed execution trace
- **perf** - Linux system profiling
- **Flamegraph** - Visualization

### Installation

```bash
# wrk
sudo apt-get install wrk  # Debian/Ubuntu
brew install wrk          # macOS

# graphviz (for pprof web view)
sudo apt-get install graphviz

# Go profiling tools (included with Go)
go tool pprof
go tool trace
```

---

## Further Reading

- [Go pprof Documentation](https://pkg.go.dev/runtime/pprof)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [GORM Performance Tips](https://gorm.io/docs/performance.html)
- [Database Indexing Strategies](https://use-the-index-luke.com/)
