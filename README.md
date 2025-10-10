# go-odata

[![CI](https://github.com/NLstn/go-odata/actions/workflows/ci.yml/badge.svg)](https://github.com/NLstn/go-odata/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nlstn/go-odata)](https://goreportcard.com/report/github.com/nlstn/go-odata)
[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/NLstn/go-odata)

A Go library for building services that expose OData APIs with automatic handling of OData logic.

## Overview

`go-odata` allows you to define Go structs representing entities and automatically handles the necessary OData protocol logic, making it easy to build OData-compliant APIs.

## Features

- ✅ Automatic OData endpoint generation from Go structs
- ✅ GORM database integration
- ✅ Entity collection retrieval (GET /EntitySet)
- ✅ Individual entity retrieval (GET /EntitySet(key))
- ✅ OData-compliant JSON responses with @odata.context
- ✅ Service document generation
- ✅ Basic metadata document
- ✅ Proper HTTP headers and error handling
- ✅ OData query operations ($filter, $select, $orderby)
- ✅ **Pagination support ($top, $skip, $count, @odata.nextLink)**
- 🔄 Complete metadata document generation - Coming soon
- 🔄 Entity relationship handling - Coming soon

## Installation

```bash
go get github.com/nlstn/go-odata
```

## Development Environment

### GitHub Codespaces

The easiest way to start developing is with GitHub Codespaces. Click the badge above or:

1. Click the "Code" button on the repository
2. Select the "Codespaces" tab
3. Click "Create codespace on main"

The development environment includes:
- Go 1.24 with all tools pre-installed
- VS Code with Go extension and language server
- golangci-lint for code quality
- Automatic dependency installation
- Pre-configured formatting and linting on save

### VS Code Dev Containers

Alternatively, you can use VS Code Dev Containers:

1. Install the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Open the repository in VS Code
3. Press `F1` and select "Dev Containers: Reopen in Container"

### Local Development

If you prefer to develop locally, ensure you have:
- Go 1.21 or later installed
- A GORM-compatible database driver (SQLite is used in examples)

## Quick Start

```go
package main

import (
    "log"
    
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
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
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
    
    // Start server
    service.ListenAndServe(":8080")
}
```

## Available Endpoints

Once your service is running, the following endpoints will be available:

- **Service Document**: `GET /` - Lists all available entity sets
- **Metadata**: `GET /$metadata` - OData metadata document
- **Entity Collection**: 
  - `GET /Products` - All products
  - `POST /Products` - Create a new product
- **Individual Entity**: 
  - `GET /Products(1)` - Product with ID 1
  - `PATCH /Products(1)` - Update product with ID 1
  - `DELETE /Products(1)` - Delete product with ID 1

## OData Query Options

The library supports the following OData v4 query options:

### Filtering (`$filter`)
Filter entities based on property values:
```
GET /Products?$filter=Price gt 100
GET /Products?$filter=Category eq 'Electronics'
GET /Products?$filter=contains(Name,'Laptop')
```

Supported operators: `eq`, `ne`, `gt`, `ge`, `lt`, `le`, `contains`, `startswith`, `endswith`

### Selection (`$select`)
Select specific properties to return:
```
GET /Products?$select=Name,Price
```

### Ordering (`$orderby`)
Sort results by one or more properties:
```
GET /Products?$orderby=Price desc
GET /Products?$orderby=Category asc,Price desc
```

### Pagination (`$top`, `$skip`)
Control the number of results returned:
```
GET /Products?$top=10              # Get first 10 products
GET /Products?$skip=10&$top=10     # Get products 11-20
```

When using `$top`, if more results are available, the response will include an `@odata.nextLink` with the URL for the next page:
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products",
  "@odata.nextLink": "http://localhost:8080/Products?$skip=10&$top=10",
  "value": [ /* ... */ ]
}
```

### Count (`$count`)
Get the total count of items matching the query:
```
GET /Products?$count=true          # Returns count with results
GET /Products?$filter=Price gt 100&$count=true
```

Response includes the total count:
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products",
  "@odata.count": 42,
  "value": [ /* ... */ ]
}
```

### Combining Query Options
You can combine multiple query options:
```
GET /Products?$filter=Category eq 'Electronics'&$orderby=Price desc&$top=5&$count=true
```

## Entity Definition

Define your entities using Go structs with appropriate tags:

```go
type Product struct {
    ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
    Name        string  `json:"Name" gorm:"not null" odata:"required"`
    Description string  `json:"Description"`
    Price       float64 `json:"Price" gorm:"not null"`
    Category    string  `json:"Category" gorm:"not null"`
}
```

### Supported Tags

- `odata:"key"` - Marks the field as the entity key (required)
- `odata:"required"` - Marks the field as required
- `json:"fieldname"` - Specifies the JSON field name
- `gorm:"..."` - GORM database tags

## Development Server

A development server is included for testing:

```bash
cd cmd/devserver
go run .
```

This starts a server on `http://localhost:8080` with sample Product data.

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

### Create Entity (`POST /Products`)

Request body:
```json
{
  "Name": "Mouse",
  "Description": "Wireless mouse",
  "Price": 29.99,
  "Category": "Accessories"
}
```

Response (201 Created):
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products/$entity",
  "ID": 2,
  "Name": "Mouse",
  "Description": "Wireless mouse",
  "Price": 29.99,
  "Category": "Accessories"
}
```

The response includes:
- Status: `201 Created`
- Header `Location`: URL of the created entity (e.g., `http://localhost:8080/Products(2)`)
- Header `OData-Version`: `4.0`
- Body: The created entity with all properties

## Requirements

- Go 1.21 or later
- GORM-compatible database driver

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Running Tests Locally

Before submitting a PR, make sure to:

1. Run all tests: `go test ./...`
2. Run tests with race detection: `go test -race ./...`
3. Format your code: `go fmt ./...`
4. Run go vet: `go vet ./...`

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.