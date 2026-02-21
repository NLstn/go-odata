# Performance Testing Server

This is the performance testing server for the go-odata library. It provides extensive test data and performance profiling capabilities.

## Features

- **Extensive Data Seeding**: Generates 10,000 products, 100 categories, and 30,000 product descriptions by default
- **CPU Profiling**: Built-in CPU profiling support for performance analysis
- **SQL Query Tracing**: Detailed SQL query tracking with optimization recommendations
- **Database Support**: Works with both SQLite and PostgreSQL

## Running the Server

### SQLite (In-Memory)
```bash
cd cmd/perfserver
go run . -db sqlite -dsn :memory:
```

### SQLite (File-Based)
```bash
cd cmd/perfserver
go run . -db sqlite -dsn perf.db
```

### PostgreSQL
```bash
cd cmd/perfserver
go run . -db postgres -dsn "postgresql://user:password@localhost:5432/dbname?sslmode=disable"
```

## Performance Profiling Options

### CPU Profiling
```bash
go run . -cpuprofile cpu.prof
```

Analyze the profile with:
```bash
go tool pprof cpu.prof
```

### SQL Query Tracing
```bash
go run . -trace-sql -trace-sql-file sql-trace.txt
```

This will:
- Log all SQL queries with execution time
- Highlight slow queries (>100ms)
- Generate optimization recommendations
- Detect N+1 query problems
- Export detailed analysis to a file

### Combined Profiling
```bash
go run . -cpuprofile cpu.prof -trace-sql -trace-sql-file sql-trace.txt
```

## Seeding Options

### Standard Seeding
```bash
go run . -extensive=false
```

Uses a small dataset similar to the compliance server (5 products, 3 categories).

### Extensive Seeding (Default)
```bash
go run . -extensive=true
```

Generates large datasets for realistic performance testing:
- 10,000 products with varied attributes
- 100 categories
- 30,000 product descriptions (3 languages per product)
- 1,000 API keys with server-generated UUID identifiers

The API key dataset is deliberately write-heavy to highlight server-managed key generation during benchmarking workloads.

## Endpoints

### OData Service
- `GET /` - Service document
- `GET /$metadata` - Metadata document
- `GET /Categories` - Category list
- `GET /Products` - Product list
- `GET /ProductDescriptions` - Product description list
- `GET /Company` - Company info singleton
- `GET /APIKeys` - API key list with server-generated identifiers

### Performance Testing
- `POST /Reseed` - Reset database to extensive performance testing state

## VS Code Integration

Use the tasks defined in `.vscode/tasks.json`:

### Server Tasks
- "Start Dev Server (SQLite)" - Launch dev server with SQLite in-memory database
- "Start Dev Server (PostgreSQL)" - Launch dev server with PostgreSQL database

### Load Testing Tasks
- "Run load tests (SQLite)" - Automated load tests with wrk
- "Run load tests (PostgreSQL)" - PostgreSQL load tests with wrk
- "Run load tests with CPU profiling (SQLite)" - With CPU profiling enabled
- "Run load tests with SQL tracing (SQLite)" - With SQL query tracing
- "Run load tests with full profiling (SQLite)" - With both CPU and SQL profiling

**Note:** Load testing tasks automatically start and stop the perfserver.

## Performance Testing Scenarios

### Automated Load Testing

Use the included load testing script to run comprehensive performance tests:

```bash
# Run all load tests with wrk (auto-starts perfserver)
./run_load_tests.sh

# Custom configuration
./run_load_tests.sh -d 60s -t 12 -C 200 -o ./my-results

# Use PostgreSQL
./run_load_tests.sh --db postgres --dsn "postgresql://user:pass@localhost/dbname"

# With CPU profiling
./run_load_tests.sh --cpu-profile

# With SQL tracing
./run_load_tests.sh --sql-trace

# Use external/already running server
./run_load_tests.sh --external-server
```

The script automatically:
- Builds and starts the perfserver
- Runs 38 different test scenarios (read + write)
- Saves detailed results to `./load-test-results/`
- Stops the server when complete

**Prerequisites:** Install wrk:
```bash
sudo apt-get install wrk  # Debian/Ubuntu
brew install wrk          # macOS
```

### Test Coverage

#### Read Tests (1–30) — URL-based, no Lua script required

| # | Scenario | OData Feature |
|---|----------|---------------|
| 1 | Service document | Service document |
| 2 | Metadata document (XML) | Metadata |
| 3 | Simple collection | Entity set |
| 4 | Products `$top=100` | Pagination |
| 5 | Products `$top=500` | Pagination |
| 6 | `$filter=Price gt 500` | Basic filter |
| 7 | `$orderby=Price desc` | Ordering |
| 8 | `$top=100&$skip=1000` | Skip pagination |
| 9 | `$select=ID,Name,Price` | Field projection |
| 10 | `$expand=Category` | Navigation expand |
| 11 | Filter + OrderBy + Expand | Complex query |
| 12 | `/$count?$filter=…` | Count with filter |
| 13 | Single entity by key | Key lookup |
| 14 | Singleton access | Singleton |
| 15 | `$apply` groupby + aggregate | Aggregation |
| 16 | `$search=Widget` | Full-text search |
| 17 | `$metadata?$format=json` | JSON metadata |
| 18 | `contains(Name,'Premium')` | String functions |
| 19 | `CreatedAt gt 2024-01-01` | Date comparison |
| 20 | Multi-condition filter (and/or) | Complex filter |
| 21 | `$orderby=Price desc,Name asc` | Multi-field ordering |
| 22 | `$expand=Category,Descriptions($top=2)` | Multi-level expand |
| 23 | `$expand=Category($select=ID,Name)` | Nested expand + select |
| 24 | `$apply=filter(…)/groupby(…)` | Apply filter pipeline |
| 25 | `GetTopProducts(count=10)` | Unbound function |
| 26 | `GetProductStats()` | Unbound function |
| 27 | `/Products(1)/Name/$value` | Property value path |
| 28 | `ProductDescriptions?$filter=LanguageKey eq 'EN'` | Related entity filter |
| 29 | `Categories?$expand=Products($top=3)` | Reverse expand |
| 30 | `/Products/$count` (full collection) | Collection count |

#### Write & Mutation Tests (31–38) — wrk Lua scripts in `wrk-scripts/`

| # | Scenario | OData Feature | Lua Script |
|---|----------|---------------|------------|
| 31 | `POST /Products` | Entity creation | `post_product.lua` |
| 32 | `PATCH /Products(id)` | Partial update | `patch_product.lua` |
| 33 | `POST /Products(id)/ApplyDiscount` | Bound action | `apply_discount.lua` |
| 34 | `GET /Products(id)` with `If-None-Match` | ETag — cache miss | `etag_conditional_get.lua` |
| 35 | `PATCH /Products(id)` with `If-Match: *` | Conditional update | `conditional_patch.lua` |
| 36 | `POST /$batch` (5 GETs per body) | JSON batch | `batch_get.lua` |
| 37 | `DELETE /Products(id)` (IDs 5001–10000) | Entity deletion | `delete_product.lua` |
| 38 | `GET /Products` with `Prefer: odata.track-changes` | Change tracking | `prefer_track_changes.lua` |

> **Note:** Write tests modify database state. After running, you can restore the
> dataset with `curl -X POST http://localhost:9091/Reseed`.

### Manual Query Performance Tests
Test various OData query patterns:
```bash
# Filter large datasets
curl "http://localhost:9091/Products?\$filter=Price gt 500"

# Complex queries with expand
curl "http://localhost:9091/Products?\$expand=Category,Descriptions"

# Pagination
curl "http://localhost:9091/Products?\$top=100&\$skip=1000"

# Aggregation
curl "http://localhost:9091/Products?\$apply=groupby((CategoryID),aggregate(Price with average as AvgPrice))"
```

### Manual Load Testing

Use wrk for load testing:
```bash
# Basic load test
wrk -t10 -c100 -d30s http://localhost:9091/Products

# With more threads and connections
wrk -t12 -c200 -d60s --latency http://localhost:9091/Products
```

## Analyzing Results

### CPU Profile Analysis
```bash
# Interactive analysis
go tool pprof load-test-results/cpu.prof
> top10
> web

# Generate SVG graph
go tool pprof -svg load-test-results/cpu.prof > cpu.svg

# Web UI (recommended)
go tool pprof -http=:8080 load-test-results/cpu.prof
```

### SQL Trace Analysis
The SQL trace file contains:
- Execution count for each query pattern
- Total, average, min, and max execution times
- Example queries for each pattern
- Optimization recommendations

```bash
# View the trace
cat load-test-results/sql-trace.txt

# Find slow queries
grep SLOW load-test-results/sql-trace.txt
```

### Complete Performance Analysis Guide

For detailed instructions on analyzing performance bottlenecks, see:

📖 **[PERFORMANCE_ANALYSIS.md](PERFORMANCE_ANALYSIS.md)** - Comprehensive guide covering:
- How to interpret HTTP metrics
- CPU profiling techniques
- SQL query optimization
- Common bottlenecks and solutions
- Performance testing best practices

