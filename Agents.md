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
- All tests use GORM with SQLite in-memory database

#### Test Organization

- **Integration tests**: All integration tests for the main OData service are located in the `test/` directory
  - These tests use the `odata_test` package and import the `odata` package
  - They test the public API of the service from an external perspective
- **Unit tests**: Internal package tests remain in their respective `internal/` subdirectories
  - These tests are in the same package as the code they test
- **White-box tests**: The root-level `odata_test.go` contains white-box tests that need access to unexported fields

When adding new tests:
- Place integration tests in the `test/` directory
- Use package `odata_test` and import `odata "github.com/nlstn/go-odata"`
- Place unit tests for internal packages in the same directory as the code

### Requirements

- Go 1.21 or later
- GORM-compatible database driver
- MIT License

---

## Code Review Instructions

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
