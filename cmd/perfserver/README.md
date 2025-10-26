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

## Endpoints

### OData Service
- `GET /` - Service document
- `GET /$metadata` - Metadata document
- `GET /Categories` - Category list
- `GET /Products` - Product list
- `GET /ProductDescriptions` - Product description list
- `GET /Company` - Company info singleton

### Performance Testing
- `POST /Reseed` - Reset database to extensive performance testing state

## VS Code Integration

Use the tasks defined in `.vscode/tasks.json`:
- "Start Perf Server (SQLite)" - Launch with SQLite in-memory database
- "Start Perf Server (PostgreSQL)" - Launch with PostgreSQL database

## Performance Testing Scenarios

### Query Performance
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

### Load Testing
Use tools like Apache Bench or wrk:
```bash
# Apache Bench
ab -n 1000 -c 10 http://localhost:9091/Products

# wrk
wrk -t10 -c100 -d30s http://localhost:9091/Products
```

## Analyzing Results

### CPU Profile Analysis
```bash
# Interactive analysis
go tool pprof cpu.prof
> top10
> web

# Generate SVG graph
go tool pprof -svg cpu.prof > cpu.svg
```

### SQL Trace Analysis
The SQL trace file contains:
- Execution count for each query pattern
- Total, average, min, and max execution times
- Example queries for each pattern
- Optimization recommendations

Press Ctrl+C to gracefully shutdown and generate the analysis report.
