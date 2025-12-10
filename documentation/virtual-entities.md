# Virtual Entities

Virtual entities are a powerful feature that allows you to expose OData endpoints without requiring a database backing store. This is particularly useful when you want to:

- Expose data from external APIs or services
- Create computed or aggregated views
- Implement custom business logic without database persistence
- Bridge between OData and legacy systems

## How Virtual Entities Work

Unlike regular entities that are backed by database tables, virtual entities require **overwrite handlers** for all operations. When a client attempts to access a virtual entity without the corresponding overwrite handler, the service will return HTTP 405 Method Not Allowed.

This design ensures that:
1. You explicitly define the behavior for each operation
2. No unexpected database operations occur
3. The service clearly communicates which operations are supported

## Registering a Virtual Entity

To register a virtual entity, use the `RegisterVirtualEntity` method:

```go
type ExternalProduct struct {
    ID          int     `json:"id" odata:"key"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    ExternalURL string  `json:"externalUrl"`
}

// Register the virtual entity
err := service.RegisterVirtualEntity(&ExternalProduct{})
if err != nil {
    log.Fatal(err)
}
```

The entity struct follows the same conventions as regular entities (tags, key properties, etc.), but no database table is created.

## Setting Overwrite Handlers

After registering a virtual entity, you must provide overwrite handlers for the operations you want to support:

```go
// Set up overwrite handlers
err = service.SetEntityOverwrite("ExternalProducts", &odata.EntityOverwrite{
    GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
        // Fetch products from external API
        products, err := externalAPI.GetProducts()
        if err != nil {
            return nil, err
        }
        return &odata.CollectionResult{Items: products}, nil
    },
    
    GetEntity: func(ctx *odata.OverwriteContext) (interface{}, error) {
        // Fetch single product by key from external API
        product, err := externalAPI.GetProduct(ctx.EntityKey)
        if err != nil {
            return nil, err
        }
        return product, nil
    },
    
    Create: func(ctx *odata.OverwriteContext, entity interface{}) (interface{}, error) {
        // Create product via external API
        product := entity.(*ExternalProduct)
        created, err := externalAPI.CreateProduct(product)
        if err != nil {
            return nil, err
        }
        return created, nil
    },
    
    Update: func(ctx *odata.OverwriteContext, data map[string]interface{}, isFullReplace bool) (interface{}, error) {
        // Update product via external API
        updated, err := externalAPI.UpdateProduct(ctx.EntityKey, data, isFullReplace)
        if err != nil {
            return nil, err
        }
        return updated, nil
    },
    
    Delete: func(ctx *odata.OverwriteContext) error {
        // Delete product via external API
        return externalAPI.DeleteProduct(ctx.EntityKey)
    },
})
```

## Partial Implementation

You don't have to implement all operations. For example, if you only want to expose read-only data:

```go
// Only implement read operations
err = service.SetEntityOverwrite("ExternalProducts", &odata.EntityOverwrite{
    GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
        products, err := externalAPI.GetProducts()
        if err != nil {
            return nil, err
        }
        return &odata.CollectionResult{Items: products}, nil
    },
    
    GetEntity: func(ctx *odata.OverwriteContext) (interface{}, error) {
        return externalAPI.GetProduct(ctx.EntityKey)
    },
    // Create, Update, and Delete are not implemented
    // Attempting these operations will return HTTP 405
})
```

## Complete Example

Here's a complete example that integrates with a hypothetical external API:

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    
    "github.com/nlstn/go-odata"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// ExternalProduct represents a product from an external system
type ExternalProduct struct {
    ID          int     `json:"id" odata:"key"`
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Price       float64 `json:"price"`
    StockLevel  int     `json:"stockLevel"`
}

// MockExternalAPI simulates an external API
type MockExternalAPI struct {
    products map[int]*ExternalProduct
    nextID   int
}

func NewMockExternalAPI() *MockExternalAPI {
    return &MockExternalAPI{
        products: map[int]*ExternalProduct{
            1: {ID: 1, Name: "External Product 1", Description: "From external system", Price: 99.99, StockLevel: 50},
            2: {ID: 2, Name: "External Product 2", Description: "Also from external system", Price: 149.99, StockLevel: 30},
        },
        nextID: 3,
    }
}

func (api *MockExternalAPI) GetProducts() ([]ExternalProduct, error) {
    products := make([]ExternalProduct, 0, len(api.products))
    for _, p := range api.products {
        products = append(products, *p)
    }
    return products, nil
}

func (api *MockExternalAPI) GetProduct(id string) (*ExternalProduct, error) {
    var productID int
    fmt.Sscanf(id, "%d", &productID)
    
    if product, exists := api.products[productID]; exists {
        return product, nil
    }
    return nil, fmt.Errorf("product not found")
}

func (api *MockExternalAPI) CreateProduct(product *ExternalProduct) (*ExternalProduct, error) {
    product.ID = api.nextID
    api.nextID++
    api.products[product.ID] = product
    return product, nil
}

func (api *MockExternalAPI) UpdateProduct(id string, data map[string]interface{}) (*ExternalProduct, error) {
    var productID int
    fmt.Sscanf(id, "%d", &productID)
    
    product, exists := api.products[productID]
    if !exists {
        return nil, fmt.Errorf("product not found")
    }
    
    // Update fields from data map
    if name, ok := data["name"].(string); ok {
        product.Name = name
    }
    if description, ok := data["description"].(string); ok {
        product.Description = description
    }
    if price, ok := data["price"].(float64); ok {
        product.Price = price
    }
    
    return product, nil
}

func (api *MockExternalAPI) DeleteProduct(id string) error {
    var productID int
    fmt.Sscanf(id, "%d", &productID)
    
    if _, exists := api.products[productID]; !exists {
        return fmt.Errorf("product not found")
    }
    delete(api.products, productID)
    return nil
}

func main() {
    // Initialize database (still needed for the service, even though virtual entities don't use it)
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }
    
    // Create OData service
    service := odata.NewService(db)
    
    // Create mock external API
    externalAPI := NewMockExternalAPI()
    
    // Register virtual entity
    if err := service.RegisterVirtualEntity(&ExternalProduct{}); err != nil {
        log.Fatal(err)
    }
    
    // Set overwrite handlers to integrate with external API
    if err := service.SetEntityOverwrite("ExternalProducts", &odata.EntityOverwrite{
        GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
            products, err := externalAPI.GetProducts()
            if err != nil {
                return nil, err
            }
            
            // Apply OData query options if needed
            // For this example, we're returning all products
            return &odata.CollectionResult{Items: products}, nil
        },
        
        GetEntity: func(ctx *odata.OverwriteContext) (interface{}, error) {
            return externalAPI.GetProduct(ctx.EntityKey)
        },
        
        Create: func(ctx *odata.OverwriteContext, entity interface{}) (interface{}, error) {
            product := entity.(*ExternalProduct)
            return externalAPI.CreateProduct(product)
        },
        
        Update: func(ctx *odata.OverwriteContext, data map[string]interface{}, isFullReplace bool) (interface{}, error) {
            return externalAPI.UpdateProduct(ctx.EntityKey, data)
        },
        
        Delete: func(ctx *odata.OverwriteContext) error {
            return externalAPI.DeleteProduct(ctx.EntityKey)
        },
    }); err != nil {
        log.Fatal(err)
    }
    
    // Start server
    log.Println("Virtual entity OData service running on :8080")
    log.Println("Try: http://localhost:8080/ExternalProducts")
    log.Fatal(http.ListenAndServe(":8080", service))
}
```

## Query Options

Virtual entities support the same OData query options as regular entities ($filter, $select, $expand, etc.). The parsed query options are available in the `OverwriteContext` parameter, allowing you to apply them to your custom data source:

```go
GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
    // Access query options
    if ctx.QueryOptions.Top != nil {
        // Apply top/limit
    }
    if ctx.QueryOptions.Skip != nil {
        // Apply skip/offset
    }
    if ctx.QueryOptions.Filter != nil {
        // Apply filter expression
    }
    if ctx.QueryOptions.OrderBy != nil {
        // Apply ordering
    }
    
    // Fetch and return data
    // ...
}
```

The library validates the query syntax before calling your handler, so you can trust that the query options are well-formed.

## Use Cases

### 1. External API Integration

Expose data from REST APIs, GraphQL endpoints, or other external services through a unified OData interface:

```go
// Integrate with Stripe API
service.RegisterVirtualEntity(&Payment{})
service.SetEntityOverwrite("Payments", &odata.EntityOverwrite{
    GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
        payments, err := stripeClient.Payments.List(...)
        return &odata.CollectionResult{Items: payments}, err
    },
})
```

### 2. Computed Views

Create dynamic, computed entities without storing them in the database:

```go
// Real-time analytics view
type SalesMetrics struct {
    Period    string  `json:"period" odata:"key"`
    Revenue   float64 `json:"revenue"`
    OrderCount int    `json:"orderCount"`
}

service.RegisterVirtualEntity(&SalesMetrics{})
service.SetEntityOverwrite("SalesMetrics", &odata.EntityOverwrite{
    GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
        metrics := computeSalesMetrics(db) // Compute from existing data
        return &odata.CollectionResult{Items: metrics}, nil
    },
})
```

### 3. Legacy System Integration

Bridge the gap between modern OData clients and legacy systems:

```go
// Wrap legacy SOAP service
service.RegisterVirtualEntity(&LegacyCustomer{})
service.SetEntityOverwrite("LegacyCustomers", &odata.EntityOverwrite{
    GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
        customers, err := legacySOAPClient.GetCustomers()
        return &odata.CollectionResult{Items: customers}, err
    },
})
```

### 4. File System Access

Expose file system entities through OData:

```go
type FileInfo struct {
    Path         string    `json:"path" odata:"key"`
    Name         string    `json:"name"`
    Size         int64     `json:"size"`
    ModifiedTime time.Time `json:"modifiedTime"`
}

service.RegisterVirtualEntity(&FileInfo{})
// Implement handlers to list/read files
```

## Best Practices

1. **Error Handling**: Return appropriate errors from your handlers. The library will convert them to proper OData error responses.

2. **Performance**: Consider caching and pagination when dealing with large datasets from external sources.

3. **Security**: Implement authorization checks within your handlers, especially when exposing external data.

4. **Query Options**: Leverage the parsed query options to implement efficient filtering and pagination at the source.

5. **Documentation**: Document which operations are supported for each virtual entity.

## Limitations

- Virtual entities cannot be used with database-specific features like GORM migrations
- Change tracking ($deltatoken) is not supported for virtual entities
- Full-text search requires manual implementation in handlers
- Navigation properties to/from virtual entities need careful consideration

## See Also

- [Overwrite Handlers](advanced-features.md#overwrite-handlers) - Learn more about overwrite handlers
- [Actions and Functions](actions-and-functions.md) - Extend virtual entities with custom operations
- [Advanced Features](advanced-features.md) - Other advanced OData features
