# Contributing to go-odata

Thank you for your interest in contributing to go-odata! We welcome contributions from the community and appreciate your help in making this library better.

This guide will help you understand our development process, coding standards, and expectations for contributions.

## Table of Contents

- [Project Scope & Philosophy](#project-scope--philosophy)
- [Prerequisites](#prerequisites)
- [Setup & Running Tests](#setup--running-tests)
- [OData Spec Compliance](#odata-spec-compliance)
- [Coding Standards](#coding-standards)
- [Pull Request Process](#pull-request-process)
- [Database Compatibility Rules](#database-compatibility-rules)
- [Versioning & Breaking Changes](#versioning--breaking-changes)
- [Security](#security)
- [License](#license)

## Project Scope & Philosophy

go-odata is focused on providing a production-ready OData v4 server toolkit for Go. Our guiding principles are:

### OData v4 Specification Compliance

- **Correctness first**: We prioritize strict adherence to the [OData v4.01 specification](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html) over shortcuts or convenience
- **100% compliance**: All features must implement the specification correctly, including edge cases and error handling
- **Validated behavior**: Protocol behavior is validated through comprehensive compliance tests

### Production Readiness

- **Reliability**: Code must be robust, well-tested, and handle errors gracefully
- **Performance**: Efficient query processing and response generation
- **Database abstraction**: Works with any GORM-compatible database without vendor lock-in
- **Clear documentation**: All public APIs must be documented

### Clean Architecture

- **Core vs optional features**: Clear separation between core OData functionality and optional features
  - **Core**: Query processing, metadata generation, CRUD operations, protocol compliance
  - **Optional**: Observability (tracing/metrics), authentication adapters, lifecycle hooks, custom operations
- **Extensibility**: Hooks and adapters allow customization without modifying core behavior
- **No implicit magic**: Explicit configuration and clear error messages

## Prerequisites

Before contributing, ensure you have the following installed:

### Required

- **Go 1.24 or later**: This project requires Go 1.24.0+
  ```bash
  go version  # Should show 1.24.0 or higher
  ```

- **golangci-lint**: For code linting
  ```bash
  # Install via go install
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  
  # Or see https://golangci-lint.run/usage/install/ for other methods
  ```

### Recommended

- **Docker**: For running database-specific compliance tests
  - Required for PostgreSQL, MySQL, and MariaDB testing
  - SQLite tests run without Docker

### Supported Databases

The library is tested against:

- ✅ **SQLite**: Fully supported (default for development)
- ✅ **PostgreSQL 17**: Fully supported with native full-text search
- ✅ **MySQL 8**: Fully supported
- ✅ **MariaDB 11**: Fully supported
- ⚠️ **Other GORM-compatible databases**: May work but not covered by CI

## Setup & Running Tests

### Clone the Repository

```bash
git clone https://github.com/nlstn/go-odata.git
cd go-odata
```

### Install Dependencies

```bash
go mod download
```

### Running Tests

#### Unit Tests

Run all unit tests (includes white-box and internal package tests):

```bash
go test ./...
```

Run tests with race detection:

```bash
go test -race ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

#### Integration Tests

Integration tests are located in the `test/` directory and use an in-memory SQLite database:

```bash
go test ./test/...
```

#### OData v4 Compliance Tests

The compliance test suite validates strict OData v4 specification adherence:

```bash
cd compliance-suite

# Run all compliance tests (4.0 + 4.01)
go run .

# Run only OData 4.0 tests
go run . -version 4.0

# Run only OData 4.01 tests
go run . -version 4.01

# Run specific tests by pattern
go run . -pattern filter

# Run with verbose debug output
go run . -debug

# Test against PostgreSQL instead of SQLite
go run . -db postgres -dsn "host=localhost user=postgres password=postgres dbname=odata_test sslmode=disable"
```

The compliance tests are **critical** for validating protocol correctness. All protocol changes must pass these tests.

### Development Server

Run the development server to manually test your changes:

```bash
cd cmd/devserver

# Run with SQLite (default)
go run .

# Run with PostgreSQL
go run . -db postgres -dsn "host=localhost user=postgres password=postgres dbname=odata_dev sslmode=disable"
```

The server runs on `http://localhost:8080` with sample data (Products, Orders, Customers).

### Linting and Formatting

Before committing, ensure your code passes all checks:

```bash
# Format code
go fmt ./...

# Run go vet
go vet ./...

# Run golangci-lint
golangci-lint run
```

**Important**: All code must pass linting before it can be merged. Zero linting errors are required.

## OData Spec Compliance

**Strict compliance with the OData v4 specification is mandatory.**

### Requirements

1. **Specification references**: All behavior changes must reference the relevant section of the [OData v4.01 specification](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html)
   
2. **Test coverage**: New features must include:
   - **Positive test cases**: Verify correct behavior with valid inputs
   - **Negative test cases**: Verify correct error handling with invalid inputs
   - **Edge cases**: Test boundary conditions and unusual scenarios

3. **Protocol validation**: All protocol behavior must be validated against:
   - **Metadata correctness**: Entity types, relationships, annotations
   - **Query processing**: Correct parsing and execution of query options
   - **Response formatting**: Proper OData JSON format with required annotations
   - **Error responses**: Correct error structure and HTTP status codes
   - **HTTP headers**: Correct Content-Type, OData-Version, etc.

4. **Compliance tests**: Protocol changes must pass the compliance test suite
   - Located in `compliance-suite/`
   - Tests are organized by OData version (v4.0, v4.01)
   - **Never make tests more lenient** to accommodate current implementation
   - If a compliance test fails, fix the implementation, not the test

### When Adding Features

- Read the relevant specification sections first
- Understand both the happy path and error cases
- Check existing compliance tests for similar features
- Add new compliance tests for new protocol features
- Validate against multiple databases (at least SQLite + PostgreSQL)

## Coding Standards

### Code Style

- **Use `gofmt`**: All code must be formatted with `gofmt`
- **Pass `go vet`**: No vet warnings allowed
- **Pass `golangci-lint`**: See `.golangci.yml` for enabled linters
- **Follow Go conventions**: Use standard Go naming, package structure, and idioms

### Documentation

- **Document all public APIs**: Every exported type, function, and method must have a doc comment
  - Start with the name of the item being documented
  - Explain what it does, not how it works
  - Include usage examples for complex features

  ```go
  // Service represents an OData v4 service backed by a GORM database.
  // It handles HTTP requests, processes OData query options, and generates
  // OData-compliant responses.
  type Service struct { ... }
  ```

- **Keep comments up to date**: When changing code, update related comments
- **No commented-out code**: Remove dead code instead of commenting it out

### Error Handling

- **Handle all errors**: Never ignore error returns
  - Use `t.Fatalf()` in tests
  - Use `log.Fatalf()` in cmd/ files
  - Return errors to callers in library code

  ```go
  // Good
  service, err := odata.NewService(db)
  if err != nil {
      return fmt.Errorf("failed to create service: %w", err)
  }

  // Bad - ignoring error
  service, _ := odata.NewService(db)
  ```

- **Wrap errors with context**: Use `fmt.Errorf` with `%w` to add context
- **Clear error messages**: Errors should help users understand what went wrong and how to fix it

### Context Usage

- **Always accept context**: HTTP handlers and long-running operations should accept `context.Context`
- **Propagate context**: Pass context through the call stack
- **Check for cancellation**: Respect context cancellation in long operations

### Code Organization

- **Small functions**: Keep functions focused on a single task
- **Package structure**:
  - Integration tests → `test/` directory (package `odata_test`)
  - Internal unit tests → `internal/*/` directories (same package as code)
  - Compliance tests → `compliance-suite/tests/`
- **No cyclic dependencies**: Keep package dependencies acyclic

### Breaking Changes

**No silent breaking changes allowed.**

- Any change that affects:
  - Public API signatures
  - Observable behavior
  - Metadata structure
  - Protocol responses
  - Configuration options

Must be clearly documented and discussed before implementation (see [Versioning & Breaking Changes](#versioning--breaking-changes)).

## Pull Request Process

### Before You Start

1. **Check existing issues**: Search for related issues or discussions
2. **Create an issue**: For significant changes, create an issue first to discuss the approach
3. **Fork the repository**: Create your own fork to work in

### Development Workflow

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/my-feature
   # or
   git checkout -b fix/issue-123
   ```

2. **Make your changes**:
   - One logical change per commit
   - Write clear commit messages
   - Keep commits focused and atomic

3. **Test thoroughly**:
   ```bash
   # Run all tests
   go test ./...
   
   # Run with race detection
   go test -race ./...
   
   # Run compliance tests
   cd compliance-suite && go run .
   
   # Lint your code
   golangci-lint run
   
   # Format your code
   go fmt ./...
   ```

4. **Update documentation**:
   - Update relevant documentation in `documentation/`
   - Update README.md if adding user-facing features
   - Add or update code comments

5. **Commit your changes**:
   ```bash
   git add .
   git commit -m "Add feature: brief description"
   ```

### Pull Request Guidelines

1. **Small, reviewable PRs**: Keep pull requests focused and reasonably sized
   - Aim for <500 lines of changes when possible
   - Split large features into multiple PRs
   - Each PR should be independently reviewable

2. **Clear description**: Include:
   - **What**: What does this PR do?
   - **Why**: Why is this change needed?
   - **How**: How does it work?
   - **Spec reference**: Link to relevant OData specification sections
   - **Testing**: What tests were added/modified?

3. **Link issues**: Use keywords to link related issues:
   ```
   Fixes #123
   Relates to #456
   ```

4. **Pass CI**: All CI checks must pass before merging
   - Tests (unit + integration + compliance)
   - Linting (golangci-lint)
   - Build verification

5. **Code review**: Address all review comments
   - Be open to feedback
   - Explain your reasoning when needed
   - Make requested changes or discuss alternatives

6. **Keep it updated**: Rebase on main if conflicts arise
   ```bash
   git fetch origin
   git rebase origin/main
   ```

### Commit Message Format

Use clear, descriptive commit messages:

```
Add support for $search on navigation properties

- Implement search query parsing for navigation paths
- Add compliance tests for $search with $expand
- Update metadata to include searchable annotations

Fixes #123
```

## Database Compatibility Rules

go-odata uses GORM for database abstraction, enabling support for multiple databases.

### Requirements

1. **GORM-compatible code only**: All database operations must use GORM APIs
   - No direct SQL queries unless absolutely necessary
   - No database-specific SQL without guards

2. **No unguarded DB-specific SQL**: If you must write raw SQL:
   - Detect the database type first
   - Provide implementations for all supported databases
   - Document the limitation if some databases can't support the feature

   ```go
   // Good - database-specific with guards
   if db.Dialector.Name() == "postgres" {
       db.Exec("SELECT ... USING GIN INDEX")
   } else if db.Dialector.Name() == "sqlite" {
       db.Exec("SELECT ... USING FTS")
   }

   // Bad - unguarded database-specific SQL
   db.Exec("SELECT ... USING GIN INDEX")  // Only works on Postgres
   ```

3. **Test on multiple databases**: At minimum, test changes on:
   - **SQLite**: Required (easiest for local development)
   - **PostgreSQL**: Strongly recommended (most common production database)
   - **MySQL/MariaDB**: Optional but appreciated

4. **CI validation**: All databases in CI must pass
   - SQLite (always runs)
   - PostgreSQL (always runs)
   - MySQL (always runs)
   - MariaDB (always runs)

### Database-Specific Features

Some features have database-specific implementations:

- **Full-text search ($search)**:
  - SQLite: Uses FTS3/FTS4/FTS5
  - PostgreSQL: Uses tsvector/GIN indexes
  - MySQL/MariaDB: In-memory filtering (slower)

Document any limitations in user-facing documentation.

## Versioning & Breaking Changes

go-odata follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** (x.0.0): Breaking changes
- **MINOR** (0.x.0): New features (backward compatible)
- **PATCH** (0.0.x): Bug fixes (backward compatible)

### What Counts as a Breaking Change?

A change is **breaking** if it:

1. **Changes public API signatures**:
   - Removing exported functions, methods, or types
   - Changing function signatures (parameters or return values)
   - Renaming exported identifiers

2. **Changes observable behavior**:
   - Different query results for the same input
   - Different HTTP status codes for the same requests
   - Different error messages (that users may rely on)

3. **Changes metadata structure**:
   - Different entity types or properties in $metadata
   - Different annotations or vocabulary terms
   - Different relationship definitions

4. **Changes protocol responses**:
   - Different OData JSON format
   - Different HTTP headers
   - Different URL conventions

5. **Changes configuration behavior**:
   - Different defaults
   - Required configuration that was optional
   - Removed configuration options

### Before Making Breaking Changes

1. **Discuss first**: Open an issue to discuss the breaking change
   - Explain why it's necessary
   - Explore alternatives
   - Get maintainer buy-in

2. **Migration path**: Provide a clear upgrade path
   - Document the change
   - Provide migration examples
   - Consider deprecation before removal

3. **Version planning**: Breaking changes are batched into major releases

### Non-Breaking Changes

These are generally **not** breaking changes:

- Adding new optional fields to structs
- Adding new exported functions or methods
- Adding new query options or features
- Fixing bugs that made behavior non-compliant with OData spec
- Improving error messages (with care)
- Internal refactoring (no API changes)

## Security

### Reporting Vulnerabilities

**Do not open public issues for security vulnerabilities.**

If you discover a security vulnerability, please report it privately:

1. **Email the maintainers**: Send details to the repository maintainers via GitHub's security advisory feature or direct email
2. **Include details**:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if you have one)

3. **Response time**: We aim to respond within 48 hours
4. **Disclosure**: We'll work with you on responsible disclosure timing

### Security Considerations

When contributing, keep security in mind:

- **Input validation**: Validate all user inputs (query parameters, request bodies, headers)
- **SQL injection**: Use GORM's parameter binding, never string concatenation
- **Path traversal**: Validate file paths and entity names
- **Resource limits**: Prevent DoS through unbounded queries or large payloads
- **Error messages**: Don't leak sensitive information in error responses
- **Authentication**: Use the auth adapter, don't roll your own

## License

By contributing to go-odata, you agree that your contributions will be licensed under the [MIT License](LICENSE).

All contributions must be your original work or properly attributed if based on other work. Ensure you have the right to contribute the code under the MIT License.

---

## Questions?

- **Documentation**: See the [`documentation/`](documentation/) directory
- **Issues**: Open an issue for bugs or feature requests
- **Discussions**: Use GitHub Discussions for questions and ideas

Thank you for contributing to go-odata! 🎉
