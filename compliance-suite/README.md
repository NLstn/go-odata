# OData v4 Compliance Test Suite (Go)

A Go-based compliance test suite for validating OData v4 protocol implementations. This replaces the previous Bash-based test suite with a more maintainable, type-safe, and developer-friendly Go implementation.

## Overview

The compliance test suite validates that an OData service correctly implements the OData v4 specification. Tests are organized by OData version (4.0 and 4.01) and cover various aspects of the protocol including:

- Service document and metadata
- Query options ($filter, $select, $orderby, etc.)
- Query option combinations that validate $count with $search responses
- CRUD operations
- HTTP headers and content negotiation
- Entity relationships and navigation
- Batch requests
- And more...

The test suite runs on both **SQLite** and **PostgreSQL** databases to ensure cross-database compatibility. All 105 test suites pass on both databases with a 93% individual test pass rate (620 passing tests out of 666 total).

## Project Structure

```
compliance-suite/
├── main.go                    # Main test runner
├── go.mod                     # Go module definition
├── framework/
│   └── framework.go           # Test framework with HTTP client and assertions
├── tests/
│   ├── v4_0/                  # OData 4.0 compliance tests
│   │   └── introduction.go   # Example: 1.1 Introduction tests
│   └── v4_01/                 # OData 4.01 compliance tests
└── README.md                  # This file
```

## Features

### Test Framework

The `framework` package provides:

- **TestSuite**: Organizes related tests with metadata (name, description, spec URL)
- **TestContext**: Provides HTTP client and assertion helpers for each test
- **HTTP Methods**: GET, POST, PATCH, PUT, DELETE with automatic debugging
- **Assertions**: Status code, headers, JSON fields, body content
- **Debug Mode**: Full HTTP request/response logging
- **Skip Support**: Mark tests as skipped with reasons

### Test Runner

The main test runner (`main.go`) provides:

- Automatic server startup and shutdown
- Support for external servers
- Test filtering by pattern
- Multiple OData versions (4.0, 4.01, all)
- Database configuration (SQLite, PostgreSQL)
- Comprehensive test reporting
- Exit codes for CI/CD integration

### Output Modes

The compliance suite supports two output modes:

**Normal Mode (default)**
- Shows a single progress line with suite and test counts
- Prints only the overall result summary (pass/fail/skip totals)
- Ideal for CI/CD pipelines and quick local testing
- Use `-verbose` to see per-suite and per-test details

**Verbose Mode (`-verbose`)**
- Shows full suite description and spec reference
- Shows individual test results: ✓ PASS, ✗ FAIL, ⊘ SKIP
- Shows detailed error messages for each failure
- Ideal for debugging and development

Example outputs:

```bash
# Normal mode - concise output
Running 105 suites (666 total tests)

Progress: suites 105/105 | tests 666/666 | passed 666 | failed 0 | skipped 0

╔════════════════════════════════════════════════════════╗
║                  OVERALL SUMMARY                       ║
╚════════════════════════════════════════════════════════╝
Test Scripts: 105/105 passed (100%)
Individual Tests:
    - Total: 666
    - Passing: 666
    - Failing: 0
    - Skipped: 0
    - Pass Rate: 100%

# Verbose mode - detailed output
✓ PASS: Test should validate entity creation
✓ PASS: Test should handle concurrent requests
✗ FAIL: Test should validate deep insert
    Error: expected status code 201 but got 500
```

## Usage

### Running Tests

```bash
# Run all tests (auto-starts compliance server)
cd compliance-suite
go run main.go

# Run with verbose mode to see all test results
go run main.go -verbose

# Run with debug mode for full HTTP details
go run main.go -debug

# Run only OData 4.0 tests
go run main.go -version 4.0

# Run only OData 4.01 tests
go run main.go -version 4.01

# Run specific tests by pattern
go run main.go -pattern introduction

# Use an external server
go run main.go -external-server -server http://localhost:8080

# Use PostgreSQL instead of SQLite
go run main.go -db postgres -dsn "postgresql://user:pass@localhost/db"

# Show help
go run main.go -help
```

### Building

```bash
# Build the test runner
cd compliance-suite
go build -o compliance-test

# Run the built binary
./compliance-test
```

### Running as Go Test

You can also run the compliance tests using Go's native test runner:

```bash
cd compliance-suite
go test -v ./...
```

## Writing New Tests

### Step 1: Create a Test File

Create a new file in `tests/v4_0/` (or `v4_01/` for 4.01-specific features):

```go
package v4_0

import "github.com/nlstn/go-odata/compliance-suite/framework"

func MyNewTest() *framework.TestSuite {
    suite := framework.NewTestSuite(
        "Section Name",
        "Description of what this test validates",
        "https://link-to-odata-spec-section",
    )

    // Add tests
    suite.AddTest(
        "test_function_name",
        "Human-readable test description",
        func(ctx *framework.TestContext) error {
            // Perform HTTP request
            resp, err := ctx.GET("/EntitySet")
            if err != nil {
                return err
            }

            // Assert expectations
            if err := ctx.AssertStatusCode(resp, 200); err != nil {
                return err
            }

            return ctx.AssertJSONField(resp, "@odata.context")
        },
    )

    return suite
}
```

### Step 2: Register the Test

Add your test to `main.go`:

```go
// In the main() function, add to testSuites:
testSuites = append(testSuites, TestSuiteInfo{
    Name:    "my_new_test",
    Version: "4.0",
    Suite:   v4_0.MyNewTest,
})
```

### Available Assertion Methods

The `TestContext` provides many assertion helpers:

```go
// HTTP Methods
resp, err := ctx.GET("/path")
resp, err := ctx.POST("/path", jsonBody, headers...)
resp, err := ctx.PATCH("/path", jsonBody, headers...)
resp, err := ctx.PUT("/path", jsonBody, headers...)
resp, err := ctx.DELETE("/path")

// Status Code Assertions
ctx.AssertStatusCode(resp, 200)

// Header Assertions
ctx.AssertHeader(resp, "Content-Type", "application/json")
ctx.AssertHeaderContains(resp, "Content-Type", "json")

// JSON Assertions
ctx.AssertJSONField(resp, "@odata.context")
ctx.GetJSON(resp, &targetStruct)
ctx.IsValidJSON(resp)

// Body Assertions
ctx.AssertBodyContains(resp, "expected text")

// Skip Test
return ctx.Skip("Feature not yet implemented")
```

### Example Test

Here's the complete 1.1 Introduction test as an example:

```go
func Introduction() *framework.TestSuite {
    suite := framework.NewTestSuite(
        "1.1 Introduction",
        "Tests basic service requirements",
        "https://docs.oasis-open.org/odata/...",
    )

    suite.AddTest(
        "test_service_root_accessible",
        "Service root is accessible",
        func(ctx *framework.TestContext) error {
            resp, err := ctx.GET("/")
            if err != nil {
                return err
            }
            return ctx.AssertStatusCode(resp, 200)
        },
    )

    // Add more tests...

    return suite
}
```

## Command-Line Options

```
-server string
    OData server URL (default "http://localhost:9090")

-db string
    Database type: sqlite or postgres (default "sqlite")

-dsn string
    Database DSN/connection string

-version string
    OData version to test: 4.0, 4.01, or all (default "all")

-pattern string
    Run only tests matching pattern

-verbose
    Enable verbose mode to show all test results (default: only shows progress and summary)

-debug
    Enable debug mode with full HTTP details

-external-server
    Use an external server (don't start/stop)

-output string
    Output file for the report (default "compliance-report.md")
```

## CI/CD Integration

The test runner returns appropriate exit codes:

- `0`: All tests passed
- `1`: One or more tests failed

Example GitHub Actions workflow:

```yaml
- name: Run Compliance Tests
  run: |
    cd compliance-suite
    go run main.go -db postgres -dsn "$DATABASE_URL"
```

## Progress: Ported Tests

The following tests have been ported from Bash to Go (8 test suites):

### OData 4.0 Tests
- ✅ 1.1 Introduction - Basic service requirements
- ✅ 2.1 Conformance - Service conformance to OData v4
- ✅ 3.1 EDMX Element - Metadata EDMX structure validation
- ✅ 8.1.1 Header Content-Type - Content-Type header validation
- ✅ 8.2.6 Header OData-Version - OData-Version header and negotiation
- ✅ 9.1 Service Document - Service document structure
- ✅ 9.2 Metadata Document - Metadata document XML validation
- ✅ 11.2.5.1 Query Filter - $filter query option tests

### Still in Bash (77 remaining)
The remaining 77 test scripts in `compliance/v4.0/` and `compliance/v4.01/` are awaiting migration.

## Migrating from Bash Tests

To migrate an existing Bash test to Go:

1. Create a new `.go` file in `tests/v4_0/` or `tests/v4_01/`
2. Create a function that returns `*framework.TestSuite`
3. For each `run_test` call in Bash, create `suite.AddTest()` in Go
4. Replace `http_get`, `check_status`, etc. with `ctx.GET()`, `ctx.AssertStatusCode()`, etc.
5. Replace `check_json_field` with `ctx.AssertJSONField()`
6. Register the suite in `main.go`

## Benefits Over Bash

- **Type Safety**: Catch errors at compile time
- **Better IDE Support**: Autocomplete, refactoring, debugging
- **Easier Testing**: Unit test the test framework itself
- **Better Error Messages**: Stack traces and structured errors
- **Maintainability**: Clear structure and reusable components
- **Performance**: Faster execution, better concurrency support
- **Cross-Platform**: Works on Windows without WSL

## Development

### Running Framework Tests

```bash
cd framework
go test -v
```

### Linting

```bash
cd compliance-suite
golangci-lint run ./...
```

### Formatting

```bash
cd compliance-suite
gofmt -w .
```

## Contributing

When adding new compliance tests:

1. Follow the existing test structure
2. Include clear descriptions and spec references
3. Use appropriate assertion methods
4. Add comments for complex test logic
5. Run linting and formatting before committing
6. Update this README if adding new features

## License

MIT License - Same as the parent go-odata project
