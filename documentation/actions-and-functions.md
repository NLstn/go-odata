# Actions and Functions

This guide covers how to register and implement custom OData actions and functions in go-odata.

## Table of Contents

- [Overview](#overview)
- [Type Definitions](#type-definitions)
  - [ActionDefinition](#actiondefinition)
  - [FunctionDefinition](#functiondefinition)
  - [ParameterDefinition](#parameterdefinition)
  - [Handler Signatures](#handler-signatures)
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

## Type Definitions

### ActionDefinition

`ActionDefinition` describes an OData action that can modify data.

```go
type ActionDefinition struct {
    Name                string           // Action name (e.g., "ApplyDiscount")
    IsBound             bool             // true for bound actions, false for unbound
    EntitySet           string           // Required for bound actions (e.g., "Products")
    Handler             ActionHandler    // Function implementing the action logic
    Parameters          []ParameterDefinition // Action parameters (from JSON body)
    ParameterStructType reflect.Type     // Optional: derive parameters from struct
    ReturnType          reflect.Type     // Return type, or nil for no return value
}
```

**Field Details:**

- **Name**: The action name as it appears in URLs. For bound actions: `/Products(1)/ApplyDiscount`. For unbound: `/ApplyDiscount`.

- **IsBound**: 
  - `true`: Action is bound to an entity or collection. The `ctx` parameter contains the entity instance.
  - `false`: Action is unbound (service-level). The `ctx` parameter is nil.

- **EntitySet**: Required when `IsBound` is `true`. Must match a registered entity set name (e.g., "Products").

- **Handler**: The function implementing the action. See [ActionHandler](#actionhandler) for signature details.

- **Parameters**: List of parameter definitions. Parameters are passed in the JSON request body. Can be empty.

- **ParameterStructType**: Optional. When set, parameters are automatically derived from struct fields using reflection. Use `reflect.TypeOf(MyStructType{})`.

- **ReturnType**: 
  - `nil`: Action returns no value (should respond with HTTP 204 No Content)
  - `reflect.TypeOf(MyType{})`: Action returns a value of the specified type

### FunctionDefinition

`FunctionDefinition` describes an OData function that computes and returns values.

```go
type FunctionDefinition struct {
    Name                string            // Function name (e.g., "GetTotalPrice")
    IsBound             bool              // true for bound functions, false for unbound
    EntitySet           string            // Required for bound functions (e.g., "Products")
    Handler             FunctionHandler   // Function implementing the computation logic
    Parameters          []ParameterDefinition // Function parameters (from URL)
    ParameterStructType reflect.Type      // Optional: derive parameters from struct
    ReturnType          reflect.Type      // Required: return type
}
```

**Field Details:**

- **Name**: The function name as it appears in URLs. For bound functions: `/Products(1)/GetTotalPrice(taxRate=0.08)`. For unbound: `/GetTopProducts(count=10)`.

- **IsBound**: 
  - `true`: Function is bound to an entity or collection. The `ctx` parameter contains the entity instance.
  - `false`: Function is unbound (service-level). The `ctx` parameter is nil.

- **EntitySet**: Required when `IsBound` is `true`. Must match a registered entity set name (e.g., "Products").

- **Handler**: The function implementing the computation. See [FunctionHandler](#functionhandler) for signature details.

- **Parameters**: List of parameter definitions. Parameters are passed in the URL query string or function call syntax. Can be empty.

- **ParameterStructType**: Optional. When set, parameters are automatically derived from struct fields using reflection. Use `reflect.TypeOf(MyStructType{})`.

- **ReturnType**: Required. The Go type of the return value. Use `reflect.TypeOf(MyType{})`.

### ParameterDefinition

`ParameterDefinition` describes a single parameter for an action or function.

```go
type ParameterDefinition struct {
    Name     string        // Parameter name
    Type     reflect.Type  // Expected Go type
    Required bool          // true if parameter must be present
}
```

**Field Details:**

- **Name**: The parameter name as it appears in requests (JSON body for actions, URL for functions).

- **Type**: The expected Go type, obtained via `reflect.TypeOf()`. Common examples:
  - `reflect.TypeOf("")` for string
  - `reflect.TypeOf(int64(0))` for int64
  - `reflect.TypeOf(float64(0))` for float64
  - `reflect.TypeOf(false)` for bool
  - `reflect.TypeOf(MyStruct{})` for complex types
  - `reflect.TypeOf([]MyStruct{})` for slices

- **Required**: 
  - `true`: Parameter must be present in the request
  - `false`: Parameter is optional and may be omitted

### Handler Signatures

#### ActionHandler

```go
type ActionHandler func(
    w http.ResponseWriter,
    r *http.Request,
    ctx interface{},
    params map[string]interface{},
) error
```

**Parameters:**
- `w`: HTTP response writer for setting headers and writing the response
- `r`: HTTP request containing headers, URL, and parsed body
- `ctx`: For bound actions, contains the entity instance (cast to appropriate type). For unbound actions, nil.
- `params`: Map of parameter names to parsed values (use type assertions to access)

**Returns:**
- `error`: Return an error to abort and send error response. Return nil after writing the HTTP response.

**Responsibilities:**
- Write HTTP status code (e.g., `w.WriteHeader(http.StatusOK)` or `http.StatusNoContent`)
- Set response headers (e.g., `w.Header().Set("Content-Type", "application/json")`)
- Write response body (if any)

#### FunctionHandler

```go
type FunctionHandler func(
    w http.ResponseWriter,
    r *http.Request,
    ctx interface{},
    params map[string]interface{},
) (interface{}, error)
```

**Parameters:**
- `w`: HTTP response writer (usually handled automatically by the framework)
- `r`: HTTP request containing headers, URL, and parsed parameters
- `ctx`: For bound functions, contains the entity instance (cast to appropriate type). For unbound functions, nil.
- `params`: Map of parameter names to parsed values (use type assertions to access)

**Returns:**
- `interface{}`: The computed result (must match `ReturnType`)
- `error`: Return nil on success, or an error to abort and send error response

**Framework Responsibilities:**
- Automatically serializes return value to JSON
- Sets appropriate HTTP status code (200 OK)
- Adds OData context annotations

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
  "@odata.context": "$metadata#Products",
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
  "@odata.context": "$metadata#Edm.Double",
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
    {Name: "addresses", Type: reflect.TypeOf([]Address{}), Required: false}, // slice of structs
    {Name: "options", Type: reflect.TypeOf(map[string]interface{}{}), Required: false}, // map
    {Name: "shipping", Type: reflect.TypeOf(&Address{}), Required: false}, // pointer to struct
}
```

### Binding Parameters to Structs

When a payload naturally maps to a Go struct, you can eliminate manual `ParameterDefinition`
lists by setting `ParameterStructType`. go-odata will derive the correct parameter metadata
and keep the legacy parameter map populated for backwards compatibility.

Inside the handler, use the generic helper `actions.BindParams[T]` to obtain a strongly typed
struct without repetitive type assertions:

```go
import (
    odata "github.com/nlstn/go-odata"
    "github.com/nlstn/go-odata/actions"
)

type ApplyDiscountInput struct {
    Percentage float64 `mapstructure:"percentage"`
    Reason     *string `mapstructure:"reason,omitempty"`
}

service.RegisterAction(odata.ActionDefinition{
    Name:                "ApplyDiscount",
    EntitySet:           "Products",
    ParameterStructType: reflect.TypeOf(ApplyDiscountInput{}),
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
        input, err := actions.BindParams[ApplyDiscountInput](params)
        if err != nil {
            return err
        }

        // input.Percentage is a float64, input.Reason is *string (nil when omitted)
        // ... perform work ...
        return nil
    },
})
```

Pointers or fields tagged with `,omitempty` are treated as optional. You can also request
`*ApplyDiscountInput` by calling `actions.BindParams[*ApplyDiscountInput](params)` if you prefer
pointer semantics.

The same pattern works for functions—set `ParameterStructType` on the `FunctionDefinition`
and call `actions.BindParams` inside the function handler.

Supported types:
- `string` - Text values
- `int`, `int32`, `int64` - Integer values
- `float32`, `float64` - Decimal values
- `bool` - Boolean values (`true`/`false`)
- Structs (and pointers to structs) - JSON objects map directly to Go structs
- Slices and arrays - JSON arrays are converted to the Go slice/array element type
- Maps - JSON objects are decoded into Go map keys/values

### Composite Parameters

Complex parameter payloads are decoded using reflection, so you can accept nested JSON objects and arrays without manual parsing.

#### Actions

When invoking an action, include JSON in the request body matching the target Go types:

```http
POST /Orders/Process
Content-Type: application/json

{
  "order": {
    "address": {
      "street": "Main St",
      "tags": ["primary", "billing"],
      "metadata": {"zone": "north"}
    },
    "lines": [
      {"sku": "A-100", "quantity": 2},
      {"sku": "B-200", "quantity": 1}
    ]
  },
  "notify": true
}
```

```go
type OrderLine struct {
    SKU      string `json:"sku"`
    Quantity int    `json:"quantity"`
}

type OrderInput struct {
    Address Address     `json:"address"`
    Lines   []OrderLine `json:"lines"`
}

service.RegisterAction(odata.ActionDefinition{
    Name: "Process",
    EntitySet: "Orders",
    Parameters: []odata.ParameterDefinition{
        {Name: "order", Type: reflect.TypeOf(OrderInput{}), Required: true},
        {Name: "notify", Type: reflect.TypeOf(false), Required: false},
    },
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
        input := params["order"].(OrderInput)
        // Use input.Address and input.Lines directly
        return nil
    },
})
```

#### Functions

Functions support JSON fragments for composite parameters supplied in the URL query string or function-call syntax:

```go
service.RegisterFunction(odata.FunctionDefinition{
    Name: "EstimateShipping",
    Parameters: []odata.ParameterDefinition{
        {Name: "addresses", Type: reflect.TypeOf([]Address{}), Required: true},
        {Name: "options", Type: reflect.TypeOf(map[string]interface{}{}), Required: false},
    },
    ReturnType: reflect.TypeOf(float64(0)),
    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
        addresses := params["addresses"].([]Address)
        options, _ := params["options"].(map[string]interface{})
        return calculate(addresses, options), nil
    },
})
```

```bash
# Query string JSON fragments (URL encoded)
GET /EstimateShipping?addresses=%5B%7B%22street%22%3A%22Main%22%7D%5D&options=%7B%22priority%22%3Atrue%7D

# Function call syntax with encoded JSON object
GET /EstimateShipping(options=%7B%22priority%22%3Atrue%7D)
```

The framework instantiates zero values for each parameter definition, unmarshals JSON into the correct Go types, and validates assignability—including pointer targets—before invoking your handler.

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

### Navigation Targets and Renamed Properties

When invoking a bound action or function after traversing a navigation property, the router now resolves the target entity set using metadata rather than assuming the property name matches the entity set. This enables patterns such as:

```text
/Stores(1)/FeaturedItems/GetAveragePrice()
```

Where `FeaturedItems` is a navigation property that targets the `NavigationFunctionItems` entity set. The `internal/service/router.EntityHandler` interface exposes a `NavigationTargetSet(name string) (string, bool)` helper so custom handlers can surface the correct entity set names. The built-in entity handler automatically implements this by inspecting the metadata collected during `Service.RegisterEntity`, so no extra configuration is required when your navigation properties use descriptive names.

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
