# Memory Profiling Baseline Analysis

**Date:** 2026-01-23
**Platform:** Linux amd64
**CPU:** Intel Xeon Platinum 8581C @ 2.10GHz

## Executive Summary

This document provides baseline memory profiling results for the go-odata library. The analysis identifies memory allocation hotspots and prioritizes areas for optimization.

## Benchmark Results

### 1. Tokenizer Benchmarks

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Tokenizer_Simple | 396 | 328 | 3 |
| Tokenizer_Complex | 1,677 | 1,136 | 4 |
| Tokenizer_ManyTokens | 2,227 | 1,472 | 6 |
| Tokenizer_DateTimeLiteral | 1,390 | 944 | 12 |
| Tokenizer_String | 722 | 800 | 2 |
| Tokenizer_GUID | 795 | 568 | 6 |
| Tokenizer_NextToken | 490 | 264 | 2 |

### 2. Query Options Parsing Benchmarks

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| ParseQueryOptions_Simple | 3,316 | 1,980 | 24 |
| ParseQueryOptions_Complex | 10,663 | 5,099 | 85 |
| ParseQueryOptions_ManyConditions | 10,267 | 4,745 | 59 |
| ParseQueryOptions_WithNavigationPaths | 8,900 | 4,141 | 60 |
| ParseQueryOptions_ComplexNavigationPaths | 15,357 | 6,274 | 100 |

### 3. AST Parser Pooling Benchmarks

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| ASTParserPooling_Simple | 606 | 337 | 4 |
| ASTParserPooling_Complex | 3,480 | 1,264 | 13 |
| **ASTParserPooling_WithoutRelease** | **5,105** | **1,956** | **33** |
| ASTParserPooling_ManyLiterals | 3,073 | 2,081 | 18 |
| ASTParserPooling_ArithmeticExpression | 2,530 | 1,197 | 11 |

**Key Finding:** AST pooling provides ~32% speed improvement, ~35% memory reduction, and ~60% allocation reduction compared to non-pooled parsing.

### 4. Filter Expression Pool Benchmarks

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| FilterExpressionPool_Simple | 3,260 | 1,890 | 19 |
| FilterExpressionPool_Complex | 7,308 | 3,775 | 39 |
| FilterExpressionPool_ManyConditions | 10,678 | 4,745 | 59 |
| FilterExpressionPool_Navigation | 6,383 | 3,062 | 39 |

### 5. Metadata Handler Benchmarks

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| MetadataXML | 5,868 | 7,947 | 26 |
| MetadataJSON | 5,607 | 7,062 | 28 |
| MetadataXML_ConcurrentCacheHit | 3,853 | 7,819 | 26 |
| MetadataJSON_ConcurrentCacheHit | 3,510 | 6,999 | 28 |

---

## Memory Allocation Hotspots

### By Allocation Space (Memory Volume)

| Rank | Function | % of Total | Cumulative |
|------|----------|------------|------------|
| 1 | `newParserCache` | 17.88% | 17.88% |
| 2 | `NewTokenizer` | 14.14% | 32.02% |
| 3 | `errors.go init` (pre-defined errors) | 13.66% | 45.68% |
| 4 | `fmt.Errorf` | 10.01% | 55.69% |
| 5 | `Tokenizer.getToken` | 7.22% | 62.91% |
| 6 | `strings.genSplit` | 6.92% | 69.83% |
| 7 | `Tokenizer.TokenizeAll` | 5.56% | 75.39% |
| 8 | `mergeNavigationSelects` | 4.50% | 79.89% |

### By Allocation Count (GC Pressure)

| Rank | Function | % of Total | Impact |
|------|----------|------------|--------|
| 1 | `strings.genSplit` | 19.90% | High - from path splitting |
| 2 | `ResolveSingleEntityNavigationPath` | 13.08% | High - navigation resolution |
| 3 | `fmt.Errorf` | 10.43% | Medium - dynamic errors |
| 4 | `errors.go init` | 9.49% | Low - one-time at startup |
| 5 | `errors.New` | 8.13% | Medium - static errors |
| 6 | `newParserCache` | 5.53% | High - per-parse allocation |
| 7 | `strings.Builder.grow` | 4.42% | Medium - string building |
| 8 | `ASTParser.parseLiteral` | 3.71% | Medium - literal parsing |

---

## Priority Improvement Areas

### Priority 1: High Impact - Parser Cache Pooling

**Current State:**
- `newParserCache()` allocates a new map for every parse operation
- Accounts for ~18% of total memory allocation

**Recommendation:**
```go
var parserCachePool = sync.Pool{
    New: func() interface{} {
        return &parserCache{
            resolvedPaths: make(map[string]bool, defaultCacheCapacity),
        }
    },
}

func acquireParserCache() *parserCache {
    return parserCachePool.Get().(*parserCache)
}

func releaseParserCache(c *parserCache) {
    clear(c.resolvedPaths) // Go 1.21+
    parserCachePool.Put(c)
}
```

**Expected Impact:** ~15-18% reduction in allocation space

---

### Priority 2: High Impact - Tokenizer Pooling

**Current State:**
- `NewTokenizer()` allocates a new token buffer slice for every tokenization
- Accounts for ~14% of total memory allocation

**Recommendation:**
```go
var tokenizerPool = sync.Pool{
    New: func() interface{} {
        return &Tokenizer{
            tokenBuffer: make([]Token, 0, minTokenSliceCapacity),
        }
    },
}

func AcquireTokenizer(input string) *Tokenizer {
    t := tokenizerPool.Get().(*Tokenizer)
    t.input = input
    t.pos = 0
    t.tokenIndex = 0
    t.tokenBuffer = t.tokenBuffer[:0]
    if len(input) > 0 {
        t.ch = rune(input[0])
    }
    return t
}

func ReleaseTokenizer(t *Tokenizer) {
    t.input = ""
    tokenizerPool.Put(t)
}
```

**Expected Impact:** ~10-14% reduction in allocation space

---

### Priority 3: Medium Impact - Navigation Path Resolution Caching

**Current State:**
- `ResolveSingleEntityNavigationPath` uses `strings.Split` repeatedly
- `strings.genSplit` accounts for 20% of allocation count
- Same paths are resolved multiple times per query

**Recommendation:**
- Add a path resolution cache at the EntityMetadata level
- Cache resolved navigation segments to avoid repeated string splitting
- Consider using a faster path splitting approach for known delimiters

**Expected Impact:** ~10-15% reduction in allocation count

---

### Priority 4: Medium Impact - Error Handling Optimization

**Current State:**
- `fmt.Errorf` accounts for ~10% of memory allocations
- Many errors are dynamically created with variable interpolation

**Recommendation:**
- Expand pre-defined errors in `errors.go`
- Use error wrapping for context: `fmt.Errorf("context: %w", predefinedErr)`
- Consider error pooling for frequent error paths

**Current Coverage:**
```
errors.go: 61 pre-defined errors (good coverage)
```

**Remaining Opportunities:**
- `internal/query/helpers.go:95,100,104` - navigation path errors
- `internal/query/orderby_parser.go:27,39,48,51` - orderby validation errors
- `internal/metadata/analyzer.go` - multiple fmt.Errorf calls

---

### Priority 5: Low Impact - String Builder Optimization in Tokenizer

**Current State:**
- `readNumber()` creates a new `strings.Builder` per number
- `readDateLiteral()`, `readTimeLiteral()`, `readGUIDLiteral()` all allocate builders

**Recommendation:**
- Use substring slicing when possible (already done for `readIdentifier`)
- For numeric literals, track start/end positions and return `input[start:end]`

**Example Optimization:**
```go
func (t *Tokenizer) readNumber() string {
    start := t.pos
    // ... advance through number characters ...
    return t.input[start:t.pos] // Zero allocation
}
```

**Expected Impact:** ~3-5% reduction in allocation count

---

## Existing Optimizations (Already Implemented)

The codebase already includes several memory optimizations:

1. **AST Node Pooling** (`ast_pool.go`)
   - 9 different node types pooled via `sync.Pool`
   - ~35% allocation reduction demonstrated

2. **Filter Expression Pooling** (`filter_pool.go`)
   - Reuses `FilterExpression` structs
   - Includes recursive tree release

3. **Pre-defined Errors** (`errors.go`)
   - 61 common errors pre-allocated
   - Reduces `errors.New` allocations

4. **Token Buffer Reuse** (`tokenizer.go`)
   - Pre-allocated token buffer per tokenizer instance
   - Dynamic growth with 50% expansion

5. **Identifier Slicing** (`tokenizer.go:262`)
   - Returns substring slice instead of copying
   - Zero-allocation identifier reading

6. **Navigation Target Index** (`analyzer.go:1314`)
   - O(1) map lookup instead of O(n) iteration
   - Reduces repeated lookups

---

## Profiling Commands Reference

```bash
# Run benchmarks with memory profiling
go test -bench=. -benchmem ./internal/query/

# Generate memory profile
go test -bench=BenchmarkParseQueryOptions_Complex -memprofile=mem.prof ./internal/query/

# Analyze memory profile
go tool pprof -text -alloc_space mem.prof
go tool pprof -text -alloc_objects mem.prof

# Interactive analysis
go tool pprof -http=:8080 mem.prof

# CPU profiling with perfserver
go run ./cmd/perfserver/main.go -cpuprofile=cpu.prof
```

---

## Next Steps

1. Implement parser cache pooling (Priority 1)
2. Implement tokenizer pooling (Priority 2)
3. Add navigation path resolution caching (Priority 3)
4. Expand pre-defined errors (Priority 4)
5. Optimize tokenizer string building (Priority 5)
6. Re-run benchmarks after each optimization to measure impact
