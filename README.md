# go-odata

[![CI](https://github.com/NLstn/go-odata/actions/workflows/ci.yml/badge.svg)](https://github.com/NLstn/go-odata/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nlstn/go-odata)](https://goreportcard.com/report/github.com/nlstn/go-odata)

A Go library for building services that expose OData v4 APIs with automatic handling of OData protocol logic.

## Overview

`go-odata` allows you to define Go structs representing entities and automatically handles the necessary OData protocol logic, making it easy to build OData-compliant APIs.

### Key Features

- ✅ **Full OData v4 support** - 100% compliant with OData v4 specification
- 🚀 **Simple API** - Define structs, register entities, and you're done
- 🔍 **Rich querying** - Supports all OData query options ($filter, $select, $expand, etc.)
- 🌍 **Geospatial functions** - Query geographic data with geo.distance, geo.length, and geo.intersects
- 💾 **GORM integration** - Works with any GORM-compatible database
- 🔒 **Optimistic concurrency** - Built-in ETag support
- 🧰 **Lifecycle & read hooks** - Inject business logic, tenant filters, and response redaction
- 🎯 **Custom operations** - Easy registration of actions and functions
- 📊 **Data aggregation** - Supports $apply transformations
- 🧪 **Fully tested** - 85+ compliance tests ensuring OData v4 adherence
- 🔑 **Server-side key generation** - Validate directives during metadata analysis and plug in custom generators

### OData v4 Specification

This library implements the [OData v4.01 specification](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html).

## Installation

```bash
go get github.com/nlstn/go-odata
```

## Quick Start

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/nlstn/go-odata"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type Product struct {
    ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
    Name        string  `json:"Name" gorm:"not null" odata:"required"`
    Description string  `json:"Description"`
    Price       float64 `json:"Price" gorm:"not null"`
    Category    string  `json:"Category" gorm:"not null"`
}

func main() {
    // Initialize database
    db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }
    
    // Auto-migrate
    db.AutoMigrate(&Product{})
    
    // Create some sample data
    db.Create(&Product{Name: "Laptop", Price: 999.99, Category: "Electronics"})
    
    // Initialize OData service
    service := odata.NewService(db)
    
    // Register entity
    if err := service.RegisterEntity(&Product{}); err != nil {
        log.Fatal(err)
    }

    // Enable change tracking for Products if you need $deltatoken responses
    if err := service.EnableChangeTracking("Products"); err != nil {
        log.Fatal(err)
    }

    // To persist change history across restarts, build the service with:
    // service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{PersistentChangeTracking: true})
    // and handle err accordingly. The tracker stores events in the reserved `_odata_change_log` table.
    
    // Create HTTP mux and register the OData service as a handler
    mux := http.NewServeMux()
    mux.Handle("/", service)
    
    // Start server
    log.Println("Starting OData service on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

This creates a fully functional OData v4 service accessible at `http://localhost:8080`. Make sure to surface registration
errors—invalid struct tags or duplicate entity names will cause `RegisterEntity` to fail and should be addressed immediately.

## Documentation

- **[Entity Definition](documentation/entities.md)** - Define entities with rich metadata and relationships
- **[End-to-End Tutorial](documentation/tutorial.md)** - Build a multi-entity Products/Orders/Customers backend with hooks and custom operations
- **[Server Configuration](documentation/server-configuration.md)** - Configure the service, add middleware, and integrate with your application
- **[Actions and Functions](documentation/actions-and-functions.md)** - Implement custom OData operations
- **[Geospatial Functions](documentation/geospatial.md)** - Query geographic data with spatial functions
- **[Advanced Features](documentation/advanced-features.md)** - Singletons, ETags, lifecycle hooks, and read hooks for tenant filtering or redaction
- **[Testing](documentation/testing.md)** - Unit tests, compliance tests, and performance profiling

## Hooks: Inject Custom Logic

Entity types can implement optional hook methods to inject business logic at specific points in the request lifecycle. Hooks are automatically discovered via reflection - no registration needed.

### Lifecycle Hooks

Execute custom validation, authorization, and logging logic:

```go
type Product struct {
    ID        uint      `json:"ID" gorm:"primaryKey" odata:"key"`
    Name      string    `json:"Name" odata:"required"`
    Price     float64   `json:"Price"`
    CreatedAt time.Time `json:"CreatedAt"`
}

// BeforeCreate validates input before creating
func (p *Product) BeforeCreate(ctx context.Context, r *http.Request) error {
    if p.Price < 0 {
        return fmt.Errorf("price cannot be negative")
    }
    p.CreatedAt = time.Now()
    return nil
}

// AfterCreate logs the creation
func (p *Product) AfterCreate(ctx context.Context, r *http.Request) error {
    log.Printf("Product created: %s", p.Name)
    return nil
}

// Also available: BeforeUpdate, AfterUpdate, BeforeDelete, AfterDelete
```

### Read Hooks

Customize queries with tenant filters and redact sensitive data:

```go
// Apply tenant filter before querying
func (p Product) BeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
    tenantID := r.Header.Get("X-Tenant-ID")
    if tenantID == "" {
        return nil, fmt.Errorf("missing tenant header")
    }
    return []func(*gorm.DB) *gorm.DB{
        func(db *gorm.DB) *gorm.DB { return db.Where("tenant_id = ?", tenantID) },
    }, nil
}

// Redact sensitive fields before returning
func (p Product) AfterReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, error) {
    product, ok := entity.(*Product)
    if !ok || isPrivileged(r) {
        return entity, nil
    }
    product.CostPrice = 0 // Hide from non-privileged users
    return product, nil
}
```

All hook methods are optional. See [EntityHook](https://pkg.go.dev/github.com/nlstn/go-odata#EntityHook) and the [Advanced Features documentation](documentation/advanced-features.md) for details.

## What You Get

Once your service is running, it automatically provides OData v4 endpoints:

**Service Root:**
- `GET /` - Service document listing all entity sets
- `GET /$metadata` - Metadata document (XML and JSON/CSDL formats)

**CRUD Operations:**
- `GET /Products` - List all products
- `GET /Products(1)` - Get product by ID
- `POST /Products` - Create new product
- `PATCH /Products(1)` - Update product (partial)
- `PUT /Products(1)` - Replace product (full)
- `DELETE /Products(1)` - Delete product

**Query Options:**
All standard OData v4 query options are supported:
- `$filter` - Filter results with complex expressions
- `$select` - Choose specific properties
- `$expand` - Include related entities
- `$orderby` - Sort results
- `$top` / `$skip` - Pagination
- `$count` - Include total count
- `$search` - Full-text search
- `$apply` - Data aggregation
- `$deltatoken` - Change tracking (enable per entity with `EnableChangeTracking`)

**Additional Features:**
- Property access: `GET /Products(1)/Name`
- Navigation properties: `GET /Products(1)/Category`
- Composite keys: `GET /OrderItems(OrderID=1,ProductID=5)`
- Singletons: `GET /Company`
- Batch requests: `POST /$batch`
- Custom actions and functions

> **Note on UUID/GUID Keys:** When using UUID strings as entity keys, don't use quotes in the path (`/Entity(uuid-value)`), but do use quotes in filter expressions (`?$filter=field eq 'uuid-value'`). See [Working with UUID/GUID Keys](documentation/entities.md#working-with-uuidguid-keys) for details.

See the [OData v4 specification](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html) for complete protocol details.

## Asynchronous Processing (`Prefer: respond-async`)

Long-running requests can be offloaded to background workers when clients send
`Prefer: respond-async`. Enable the behaviour explicitly on your service:

```go
if err := service.EnableAsyncProcessing(odata.AsyncConfig{
    MonitorPathPrefix:    "/$async/jobs/", // default when empty
    DefaultRetryInterval: 5 * time.Second,  // Retry-After header while pending
    MaxQueueSize:         8,                // Optional worker limit
    JobRetention:         15 * time.Minute, // Overrides the 24h default retention window
}); err != nil {
    log.Fatalf("enable async processing: %v", err)
}

defer service.Close()
```

With async processing enabled:

- Initial responses return `202 Accepted` with `Preference-Applied: respond-async`,
  a `Location` header of the form `/{prefix}{jobID}`, and (when configured) a
  numeric `Retry-After` header.
- Polling the monitor URL with `GET` or `HEAD` returns 202 while the job is
  pending and replays the stored status, headers, and body once the job
  completes.
- Sending `DELETE` to the monitor URL cancels running work when possible and
  removes completed jobs.
- Job metadata is persisted in the reserved `_odata_async_jobs` table. The table
  keeps monitor state isolated from application models and allows a fresh
  `async.Manager` to serve completed job results until the retention TTL deletes
  the row.

Completed jobs are kept for 24 hours by default (`async.DefaultJobRetention`).
Setting `JobRetention` to zero uses that default, while applications that must
retain data indefinitely can opt out by setting `DisableRetention: true` and
managing cleanup manually.

Call `service.Close()` during shutdown to stop the background async manager and
release its resources. The method is idempotent so repeated shutdown hooks are
safe.

The development server (`cmd/devserver`) enables async processing by default
using the standard `/$async/jobs/` prefix and advertises the monitor endpoint in
its startup banner. Compliance tests assume that path when validating async
behaviour.

## Example Responses

### Service Document (`GET /`)
```json
{
  "@odata.context": "http://localhost:8080/$metadata",
  "value": [
    {
      "kind": "EntitySet",
      "name": "Products", 
      "url": "Products"
    }
  ]
}
```

### Entity Collection (`GET /Products`)
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products",
  "value": [
    {
      "ID": 1,
      "Name": "Laptop",
      "Description": "High-performance laptop",
      "Price": 999.99,
      "Category": "Electronics"
    }
  ]
}
```

### Individual Entity (`GET /Products(1)`)
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products/$entity",
  "ID": 1,
  "Name": "Laptop",
  "Description": "High-performance laptop",
  "Price": 999.99,
  "Category": "Electronics"
}
```

## Versioning and Changelog

This project follows [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).
Patch releases deliver backward-compatible fixes, minor releases add new
backward-compatible functionality, and major releases are reserved for breaking
changes. Planned release tags will start with `v0.1.0` and continue with the
`vMAJOR.MINOR.PATCH` pattern.

See the [CHANGELOG](CHANGELOG.md) for a curated list of notable updates and the
release plan so downstream integrations can assess compatibility expectations.

## Requirements

- Go 1.24 or later
- GORM-compatible database driver

## Supported Databases

The library works with any GORM-compatible database, but testing and active support vary by database:

- ✅ **SQLite** - Fully supported and tested. All features work reliably.
- ✅ **PostgreSQL** - Fully supported and tested. All compliance tests pass on PostgreSQL 17.
- ❓ **Other databases** (MySQL, SQL Server, etc.) - Should work through GORM compatibility, but not actively tested.

### Using Other Databases

If you'd like to use a database that isn't listed above or encounter issues:

1. Open an issue on [GitHub Issues](https://github.com/NLstn/go-odata/issues)
2. Share your use case and any errors you encounter
3. We'll work with you to add official support

While the library is designed to work with any GORM-compatible database through GORM's abstraction layer, we focus our testing efforts on SQLite and PostgreSQL to ensure the best experience.

## Testing

Run the test suite:

```bash
# Run all unit tests
go test ./...

# Run all compliance tests
cd compliance-suite
go run .

# Run tests with race detection
go test -race ./...
```

See the [Testing documentation](documentation/testing.md) for detailed information about unit tests, compliance tests, and performance profiling.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Before Submitting a PR

1. Run all unit tests: `go test ./...`
2. Run tests with race detection: `go test -race ./...`
3. Run compliance tests: `cd compliance-suite && go run .`
4. Format your code: `go fmt ./...`
5. Run go vet: `go vet ./...`
6. Run linter: `golangci-lint run`

All tests run automatically in CI/CD on every push and pull request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
