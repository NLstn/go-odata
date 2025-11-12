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
- Runs 14 different test scenarios
- Saves detailed results to `./load-test-results/`
- Stops the server when complete

**Prerequisites:** Install wrk:
```bash
sudo apt-get install wrk  # Debian/Ubuntu
brew install wrk          # macOS
```

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

