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
    service, err := odata.NewService(db)
    if err != nil {
        log.Fatal(err)
    }
    
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

            // Apply basic OData query options in-memory for external data sources.
            filtered, err := odata.ApplyQueryOptionsToSlice(products, ctx.QueryOptions, func(item ExternalProduct, filter *odata.FilterExpression) (bool, error) {
                if filter == nil {
                    return true, nil
                }
                if filter.Property == "name" && filter.Operator == "eq" {
                    if value, ok := filter.Value.(string); ok {
                        return item.Name == value, nil
                    }
                }
                // Unsupported filter expression; let the caller decide how to handle it.
                return false, nil
            })
            if err != nil {
                return nil, err
            }

            return &odata.CollectionResult{Items: filtered}, nil
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

## Error Handling in Overwrite Handlers

Errors returned from overwrite handlers are automatically mapped to OData-compliant error responses. The library provides several ways to control error responses:

### Using Standard Errors

Simple errors are automatically converted to HTTP 500 Internal Server Error:

```go
GetEntity: func(ctx *odata.OverwriteContext) (interface{}, error) {
    entity, err := externalAPI.GetEntity(ctx.EntityKey)
    if err != nil {
        return nil, err // Returns HTTP 500 with error message
    }
    return entity, nil
}
```

### Using Sentinel Errors

The library provides sentinel errors that map to specific HTTP status codes:

```go
GetEntity: func(ctx *odata.OverwriteContext) (interface{}, error) {
    entity, err := externalAPI.GetEntity(ctx.EntityKey)
    if err != nil {
        if errors.Is(err, externalAPI.ErrNotFound) {
            return nil, odata.ErrEntityNotFound // Returns HTTP 404
        }
        if errors.Is(err, externalAPI.ErrUnauthorized) {
            return nil, odata.ErrUnauthorized // Returns HTTP 401
        }
        return nil, err // Returns HTTP 500 for other errors
    }
    return entity, nil
}
```

Available sentinel errors:
- `odata.ErrEntityNotFound` → HTTP 404 Not Found
- `odata.ErrValidationError` → HTTP 400 Bad Request
- `odata.ErrUnauthorized` → HTTP 401 Unauthorized
- `odata.ErrForbidden` → HTTP 403 Forbidden
- `odata.ErrMethodNotAllowed` → HTTP 405 Method Not Allowed
- `odata.ErrConflict` → HTTP 409 Conflict
- `odata.ErrPreconditionFailed` → HTTP 412 Precondition Failed
- `odata.ErrUnsupportedMediaType` → HTTP 415 Unsupported Media Type
- `odata.ErrInternalServerError` → HTTP 500 Internal Server Error

### Using ODataError for Fine-Grained Control

For complete control over error responses, use `odata.ODataError`:

```go
Create: func(ctx *odata.OverwriteContext, entity interface{}) (interface{}, error) {
    product := entity.(*Product)
    
    // Validate input
    if product.Price < 0 {
        return nil, &odata.ODataError{
            StatusCode: http.StatusBadRequest,
            Code:       odata.ErrorCodeBadRequest,
            Message:    "Invalid product data",
            Target:     "Price",
            Details: []odata.ErrorDetail{
                {
                    Code:    "NegativePrice",
                    Target:  "Price",
                    Message: "Price must be non-negative",
                },
            },
        }
    }
    
    // Call external API
    created, err := externalAPI.CreateProduct(product)
    if err != nil {
        return nil, &odata.ODataError{
            StatusCode: http.StatusInternalServerError,
            Code:       odata.ErrorCodeInternalServerError,
            Message:    "Failed to create product in external system",
            Err:        err, // Wrap the original error
        }
    }
    
    return created, nil
}
```

### Using HookError

Similar to entity hooks, overwrite handlers can return `odata.HookError` for custom status codes:

```go
GetEntity: func(ctx *odata.OverwriteContext) (interface{}, error) {
    entity, err := externalAPI.GetEntity(ctx.EntityKey)
    if err != nil {
        if errors.Is(err, externalAPI.ErrNotFound) {
            return nil, odata.NewHookError(http.StatusNotFound, "Entity not found")
        }
        return nil, odata.NewHookError(http.StatusBadGateway, "External service unavailable")
    }
    return entity, nil
}
```

## Implementing Pagination for Virtual Entities

Virtual entities should implement pagination to handle large result sets efficiently. The library provides query options that you can use to implement pagination:

### Basic Pagination with $top and $skip

```go
GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
    // Get pagination parameters
    skip := 0
    if ctx.QueryOptions.Skip != nil {
        skip = *ctx.QueryOptions.Skip
    }
    
    top := 100 // Default page size
    if ctx.QueryOptions.Top != nil {
        top = *ctx.QueryOptions.Top
    }
    
    // Fetch paginated data from external source
    items, total, err := externalAPI.GetItemsPaginated(skip, top)
    if err != nil {
        return nil, err
    }
    
    // Return with count if requested
    var count *int64
    if ctx.QueryOptions.Count {
        c := int64(total)
        count = &c
    }
    
    return &odata.CollectionResult{
        Items: items,
        Count: count,
    }, nil
}
```

### Server-Driven Pagination with NextLink

For very large datasets, implement server-driven pagination:

```go
GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
    pageSize := 100
    
    // Check for skip token (continuation)
    var skipToken string
    if ctx.QueryOptions.SkipToken != nil {
        skipToken = *ctx.QueryOptions.SkipToken
    }
    
    // Fetch one extra item to detect if there's a next page
    items, err := externalAPI.GetItems(skipToken, pageSize+1)
    if err != nil {
        return nil, err
    }
    
    // Check if there are more results
    hasMore := len(items) > pageSize
    if hasMore {
        items = items[:pageSize] // Trim to page size
        
        // Generate next skip token (implementation depends on your data source)
        // For example, use the last item's ID or timestamp
        // Note: Implement generateSkipToken based on your pagination strategy
        // (e.g., encode the last item's ID, timestamp, or offset)
        // lastItem := items[len(items)-1]
        // nextSkipToken := generateSkipToken(lastItem)
        
        // The library will automatically add @odata.nextLink to the response
        // when you configure server-driven paging in your service
        // Note: The library handles this automatically based on service configuration
    }
    
    return &odata.CollectionResult{
        Items: items,
    }, nil
}
```

### In-Memory Pagination with ApplyQueryOptionsToSlice

For smaller datasets or when fetching all data at once, use the built-in helper:

```go
GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
    // Fetch all items from external source
    allItems, err := externalAPI.GetAllItems()
    if err != nil {
        return nil, err
    }
    
    // Apply OData query options including $top, $skip, $filter, $orderby
    filtered, err := odata.ApplyQueryOptionsToSlice(
        allItems,
        ctx.QueryOptions,
        myFilterFunc, // Custom filter evaluator
    )
    if err != nil {
        return nil, err
    }
    
    // Calculate count if requested
    var count *int64
    if ctx.QueryOptions.Count {
        c := int64(len(filtered))
        count = &c
    }
    
    return &odata.CollectionResult{
        Items: filtered,
        Count: count,
    }, nil
}
```

## Handling $count Requests

The `$count` query option requests the total number of entities in the collection. There are two ways to handle this:

### 1. Inline Count with $count=true

When `$count=true` is specified, include the total count in the result:

```go
GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
    // Fetch paginated items
    items, err := externalAPI.GetItems(skip, top)
    if err != nil {
        return nil, err
    }
    
    // If count is requested, fetch total count
    var count *int64
    if ctx.QueryOptions.Count {
        total, err := externalAPI.GetTotalCount()
        if err != nil {
            return nil, err
        }
        c := int64(total)
        count = &c
    }
    
    return &odata.CollectionResult{
        Items: items,
        Count: count, // Include count in response
    }, nil
}
```

### 2. $count Segment (GET /EntitySet/$count)

To support direct count queries (`GET /EntitySet/$count`), implement the `GetCount` handler:

```go
GetCount: func(ctx *odata.OverwriteContext) (int64, error) {
    // Apply any filters from query options
    var filter string
    if ctx.QueryOptions.Filter != nil {
        // Convert filter expression to your external API's format.
        // Note: convertFilterToAPIFormat is a placeholder helper; implement this to match your API's filtering syntax.
        // Example: convert OData filter "Name eq 'Product1'" to your API's query format
        filter = convertFilterToAPIFormat(ctx.QueryOptions.Filter)
    }
    
    // Get count from external source
    count, err := externalAPI.GetCount(filter)
    if err != nil {
        return 0, err
    }
    
    return count, nil
}
```

### Efficient Count Implementation

For performance, consider these strategies:

```go
GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
    // Strategy 1: If your API returns count with results, use it
    response, err := externalAPI.GetItemsWithCount(skip, top)
    if err != nil {
        return nil, err
    }
    
    var count *int64
    if ctx.QueryOptions.Count && response.TotalCount != nil {
        c := int64(*response.TotalCount)
        count = &c
    }
    
    return &odata.CollectionResult{
        Items: response.Items,
        Count: count,
    }, nil
}

// Strategy 2: Cache count for frequently accessed, relatively static data
// Note: This example requires importing "sync" and "time" packages
var countCache = struct {
    sync.RWMutex
    value     int64
    expiresAt time.Time
}{}

GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
    items, err := externalAPI.GetItems(skip, top)
    if err != nil {
        return nil, err
    }
    
    var count *int64
    if ctx.QueryOptions.Count {
        countCache.RLock()
        if time.Now().Before(countCache.expiresAt) {
            c := countCache.value
            count = &c
            countCache.RUnlock()
        } else {
            countCache.RUnlock()
            
            // Fetch fresh count
            total, err := externalAPI.GetTotalCount()
            if err != nil {
                return nil, err
            }
            
            // Update cache
            countCache.Lock()
            countCache.value = int64(total)
            countCache.expiresAt = time.Now().Add(5 * time.Minute)
            countCache.Unlock()
            
            c := int64(total)
            count = &c
        }
    }
    
    return &odata.CollectionResult{
        Items: items,
        Count: count,
    }, nil
}
```

## Best Practices

1. **Error Handling**: Use `odata.ODataError` or sentinel errors for precise error responses. Wrap underlying errors to preserve error chains.

2. **Pagination**: Always implement pagination for collections. Use $top and $skip for client-driven paging, or server-driven paging for very large datasets.

3. **Count Performance**: Optimize $count operations by caching counts for static data or requesting counts alongside results from your data source.

4. **Security**: Implement authorization checks within your handlers, especially when exposing external data.

5. **Query Options**: Leverage the parsed query options to implement efficient filtering and pagination at the source rather than fetching all data.

6. **Documentation**: Document which operations and query options are supported for each virtual entity.

7. **Testing**: Write tests for your overwrite handlers, including error cases and edge conditions.

## Limitations

- Virtual entities cannot be used with database-specific features like GORM migrations
- Change tracking ($deltatoken) is not supported for virtual entities
- Full-text search requires manual implementation in handlers
- Navigation properties to/from virtual entities need careful consideration

## See Also

- [Overwrite Handlers](advanced-features.md#overwrite-handlers) - Learn more about overwrite handlers
- [Actions and Functions](actions-and-functions.md) - Extend virtual entities with custom operations
- [Advanced Features](advanced-features.md) - Other advanced OData features
- Error Handling - See the `errors.go` file in the root of the repository for complete error types reference
