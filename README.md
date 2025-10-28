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
- 💾 **GORM integration** - Works with any GORM-compatible database
- 🔒 **Optimistic concurrency** - Built-in ETag support
- 🧰 **Lifecycle & read hooks** - Inject business logic, tenant filters, and response redaction
- 🎯 **Custom operations** - Easy registration of actions and functions
- 📊 **Data aggregation** - Supports $apply transformations
- 🧪 **Fully tested** - 85+ compliance tests ensuring OData v4 adherence

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
    service.RegisterEntity(&Product{})
    
    // Create HTTP mux and register the OData service as a handler
    mux := http.NewServeMux()
    mux.Handle("/", service)
    
    // Start server
    log.Println("Starting OData service on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

This creates a fully functional OData v4 service accessible at `http://localhost:8080`.

## Documentation

- **[Entity Definition](documentation/entities.md)** - Define entities with rich metadata and relationships
- **[Server Configuration](documentation/server-configuration.md)** - Configure the service, add middleware, and integrate with your application
- **[Actions and Functions](documentation/actions-and-functions.md)** - Implement custom OData operations
- **[Advanced Features](documentation/advanced-features.md)** - Singletons, ETags, lifecycle hooks, and read hooks for tenant filtering or redaction
- **[Testing](documentation/testing.md)** - Unit tests, compliance tests, and performance profiling

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

**Additional Features:**
- Property access: `GET /Products(1)/Name`
- Navigation properties: `GET /Products(1)/Category`
- Composite keys: `GET /OrderItems(OrderID=1,ProductID=5)`
- Singletons: `GET /Company`
- Batch requests: `POST /$batch`
- Custom actions and functions

See the [OData v4 specification](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html) for complete protocol details.

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

## Requirements

- Go 1.21 or later
- GORM-compatible database driver

## Testing

Run the test suite:

```bash
# Run all unit tests
go test ./...

# Run all compliance tests
./compliance/run_compliance_tests.sh

# Run tests with race detection
go test -race ./...
```

See the [Testing documentation](documentation/testing.md) for detailed information about unit tests, compliance tests, and performance profiling.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Before Submitting a PR

1. Run all unit tests: `go test ./...`
2. Run tests with race detection: `go test -race ./...`
3. Run compliance tests: `./compliance/run_compliance_tests.sh`
4. Format your code: `go fmt ./...`
5. Run go vet: `go vet ./...`
6. Run linter: `golangci-lint run`

All tests run automatically in CI/CD on every push and pull request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
