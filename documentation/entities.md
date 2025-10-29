# Entity Definition

This guide covers how to define entities in go-odata using Go structs with appropriate tags.

## Table of Contents

- [Basic Entity](#basic-entity)
- [Entity with Rich Metadata](#entity-with-rich-metadata)
- [Entity with Relationships](#entity-with-relationships)
- [Composite Keys](#composite-keys)
- [Supported Tags](#supported-tags)
- [Read Hooks and Query Options](#read-hooks-and-query-options)

## Basic Entity

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

**Register the entity:**

```go
service := odata.NewService(db)
if err := service.RegisterEntity(&Product{}); err != nil {
    // Surface registration errors like invalid tags or duplicate entity set names.
    return err
}
```

Always bubble up registration errors—problems such as malformed tags or duplicate entity names are detected during
`RegisterEntity` and should be fixed before serving requests.

## Entity with Rich Metadata

Add metadata facets to provide richer type information in the OData metadata document:

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

These metadata facets will be reflected in the generated `$metadata` document, helping clients understand the data model constraints.

## Entity with Relationships

Define relationships between entities using GORM foreign key tags:

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

**Accessing related entities:**

```bash
# Get customer with expanded orders
GET /Customers(1)?$expand=Orders

# Get order with customer details
GET /Orders(1)?$expand=Customer
```

## Composite Keys

For entities with composite keys, mark multiple fields with `odata:"key"`:

```go
type OrderItem struct {
    OrderID   int     `json:"orderId" gorm:"primaryKey" odata:"key"`
    ProductID int     `json:"productId" gorm:"primaryKey" odata:"key"`
    Quantity  int     `json:"quantity" odata:"required"`
    Price     float64 `json:"price" odata:"precision=10,scale=2"`
}
```

**Accessing entities with composite keys:**

```bash
GET /OrderItems(OrderID=1,ProductID=5)
```

## Supported Tags

### OData Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `key` | Marks the field as the entity key (required) | `odata:"key"` |
| `etag` | Marks the field to be used for ETag generation | `odata:"etag"` |
| `required` | Marks the field as required | `odata:"required"` |
| `maxlength=N` | Sets the maximum length for string properties | `odata:"maxlength=100"` |
| `precision=N` | Sets the precision for numeric properties | `odata:"precision=10"` |
| `scale=N` | Sets the scale for decimal properties | `odata:"scale=2"` |
| `default=VALUE` | Sets the default value for the property | `odata:"default=AUTO"` |
| `nullable` | Explicitly marks the field as nullable | `odata:"nullable"` |
| `nullable=false` | Explicitly marks the field as non-nullable | `odata:"nullable=false"` |
| `searchable` | Marks field as searchable for `$search` queries | `odata:"searchable"` |
| `fuzziness=N` | Sets fuzzy matching tolerance for search (1=exact, 2+=fuzzy) | `odata:"searchable,fuzziness=2"` |
| `similarity=X` | Sets similarity score threshold for search (0.0-1.0, where 0.95=95% similar) | `odata:"searchable,similarity=0.8"` |

### JSON Tags

Use `json` tags to specify field names in JSON responses:

```go
type Product struct {
    ID   int    `json:"id" odata:"key"`
    Name string `json:"name" odata:"required"`
}
```

### GORM Tags

Use standard GORM tags for database configuration:

```go
type Product struct {
    ID          int       `gorm:"primaryKey"`
    Name        string    `gorm:"not null;index"`
    Description string    `gorm:"type:text"`
    CreatedAt   time.Time `gorm:"autoCreateTime"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}
```

For relationships:

```go
type Order struct {
    CustomerID int       `gorm:"not null;index"`
    Customer   *Customer `gorm:"foreignKey:CustomerID;references:ID"`
}
```

## Type Mappings

Go types are automatically mapped to EDM types in the OData metadata:

| Go Type | EDM Type |
|---------|----------|
| `string` | `Edm.String` |
| `int`, `int32` | `Edm.Int32` |
| `int64` | `Edm.Int64` |
| `int16` | `Edm.Int16` |
| `int8` | `Edm.SByte` |
| `uint`, `uint32` | `Edm.Int64` |
| `uint64` | `Edm.Int64` |
| `uint16` | `Edm.Int32` |
| `uint8`, `byte` | `Edm.Byte` |
| `float32` | `Edm.Single` |
| `float64` | `Edm.Double` |
| `bool` | `Edm.Boolean` |
| `time.Time` | `Edm.DateTimeOffset` |
| `[]byte` | `Edm.Binary` |

## Read Hooks and Query Options

Read hooks run alongside the entity metadata you define here. When you implement `BeforeReadCollection` or `BeforeReadEntity`, return one or more [GORM scopes](https://gorm.io/docs/scopes.html). `go-odata` applies those scopes to the base query *before* it executes OData options such as `$filter`, `$orderby`, `$top`, `$skip`, `$expand`, and `$count`.

Best practices:

- **Return scopes, not mutations.** Always return scopes from `BeforeRead*` hooks instead of modifying the `*gorm.DB` manually. This keeps the handler free to compose query options, pagination, `$count`, and `$expand` requests using the same filtered query.
- **Handle `$count` transparently.** The same scopes are reused when clients request `$count=true`, so tenant filters or soft-delete predicates remain in sync.
- **Keep After hooks pure.** `AfterReadEntity` and `AfterReadCollection` receive the materialized result *after* pagination and projections. Use them to redact or enrich the payload, but avoid additional database work to keep responses fast.

Example multi-tenant hook that preserves pagination and `$count` alignment:

```go
// Requires: import "fmt" and "gorm.io/gorm"
func (Order) BeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
    tenantID := r.Header.Get("X-Tenant-ID")
    if tenantID == "" {
        return nil, fmt.Errorf("missing tenant header")
    }

    scope := func(db *gorm.DB) *gorm.DB {
        return db.Where("tenant_id = ?", tenantID)
    }
    return []func(*gorm.DB) *gorm.DB{scope}, nil
}
```

Because the hook returns a scope, the filtered tenant query feeds directly into `$top/$skip` pagination, navigation property reads, and `$count` calculations without extra code.

## Configuring Search

By default, all string properties are searchable when using `$search`. You can control this:

```go
type Product struct {
    ID          int    `json:"ID" odata:"key"`
    Name        string `json:"Name" odata:"searchable"`           // Searchable
    Description string `json:"Description" odata:"searchable"`    // Searchable
    SKU         string `json:"SKU"`                               // Not searchable
    Category    string `json:"Category"`                          // Not searchable
}
```

### Fuzzy Matching

Configure fuzzy matching tolerance using the `fuzziness` parameter:

```go
type Product struct {
    Name  string `odata:"searchable,fuzziness=1"`  // Exact match only
    Email string `odata:"searchable,fuzziness=2"`  // 1 char difference allowed
    Tags  string `odata:"searchable,fuzziness=3"`  // 2 char differences allowed
}
```

The fuzziness value determines how many character differences are allowed when matching:
- `fuzziness=1`: Exact substring match (default)
- `fuzziness=2`: Allows 1 character difference
- `fuzziness=3`: Allows 2 character differences

### Similarity-Based Matching

Configure similarity-based matching using the `similarity` parameter (value between 0.0 and 1.0):

```go
type User struct {
    ID       int    `json:"ID" odata:"key"`
    Name     string `odata:"searchable,similarity=0.8"`   // Must be at least 80% similar
    Email    string `odata:"searchable,similarity=0.9"`   // Must be at least 90% similar
    Username string `odata:"searchable,similarity=0.95"`  // Must be at least 95% similar
}
```

The similarity value represents the minimum similarity threshold:
- `similarity=0.95`: Field must be at least 95% similar to the search term
- `similarity=0.8`: Field must be at least 80% similar to the search term
- Similarity is calculated using normalized Levenshtein distance

**Important**: A field cannot have both `fuzziness` and `similarity` defined. You must choose one or the other. An error will be raised at startup if both are specified for the same field.

```go
// ❌ INVALID - Cannot use both fuzziness and similarity
type Product struct {
    Name string `odata:"searchable,fuzziness=2,similarity=0.8"`  // Error!
}

// ✅ VALID - Use either fuzziness or similarity
type Product struct {
    Name  string `odata:"searchable,fuzziness=2"`    // OK
    Email string `odata:"searchable,similarity=0.9"`  // OK
}
```

