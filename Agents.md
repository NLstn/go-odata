# AI Agent Instructions for go-odata

## Library Description

**Note:** This library implements OData v4 (specifically the OData v4.01 specification).

`go-odata` is a Go library for building services that expose OData APIs with automatic handling of OData protocol logic. It allows you to define Go structs representing entities and automatically handles the necessary OData protocol logic, making it easy to build OData-compliant APIs.

### Architecture

The library is structured with:
- **Core service** (`odata.go`, `server.go`): Main OData service and HTTP server
- **Internal handlers** (`internal/handlers/`): Request handlers for entities, metadata, and service documents
- **Metadata processing** (`internal/metadata/`): Entity metadata extraction and analysis
- **Query processing** (`internal/query/`): OData query option parsing and execution
- **Response formatting** (`internal/response/`): OData-compliant response generation
- **Development server** (`cmd/devserver/`): Example implementation with sample data

### Testing

The project includes comprehensive tests:
- Unit tests for handlers, metadata, query processing, and responses (located in `internal/*/`)
- Integration tests for the main OData service (located in `test/`)
- OData v4 compliance tests (located in `compliance/v4/`)
- All tests use GORM with SQLite in-memory database

#### Test Organization

- **Integration tests**: All integration tests for the main OData service are located in the `test/` directory
  - These tests use the `odata_test` package and import the `odata` package
  - They test the public API of the service from an external perspective
- **Unit tests**: Internal package tests remain in their respective `internal/` subdirectories
  - These tests are in the same package as the code they test
- **White-box tests**: The root-level `odata_test.go` contains white-box tests that need access to unexported fields
- **Compliance tests**: OData v4 specification compliance tests in `compliance/v4/`
  - Shell scripts that test against a running development server
  - Validate strict adherence to the OData v4 specification
  - Must be kept in sync with specification requirements

When adding new tests:
- Place integration tests in the `test/` directory
- Use package `odata_test` and import `odata "github.com/nlstn/go-odata"`
- Place unit tests for internal packages in the same directory as the code

#### OData v4 Compliance Testing

**CRITICAL: Compliance tests MUST strictly adhere to the OData v4 specification.**

The `compliance-suite/` directory contains a Go-based test suite that validates the library's compliance with the official OData v4 specification. Tests are organized by OData version:

- **`tests/v4_0/`** - OData 4.0 specification compliance tests
- **`tests/v4_01/`** - OData 4.01-specific compliance tests
- **`framework/`** - Test framework with HTTP client and assertions
- **`main.go`** - Test runner with server management and reporting

##### Compliance Test Requirements

1. **Strict Specification Adherence**: Tests must validate EXACT compliance with the OData v4 spec
   - If the spec requires HTTP status 400, the test must fail if 500 is returned
   - If the spec requires specific headers, those exact headers must be present
   - Error response formats must match the specification exactly
   - No lenient behavior or "close enough" validations

2. **Test Structure**: Each compliance test:
   - Tests one specific section of the OData v4 specification
   - Is named according to the spec section (e.g., `query_filter.go`)
   - Is placed in `tests/v4_0/` for OData 4.0 features, or `tests/v4_01/` for 4.01-specific features
   - Includes spec reference URLs in the TestSuite definition
   - Can run independently or as part of the full suite
   - Returns appropriate exit codes for CI/CD integration
   - Cleans up any test data it creates (non-destructive testing)

3. **When Modifying Compliance Tests**:
   - **NEVER make tests more lenient** to accommodate current implementation
   - If a test fails, the implementation must be fixed, not the test
   - Tests should reveal gaps between implementation and specification
   - Document any intentional deviations from the spec with clear justification
   - Update tests only when the OData specification itself changes

4. **Running Compliance Tests**:
   
   **IMPORTANT: Always run compliance tests using the Go-based test suite.**
   
   The test suite at `compliance-suite/` ensures:
   - Proper test environment setup and cleanup
   - Consistent execution across all test versions
   - Comprehensive reporting and error tracking
   - Automatic server management
   
   ```bash
   # Run all compliance tests (4.0 + 4.01) - PREFERRED METHOD
   cd compliance-suite
   go run .
   
   # Run only OData 4.0 tests
   go run . -version 4.0
   
   # Run only OData 4.01 tests
   go run . -version 4.01
   
   # Run specific tests by pattern
   go run . -pattern filter
   ```

5. **Test Coverage**: The Go-based test suite covers:
   - HTTP headers (Content-Type, OData-Version, OData-MaxVersion, Accept, Prefer, Error responses)
   - Service document and metadata document
   - URL conventions (entity addressing, canonical URLs, property access, metadata levels, delta links)
   - Query options ($filter, $select, $orderby, $top, $skip, $count, $expand, $search, $format, $apply)
   - CRUD operations (GET, POST, PATCH, PUT, DELETE)
   - Conditional requests (ETags, If-Match, If-None-Match)
   - Relationship management ($ref)
   - Batch requests
   
   The suite is actively being expanded with tests ported from the legacy bash implementation.

6. **Adding New Compliance Tests**:
   - Choose the correct directory:
     - Add to `tests/v4_0/` for OData 4.0 features (applies to both 4.0 and 4.01)
     - Add to `tests/v4_01/` only for features new or different in OData 4.01
   - Create a Go file with a function that returns `*framework.TestSuite`
   - Reference the official OData v4 specification sections
   - Include spec URL in the TestSuite definition
   - Use the framework's HTTP methods (GET, POST, etc.) and assertions
   - Test both success cases and error cases
   - Validate response status codes, headers, and body structure
   - Ensure tests are idempotent and don't leave test data
   - Register the test suite in `compliance-suite/main.go`
   - Update `compliance-suite/README.md` with new test description

### Requirements

- Go 1.24 or later
- GORM-compatible database driver
- MIT License

---

## Code Review Instructions

### General Instructions

- **DO NOT generate summary documents of your changes**

### Code Quality Requirements

When reviewing or making code changes, ensure the following quality checks are performed:

#### Linting
- **ALWAYS run golangci-lint** before finalizing any code changes
- Run: `golangci-lint run ./...`
- Fix all linting errors before committing
- Zero linting errors are required for code to be considered complete

#### Testing
- Run all tests with: `go test ./...`
- All existing tests must pass
- Add new tests for new functionality
- Maintain or improve code coverage

#### Formatting
- Run `gofmt -w .` to format all Go files
- Follow Go standard formatting conventions

#### Building
- Verify the code builds without errors: `go build ./...`
- Check for any compilation warnings

### Workflow

1. Make code changes
2. Add tests in appropriate location:
   - Integration tests → `test/` directory
   - Unit tests → same directory as source code
3. Format code: `gofmt -w .`
4. Run linter: `golangci-lint run ./...`
5. Fix all linting errors
6. Run tests: `go test ./...`
7. Build: `go build ./...`
8. Commit only after all checks pass
9. **CRITICAL**: Update `CHANGELOG.md` with any user-facing changes or release planning adjustments

### Configuration

The project uses `.golangci.yml` for linter configuration. Respect the settings defined there.
