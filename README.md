# go-odata

A Go library for building services that expose OData APIs with automatic handling of OData logic.

## Overview

`go-odata` allows you to define Go structs representing entities and automatically handles the necessary OData protocol logic, making it easy to build OData-compliant APIs.

## Features (Planned)

- Automatic OData endpoint generation from Go structs
- Support for OData query operations ($filter, $select, $orderby, etc.)
- Metadata document generation
- Entity relationship handling
- Pagination support
- Type-safe query building

## Installation

```bash
go get github.com/nlstn/go-odata
```

## Quick Start

```go
// Example usage (to be implemented)
package main

import (
    "github.com/nlstn/go-odata"
)

type Product struct {
    ID    int    `odata:"key"`
    Name  string `odata:"required"`
    Price float64
}

func main() {
    // Initialize OData service
    service := odata.NewService()
    
    // Register entity
    service.RegisterEntity(&Product{})
    
    // Start server
    service.ListenAndServe(":8080")
}
```

## Examples

See the [examples](./examples/) directory for complete usage examples.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.