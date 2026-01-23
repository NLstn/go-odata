# go-odata Performance Analysis Results

**Analysis Date:** January 23, 2026
**Test Environment:** Linux 4.4.0, Intel Xeon Platinum 8581C @ 2.10GHz, 16 cores
**Database:** SQLite (file-based for load tests)
**Test Duration:** 10 seconds per endpoint with 4 threads and 50 connections

---

## Executive Summary

This deep performance analysis used the repository's existing performance tools (benchmarks, CPU profiling, SQL tracing, and wrk load testing) to identify optimization opportunities. The analysis revealed several key areas for improvement, with the most impactful being related to **memory allocations**, **JSON serialization**, **reflection overhead**, and **database connection handling**.

---

## Load Test Results

| Endpoint | Requests/sec | Avg Latency | p50 | p99 | Notes |
|----------|-------------|-------------|-----|-----|-------|
| Service Document (`/`) | 15,462 | 1.85ms | 1.57ms | 5.26ms | Excellent baseline |
| Metadata (`/$metadata`) | 13,899 | 2.09ms | 1.74ms | 6.95ms | Good - cached |
| Categories (100 items) | 2,868 | 19.10ms | 14.21ms | 65.78ms | Collection overhead visible |
| Products Top 100 | 2,386 | 21.32ms | 16.79ms | 66.96ms | Standard CRUD |
| Products Top 500 | 724 | 68.02ms | 60.67ms | 194.34ms | **Scales poorly with size** |
| Filter Query | 1,698 | 29.68ms | 26.51ms | 91.75ms | Filter parsing + execution |
| OrderBy Query | 1,700 | 29.52ms | 25.92ms | 91.79ms | Similar to filter |
| Pagination (skip 1000) | 2,105 | 24.61ms | 19.08ms | 82.86ms | Skip penalty visible |
| Select (3 fields) | 2,933 | 19.95ms | 13.36ms | 71.21ms | **23% faster than full entity** |
| Expand (Category) | 1,582 | 31.90ms | 27.07ms | 93.01ms | Navigation adds overhead |
| Complex Query | 1,350 | 36.51ms | 32.62ms | 96.72ms | Combined operations |
| Count with Filter | 3,504 | 18.38ms | 8.70ms | 69.35ms | Good - minimal serialization |
| Single Entity | 3,528 | 17.17ms | 9.16ms | 64.83ms | Good - single row |
| Singleton | 4,518 | 13.22ms | 7.46ms | 55.43ms | Excellent |
| Apply/Aggregate | 1,049 | 88.72ms | 38.05ms | **765.71ms** | **Needs optimization** |

### Key Observations:

1. **Serialization-light operations are fast**: Service doc, metadata, count, and singleton all perform well (>3,500 req/s)
2. **Collection size has major impact**: 500 items is 3.3x slower than 100 items
3. **Select optimization works**: Selecting 3 fields vs all is 23% faster
4. **Apply/Aggregate is the slowest**: p99 of 765ms indicates potential issues

---

## CPU Profile Analysis

### Top CPU Consumers (% of total CPU time)

```
53.76%  runtime.cgocall          - CGO calls to SQLite driver
18.30%  SQLiteRows.nextSyncLocked - Row iteration
13.98%  gorm.Scan                - GORM result scanning
11.40%  syscall.Syscall6         - System calls
11.06%  WriteODataCollection...  - Response serialization
 6.36%  encoding/json            - JSON encoding
 5.91%  runtime.scanobject       - GC scanning
 4.68%  runtime.mallocgc         - Memory allocation
 3.53%  OrderedMap.MarshalJSON   - Custom JSON marshaling
 3.12%  processStructEntityOrdered - Entity processing
```

### Detailed Hot Spots

#### 1. OrderedMap.MarshalJSON (53.47s cumulative, 3.53%)
**Location:** `internal/response/ordered_map.go:96-154`

```
Line 142: json.Marshal(om.values[key])  - 43.07s (80% of function time)
Line 152: make([]byte, buf.Len())       - 2.62s (copy overhead)
```

**Issue:** For each key in the ordered map, `json.Marshal` is called individually, creating many small allocations. The final result requires a copy from the pooled buffer.

#### 2. NewOrderedMapWithCapacity (11.16s cumulative, 0.74%)
**Location:** `internal/response/ordered_map.go:34-38`

```
Line 37: values: make(map[string]interface{}, capacity) - 8.13s
Line 36: keys: make([]string, 0, capacity)             - 1.84s
```

**Issue:** Every entity in a collection response creates a new OrderedMap. With 100+ entities per request, this adds up significantly.

#### 3. ETag Generation (6.12s cumulative, 0.4%)
**Location:** `internal/etag/etag.go:86-153`

```
Line 103: getFieldIndex(...) - 1.22s (field lookup)
Line 134: sha256.Sum256(...) - 1.36s (hashing)
Line 149: hex.EncodeToString - 2.01s (hex encoding)
```

**Issue:** SHA256 is cryptographically secure but slower than needed for ETags. Field lookup via reflection adds overhead.

#### 4. Key Segment Building (4.21s cumulative, 0.28%)
**Location:** `internal/response/navigation_links.go:320-332`

```
Line 327: entity.FieldByName(...) - 1.46s
Line 329: fmt.Sprintf("%v", ...)  - 2.59s
```

**Issue:** `fmt.Sprintf("%v")` is convenient but slow. Direct type conversion with `strconv` would be faster.

#### 5. Cache Mutex Contention (3.58s cumulative)
**Locations:** `internal/response/field_cache.go`

```
getCachedPropertyMetadataMap: 2.40s (mutex RLock: 1.12s)
getFieldInfos:                1.18s (mutex RLock: 0.67s)
```

**Issue:** Read locks are held during map lookups. Under high concurrency, this creates contention even for cache hits.

#### 6. Database Fetch Operations (487.80s cumulative, 32.22%)
**Location:** `internal/handlers/collection_read.go:237-313`

```
Line 287: db.Find(results).Error - 460.87s
```

**Issue:** Most time is spent in database operations, which is expected but could be optimized with connection pooling.

---

## Benchmark Results

### Query Parsing Benchmarks

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Tokenizer_Simple | 405.0 | 320 | 2 |
| Tokenizer_Complex | 1,745 | 1,120 | 2 |
| Tokenizer_ManyTokens | 2,331 | 1,440 | 2 |
| ParseQueryOptions_Simple | 2,831 | 790 | 19 |
| ParseQueryOptions_Complex | 10,447 | 3,197 | 82 |
| ParseQueryOptions_ManyConditions | 10,079 | 3,020 | 59 |
| ParseQueryOptions_ComplexNavigation | 14,765 | 4,135 | 98 |

**Observation:** Query parsing is efficient. Tokenizer has only 2 allocations even for complex queries. Full parsing has reasonable allocation counts.

### AST Pooling Benchmarks

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| ASTParserPooling_Simple | 620.8 | 328 | 3 |
| ASTParserPooling_Complex | 3,471 | 1,247 | 11 |
| ASTParserPooling_WithoutRelease | 4,769 | 1,937 | 31 |
| ASTParserPooling_ManyLiterals | 3,032 | 2,080 | 18 |

**Observation:** Object pooling is working - "WithoutRelease" shows 65% more allocations than the pooled version.

### Metadata Benchmarks

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| MetadataXML | 8,071 | 7,954 | 26 |
| MetadataJSON | 7,797 | 7,069 | 28 |
| MetadataXML_ConcurrentCacheHit | 5,066 | 7,822 | 26 |
| MetadataJSON_ConcurrentCacheHit | 4,876 | 7,003 | 28 |

**Observation:** Caching provides ~37% improvement. Good concurrent cache hit performance.

---

## Most Valuable Improvement Opportunities

Ranked by estimated impact (combination of CPU time, frequency, and implementation effort):

### 1. Pool OrderedMap Instances (HIGH IMPACT)
**Current:** Each entity creates a new OrderedMap with map allocation
**Improvement:** Use `sync.Pool` for OrderedMap instances
**Expected Gain:** 5-10% reduction in allocation overhead for collection responses
**Implementation:** Add pool for `*OrderedMap`, reset keys/values on release

```go
var orderedMapPool = sync.Pool{
    New: func() interface{} {
        return &OrderedMap{
            keys:   make([]string, 0, 16),
            values: make(map[string]interface{}, 16),
        }
    },
}

func (om *OrderedMap) Reset() {
    om.keys = om.keys[:0]
    for k := range om.values {
        delete(om.values, k)
    }
}
```

### 2. Streaming JSON Encoder for OrderedMap (HIGH IMPACT)
**Current:** json.Marshal called per value, then results concatenated
**Improvement:** Use `json.Encoder` with streaming output
**Expected Gain:** 30-40% improvement in MarshalJSON time
**Implementation:** Write directly to encoder instead of marshaling each value separately

```go
func (om *OrderedMap) MarshalJSON() ([]byte, error) {
    buf := bufferPool.Get().(*bytes.Buffer)
    buf.Reset()
    defer bufferPool.Put(buf)

    enc := json.NewEncoder(buf)
    buf.WriteByte('{')
    for i, key := range om.keys {
        if i > 0 {
            buf.WriteByte(',')
        }
        // Write key
        buf.WriteByte('"')
        buf.WriteString(key)
        buf.WriteString("\":")
        // Encode value directly
        enc.Encode(om.values[key])
        // Remove trailing newline from encoder
        buf.Truncate(buf.Len() - 1)
    }
    buf.WriteByte('}')
    return bytes.Clone(buf.Bytes()), nil
}
```

### 3. Replace fmt.Sprintf with strconv (MEDIUM IMPACT)
**Current:** `fmt.Sprintf("%v", keyFieldValue.Interface())` for key building
**Improvement:** Type-switch and use strconv directly
**Expected Gain:** 40-60% improvement in key segment building
**Implementation:**

```go
func formatKeyValue(v reflect.Value) string {
    switch v.Kind() {
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return strconv.FormatInt(v.Int(), 10)
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        return strconv.FormatUint(v.Uint(), 10)
    case reflect.String:
        return v.String()
    default:
        return fmt.Sprintf("%v", v.Interface())
    }
}
```

### 4. Use Faster Hash for ETags (MEDIUM IMPACT)
**Current:** SHA256 (cryptographic, slow)
**Improvement:** Use xxhash or FNV-1a (non-cryptographic, fast)
**Expected Gain:** 60-70% improvement in ETag generation
**Implementation:**

```go
import "github.com/cespare/xxhash/v2"

func generateETag(source string) string {
    h := xxhash.Sum64String(source)
    return fmt.Sprintf("W/\"%016x\"", h)
}
```

### 5. Reduce Cache Lock Contention (MEDIUM IMPACT)
**Current:** `sync.RWMutex` for all cache lookups
**Improvement:** Use `sync.Map` for read-heavy caches
**Expected Gain:** 20-30% improvement under high concurrency
**Implementation:** Replace `map + RWMutex` with `sync.Map`

### 6. Pre-compute Field Indices at Registration (MEDIUM IMPACT)
**Current:** `entity.FieldByName(keyProps[0].Name)` called per entity
**Improvement:** Store field index in metadata at entity registration time
**Expected Gain:** Eliminate reflection overhead for known fields
**Implementation:**

```go
type PropertyMetadata struct {
    Name       string
    FieldIndex int // Pre-computed at registration
}

// Then use: entity.Field(prop.FieldIndex) instead of FieldByName
```

### 7. Connection Pool Configuration for SQLite (LOW-MEDIUM IMPACT)
**Current:** Default connection pool settings
**Improvement:** Tune `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`
**Expected Gain:** Reduce connection open/close overhead
**Implementation:**

```go
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(25)
sqlDB.SetConnMaxLifetime(5 * time.Minute)
```

### 8. Avoid OrderedMap for Simple Responses (LOW IMPACT)
**Current:** All entity responses use OrderedMap for field ordering
**Improvement:** For structs with `json` tags, rely on struct field order
**Expected Gain:** Eliminate OrderedMap overhead for standard responses
**Implementation:** Use direct struct marshaling when custom ordering isn't needed

---

## SQL Query Patterns Observed

The SQL tracer identified these query patterns during load testing:

| Query Pattern | Description |
|--------------|-------------|
| `SELECT * FROM products LIMIT ?` | Standard pagination |
| `SELECT * FROM products WHERE price > ? LIMIT ?` | Filter with limit |
| `SELECT * FROM categories WHERE id IN (?)` | Batch loading for $expand |
| `SELECT count(*) FROM products WHERE ...` | Count queries |
| `SELECT id,name,price FROM products LIMIT ?` | Column projection ($select) |
| `SELECT ... GROUP BY ... aggregate(...)` | Apply transformations |

**Good News:** No N+1 query patterns detected. The `$expand` implementation correctly uses batch loading with `IN (?)` clauses.

---

## Recommendations Summary

### Quick Wins (1-2 hours each)
1. Replace `fmt.Sprintf` with `strconv` in key building
2. Configure SQLite connection pool
3. Use xxhash instead of SHA256 for ETags

### Medium Effort (4-8 hours each)
4. Pool OrderedMap instances
5. Replace `sync.RWMutex` with `sync.Map` for caches
6. Pre-compute field indices at registration

### Larger Refactoring (1-2 days)
7. Streaming JSON encoder for collections
8. Conditional OrderedMap usage (only when needed)

---

## Next Steps

1. Implement quick wins first for immediate gains
2. Run benchmarks after each change to measure improvement
3. Re-run load tests to validate real-world performance gains
4. Consider implementing the streaming JSON encoder for collections as it addresses the largest bottleneck

---

## Appendix: Running These Tests

```bash
# Run benchmarks
go test -bench=. -benchmem ./internal/query
go test -bench=. -benchmem ./internal/handlers

# Run load tests with profiling
cd cmd/perfserver
./run_load_tests.sh --cpu-profile --sql-trace -d 10s

# Analyze CPU profile
go tool pprof load-test-results/cpu.prof
go tool pprof -http=:8080 load-test-results/cpu.prof  # Interactive web UI
```
