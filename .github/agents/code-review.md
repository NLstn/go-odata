# Code Review Instructions for Copilot

## Code Quality Requirements

When reviewing or making code changes, ensure the following quality checks are performed:

### Linting
- **ALWAYS run golangci-lint** before finalizing any code changes
- Run: `golangci-lint run ./...`
- Fix all linting errors before committing
- Zero linting errors are required for code to be considered complete

### Testing
- Run all tests with: `go test ./...`
- All existing tests must pass
- Add new tests for new functionality
- Maintain or improve code coverage

### Formatting
- Run `gofmt -w .` to format all Go files
- Follow Go standard formatting conventions

### Building
- Verify the code builds without errors: `go build ./...`
- Check for any compilation warnings

## Workflow

1. Make code changes
2. Format code: `gofmt -w .`
3. Run linter: `golangci-lint run ./...`
4. Fix all linting errors
5. Run tests: `go test ./...`
6. Build: `go build ./...`
7. Commit only after all checks pass

## Configuration

The project uses `.golangci.yml` for linter configuration. Respect the settings defined there.
