# Actions and Functions

This guide covers how to register and implement custom OData actions and functions in go-odata.

## Table of Contents

- [Overview](#overview)
- [Functions](#functions)
- [Actions](#actions)
- [Parameter Types](#parameter-types)
- [Best Practices](#best-practices)

## Overview

OData v4 supports custom operations beyond standard CRUD through Actions and Functions:

- **Functions**: Read-only operations that compute and return values. Invoked with GET.
- **Actions**: Operations that can have side effects (create, update, delete data). Invoked with POST.

Both can be:
- **Unbound**: Standalone operations accessible at the service root
- **Bound**: Operations tied to specific entities or entity sets

## Functions

Functions are side-effect free operations that return computed values.

### Unbound Function

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

**Invoke:**
```bash
GET /GetTopProducts?count=3
```

**Response:**
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

### Bound Function

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
        product := ctx.(*Product)
        taxRate := params["taxRate"].(float64)
        totalPrice := product.Price * (1 + taxRate)
        return totalPrice, nil
    },
})
```

**Invoke:**
```bash
GET /Products(1)/GetTotalPrice?taxRate=0.08
```

**Response:**
```json
{
  "@odata.context": "$metadata#Edm.String",
  "value": 1079.99
}
```

## Actions

Actions can modify data and are invoked with POST.

### Unbound Action

```go
// Register an action that resets all product prices
service.RegisterAction(odata.ActionDefinition{
    Name:       "ResetAllPrices",
    IsBound:    false,
    Parameters: []odata.ParameterDefinition{},
    ReturnType: nil, // No return value
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
        if err := db.Model(&Product{}).Update("Price", 0).Error; err != nil {
            return err
        }
        w.Header().Set("OData-Version", "4.0")
        w.WriteHeader(http.StatusNoContent)
        return nil
    },
})
```

**Invoke:**
```bash
POST /ResetAllPrices
```

**Response:**
```
HTTP/1.1 204 No Content
OData-Version: 4.0
```

### Bound Action

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
        product := ctx.(*Product)
        percentage := params["percentage"].(float64)
        
        // Apply discount
        product.Price = product.Price * (1 - percentage/100)
        if err := db.Save(product).Error; err != nil {
            return err
        }
        
        // Return updated product
        response := map[string]interface{}{
            "@odata.context": "$metadata#Products/$entity",
            "value": product,
        }
        w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
        w.Header().Set("OData-Version", "4.0")
        return json.NewEncoder(w).Encode(response)
    },
})
```

**Invoke:**
```bash
POST /Products(1)/ApplyDiscount
Content-Type: application/json

{"percentage": 10}
```

**Response:**
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

## Parameter Types

Actions and functions support various parameter types:

```go
Parameters: []odata.ParameterDefinition{
    {Name: "name", Type: reflect.TypeOf(""), Required: true},              // string
    {Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},       // int64
    {Name: "price", Type: reflect.TypeOf(float64(0)), Required: true},     // float64
    {Name: "active", Type: reflect.TypeOf(false), Required: true},         // bool
    {Name: "filter", Type: reflect.TypeOf(""), Required: false},           // optional string
}
```

Supported types:
- `string` - Text values
- `int`, `int32`, `int64` - Integer values
- `float32`, `float64` - Decimal values
- `bool` - Boolean values (`true`/`false`)

## Best Practices

### Error Handling

Always handle errors appropriately in your handlers:

```go
Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
    count := params["count"].(int64)
    
    if count <= 0 {
        return nil, fmt.Errorf("count must be greater than 0")
    }
    
    var products []Product
    if err := db.Order("price DESC").Limit(int(count)).Find(&products).Error; err != nil {
        return nil, err
    }
    
    return products, nil
}
```

### Context Access

For bound operations, the context contains the entity instance:

```go
Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
    // For bound function on Products
    product := ctx.(*Product)
    
    // Use the product data
    return product.Price * 1.1, nil
}
```

### Response Formatting

The library handles most response formatting automatically, but you can customize when needed:

```go
Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
    // Set headers
    w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
    w.Header().Set("OData-Version", "4.0")
    
    // Set status
    w.WriteHeader(http.StatusOK)
    
    // Write response
    response := map[string]interface{}{
        "@odata.context": "$metadata#Products",
        "value": results,
    }
    return json.NewEncoder(w).Encode(response)
}
```

### Function vs Action Decision

| Use Function When | Use Action When |
|-------------------|------------------|
| Read-only operation | Modifies data |
| Idempotent | May have side effects |
| Can be cached | Should not be cached |
| Computing values | Creating/updating/deleting |
| Querying data | Processing operations |

### Examples

**Good Function Use Cases:**
- Calculate total order value
- Get top N products
- Search with complex logic
- Compute statistics

**Good Action Use Cases:**
- Apply discount to products
- Process order
- Reset passwords
- Import/export data
- Send notifications
