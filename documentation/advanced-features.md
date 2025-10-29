# Advanced Features

This guide covers advanced features of go-odata including singletons, ETags, and lifecycle hooks.

## Table of Contents

- [Singletons](#singletons)
- [ETags (Optimistic Concurrency Control)](#etags-optimistic-concurrency-control)
- [Lifecycle Hooks](#lifecycle-hooks)
  - [Read Hooks](#read-hooks)
  - [Tenant Filtering Example](#tenant-filtering-example)
  - [Redacting Sensitive Data](#redacting-sensitive-data)
- [Asynchronous Processing](#asynchronous-processing)

## Singletons

Singletons are special entities that represent a single instance of an entity type, accessible directly by name without requiring a key.

### When to Use Singletons

Use singletons for unique resources like:
- Company information
- Application settings
- User profiles
- System configuration

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

- ✅ **Direct Access**: No key required - access via `/Company` instead of `/Companies(1)`
- ✅ **Full CRUD Support**: Supports GET, PATCH, and PUT operations
- ✅ **Metadata Integration**: Automatically appears in service document and metadata
- ✅ **ETag Support**: Full support for optimistic concurrency control
- ✅ **Prefer Header**: Supports `return=representation` and `return=minimal`
- ✅ **Navigation Properties**: Can have navigation properties to other entities

❌ POST (Create) and DELETE are not applicable to singletons

### Singleton vs Entity Set

| Feature | Entity Set | Singleton |
|---------|-----------|-----------|
| URL Pattern | `/EntitySet(key)` | `/SingletonName` |
| Key Required | Yes | No |
| Multiple Instances | Yes | No |
| POST (Create) | ✅ | ❌ |
| GET (Read) | ✅ | ✅ |
| PATCH (Update) | ✅ | ✅ |
| PUT (Replace) | ✅ | ✅ |
| DELETE | ✅ | ❌ |

### Service Document with Singleton

Singletons automatically appear in the service document:

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

## ETags (Optimistic Concurrency Control)

ETags enable optimistic concurrency control, preventing concurrent updates from overwriting each other's changes.

### Defining an ETag Property

Mark a field with the `odata:"etag"` tag:

```go
type Product struct {
    ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
    Name     string  `json:"Name" odata:"required"`
    Price    float64 `json:"Price"`
    Version  int     `json:"Version" odata:"etag"` // Used for ETag generation
}
```

You can use any field type for ETags:
- **Integer fields** (e.g., `Version int`) - Common for version numbers
- **Timestamp fields** (e.g., `LastModified time.Time`) - Tracks last modification
- **String fields** - Custom version identifiers

### How ETags Work

1. **GET requests** return an `ETag` header and `@odata.etag` annotation
2. **Clients** store the ETag and send it back in an `If-Match` header when updating
3. **UPDATE/DELETE operations** validate that the `If-Match` header matches the current ETag
4. If ETags don't match, a `412 Precondition Failed` response is returned

### Using ETags

**Step 1: Retrieve an entity**
```bash
GET /Products(1)
```

Response headers:
```
HTTP/1.1 200 OK
ETag: W/"abc123def456..."
OData-Version: 4.0
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

**Step 2: Update with If-Match header**
```bash
PATCH /Products(1)
If-Match: W/"abc123def456..."
Content-Type: application/json

{
  "Price": 899.99
}
```

**If ETag matches:**
```
HTTP/1.1 204 No Content
OData-Version: 4.0
```

**If ETag doesn't match (modified by another client):**
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

- **Specific ETag**: `If-Match: W/"abc123..."` - Match only if ETag is exactly this value
- **Wildcard**: `If-Match: *` - Match if the entity exists (any ETag value)
- **No header**: Update proceeds without validation

### ETag Generation

ETags are automatically generated as weak ETags (format: `W/"hash"`) using SHA-256 hash of the ETag field value. The same field value always produces the same ETag.

Per OData v4 specification, ETags are included in:
1. **HTTP Header**: `ETag: W/"abc123..."` - Used by clients for conditional requests
2. **Response Body**: `"@odata.etag": "W/\"abc123...\""` - Included in JSON response

### Operations Supporting If-Match

- ✅ `PATCH /EntitySet(key)` - Partial update with ETag validation
- ✅ `PUT /EntitySet(key)` - Full replacement with ETag validation
- ✅ `DELETE /EntitySet(key)` - Delete with ETag validation

### Best Practices

1. **Use version numbers** for simple counter-based concurrency control
2. **Use timestamps** when you need to track when entities were last modified
3. **Always check for 412 responses** in your client code
4. **Refresh data** when receiving a 412 response before retrying

## Lifecycle Hooks

Lifecycle hooks allow you to execute custom logic at specific points in the entity lifecycle.

## Change Tracking and Delta Tokens

OData delta responses allow clients to synchronize changes efficiently, but they are optional in the specification. The library
now requires an explicit opt-in per entity set before emitting delta links or processing `$deltatoken` requests.

1. Register your entity as usual:

```go
if err := service.RegisterEntity(&Product{}); err != nil {
    log.Fatalf("register product: %v", err)
}
```

2. Enable change tracking for that entity set:

```go
if err := service.EnableChangeTracking("Products"); err != nil {
    log.Fatalf("enable change tracking: %v", err)
}
```

When enabled, the entity handler records inserts, updates, and deletions so that collection requests with the `Prefer:
odata.track-changes` header include an `@odata.deltaLink`. Subsequent requests using the returned `$deltatoken` yield only the
changes since the previous checkpoint.

**Important notes:**

- Change tracking is disabled by default for every entity set.
- Singletons do not support change tracking. Attempting to enable it returns an error.
- If a client supplies `$deltatoken` for an entity set without change tracking enabled, the service returns `501 Not
  Implemented` with an explanatory error message.
- The change tracker keeps an in-memory history per entity set. Restarting the process clears the tracked history.

### Available Hooks

The library supports the following hooks:

- `BeforeCreate` - Called before creating a new entity
- `AfterCreate` - Called after creating a new entity
- `BeforeUpdate` - Called before updating an entity
- `AfterUpdate` - Called after updating an entity
- `BeforeDelete` - Called before deleting an entity
- `AfterDelete` - Called after deleting an entity
- `BeforeReadCollection` - Called before reading a collection (applies additional GORM scopes)
- `AfterReadCollection` - Called after reading a collection (allows mutating/overriding the result)
- `BeforeReadEntity` - Called before reading a single entity (applies additional GORM scopes)
- `AfterReadEntity` - Called after reading a single entity (allows mutating/overriding the result)

### Implementing Hooks

Define hooks by implementing the corresponding interface:

> **Why the extra parameters?** Every hook receives the request context and the active `*http.Request` so that you can check
> cancellation, deadlines, authentication headers, or other per-request metadata before deciding how to handle the entity.
> Returning a non-`nil` error from any hook aborts the pipeline immediately and propagates the error back to the client.

```go
type Product struct {
    ID        uint      `json:"ID" gorm:"primaryKey" odata:"key"`
    Name      string    `json:"Name" odata:"required"`
    Price     float64   `json:"Price"`
    CreatedAt time.Time `json:"CreatedAt"`
    UpdatedAt time.Time `json:"UpdatedAt"`
}

// BeforeCreate hook
func (p *Product) BeforeCreate(ctx context.Context, r *http.Request) error {
    p.CreatedAt = time.Now()
    if p.Price < 0 {
        return fmt.Errorf("price cannot be negative")
    }
    return nil
}

// BeforeUpdate hook
func (p *Product) BeforeUpdate(ctx context.Context, r *http.Request) error {
    p.UpdatedAt = time.Now()
    if p.Price < 0 {
        return fmt.Errorf("price cannot be negative")
    }
    return nil
}

// AfterCreate hook
func (p *Product) AfterCreate(ctx context.Context, r *http.Request) error {
    log.Printf("Product created: %s (ID: %d)", p.Name, p.ID)
    return nil
}
```

### Hook Use Cases

**Validation:**
```go
func (p *Product) BeforeCreate(_ context.Context, _ *http.Request) error {
    if p.Price < 0 {
        return fmt.Errorf("price cannot be negative")
    }
    if len(p.Name) < 3 {
        return fmt.Errorf("name must be at least 3 characters")
    }
    return nil
}
```

**Timestamps:**
```go
func (p *Product) BeforeCreate(_ context.Context, _ *http.Request) error {
    p.CreatedAt = time.Now()
    p.UpdatedAt = time.Now()
    return nil
}

func (p *Product) BeforeUpdate(_ context.Context, _ *http.Request) error {
    p.UpdatedAt = time.Now()
    return nil
}
```

**Audit Logging:**
```go
func (p *Product) AfterUpdate(ctx context.Context, r *http.Request) error {
    auditLog := AuditLog{
        EntityType: "Product",
        EntityID:   p.ID,
        Action:     "UPDATE",
        Timestamp:  time.Now(),
    }
    return db.Create(&auditLog).Error
}
```

**Business Logic:**
```go
func (o *Order) BeforeCreate(ctx context.Context, r *http.Request) error {
    // Calculate total from items
    var total float64
    for _, item := range o.Items {
        total += item.Price * float64(item.Quantity)
    }
    o.TotalAmount = total
    return nil
}
```

### Hook Error Handling

If a hook returns an error:
- The operation is aborted
- The error is returned to the client
- Database changes are rolled back (for operations in transactions)

```go
func (p *Product) BeforeCreate(_ context.Context, _ *http.Request) error {
    if p.Price > 100000 {
        return fmt.Errorf("price exceeds maximum allowed value")
    }
    return nil
}
```

Client receives:
```json
{
  "error": {
    "code": "400",
    "message": "price exceeds maximum allowed value"
  }
}
```

### Read Hooks

Read hooks let you shape read behavior without forking handlers:

- **Before hooks** (`BeforeReadCollection` / `BeforeReadEntity`) return additional [GORM scopes](https://gorm.io/docs/scopes.html). Each scope is applied to the underlying query *before* OData options like `$filter`, `$orderby`, `$top`, `$skip`, and `$count` execute. This is the preferred place for authorization filters, tenant scoping, or eager-loading navigation properties.
- **After hooks** (`AfterReadCollection` / `AfterReadEntity`) receive the fetched results after all query options and pagination have been applied. They can mutate or replace the response payload (e.g., redact fields, append computed properties) before it is sent to the client.

Hook signatures:

```go
func (Product) BeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)
func (Product) AfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error)
func (Product) BeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)
func (Product) AfterReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, error)
```

Each hook receives the active HTTP request, context, and parsed OData query options. Returning an error aborts the request and surfaces the error to the client.
Return `(nil, nil)` from an After hook to keep the original response body.

### Tenant Filtering Example

Apply multi-tenant filters centrally by returning scopes from `BeforeReadCollection` and `BeforeReadEntity` hooks:

```go
// Requires: import "fmt" and "gorm.io/gorm"
type Product struct {
    ID        uint   `json:"ID" gorm:"primaryKey" odata:"key"`
    Name      string `json:"Name"`
    TenantID  string `json:"TenantID"`
}

func (Product) tenantScope(tenantID string) func(*gorm.DB) *gorm.DB {
    return func(db *gorm.DB) *gorm.DB {
        return db.Where("tenant_id = ?", tenantID)
    }
}

func (Product) BeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
    tenantID := r.Header.Get("X-Tenant-ID")
    if tenantID == "" {
        return nil, fmt.Errorf("missing tenant header")
    }
    return []func(*gorm.DB) *gorm.DB{Product{}.tenantScope(tenantID)}, nil
}

func (Product) BeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
    tenantID := r.Header.Get("X-Tenant-ID")
    if tenantID == "" {
        return nil, fmt.Errorf("missing tenant header")
    }
    return []func(*gorm.DB) *gorm.DB{Product{}.tenantScope(tenantID)}, nil
}
```

By returning scopes instead of mutating the request, the same tenant filter is applied consistently across `$count`, pagination, `$expand`, and navigation reads.

### Redacting Sensitive Data

Use `AfterReadEntity` or `AfterReadCollection` to redact fields just before they leave the service:

```go
func (Product) AfterReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, error) {
    product, ok := entity.(*Product)
    if !ok {
        return entity, nil
    }

    if !isPrivileged(r) {
        product.CostPrice = 0
    }
    return product, nil
}

func (Product) AfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error) {
    products, ok := results.([]Product)
    if !ok {
        return results, nil
    }

    if !isPrivileged(r) {
        for i := range products {
            products[i].CostPrice = 0
        }
    }
    return products, nil
}
```

After hooks execute after all database work is complete, so they can safely adjust derived responses or redact sensitive information.

### Hook Order

For a single operation, hooks execute in this order:

**Create:**
1. BeforeCreate
2. Database INSERT
3. AfterCreate

**Update:**
1. BeforeUpdate
2. Database UPDATE
3. AfterUpdate

**Delete:**
1. BeforeDelete
2. Database DELETE
3. AfterDelete

## Asynchronous Processing

`go-odata` can run long-running requests asynchronously when clients send `Prefer: respond-async`. Enable it with `Service.EnableAsyncProcessing` and provide a monitor prefix (defaults to `/$async/jobs/`). Once enabled, the router reserves the monitor path and exposes the following behaviour:

- **Status monitors** live at `/{prefix}{jobID}` (for example, `/$async/jobs/6f92b3...`). Job IDs are limited to ASCII alphanumeric characters plus `_` and `-`. Requests that omit the job ID, add extra path segments, or include disallowed characters return `404 Not Found` or `400 Bad Request` before the async manager is invoked.
- **Polling** a pending job with `GET` or `HEAD` returns `202 Accepted` with `Preference-Applied: respond-async` and, when configured, a `Retry-After` header. The 202 response never includes a body.
- **Completion** replays the stored response exactly: the original status code, headers (including any `Preference-Applied` values set by the handler), and body are forwarded once the job finishes.
- **Cancellation** is available via `DELETE` on the monitor URI. The router calls the async manager’s cancellation hook and returns `204 No Content`, even if the worker finishes later.

These guarantees let clients reliably poll job status without conflicting with regular entity routing.
