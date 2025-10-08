````markdown
# go-odata

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
- 🔄 OData query operations ($filter, $select, $orderby, etc.) - Coming soon
- 🔄 Complete metadata document generation - Coming soon
- 🔄 Entity relationship handling - Coming soon
- 🔄 Pagination support - Coming soon

## Installation

```bash
go get github.com/nlstn/go-odata
```

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
- **Entity Collection**: `GET /Products` - All products
- **Individual Entity**: `GET /Products(1)` - Product with ID 1

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

## Requirements

- Go 1.21 or later
- GORM-compatible database driver

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
````