# go-odata

[![CI](https://github.com/NLstn/go-odata/actions/workflows/ci.yml/badge.svg)](https://github.com/NLstn/go-odata/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nlstn/go-odata)](https://goreportcard.com/report/github.com/nlstn/go-odata)

A Go library for building services that expose OData APIs with automatic handling of OData logic.

## Overview

`go-odata` allows you to define Go structs representing entities and automatically handles the necessary OData protocol logic, making it easy to build OData-compliant APIs.

## Features

### Core OData Protocol Support
- ✅ Automatic OData endpoint generation from Go structs
- ✅ GORM database integration
- ✅ OData-compliant JSON responses with @odata.context
- ✅ Service document generation (GET /)
- ✅ Metadata document generation in both XML and JSON (CSDL) formats (GET /$metadata)
- ✅ Proper HTTP headers and error handling

### CRUD Operations
- ✅ Entity collection retrieval (GET /EntitySet)
- ✅ Individual entity retrieval (GET /EntitySet(key))
- ✅ Entity creation (POST /EntitySet)
- ✅ Entity update (PUT and PATCH /EntitySet(key))
- ✅ Entity deletion (DELETE /EntitySet(key))

### OData Query Options
- ✅ Filtering ($filter) with operators: eq, ne, gt, ge, lt, le, contains, startswith, endswith
- ✅ Selection ($select) - choose specific properties to return
- ✅ Ordering ($orderby) - sort by one or more properties
- ✅ Pagination ($top, $skip) with automatic @odata.nextLink generation
- ✅ Count ($count) - inline count with results or standalone count endpoint
- ✅ Expand ($expand) - retrieve related entities in a single request

### Advanced Features
- ✅ Composite keys support (e.g., /EntitySet(key1=value1,key2=value2))
- ✅ Navigation properties - access related entities (e.g., /Products(1)/Category)
- ✅ Structural properties with $value endpoint (e.g., /Products(1)/Name/$value)
- ✅ Prefer header support (return=representation, return=minimal)
- ✅ Filter operations on expanded navigation properties
- ✅ **Rich metadata document generation (XML and JSON)**
  - Property facets (MaxLength, Precision, Scale, DefaultValue, Nullable)
  - Extended type support (DateTimeOffset, Guid, Binary)
  - Navigation properties with referential constraints

## Installation

```bash
go get github.com/nlstn/go-odata
```

## Development Environment

### GitHub Codespaces

The easiest way to start developing is with GitHub Codespaces:

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
- **Metadata**: `GET /$metadata` - OData metadata document (supports both XML and JSON/CSDL formats)
- **Metadata (JSON)**: `GET /$metadata?$format=json` - OData metadata document in JSON format (CSDL JSON)
- **Entity Collection**: 
  - `GET /Products` - All products
  - `POST /Products` - Create a new product
- **Individual Entity**: 
  - `GET /Products(1)` - Product with ID 1
  - `PUT /Products(1)` - Replace product with ID 1 (complete replacement)
  - `PATCH /Products(1)` - Update product with ID 1 (partial update)
  - `DELETE /Products(1)` - Delete product with ID 1
- **Count Endpoint**: `GET /Products/$count` - Get total count of products (supports filtering)
- **Navigation Properties**: `GET /Products(1)/Category` - Access related entities
- **Structural Properties**: `GET /Products(1)/Name` - Access individual property values
- **Raw Property Value**: `GET /Products(1)/Name/$value` - Get raw property value without JSON wrapping
- **Composite Keys**: `GET /EntitySet(key1=value1,key2=value2)` - Access entities with composite keys

## Metadata Document

The library generates rich OData v4 metadata documents that describe your data model. Metadata is available in both XML and JSON formats.

### Accessing Metadata

```bash
# XML format (default)
GET http://localhost:8080/$metadata

# JSON format (CSDL JSON)
GET http://localhost:8080/$metadata?$format=json
```

### Metadata Features

The metadata document includes:

- **Entity Types**: Complete type definitions for all registered entities
- **Property Facets**: 
  - `MaxLength` - Maximum string length
  - `Precision` and `Scale` - Numeric precision for decimals
  - `DefaultValue` - Default values for properties
  - `Nullable` - Nullability constraints
- **Type Mappings**:
  - `time.Time` → `Edm.DateTimeOffset`
  - `int`, `int32` → `Edm.Int32`
  - `int64` → `Edm.Int64`
  - `float64` → `Edm.Double`
  - `bool` → `Edm.Boolean`
  - `[]byte` → `Edm.Binary`
- **Navigation Properties**: Relationship definitions with referential constraints
- **Entity Container**: Entity sets and navigation property bindings

### Example Metadata (JSON)

```json
{
  "$Version": "4.0",
  "ODataService": {
    "Product": {
      "$Kind": "EntityType",
      "$Key": ["id"],
      "id": { "$Type": "Edm.Int32" },
      "name": { 
        "$Type": "Edm.String", 
        "$MaxLength": 100 
      },
      "price": { 
        "$Type": "Edm.Double",
        "$Precision": 10,
        "$Scale": 2
      },
      "createdAt": { 
        "$Type": "Edm.DateTimeOffset" 
      }
    },
    "Order": {
      "$Kind": "EntityType",
      "$Key": ["id"],
      "customer": {
        "$Kind": "NavigationProperty",
        "$Type": "ODataService.Customer",
        "$ReferentialConstraint": [{
          "Property": "CustomerID",
          "ReferencedProperty": "ID"
        }]
      }
    },
    "Container": {
      "$Kind": "EntityContainer",
      "Products": {
        "$Collection": true,
        "$Type": "ODataService.Product"
      }
    }
  }
}
```

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

### Basic Entity

```go
type Product struct {
    ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
    Name        string  `json:"Name" gorm:"not null" odata:"required"`
    Description string  `json:"Description"`
    Price       float64 `json:"Price" gorm:"not null"`
    Category    string  `json:"Category" gorm:"not null"`
}
```

### Entity with Rich Metadata

```go
type Product struct {
    ID          int       `json:"id" gorm:"primaryKey" odata:"key"`
    Name        string    `json:"name" odata:"required,maxlength=100"`
    Description string    `json:"description" odata:"maxlength=1000,nullable"`
    SKU         string    `json:"sku" odata:"maxlength=50,default=AUTO"`
    Price       float64   `json:"price" odata:"precision=10,scale=2"`
    Stock       int       `json:"stock" odata:"default=0"`
    Active      bool      `json:"active" odata:"default=true"`
    CreatedAt   time.Time `json:"createdAt"`
}
```

### Entity with Relationships

```go
type Order struct {
    ID          int       `json:"id" gorm:"primaryKey" odata:"key"`
    OrderNumber string    `json:"orderNumber" odata:"required,maxlength=50"`
    CustomerID  int       `json:"customerId" odata:"required"`
    Customer    *Customer `json:"customer" gorm:"foreignKey:CustomerID;references:ID"`
    TotalAmount float64   `json:"totalAmount" odata:"precision=10,scale=2"`
    OrderDate   time.Time `json:"orderDate"`
    Items       []OrderItem `json:"items" gorm:"foreignKey:OrderID;references:ID"`
}

type Customer struct {
    ID     int     `json:"id" gorm:"primaryKey" odata:"key"`
    Name   string  `json:"name" odata:"required,maxlength=100"`
    Email  string  `json:"email" odata:"maxlength=255"`
    Orders []Order `json:"orders" gorm:"foreignKey:CustomerID;references:ID"`
}
```

### Supported Tags

- `odata:"key"` - Marks the field as the entity key (required)
- `odata:"required"` - Marks the field as required
- `odata:"maxlength=N"` - Sets the maximum length for string properties
- `odata:"precision=N"` - Sets the precision for numeric properties
- `odata:"scale=N"` - Sets the scale for decimal properties
- `odata:"default=VALUE"` - Sets the default value for the property
- `odata:"nullable"` - Explicitly marks the field as nullable
- `odata:"nullable=false"` - Explicitly marks the field as non-nullable
- `json:"fieldname"` - Specifies the JSON field name
- `gorm:"..."` - GORM database tags (including foreign key relationships)

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

### Update Entity (`PUT /Products(1)` vs `PATCH /Products(1)`)

The library supports both PUT and PATCH for updating entities, following OData v4 specifications:

**PUT - Complete Replacement:**
- Replaces the entire entity
- All properties not included in the request are set to their default values
- Returns `204 No Content` on success

Request body (PUT):
```json
{
  "Name": "Gaming Laptop",
  "Price": 1499.99
}
```
Result: Name and Price are updated, but Description and Category are set to empty strings (defaults).

**PATCH - Partial Update:**
- Updates only the properties included in the request
- Other properties remain unchanged
- Returns `204 No Content` on success

Request body (PATCH):
```json
{
  "Price": 1499.99
}
```
Result: Only Price is updated, all other properties remain unchanged.

Both methods:
- Require the entity to exist (404 if not found)
- Cannot modify key properties
- Return proper OData v4 headers

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