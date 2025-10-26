# Testing

This guide covers testing strategies for go-odata applications, including unit tests, integration tests, and compliance tests.

## Table of Contents

- [Unit Tests](#unit-tests)
- [Integration Tests](#integration-tests)
- [Compliance Tests](#compliance-tests)

## Unit Tests

Run the unit test suite to verify core functionality:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/handlers
```

### Test Organization

- **Integration tests**: Located in `test/` directory
  - Use `odata_test` package
  - Test the public API from an external perspective
  
- **Unit tests**: Located in `internal/*/` subdirectories
  - Test internal package functionality
  - In the same package as the code they test

- **White-box tests**: Located in `odata_test.go`
  - Test unexported fields and internal behavior

### Writing Tests

When adding new functionality, add tests in the appropriate location:

```go
// test/my_feature_test.go
package odata_test

import (
    "testing"
    odata "github.com/nlstn/go-odata"
)

func TestMyFeature(t *testing.T) {
    // Test implementation
}
```

## Integration Tests

Integration tests verify end-to-end functionality using GORM with SQLite in-memory database:

```go
func TestEntityRetrieval(t *testing.T) {
    // Setup database
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    db.AutoMigrate(&Product{})
    
    // Create test data
    db.Create(&Product{Name: "Test", Price: 99.99})
    
    // Create service
    service := odata.NewService(db)
    service.RegisterEntity(&Product{})
    
    // Test requests
    req := httptest.NewRequest("GET", "/Products", nil)
    w := httptest.NewRecorder()
    service.ServeHTTP(w, req)
    
    // Verify response
    if w.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", w.Code)
    }
}
```

## Compliance Tests

The library includes a comprehensive OData v4 compliance test suite with 85+ test scripts covering the OData specification.

### Running Compliance Tests

**IMPORTANT: Always use the master `run_compliance_tests.sh` script.**

```bash
# Run all compliance tests (4.0 + 4.01) - RECOMMENDED
./compliance/run_compliance_tests.sh

# Run only OData 4.0 tests
./compliance/run_compliance_tests.sh --version 4.0

# Run only OData 4.01 tests
./compliance/run_compliance_tests.sh --version 4.01

# Run a specific test
./compliance/run_compliance_tests.sh 11.4.3_update_entity.sh

# Run with verbose output
./compliance/run_compliance_tests.sh -v

# Run with failures only
./compliance/run_compliance_tests.sh -f

# Save report to file
./compliance/run_compliance_tests.sh -o compliance-report.md
```

### What Compliance Tests Cover

The compliance tests verify:
- ✅ HTTP headers (Content-Type, OData-Version, OData-MaxVersion, Accept, Prefer)
- ✅ Service document and metadata document
- ✅ URL conventions (entity addressing, canonical URLs, property access)
- ✅ Query options ($filter, $select, $orderby, $top, $skip, $count, $expand, $search, $format, $apply)
- ✅ CRUD operations (GET, POST, PATCH, PUT, DELETE)
- ✅ Conditional requests (ETags, If-Match, If-None-Match)
- ✅ Relationship management ($ref)
- ✅ Batch requests
- ✅ Error responses

### Compliance Test Structure

Tests are organized by OData version:
- `compliance/v4.0/` - OData 4.0 specification tests (82 scripts)
- `compliance/v4.01/` - OData 4.01-specific tests (3 scripts)

Each test script:
- Tests one specific section of the OData specification
- Is named according to the spec section (e.g., `11.4.3_update_entity.sh`)
- Includes spec reference URLs in comments
- Is executable and can run independently
- Returns exit code 0 on success, 1 on failure

### Adding New Compliance Tests

When adding new compliance tests:

1. Choose the correct directory:
   - Add to `v4.0/` for OData 4.0 features
   - Add to `v4.01/` only for features new in OData 4.01

2. Reference the official OData v4 specification sections

3. Include spec URL in script header:
```bash
#!/bin/bash
# Test: OData v4.0 - Section 11.4.3 Update Entity
# Reference: https://docs.oasis-open.org/odata/odata/v4.01/...
```

4. Source the test framework:
```bash
source "$SCRIPT_DIR/../test_framework.sh"
```

5. Write tests for both success and error cases

6. Ensure tests are idempotent (don't leave test data)

7. Update `compliance/README.md` with new test description

### Continuous Integration

All tests run automatically on every push and pull request via GitHub Actions:
- Unit tests
- Compliance tests (100% OData v4 compliance required)
- Code builds
- Linting

## Performance Profiling

Profile CPU usage and SQL queries during testing to identify performance bottlenecks.

### Performance Testing Server

A dedicated performance testing server is available at `cmd/perfserver` with extensive data seeding (10,000 products, 100 categories, 30,000 descriptions) for realistic performance testing.

```bash
# Start perfserver with extensive seeding
cd cmd/perfserver
go run . -extensive=true

# Start with CPU profiling
go run . -cpuprofile cpu.prof

# Start with SQL query tracing
go run . -trace-sql -trace-sql-file sql-trace.txt

# Start with both CPU and SQL profiling
go run . -cpuprofile cpu.prof -trace-sql -trace-sql-file sql-trace.txt

# Use PostgreSQL instead of SQLite
go run . -db postgres -dsn "postgresql://username:password@localhost:5432/dbname?sslmode=disable"
```

See `cmd/perfserver/README.md` for detailed usage and performance testing scenarios.

### VS Code Integration

Use the VS Code tasks (defined in `.vscode/tasks.json`) for easy server launching:
- **Start Perf Server (SQLite)** - Launch with extensive seeding on SQLite
- **Start Perf Server (PostgreSQL)** - Launch with PostgreSQL database
- **Start Perf Server with CPU Profiling (SQLite)** - Launch with CPU profiling enabled
- **Start Perf Server with SQL Tracing (SQLite)** - Launch with SQL query tracing

Access these tasks in VS Code via **Terminal > Run Task...** or the Command Palette (Ctrl/Cmd+Shift+P).

### Running with CPU Profiling

```bash
# Run compliance tests with CPU profiling
./compliance/run_compliance_tests.sh --cpuprofile /tmp/cpu.prof

# Analyze the profile with pprof
go tool pprof /tmp/cpu.prof

# Generate interactive web-based profile (requires graphviz)
go tool pprof -http=:8080 /tmp/cpu.prof

# Generate text-based reports
go tool pprof -top /tmp/cpu.prof              # Top functions by CPU time
go tool pprof -list=FunctionName /tmp/cpu.prof  # Line-by-line analysis
```

### Profiling Workflow

```bash
# 1. Run tests with profiling enabled
./compliance/run_compliance_tests.sh --cpuprofile /tmp/before.prof

# 2. Make performance improvements to the code

# 3. Run tests again with profiling
./compliance/run_compliance_tests.sh --cpuprofile /tmp/after.prof

# 4. Compare the profiles
go tool pprof -top /tmp/before.prof | head -20
go tool pprof -top /tmp/after.prof | head -20
```

### What Profiling Helps With

- Identify CPU hotspots in the library
- Measure performance improvements
- Optimize critical code paths
- Analyze execution patterns during OData operations

## SQL Query Tracing

The library includes a comprehensive SQL query tracer for identifying performance bottlenecks and optimization opportunities.

### Enabling SQL Tracing

```bash
# Enable SQL tracing
./compliance/run_compliance_tests.sh --trace-sql

# Run specific test with SQL tracing
./compliance/run_compliance_tests.sh --trace-sql 11.2.5

# Save SQL analysis to a file
./compliance/run_compliance_tests.sh --trace-sql --trace-sql-file sql-analysis.txt
```

### What You Get

When SQL tracing is enabled, you receive:

**1. Overall Statistics**
- Total queries executed
- Unique query patterns (normalized)
- Total SQL time
- Average query time

**2. Top Queries by Total Time**
Identifies queries that consume the most cumulative time - your primary optimization targets.

**3. N+1 Query Detection**
Automatically detects queries executed more than 10 times, indicating N+1 problems that should be fixed with eager loading.

**4. Slowest Individual Queries**
Shows the maximum execution time for each query pattern to identify bottlenecks.

**5. Optimization Recommendations**
Automated suggestions based on query patterns:
- N+1 Query Warnings
- Slow Query Identification
- SELECT * Detection

### Example Output

```
================================================================================
📊 SQL QUERY OPTIMIZATION ANALYSIS
================================================================================

📈 Overall Statistics:
  Total Queries Executed: 150
  Unique Query Patterns:  45
  Total SQL Time:         1250.5ms
  Average Query Time:     8.3ms

🔥 Top Queries by Total Time (Target for Optimization):
--------------------------------------------------------------------------------

  #1: Executed 50 times | Total: 450.2ms | Avg: 9.0ms | Max: 15.2ms
      SELECT * FROM `products` WHERE category_id = ?

🔁 Queries with High Execution Count (Potential N+1 Problems):
--------------------------------------------------------------------------------

  #1: Executed 50 times | Total: 450.2ms | Avg: 9.0ms | Max: 15.2ms
      SELECT * FROM `products` WHERE category_id = ?
      ⚠️  This query pattern suggests an N+1 problem!

💡 Optimization Recommendations:
--------------------------------------------------------------------------------
  1. ⚠️  N+1 Query Detected: Query executed 50 times. Consider using eager loading.
  2. 🐌 Slow Query on 'products': Avg 9.0ms. Consider adding indexes.
  3. 📋 SELECT * Detected: Consider selecting only needed columns.
```

### Use Cases

- **Find N+1 queries**: Identify queries executed excessively
- **Identify slow queries**: Find queries with high execution times
- **Optimize compliance tests**: See exactly what SQL is generated
- **Performance tuning**: Get data-driven insights for adding indexes
- **Regression detection**: Compare SQL patterns before and after changes

### Combining with CPU Profiling

Use both SQL tracing and CPU profiling together:

```bash
# Run with both SQL tracing and CPU profiling
./compliance/run_compliance_tests.sh --trace-sql --cpuprofile /tmp/cpu.prof --trace-sql-file /tmp/sql-analysis.txt

# Analyze CPU profile
go tool pprof /tmp/cpu.prof

# Review SQL analysis
cat /tmp/sql-analysis.txt
```

This helps you:
- Correlate CPU hotspots with SQL query patterns
- Identify whether issues are in query execution or application logic
- Make informed decisions about optimization priorities

## Before Submitting a PR

Ensure all quality checks pass:

1. ✅ Run all unit tests: `go test ./...`
2. ✅ Run tests with race detection: `go test -race ./...`
3. ✅ Run compliance tests: `./compliance/run_compliance_tests.sh`
4. ✅ Format your code: `go fmt ./...`
5. ✅ Run go vet: `go vet ./...`
6. ✅ Run linter: `golangci-lint run`

All tests run automatically in CI/CD on every push and pull request.
