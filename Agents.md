# AI Agent Instructions for go-odata

## Library Description

`go-odata` is a Go library for building services that expose OData APIs with automatic handling of OData protocol logic. It allows you to define Go structs representing entities and automatically handles the necessary OData protocol logic, making it easy to build OData-compliant APIs.

### Key Features

- ✅ Automatic OData endpoint generation from Go structs
- ✅ GORM database integration
- ✅ Entity collection retrieval (GET /EntitySet)
- ✅ Individual entity retrieval (GET /EntitySet(key))
- ✅ OData-compliant JSON responses with @odata.context
- ✅ Service document generation
- ✅ Basic metadata document
- ✅ Proper HTTP headers and error handling
- ✅ OData query operations ($filter, $select, $orderby)
- ✅ Pagination support ($top, $skip, $count, @odata.nextLink)
- 🔄 Complete metadata document generation - Coming soon
- 🔄 Entity relationship handling - Coming soon

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
- Unit tests for handlers, metadata, query processing, and responses
- Integration tests for relations and expand/filter combinations
- All tests use GORM with SQLite in-memory database

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
2. Format code: `gofmt -w .`
3. Run linter: `golangci-lint run ./...`
4. Fix all linting errors
5. Run tests: `go test ./...`
6. Build: `go build ./...`
7. Commit only after all checks pass

### Configuration

The project uses `.golangci.yml` for linter configuration. Respect the settings defined there.
