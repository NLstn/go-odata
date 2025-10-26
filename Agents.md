# AI Agent Instructions for go-odata

## Library Description

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

The `compliance/` directory contains executable shell scripts that validate the library's compliance with the official OData v4 specification. Tests are organized by OData version:

- **`v4.0/`** - OData 4.0 specification compliance tests (82 scripts)
- **`v4.01/`** - OData 4.01-specific compliance tests (3 scripts for features new in 4.01)
- **`test_framework.sh`** - Shared test framework
- **`run_compliance_tests.sh`** - Master test runner with version selection

##### Compliance Test Requirements

1. **Strict Specification Adherence**: Tests must validate EXACT compliance with the OData v4 spec
   - If the spec requires HTTP status 400, the test must fail if 500 is returned
   - If the spec requires specific headers, those exact headers must be present
   - Error response formats must match the specification exactly
   - No lenient behavior or "close enough" validations

2. **Test Structure**: Each compliance test script:
   - Tests one specific section of the OData v4 specification
   - Is named according to the spec section (e.g., `11.4.3_update_entity.sh`)
   - Is placed in `v4.0/` for OData 4.0 features, or `v4.01/` for 4.01-specific features
   - Includes spec reference URLs in comments
   - Is executable and can run independently
   - Returns exit code 0 on success, 1 on failure
   - Cleans up any test data it creates (non-destructive testing)

3. **When Modifying Compliance Tests**:
   - **NEVER make tests more lenient** to accommodate current implementation
   - If a test fails, the implementation must be fixed, not the test
   - Tests should reveal gaps between implementation and specification
   - Document any intentional deviations from the spec with clear justification
   - Update tests only when the OData specification itself changes

4. **Running Compliance Tests**:
   ```bash
   # Start the dev server first
   cd cmd/devserver && go run . &
   
   # Run all compliance tests (4.0 + 4.01)
   cd compliance
   ./run_compliance_tests.sh
   
   # Run only OData 4.0 tests
   ./run_compliance_tests.sh --version 4.0
   
   # Run only OData 4.01 tests
   ./run_compliance_tests.sh --version 4.01
   
   # Run specific test
   cd v4.0
   ./11.4.3_update_entity.sh
   ```

5. **Test Coverage**: Currently includes 85 test scripts covering:
   - HTTP headers (Content-Type, OData-Version, OData-MaxVersion, Accept, Prefer, Error responses)
   - Service document and metadata document
   - URL conventions (entity addressing, canonical URLs, property access, metadata levels, delta links)
   - Query options ($filter, $select, $orderby, $top, $skip, $count, $expand, $search, $format, $apply)
   - CRUD operations (GET, POST, PATCH, PUT, DELETE)
   - Conditional requests (ETags, If-Match, If-None-Match)
   - Relationship management ($ref)
   - Batch requests

6. **Adding New Compliance Tests**:
   - Choose the correct directory:
     - Add to `v4.0/` for OData 4.0 features (applies to both 4.0 and 4.01)
     - Add to `v4.01/` only for features new or different in OData 4.01
   - Reference the official OData v4 specification sections
   - Include spec URL in script header comments
   - Source the test framework: `source "$SCRIPT_DIR/../test_framework.sh"`
   - Test both success cases and error cases
   - Validate response status codes, headers, and body structure
   - Ensure tests are idempotent and don't leave test data
   - Update `compliance/README.md` with new test description

### Requirements

- Go 1.21 or later
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

### Configuration

The project uses `.golangci.yml` for linter configuration. Respect the settings defined there.
