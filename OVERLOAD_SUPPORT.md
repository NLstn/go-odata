# Function and Action Overload Support

This document demonstrates how to use the new function and action overload support in go-odata.

## Overview

As of this update, go-odata now supports function and action overloading as specified in the OData v4.01 specification. This allows you to register multiple functions or actions with the same name but different signatures.

## What are Overloads?

In OData, functions and actions can have multiple "overloads" - different implementations with the same name but different:
- Parameter counts
- Parameter types
- Parameter names
- Binding contexts (bound to different entity sets)

The service automatically selects the appropriate overload based on the parameters provided in the request.

## Examples

### Function Overloads with Different Parameter Counts

```go
service := odata.NewService(db)

// Overload 1: No parameters - get all top products
service.RegisterFunction(odata.FunctionDefinition{
    Name:       "GetTopProducts",
    IsBound:    false,
    Parameters: []odata.ParameterDefinition{},
    ReturnType: reflect.TypeOf([]Product{}),
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
        // Return top 10 products by default
        var products []Product
        db.Order("price DESC").Limit(10).Find(&products)
        return products, nil
    },
})

// Overload 2: With count parameter
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
        db.Order("price DESC").Limit(int(count)).Find(&products)
        return products, nil
    },
})

// Overload 3: With count and category parameters
service.RegisterFunction(odata.FunctionDefinition{
    Name:    "GetTopProducts",
    IsBound: false,
    Parameters: []odata.ParameterDefinition{
        {Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
        {Name: "category", Type: reflect.TypeOf(""), Required: true},
    },
    ReturnType: reflect.TypeOf([]Product{}),
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
        count := params["count"].(int64)
        category := params["category"].(string)
        var products []Product
        db.Where("category = ?", category).Order("price DESC").Limit(int(count)).Find(&products)
        return products, nil
    },
})
```

**Usage:**
```http
GET /GetTopProducts()                           # Uses overload 1
GET /GetTopProducts()?count=5                   # Uses overload 2
GET /GetTopProducts()?count=5&category=Electronics  # Uses overload 3
```

### Action Overloads with Different Parameters

```go
// Overload 1: Apply discount to all products
service.RegisterAction(odata.ActionDefinition{
    Name:    "ApplyDiscount",
    IsBound: false,
    Parameters: []odata.ParameterDefinition{
        {Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
    },
    ReturnType: nil,
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
        percentage := params["percentage"].(float64)
        // Apply discount to all products
        db.Model(&Product{}).Update("price", gorm.Expr("price * ?", 1.0-percentage/100.0))
        w.WriteHeader(http.StatusNoContent)
        return nil
    },
})

// Overload 2: Apply discount to specific category
service.RegisterAction(odata.ActionDefinition{
    Name:    "ApplyDiscount",
    IsBound: false,
    Parameters: []odata.ParameterDefinition{
        {Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
        {Name: "category", Type: reflect.TypeOf(""), Required: true},
    },
    ReturnType: nil,
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
        percentage := params["percentage"].(float64)
        category := params["category"].(string)
        // Apply discount to category
        db.Model(&Product{}).Where("category = ?", category).
            Update("price", gorm.Expr("price * ?", 1.0-percentage/100.0))
        w.WriteHeader(http.StatusNoContent)
        return nil
    },
})
```

**Usage:**
```http
POST /ApplyDiscount
Content-Type: application/json

{"percentage": 10.0}                            # Uses overload 1
```

```http
POST /ApplyDiscount
Content-Type: application/json

{"percentage": 10.0, "category": "Electronics"} # Uses overload 2
```

### Bound Function Overloads on Different Entity Sets

```go
// Register for Products entity set
service.RegisterFunction(odata.FunctionDefinition{
    Name:      "GetInfo",
    IsBound:   true,
    EntitySet: "Products",
    Parameters: []odata.ParameterDefinition{
        {Name: "format", Type: reflect.TypeOf(""), Required: true},
    },
    ReturnType: reflect.TypeOf(""),
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
        product := ctx.(*Product)
        format := params["format"].(string)
        return fmt.Sprintf("Product: %s (Price: %.2f) in %s format", product.Name, product.Price, format), nil
    },
})

// Register for Customers entity set (same name, different entity set)
service.RegisterFunction(odata.FunctionDefinition{
    Name:      "GetInfo",
    IsBound:   true,
    EntitySet: "Customers",
    Parameters: []odata.ParameterDefinition{
        {Name: "format", Type: reflect.TypeOf(""), Required: true},
    },
    ReturnType: reflect.TypeOf(""),
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
        customer := ctx.(*Customer)
        format := params["format"].(string)
        return fmt.Sprintf("Customer: %s in %s format", customer.Name, format), nil
    },
})
```

**Usage:**
```http
GET /Products(1)/GetInfo?format=json      # Calls Product-bound overload
GET /Customers(1)/GetInfo?format=json     # Calls Customer-bound overload
```

## Overload Resolution Rules

When you invoke a function or action, the service resolves the appropriate overload by:

1. **Filtering by binding context**: First, it filters overloads based on whether the call is bound (e.g., `/Products(1)/Function`) or unbound (e.g., `/Function()`)

2. **Filtering by entity set**: For bound operations, it further filters by the entity set in the URL path

3. **Matching by parameters**: Finally, it selects the overload whose parameter definitions match the parameters provided in the request:
   - All required parameters must be present
   - No extra parameters should be provided
   - Parameter names must match

## Duplicate Overload Protection

The registration functions now validate that you don't register duplicate overloads. Two overloads are considered duplicates if they have:
- Same name
- Same binding status (bound vs unbound)
- Same entity set (for bound operations)
- Same parameters (names and types)

If you attempt to register a duplicate, you'll get an error:
```go
err := service.RegisterFunction(duplicateFunction)
// Error: "function 'FunctionName' with this signature is already registered"
```

## Backward Compatibility

This feature is fully backward compatible. If you only register one function or action with a given name, the behavior is identical to the previous version. All existing tests pass without modification.

## OData Specification Reference

This implementation follows the OData v4.01 CSDL specification:
https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html#sec_FunctionandActionOverloading
