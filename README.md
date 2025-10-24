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
- ✅ **Advanced Filtering ($filter)** - AST-based parser with full OData v4 support
  - Comparison operators: `eq`, `ne`, `gt`, `ge`, `lt`, `le`
  - String functions: `contains`, `startswith`, `endswith`, `tolower`, `toupper`, `trim`, `length`, `indexof`, `substring`, `concat`
  - Date functions: `year`, `month`, `day`, `hour`, `minute`, `second`, `date`, `time`
  - Type functions: `cast`, `isof` - type conversion and type checking
  - Boolean operators: `and`, `or`, `not`
  - Parentheses for complex expressions
  - Literal types: strings, numbers, booleans, null
  - Basic arithmetic operators: `+`, `-`, `*`, `/`, `mod`
  - **Lambda operators**: `any`, `all` - collection filtering with range variables
- ✅ **Search ($search)** - database-agnostic full-text search with fuzzy matching
  - Case-insensitive substring search across entity properties
  - Configurable searchable fields using `odata:"searchable"` tag
  - Fuzzy matching support with customizable fuzziness level
  - Defaults to searching all string fields if no searchable fields defined
- ✅ Selection ($select) - choose specific properties to return
- ✅ **Computed Properties ($compute)** - extract and compute properties from data
  - Date extraction functions: `year`, `month`, `day`, `hour`, `minute`, `second`, `date`, `time`
  - Use with $select to return only computed properties
- ✅ Ordering ($orderby) - sort by one or more properties
- ✅ **Pagination ($top, $skip)** with automatic @odata.nextLink generation
  - Server-driven paging with `odata.maxpagesize` preference header
  - Stable pagination with `$skiptoken` for ordered queries
- ✅ Count ($count) - inline count with results or standalone count endpoint
- ✅ **Expand ($expand)** - retrieve related entities with nested query options
  - Nested $filter, $select, $orderby, $top, $skip on expanded properties
  - Complex filters on expanded navigation properties
- ✅ **Data Aggregation ($apply)** - analytical queries with transformations
  - `groupby` - group results by one or more properties
  - `aggregate` - perform aggregations (sum, average, min, max, count, countdistinct)
  - `filter` - apply filters before transformations
  - Multiple transformations in sequence (pipeline)

### Advanced Features
- ✅ Composite keys support (e.g., /EntitySet(key1=value1,key2=value2))
- ✅ **Singletons** - single entity instances accessible by name
  - Direct access without keys (e.g., /Company instead of /Companies(1))
  - Full CRUD support (GET, PATCH, PUT)
  - Automatic metadata and service document generation
  - ETag support for optimistic concurrency control
- ✅ Navigation properties - access related entities (e.g., /Products(1)/Category)
- ✅ Structural properties with $value endpoint (e.g., /Products(1)/Name/$value)
- ✅ **Prefer header support** 
  - `return=representation`, `return=minimal` for CRUD operations
  - `odata.maxpagesize` for server-driven paging
- ✅ **OData-MaxVersion header support** - version negotiation and validation
  - Rejects requests with versions below 4.0 (returns 406 Not Acceptable)
  - Accepts version 4.0 and above
  - Compliant with OData v4 specification
- ✅ **OData-EntityId header** - returned in 204 No Content responses for POST, PUT, and PATCH operations
- ✅ **ETag support with If-Match headers** - optimistic concurrency control
- ✅ Filter operations on expanded navigation properties
- ✅ **Rich metadata document generation (XML and JSON)**
  - Property facets (MaxLength, Precision, Scale, DefaultValue, Nullable)
  - Extended type support (DateTimeOffset, Guid, Binary)
  - Navigation properties with referential constraints
- ✅ Proper snake_case database column mapping for all operations
- ✅ **Batch requests ($batch)** - group multiple operations in a single HTTP request
  - Support for multipart/mixed format (OData v4 standard)
  - Changesets for atomic operations (transaction support)
  - Mix read and write operations in a single batch
- ✅ **Actions and Functions** - custom operations beyond CRUD
  - Bound and unbound actions (POST)
  - Bound and unbound functions (GET)
  - Parameter validation and type conversion
  - Support for return values and void operations

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
    
    // Create HTTP mux and register the OData service as a handler
    mux := http.NewServeMux()
    mux.Handle("/", service)
    
    // Start server
    log.Println("Starting OData service on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

### Alternative: Using Service as Handler Directly

The `Service` implements `http.Handler`, so you can also use it directly:

```go
// Option 1: Use the Handler() method
mux.Handle("/", service.Handler())

// Option 2: Use the service directly (equivalent)
mux.Handle("/", service)
```

### Mounting at a Custom Path

You can mount the OData service at a custom path prefix:

```go
// Mount OData service at /api/odata/
mux.Handle("/api/odata/", http.StripPrefix("/api/odata", service))

// Now access via http://localhost:8080/api/odata/Products
```

### Adding Other Handlers

Since the OData service is just a standard HTTP handler, you can easily combine it with other handlers:

```go
mux := http.NewServeMux()

// OData service at root
mux.Handle("/", service)

// Health check endpoint
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})

// Metrics endpoint
mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
    // Your metrics logic
})

log.Fatal(http.ListenAndServe(":8080", mux))
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
- **Singletons**: 
  - `GET /Company` - Access singleton entity directly by name
  - `PATCH /Company` - Update singleton (partial update)
  - `PUT /Company` - Replace singleton (full update)
- **Batch Requests**: `POST /$batch` - Execute multiple operations in a single HTTP request
- **Functions (Unbound)**: `GET /FunctionName?param=value` - Invoke custom read-only operations
- **Functions (Bound)**: `GET /Products(1)/FunctionName?param=value` - Invoke functions on specific entities
- **Actions (Unbound)**: `POST /ActionName` - Invoke custom operations that may modify data
- **Actions (Bound)**: `POST /Products(1)/ActionName` - Invoke actions on specific entities

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

The library supports comprehensive OData v4 filter expressions with an AST-based parser:

#### Basic Comparison Operators
```
GET /Products?$filter=Price gt 100
GET /Products?$filter=Category eq 'Electronics'
GET /Products?$filter=Price ne 0
GET /Products?$filter=Price ge 100
GET /Products?$filter=Price le 1000
GET /Products?$filter=Price lt 50
```

Supported operators: `eq`, `ne`, `gt`, `ge`, `lt`, `le`

#### String Functions
```
# Search functions
GET /Products?$filter=contains(Name,'Laptop')
GET /Products?$filter=startswith(Category,'Elec')
GET /Products?$filter=endswith(Name,'Pro')

# Case transformation
GET /Products?$filter=tolower(Name) eq 'laptop pro'
GET /Products?$filter=toupper(Category) eq 'ELECTRONICS'

# String manipulation
GET /Products?$filter=trim(Description) ne ''
GET /Products?$filter=length(Name) gt 10
GET /Products?$filter=indexof(Name,'Pro') gt 0
GET /Products?$filter=substring(Name,0,3) eq 'Lap'
GET /Products?$filter=substring(Name,1,5) eq 'aptop'
GET /Products?$filter=concat(Name,' Edition') eq 'Laptop Pro Edition'
```

Supported functions:
- **Search**: `contains`, `startswith`, `endswith`
- **Case**: `tolower`, `toupper`
- **Manipulation**: `trim`, `length`, `indexof`, `substring`, `concat`

#### Date Functions
```
# Date component extraction
GET /Orders?$filter=year(OrderDate) eq 2024
GET /Orders?$filter=month(OrderDate) eq 12
GET /Orders?$filter=day(OrderDate) eq 25

# Time component extraction
GET /Orders?$filter=hour(OrderDate) eq 14
GET /Orders?$filter=minute(OrderDate) eq 30
GET /Orders?$filter=second(OrderDate) eq 0

# Date and time parts
GET /Orders?$filter=date(OrderDate) eq '2024-12-25'
GET /Orders?$filter=time(OrderDate) eq '14:30:00'

# Complex date queries
GET /Orders?$filter=year(OrderDate) eq 2024 and month(OrderDate) ge 6
GET /Orders?$filter=date(OrderDate) ge '2024-01-01' and date(OrderDate) le '2024-12-31'
```

Supported date functions:
- **Component extraction**: `year`, `month`, `day`, `hour`, `minute`, `second`
- **Date/time parts**: `date`, `time`

#### Type Functions
```
# Type conversion with cast
GET /Products?$filter=cast(Price, 'Edm.String') eq '100'
GET /Products?$filter=cast(Price, 'Edm.Int32') gt 50
GET /Products?$filter=cast(Name, 'Edm.String') eq 'Laptop'

# Type checking with isof
GET /Products?$filter=isof(Price, 'Edm.Decimal') eq true
GET /Products?$filter=isof(Price, 'Edm.Int32') eq true and Price gt 100
GET /Products?$filter=isof('Edm.String') eq true

# Combined usage
GET /Products?$filter=isof(Price, 'Edm.Int32') eq true and cast(Price, 'Edm.String') eq '100'
```

Supported type functions:
- **`cast(property, 'TypeName')`**: Converts a property to the specified EDM type
- **`isof(property, 'TypeName')`**: Returns true if the property is of the specified type
- **`isof('TypeName')`**: Returns true if the current instance is of the specified type

Supported EDM types: `Edm.String`, `Edm.Int32`, `Edm.Int64`, `Edm.Decimal`, `Edm.Double`, `Edm.Single`, `Edm.Boolean`, `Edm.DateTimeOffset`, `Edm.Date`, `Edm.TimeOfDay`, `Edm.Guid`, `Edm.Binary`, `Edm.Byte`, `Edm.SByte`, `Edm.Int16`

#### Boolean Logic with Parentheses
```
GET /Products?$filter=(Price gt 100 and Category eq 'Electronics') or (Price lt 50 and Category eq 'Books')
GET /Products?$filter=((Price gt 1000 or Price lt 50) and IsAvailable eq true)
```

Parentheses can be nested to create complex boolean expressions with proper operator precedence.

#### NOT Operator
```
GET /Products?$filter=not (Category eq 'Books')
GET /Products?$filter=not (Price gt 1000) and IsAvailable eq true
GET /Products?$filter=contains(Name,'Laptop') and not (Category eq 'Used')
```

The `not` operator negates the expression that follows it.

#### Literal Types
```
GET /Products?$filter=IsAvailable eq true          # Boolean
GET /Products?$filter=IsAvailable eq false         # Boolean
GET /Products?$filter=Price eq 99.99               # Numeric (decimal)
GET /Products?$filter=Quantity eq 42               # Numeric (integer)
GET /Products?$filter=Category eq 'Electronics'    # String
GET /Products?$filter=Description eq null          # Null
```

The parser properly handles different literal types including booleans, numbers (integers and decimals), strings, and null.

#### Arithmetic Operators (Basic Support)
```
GET /Products?$filter=Quantity mod 2 eq 0          # Modulo
```

Basic arithmetic operators (`+`, `-`, `*`, `/`, `mod`) are supported in simple expressions.

#### Lambda Operators (`any`, `all`)

Lambda operators allow filtering based on collection properties. They support range variables to refer to individual elements in the collection.

**any() operator** - Returns true if any element in a collection satisfies the condition:
```
# Check if any tag equals 'Electronics'
GET /Products?$filter=Tags/any(t: t eq 'Electronics')

# Check if any order has total greater than 100
GET /Products?$filter=Orders/any(o: o/Total gt 100)

# Parameterless any - checks if collection is non-empty
GET /Products?$filter=Tags/any()

# Any with complex conditions
GET /Products?$filter=Orders/any(o: o/Total gt 100 and o/Status eq 'Completed')

# Any with string functions
GET /Products?$filter=Tags/any(t: contains(t, 'tech'))
```

**all() operator** - Returns true if all elements in a collection satisfy the condition:
```
# Check if all tags contain 'Certified'
GET /Products?$filter=Tags/all(t: contains(t, 'Certified'))

# Check if all orders are completed
GET /Products?$filter=Orders/all(o: o/Status eq 'Completed')

# Parameterless all - checks if collection is empty (vacuous truth)
GET /Products?$filter=Tags/all()
```

**Nested lambda expressions**:
```
# Check if any order has any item with price greater than 50
GET /Products?$filter=Orders/any(o: o/Items/any(i: i/Price gt 50))

# Check if all orders have all items in stock
GET /Products?$filter=Orders/all(o: o/Items/all(i: i/InStock eq true))

# Mixed any and all
GET /Products?$filter=Orders/any(o: o/Items/all(i: i/Status eq 'Valid'))
```

**Combining lambda with other operators**:
```
# Lambda with AND
GET /Products?$filter=Tags/any(t: t eq 'Electronics') and Price gt 100

# Lambda with OR
GET /Products?$filter=Tags/any(t: t eq 'Sale') or Price lt 50

# NOT with lambda
GET /Products?$filter=not (Tags/any(t: t eq 'Discontinued'))

# Multiple lambda expressions
GET /Products?$filter=Tags/any(t: t eq 'Electronics') and Categories/any(c: c/Name eq 'Computers')
```

**Lambda features**:
- **Range variables**: Use short variable names (e.g., `t:`, `o:`, `i:`) to refer to collection elements
- **Property access**: Access properties of collection elements using `/` (e.g., `o/Total`, `i/Price`)
- **Complex conditions**: Use `and`, `or`, `not`, and parentheses within lambda predicates
- **Functions**: Use any filter function within lambda predicates (e.g., `contains`, `startswith`)
- **Nesting**: Lambda expressions can be nested for multi-level collection filtering
- **Parameterless**: Use `any()` without predicate to check if collection is non-empty

#### Complex Filter Examples
```
# Multiple conditions with functions and NOT
GET /Products?$filter=(contains(Name,'Laptop') or contains(Name,'Computer')) and Price gt 500 and not (Category eq 'Used')

# Deep nesting with boolean logic
GET /Products?$filter=((Price gt 100 and not (Category eq 'Books')) or contains(Name,'Special')) and IsAvailable eq true

# Combining multiple operators and literals
GET /Products?$filter=Price gt 100.0 and Price lt 1000.0 and IsAvailable eq true and Category ne 'Luxury'
```

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

### Server-Driven Paging

The library supports OData v4 server-driven paging features:

#### `odata.maxpagesize` Preference
Control the maximum page size using the `Prefer` header:
```bash
GET /Products HTTP/1.1
Host: localhost:8080
Prefer: odata.maxpagesize=50
```

The server will limit results to the specified page size and include a `Preference-Applied` response header:
```
HTTP/1.1 200 OK
Preference-Applied: odata.maxpagesize=50
OData-Version: 4.0
Content-Type: application/json;odata.metadata=minimal
```

If both `odata.maxpagesize` and `$top` are specified, the smaller value is used.

#### `$skiptoken` for Stable Paging
When using `$orderby`, the library generates `$skiptoken`-based pagination instead of `$skip`:
```
GET /Products?$top=10&$orderby=Price
```

Response includes a `$skiptoken` in the `@odata.nextLink`:
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products",
  "@odata.nextLink": "http://localhost:8080/Products?$skiptoken=eyJrIjp7IklEIjoxMH0sIm8iOnsiUHJpY2UiOjk5Ljk5fX0%3D&$top=10&$orderby=Price",
  "value": [ /* ... */ ]
}
```

The `$skiptoken` encodes the position in the result set using the last entity's key and orderby values, providing:
- Stable pagination even when data changes between requests
- Better performance for large offsets compared to `$skip`
- Proper handling of ordered queries

**Note:** `$skiptoken` and `$skip` are mutually exclusive and cannot be used together.

### Computed Properties (`$compute`)

Extract and compute new properties from existing data using date functions and other operations:

```
# Extract date components
GET /Orders?$compute=year(OrderDate) as OrderYear&$select=OrderNo,OrderYear
GET /Orders?$compute=month(OrderDate) as Month,day(OrderDate) as Day&$select=OrderNo,Month,Day

# Extract time components
GET /Orders?$compute=hour(OrderDate) as Hour,minute(OrderDate) as Minute&$select=OrderNo,Hour,Minute

# Extract date and time parts
GET /Orders?$compute=date(OrderDate) as DatePart,time(OrderDate) as TimePart&$select=OrderNo,DatePart,TimePart

# Combine with filters
GET /Orders?$compute=year(OrderDate) as Year&$filter=year(OrderDate) eq 2024&$select=OrderNo,Year
```

Supported date extraction functions in `$compute`:
- **year(property)** - Extract year as integer
- **month(property)** - Extract month (1-12) as integer
- **day(property)** - Extract day of month (1-31) as integer
- **hour(property)** - Extract hour (0-23) as integer
- **minute(property)** - Extract minute (0-59) as integer
- **second(property)** - Extract second (0-59) as integer
- **date(property)** - Extract date part (without time)
- **time(property)** - Extract time part (without date)

Response example:
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Orders",
  "value": [
    {
      "OrderNo": "ORD001",
      "Year": 2024,
      "Month": 12,
      "Day": 25
    }
  ]
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

### Search (`$search`)

The `$search` query option provides database-agnostic full-text search capabilities with fuzzy matching support. Search is performed in-memory after fetching results, making it independent of the underlying database.

#### Basic Search

```
GET /Products?$search=laptop        # Case-insensitive search
GET /Products?$search=gaming        # Search across searchable fields
```

#### Configuring Searchable Fields

By default, all string properties are searchable. You can control which fields are searchable using the `odata:"searchable"` tag:

```go
type Product struct {
    ID          int     `json:"ID" odata:"key"`
    Name        string  `json:"Name" odata:"searchable"`           // Searchable
    Description string  `json:"Description" odata:"searchable"`    // Searchable
    Category    string  `json:"Category"`                          // Not searchable
    Price       float64 `json:"Price"`                             // Not searchable
}
```

#### Fuzzy Matching

You can configure the fuzziness level for each searchable field. Fuzziness determines how tolerant the search is to character differences:

- **fuzziness=1** (default): Exact substring match (case-insensitive)
- **fuzziness=2+**: Allows character differences based on Levenshtein distance

```go
type Product struct {
    ID    int    `json:"ID" odata:"key"`
    Name  string `json:"Name" odata:"searchable,fuzziness=1"`   // Exact match
    Email string `json:"Email" odata:"searchable,fuzziness=2"`  // 1 char difference allowed
    Tags  string `json:"Tags" odata:"searchable,fuzziness=3"`   // 2 char differences allowed
}
```

#### Combining with Other Query Options

Search can be combined with other query options:

```
GET /Products?$search=laptop&$filter=Price gt 500&$top=10
GET /Products?$search=gaming&$orderby=Price desc
GET /Products?$search=wireless&$select=Name,Price
```

#### How It Works

1. The search query is applied after fetching results from the database
2. Search is case-insensitive by default
3. If no fields are marked as `searchable`, all string properties are searched
4. Each entity is checked against the search query across all searchable fields
5. Fuzzy matching is applied based on the configured fuzziness level

### Batch Requests (`$batch`)

Batch requests allow you to group multiple operations into a single HTTP request, reducing network overhead and improving performance. The library supports OData v4 batch processing using the `multipart/mixed` format.

#### Basic Batch Request

```bash
POST /$batch HTTP/1.1
Content-Type: multipart/mixed; boundary=batch_36d5c8c6
```

```
--batch_36d5c8c6
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /Products(1) HTTP/1.1
Host: localhost
Accept: application/json


--batch_36d5c8c6
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /Products(2) HTTP/1.1
Host: localhost
Accept: application/json


--batch_36d5c8c6--
```

#### Changesets for Atomic Operations

Group write operations (POST, PUT, PATCH, DELETE) into changesets for atomic transaction support. If any operation in a changeset fails, all operations in that changeset are rolled back.

```
--batch_36d5c8c6
Content-Type: multipart/mixed; boundary=changeset_77162fcd

--changeset_77162fcd
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /Products HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 1","Price":10.00,"Category":"Electronics"}

--changeset_77162fcd
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /Products HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 2","Price":20.00,"Category":"Books"}

--changeset_77162fcd--

--batch_36d5c8c6--
```

#### Mixed Read and Write Operations

You can mix read operations (GET) outside changesets with write operations inside changesets in a single batch:

```
--batch_36d5c8c6
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /Products(1) HTTP/1.1
Host: localhost
Accept: application/json


--batch_36d5c8c6
Content-Type: multipart/mixed; boundary=changeset_77162fcd

--changeset_77162fcd
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /Products HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"New Product","Price":100.00,"Category":"Books"}

--changeset_77162fcd--

--batch_36d5c8c6
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /Products HTTP/1.1
Host: localhost
Accept: application/json


--batch_36d5c8c6--
```

#### Batch Response Format

The server responds with a `multipart/mixed` response containing the results of each operation:

```
HTTP/1.1 200 OK
Content-Type: multipart/mixed; boundary=batchresponse_36d5c8c6
OData-Version: 4.0

--batchresponse_36d5c8c6
Content-Type: application/http
Content-Transfer-Encoding: binary

HTTP/1.1 200 OK
Content-Type: application/json

{"ID":1,"Name":"Product 1","Price":99.99}

--batchresponse_36d5c8c6
Content-Type: application/http
Content-Transfer-Encoding: binary

HTTP/1.1 201 Created
Content-Type: application/json

{"ID":2,"Name":"New Product","Price":100.00}

--batchresponse_36d5c8c6--
```

#### Batch Features

- ✅ Multiple GET requests in a single batch
- ✅ Changesets for atomic write operations (POST, PUT, PATCH, DELETE)
- ✅ Mix read and write operations
- ✅ Transaction support for changesets
- ✅ Individual error handling per request
- ✅ OData v4 compliant multipart/mixed format

### Expand (`$expand`)
Retrieve related entities in a single request:
```
GET /Products?$expand=Category
GET /Authors?$expand=Books
```

#### Nested Query Options on Expand
You can apply query options to expanded navigation properties:
```
# Select specific properties from expanded entity
GET /Authors?$expand=Books($select=Title,Price)

# Filter expanded entities
GET /Authors?$expand=Books($filter=Price gt 50)

# Order expanded entities
GET /Authors?$expand=Books($orderby=Title asc)

# Paginate expanded entities
GET /Authors?$expand=Books($top=5;$skip=2)

# Combine multiple nested options
GET /Authors?$expand=Books($filter=Price gt 50;$orderby=Price desc;$top=10)
```

#### Advanced Filters on Expanded Properties
All filter features work in nested expand filters:
```
# Parentheses in nested filters
GET /Authors?$expand=Books($filter=(Price gt 50 and Category eq 'Fiction') or (Price lt 20 and Category eq 'NonFiction'))

# NOT operator in nested filters
GET /Authors?$expand=Books($filter=not (Category eq 'OutOfPrint'))

# Functions with complex logic in nested filters
GET /Authors?$expand=Books($filter=contains(Title,'Guide') and not (Price gt 100))
```

#### Multiple Expand
Expand multiple navigation properties:
```
GET /Products?$expand=Category,Reviews
GET /Authors?$expand=Books,Publisher
```

### Data Aggregation (`$apply`)

The library supports OData v4 data aggregation transformations for analytical queries:

#### GroupBy Transformation
Group results by one or more properties:
```
# Group by single property
GET /Products?$apply=groupby((Category))

# Group by multiple properties
GET /Products?$apply=groupby((Category,Name))

# Group by with aggregation
GET /Products?$apply=groupby((Category),aggregate(Price with sum as TotalPrice))
```

#### Aggregate Transformation
Perform aggregations on data:
```
# Sum aggregation
GET /Products?$apply=aggregate(Price with sum as TotalPrice)

# Count aggregation
GET /Products?$apply=aggregate($count as Total)

# Average aggregation
GET /Products?$apply=aggregate(Price with average as AvgPrice)

# Min/Max aggregation
GET /Products?$apply=aggregate(Price with min as MinPrice,Price with max as MaxPrice)

# Multiple aggregations
GET /Products?$apply=aggregate(Price with sum as TotalPrice,Quantity with sum as TotalQuantity)
```

Supported aggregation methods:
- **sum** - Sum of values
- **average** or **avg** - Average of values
- **min** - Minimum value
- **max** - Maximum value
- **count** - Count of rows (use `$count as alias`)
- **countdistinct** - Count of distinct values

#### Filter Transformation
Apply filters before other transformations:
```
# Filter then group
GET /Products?$apply=filter(Price gt 100)/groupby((Category))

# Filter then aggregate
GET /Products?$apply=filter(Category eq 'Electronics')/aggregate(Price with sum as Total)
```

#### Combining Transformations
Chain multiple transformations using `/`:
```
# Filter, then group by, then aggregate
GET /Products?$apply=filter(Price gt 100)/groupby((Category),aggregate(Price with sum as TotalPrice))

# Complex transformation pipeline
GET /Products?$apply=filter(IsAvailable eq true)/groupby((Category),aggregate(Price with average as AvgPrice,$count as Count))
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
- `odata:"etag"` - Marks the field to be used for ETag generation (optimistic concurrency control)
- `odata:"required"` - Marks the field as required
- `odata:"maxlength=N"` - Sets the maximum length for string properties
- `odata:"precision=N"` - Sets the precision for numeric properties
- `odata:"scale=N"` - Sets the scale for decimal properties
- `odata:"default=VALUE"` - Sets the default value for the property
- `odata:"nullable"` - Explicitly marks the field as nullable
- `odata:"nullable=false"` - Explicitly marks the field as non-nullable
- `json:"fieldname"` - Specifies the JSON field name
- `gorm:"..."` - GORM database tags (including foreign key relationships)

## Singletons

Singletons are special entities in OData v4 that represent a single instance of an entity type, accessible directly by name without requiring a key. They are useful for representing unique resources like company information, application settings, or user profiles.

### Defining a Singleton

```go
type CompanyInfo struct {
    ID          uint      `json:"ID" gorm:"primaryKey" odata:"key"`
    Name        string    `json:"Name" gorm:"not null" odata:"required"`
    CEO         string    `json:"CEO" gorm:"not null"`
    Founded     int       `json:"Founded"`
    HeadQuarter string    `json:"HeadQuarter"`
    Version     int       `json:"Version" gorm:"default:1" odata:"etag"`
}
```

### Registering a Singleton

```go
service := odata.NewService(db)

// Register as a singleton
err := service.RegisterSingleton(&CompanyInfo{}, "Company")
if err != nil {
    log.Fatal(err)
}
```

### Accessing Singletons

Singletons are accessed directly by their name without keys:

```bash
# Get singleton
GET /Company

# Update singleton (partial update)
PATCH /Company
Content-Type: application/json
{ "CEO": "New CEO Name" }

# Replace singleton (full update)
PUT /Company
Content-Type: application/json
{
  "Name": "Updated Company Name",
  "CEO": "New CEO",
  "Founded": 1990,
  "HeadQuarter": "New Location"
}
```

### Singleton Features

- **Direct Access**: No key required - access via `/Company` instead of `/Companies(1)`
- **Full CRUD Support**: Supports GET, PATCH, and PUT operations (POST and DELETE are not applicable)
- **Metadata Integration**: Automatically appears in service document and metadata with `$Kind: "Singleton"`
- **ETag Support**: Full support for optimistic concurrency control using ETag
- **Prefer Header**: Supports `return=representation` and `return=minimal` preferences
- **Navigation Properties**: Singletons can have navigation properties to other entities

### Singleton vs Entity Set

| Feature | Entity Set | Singleton |
|---------|-----------|-----------|
| URL Pattern | `/EntitySet(key)` | `/SingletonName` |
| Key Required | Yes | No |
| Multiple Instances | Yes | No |
| POST (Create) | ✅ Supported | ❌ Not applicable |
| GET (Read) | ✅ Supported | ✅ Supported |
| PATCH (Update) | ✅ Supported | ✅ Supported |
| PUT (Replace) | ✅ Supported | ✅ Supported |
| DELETE | ✅ Supported | ❌ Not applicable |
| Use Case | Multiple entities | Single global entity |

### Example: Service Document with Singleton

When you register a singleton, it automatically appears in the service document:

```json
{
  "@odata.context": "http://localhost:8080/$metadata",
  "value": [
    {
      "name": "Products",
      "kind": "EntitySet",
      "url": "Products"
    },
    {
      "name": "Company",
      "kind": "Singleton",
      "url": "Company"
    }
  ]
}
```

## Development Servers

Two servers are included in this repository:

### Development Server (`cmd/devserver`)
A full-featured development server with demo functionality:

```bash
cd cmd/devserver
go run .
```

This starts a server on `http://localhost:8080` with:
- Sample Product, Category, and User data
- Custom authentication middleware (demo only)
- Example actions and functions
- Lifecycle hooks (BeforeCreate, BeforeUpdate)

### Compliance Server (`cmd/complianceserver`)
A minimal server for OData compliance testing:

```bash
cd cmd/complianceserver
go run .
```

This starts a server on `http://localhost:9090` with:
- Minimal entities (Products, Categories, ProductDescriptions, Company singleton)
- No custom middleware or hooks
- Only essential Reseed action for testing
- Used by the compliance test suite in `compliance/v4/`

See [cmd/complianceserver/README.md](cmd/complianceserver/README.md) for more details.

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

When using `Prefer: return=minimal` header:
- Status: `204 No Content`
- Header `Location`: URL of the created entity
- Header `OData-EntityId`: Canonical URL of the created entity (same as Location)
- Header `OData-Version`: `4.0`
- No body content

### Update Entity (`PUT /Products(1)` vs `PATCH /Products(1)`)

The library supports both PUT and PATCH for updating entities, following OData v4 specifications:

**PUT - Complete Replacement:**
- Replaces the entire entity
- All properties not included in the request are set to their default values
- Returns `204 No Content` on success (with `OData-EntityId` header)

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
- Returns `204 No Content` on success (with `OData-EntityId` header)

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

## ETag Support (Optimistic Concurrency Control)

The library supports ETags for optimistic concurrency control, allowing you to prevent concurrent updates from overwriting each other's changes.

### Defining an ETag Property

Mark a field in your entity with the `odata:"etag"` tag. This field will be used to generate ETags:

```go
type Product struct {
    ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
    Name     string  `json:"Name" odata:"required"`
    Price    float64 `json:"Price"`
    Version  int     `json:"Version" odata:"etag"` // Used for ETag generation
}
```

You can use any field type for ETags:
- **Integer fields** (e.g., `Version int`) - Common pattern for version numbers
- **Timestamp fields** (e.g., `LastModified time.Time`) - Tracks last modification time
- **String fields** - Custom version identifiers

### How ETags Work

1. **GET requests** return an `ETag` header and `@odata.etag` annotation in the response body with a hash of the ETag field value
2. **Clients** store the ETag value and send it back in an `If-Match` header when updating
3. **UPDATE/DELETE operations** validate that the `If-Match` header matches the current ETag
4. If ETags don't match, a `412 Precondition Failed` response is returned

### Example: Using ETags

**Step 1: Retrieve an entity (GET)**
```bash
GET /Products(1)
```

Response headers:
```
HTTP/1.1 200 OK
ETag: W/"abc123def456..."
OData-Version: 4.0
Content-Type: application/json
```

Response body:
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products/$entity",
  "@odata.etag": "W/\"abc123def456...\"",
  "ID": 1,
  "Name": "Laptop",
  "Price": 999.99,
  "Version": 1
}
```

Note: The `@odata.etag` annotation in the response body is included per OData v4 specification, in addition to the `ETag` HTTP header.

**Step 2: Update the entity with If-Match header (PATCH)**
```bash
PATCH /Products(1)
If-Match: W/"abc123def456..."
Content-Type: application/json

{
  "Price": 899.99
}
```

If the ETag matches:
```
HTTP/1.1 204 No Content
OData-Version: 4.0
```

If the ETag doesn't match (entity was modified by another client):
```
HTTP/1.1 412 Precondition Failed
Content-Type: application/json

{
  "error": {
    "code": "412",
    "message": "Precondition failed",
    "details": [{
      "message": "The entity has been modified. Please refresh and try again."
    }]
  }
}
```

### If-Match Header Options

- **Specific ETag**: `If-Match: W/"abc123..."` - Match only if the ETag is exactly this value
- **Wildcard**: `If-Match: *` - Match if the entity exists (any ETag value)
- **No header**: Update proceeds without validation

### ETag Generation

ETags are automatically generated as weak ETags (format: `W/"hash"`) using SHA-256 hash of the ETag field value. The same field value always produces the same ETag, ensuring consistency.

Per OData v4 specification, ETags are included in responses in two ways:
1. **HTTP Header**: `ETag: W/"abc123..."` - Used by clients for conditional requests
2. **Response Body**: `"@odata.etag": "W/\"abc123...\""` - Included in the JSON response body for each entity

Both the header and the annotation contain the same ETag value.

### Best Practices

1. **Use version numbers** for simple counter-based concurrency control
2. **Use timestamps** when you need to track when entities were last modified
3. **Always check for 412 responses** in your client code and handle them appropriately
4. **Refresh data** when receiving a 412 response before retrying the update

### Operations Supporting If-Match

- `PATCH /EntitySet(key)` - Partial update with ETag validation
- `PUT /EntitySet(key)` - Full replacement with ETag validation
- `DELETE /EntitySet(key)` - Delete with ETag validation

## Actions and Functions

OData v4 supports custom operations beyond standard CRUD through Actions and Functions. Actions can have side effects and are invoked with POST, while Functions are side-effect free and are invoked with GET.

### Registering Functions

Functions are read-only operations that compute and return values. They can be bound to entities or unbound (standalone).

#### Unbound Function Example

```go
// Register a function that returns the top N products by price
service.RegisterFunction(odata.FunctionDefinition{
    Name:    "GetTopProducts",
    IsBound: false,
    Parameters: []odata.ParameterDefinition{
        {Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
    },
    ReturnType: reflect.TypeOf([]Product{}),
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
        count := params["count"].(int64)
        var products []Product
        if err := db.Order("price DESC").Limit(int(count)).Find(&products).Error; err != nil {
            return nil, err
        }
        return products, nil
    },
})
```

Invoke with:
```bash
GET /GetTopProducts?count=3
```

Response:
```json
{
  "@odata.context": "$metadata#Edm.String",
  "value": [
    {"ID": 1, "Name": "Laptop", "Price": 999.99},
    {"ID": 5, "Name": "Smartphone", "Price": 799.99},
    {"ID": 4, "Name": "Office Chair", "Price": 249.99}
  ]
}
```

#### Bound Function Example

```go
// Register a function that calculates total price with tax for a specific product
service.RegisterFunction(odata.FunctionDefinition{
    Name:      "GetTotalPrice",
    IsBound:   true,
    EntitySet: "Products",
    Parameters: []odata.ParameterDefinition{
        {Name: "taxRate", Type: reflect.TypeOf(float64(0)), Required: true},
    },
    ReturnType: reflect.TypeOf(float64(0)),
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
        taxRate := params["taxRate"].(float64)
        // Extract product from context or fetch from database
        // Calculate total price with tax
        return totalPrice, nil
    },
})
```

Invoke with:
```bash
GET /Products(1)/GetTotalPrice?taxRate=0.08
```

Response:
```json
{
  "@odata.context": "$metadata#Edm.String",
  "value": 1079.99
}
```

### Registering Actions

Actions can have side effects (create, update, delete data) and are invoked with POST.

#### Unbound Action Example

```go
// Register an action that resets all product prices
service.RegisterAction(odata.ActionDefinition{
    Name:       "ResetAllPrices",
    IsBound:    false,
    Parameters: []odata.ParameterDefinition{},
    ReturnType: nil, // No return value
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
        // Reset prices in database
        w.Header().Set("OData-Version", "4.0")
        w.WriteHeader(http.StatusNoContent)
        return nil
    },
})
```

Invoke with:
```bash
POST /ResetAllPrices
```

Response:
```
HTTP/1.1 204 No Content
OData-Version: 4.0
```

#### Bound Action Example

```go
// Register an action that applies a discount to a specific product
service.RegisterAction(odata.ActionDefinition{
    Name:      "ApplyDiscount",
    IsBound:   true,
    EntitySet: "Products",
    Parameters: []odata.ParameterDefinition{
        {Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
    },
    ReturnType: reflect.TypeOf(Product{}),
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
        percentage := params["percentage"].(float64)
        // Apply discount and save to database
        // Return the updated product
        response := map[string]interface{}{
            "@odata.context": "$metadata#Products/$entity",
            "value": updatedProduct,
        }
        w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
        w.Header().Set("OData-Version", "4.0")
        return json.NewEncoder(w).Encode(response)
    },
})
```

Invoke with:
```bash
POST /Products(1)/ApplyDiscount
Content-Type: application/json

{"percentage": 10}
```

Response:
```json
{
  "@odata.context": "$metadata#Products/$entity",
  "value": {
    "ID": 1,
    "Name": "Laptop",
    "Price": 899.99
  }
}
```

### Parameter Types

Actions and Functions support various parameter types:
- `string` - Text values
- `int`, `int32`, `int64` - Integer values
- `float32`, `float64` - Decimal values
- `bool` - Boolean values (`true`/`false`)

Parameters can be marked as required or optional:
```go
Parameters: []odata.ParameterDefinition{
    {Name: "filter", Type: reflect.TypeOf(""), Required: false},  // Optional
    {Name: "count", Type: reflect.TypeOf(int64(0)), Required: true}, // Required
}
```

### Key Differences

| Feature | Actions | Functions |
|---------|---------|-----------|
| HTTP Method | POST | GET |
| Side Effects | Yes (can modify data) | No (read-only) |
| Parameters | In request body (JSON) | In query string |
| Caching | Not cacheable | Cacheable |
| Use Cases | Create, update, delete operations | Calculations, queries, aggregations |

## OData Version Negotiation

The library supports the `OData-MaxVersion` header for protocol version negotiation, as specified in the OData v4 standard. This allows clients to indicate the maximum OData protocol version they support.

### How It Works

1. **Client sends OData-MaxVersion header**: The client can optionally include the `OData-MaxVersion` header in the request to specify the maximum OData version it can handle.

2. **Server validates version**: The service validates that it can provide a response compatible with the client's requested version.

3. **Version compatibility**:
   - ✅ **Accepted**: Requests with `OData-MaxVersion: 4.0` or higher (e.g., `4.01`, `5.0`)
   - ✅ **Accepted**: Requests without the `OData-MaxVersion` header (no version constraint)
   - ❌ **Rejected**: Requests with `OData-MaxVersion` below `4.0` (e.g., `3.0`, `2.0`, `1.0`)

### Examples

**Successful request with OData-MaxVersion: 4.0**
```bash
GET /Products HTTP/1.1
Host: localhost:8080
OData-MaxVersion: 4.0
```

Response:
```
HTTP/1.1 200 OK
OData-Version: 4.0
Content-Type: application/json;odata.metadata=minimal

{
  "@odata.context": "http://localhost:8080/$metadata#Products",
  "value": [...]
}
```

**Rejected request with OData-MaxVersion: 3.0**
```bash
GET /Products HTTP/1.1
Host: localhost:8080
OData-MaxVersion: 3.0
```

Response:
```
HTTP/1.1 406 Not Acceptable
OData-Version: 4.0
Content-Type: application/json;odata.metadata=minimal

{
  "error": {
    "code": "406",
    "message": "OData version not supported",
    "details": [
      {
        "message": "This service only supports OData version 4.0 and above. The maximum version specified in the OData-MaxVersion header is below 4.0."
      }
    ]
  }
}
```

### Why This Matters

The `OData-MaxVersion` header ensures that clients don't receive responses they cannot process. If a client only supports OData v3.0, it should include `OData-MaxVersion: 3.0` in its requests. Since this library only supports OData v4.0, it will properly reject such requests with a `406 Not Acceptable` status, preventing potential compatibility issues.

This is particularly important when:
- Migrating from older OData versions to v4.0
- Ensuring backward compatibility in client applications
- Following OData v4 specification requirements

## Error Handling

The library implements OData v4 compliant error responses, providing structured error information that helps clients understand and handle errors effectively.

### Error Response Structure

All errors follow the OData v4 specification with the following structure:

```json
{
  "error": {
    "code": "404",
    "message": "Entity not found",
    "target": "Products(999)",
    "details": [
      {
        "code": "EntityNotFound",
        "target": "Products(999)",
        "message": "The entity with key '999' does not exist"
      }
    ]
  }
}
```

### Error Fields

- **code**: A string error code that can be used programmatically (typically the HTTP status code)
- **message**: A human-readable error message describing the error
- **target** (optional): The target of the error (e.g., the entity set and key, or property name)
- **details** (optional): An array of detailed error information
- **innererror** (optional): Nested error information for debugging (typically used in development)

### Single Error Example

Simple validation error:

```json
{
  "error": {
    "code": "400",
    "message": "Invalid query options",
    "details": [
      {
        "message": "Unknown query option: $invalidOption"
      }
    ]
  }
}
```

### Multiple Validation Errors

When multiple validation errors occur:

```json
{
  "error": {
    "code": "ValidationError",
    "message": "Multiple validation errors occurred",
    "details": [
      {
        "code": "RequiredField",
        "target": "Name",
        "message": "Name is required"
      },
      {
        "code": "InvalidFormat",
        "target": "Email",
        "message": "Email format is invalid"
      },
      {
        "code": "OutOfRange",
        "target": "Price",
        "message": "Price must be greater than 0"
      }
    ]
  }
}
```

### Nested Error with Debug Information

For internal errors with additional context (typically in development environments):

```json
{
  "error": {
    "code": "500",
    "message": "An internal error occurred",
    "innererror": {
      "message": "Database connection failed",
      "type": "System.Data.SqlClient.SqlException",
      "innererror": {
        "message": "Network timeout",
        "stacktrace": "at Database.Connect()\n   at QueryExecutor.Execute()"
      }
    }
  }
}
```

### Common Error Scenarios

#### Entity Not Found (404)
```bash
GET /Products(999)
```
Response:
```json
{
  "error": {
    "code": "404",
    "message": "Entity not found",
    "target": "Products(999)",
    "details": [
      {
        "target": "Products(999)",
        "message": "The entity with key '999' does not exist"
      }
    ]
  }
}
```

#### Invalid Request (400)
```bash
GET /Products?$filter=invalid syntax
```
Response:
```json
{
  "error": {
    "code": "400",
    "message": "Invalid query options",
    "details": [
      {
        "message": "Failed to parse filter expression"
      }
    ]
  }
}
```

#### Method Not Allowed (405)
```bash
DELETE /
```
Response:
```json
{
  "error": {
    "code": "405",
    "message": "Method not allowed",
    "details": [
      {
        "message": "Method DELETE is not supported for entity collections"
      }
    ]
  }
}
```

All error responses include the appropriate HTTP status code and OData v4 headers:
- `Content-Type: application/json;odata.metadata=minimal`
- `OData-Version: 4.0`

## Requirements

- Go 1.21 or later
- GORM-compatible database driver

## Testing

### Unit Tests

Run the unit test suite:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...
```

### Compliance Tests

The library includes a comprehensive OData v4 compliance test suite with 67+ test scripts covering all aspects of the OData specification.

To run compliance tests:

```bash
# Start the development server
cd cmd/devserver
go run . &

# In another terminal, run compliance tests
cd compliance/v4
./run_compliance_tests.sh
```

The compliance tests verify:
- HTTP headers and status codes
- Service document and metadata generation
- Query options ($filter, $select, $orderby, $expand, etc.)
- CRUD operations (Create, Read, Update, Delete)
- Data types and validation
- Actions and functions
- Batch requests
- ETags and conditional requests

For more information about compliance tests, see [compliance/v4/README.md](compliance/v4/README.md).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Running Tests Locally

Before submitting a PR, make sure to:

1. Run all unit tests: `go test ./...`
2. Run tests with race detection: `go test -race ./...`
3. Run compliance tests (see above)
4. Format your code: `go fmt ./...`
5. Run go vet: `go vet ./...`
6. Run linter: `golangci-lint run`

### Continuous Integration

All tests (unit tests, compliance tests, builds, and linting) are automatically run on every push and pull request via GitHub Actions. The CI workflow ensures:
- All unit tests pass
- All compliance tests pass (100% compliance with OData v4 specification)
- Code builds successfully
- Code meets linting standards

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.