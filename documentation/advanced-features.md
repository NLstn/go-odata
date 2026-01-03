# Advanced Features

This guide covers advanced features of go-odata including singletons, ETags, lifecycle hooks, and HTTP method restrictions.

## Table of Contents

- [Singletons](#singletons)
- [ETags (Optimistic Concurrency Control)](#etags-optimistic-concurrency-control)
- [Disabling HTTP Methods](#disabling-http-methods)
- [Lifecycle Hooks](#lifecycle-hooks)
- [Pre-Request Hook](#pre-request-hook)
- [Change Tracking and Delta Tokens](#change-tracking-and-delta-tokens)
  - [Read Hooks](#read-hooks)
  - [Tenant Filtering Example](#tenant-filtering-example)
  - [Redacting Sensitive Data](#redacting-sensitive-data)
- [Asynchronous Processing](#asynchronous-processing)
- [Full-Text Search with Database FTS](#full-text-search-with-database-fts)

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

## Disabling HTTP Methods

You can easily restrict specific HTTP methods for entities without needing to implement hooks. This is useful when you want to make entities read-only, prevent creation, or disable deletion.

### When to Use Method Restrictions

Use method restrictions when you want to:
- Make entities read-only (disable POST, PUT, PATCH, DELETE)
- Prevent creation of new entities (disable POST)
- Prevent deletion (disable DELETE)
- Prevent modifications (disable PUT, PATCH)
- Control access at the endpoint level for specific entities

### Disabling Methods

```go
service := odata.NewService(db)

// Register entity
if err := service.RegisterEntity(&User{}); err != nil {
    log.Fatal(err)
}

// Disable POST for Users - prevents creating new users
if err := service.DisableHTTPMethods("Users", "POST"); err != nil {
    log.Fatal(err)
}
```

### Supported HTTP Methods

You can disable any of the following methods:
- `GET` - Read operations (collections, single entities, $count)
- `POST` - Create operations
- `PUT` - Full replacement update
- `PATCH` - Partial update
- `DELETE` - Delete operations

### Examples

**Read-Only Entity**
```go
// Make Products read-only
service.DisableHTTPMethods("Products", "POST", "PUT", "PATCH", "DELETE")
```

**Prevent Creation and Deletion**
```go
// Allow updates but not creation or deletion
service.DisableHTTPMethods("Categories", "POST", "DELETE")
```

**Disable Single Method**
```go
// Prevent deletion only
service.DisableHTTPMethods("Orders", "DELETE")
```

### HTTP Response

When a disabled method is called, the service returns HTTP 405 Method Not Allowed:

```bash
POST /Users
Content-Type: application/json
{"name": "John"}
```

Response:
```
HTTP/1.1 405 Method Not Allowed
Content-Type: application/json
{
  "error": {
    "code": "405",
    "message": "Method not allowed",
    "details": [{
      "message": "Method POST is not allowed for this entity"
    }]
  }
}
```

### Method Checking

The method restriction is checked before any other processing, including:
- Request body parsing
- Hook execution
- Database operations
- Authorization checks (from hooks)

This means disabled methods are rejected immediately, minimizing processing overhead.

### Combining with Hooks

You can still use hooks with entities that have disabled methods. For example, you might disable POST but use hooks for authorization checks on GET requests:

```go
// Disable POST
service.DisableHTTPMethods("Users", "POST")

// Add authorization hook for GET
type User struct {
    ID   int    `json:"id" odata:"key"`
    Name string `json:"name"`
}

func (u User) ODataBeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
    // Add tenant filter
    tenantID := r.Header.Get("X-Tenant-ID")
    if tenantID == "" {
        return nil, fmt.Errorf("missing tenant header")
    }
    return []func(*gorm.DB) *gorm.DB{
        func(db *gorm.DB) *gorm.DB {
            return db.Where("tenant_id = ?", tenantID)
        },
    }, nil
}
```

### Error Cases

The method returns an error if:
- The entity set is not registered
- An unsupported HTTP method is specified

```go
// Error: entity not registered
err := service.DisableHTTPMethods("NonExistent", "POST")
// Returns: "entity set 'NonExistent' is not registered"

// Error: invalid method
err := service.DisableHTTPMethods("Users", "INVALID")
// Returns: "unsupported HTTP method 'INVALID'; supported methods are GET, POST, PUT, PATCH, DELETE"
```

## Lifecycle Hooks

Lifecycle hooks allow you to execute custom logic at specific points in the entity lifecycle.

## Pre-Request Hook

The PreRequestHook provides a unified mechanism for request preprocessing that works consistently for both single requests and batch sub-requests. This eliminates the need for manual middleware configuration when you need authentication or context enrichment.

### When to Use PreRequestHook

Use PreRequestHook when you need to:
- Load user information from JWT tokens into request context
- Validate API keys and set tenant context
- Log all requests (including batch sub-requests)
- Set request-scoped values for downstream handlers

### Setting Up PreRequestHook

```go
service := odata.NewService(db)
service.RegisterEntity(&Product{})

// Define a context key for storing user info
type contextKey string
const userContextKey contextKey = "user"

// Set the pre-request hook
service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
    token := r.Header.Get("Authorization")
    if token == "" {
        return nil, nil // Allow anonymous access
    }
    
    // Validate token and load user
    user, err := validateAndParseToken(token)
    if err != nil {
        return nil, fmt.Errorf("authentication failed: %w", err)
    }
    
    // Add user to context for downstream handlers
    return context.WithValue(r.Context(), userContextKey, user), nil
})
```

### Hook Return Values

The hook can return:
- `(nil, nil)`: Request proceeds with the original context
- `(ctx, nil)`: Request proceeds with the returned context
- `(_, err)`: Request is aborted with HTTP 403 Forbidden

### Batch Sub-Request Support

The hook is automatically called for:
- Single HTTP requests to the service
- Each batch sub-request (both changeset and non-changeset operations)

This means authentication logic runs consistently regardless of how the request arrives:

```go
// Single request - hook is called
GET /Products(1)
Authorization: Bearer token123

// Batch request - hook is called for outer request AND each sub-request
POST /$batch
Authorization: Bearer outer-token

--batch_boundary
Content-Type: application/http

GET /Products(1) HTTP/1.1
Authorization: Bearer sub-request-token
--batch_boundary--
```

### Using Context Values in Hooks

Values set in the PreRequestHook context are available in entity lifecycle hooks:

```go
service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
    userID := r.Header.Get("X-User-ID")
    return context.WithValue(r.Context(), userContextKey, userID), nil
})

func (p *Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
    // Access the user ID set by PreRequestHook
    userID := ctx.Value(userContextKey).(string)
    p.CreatedBy = userID
    return nil
}
```

### Comparison with SetBatchSubRequestHandler

The `SetPreRequestHook` method is simpler and more reliable than `SetBatchSubRequestHandler` for most use cases:

| Feature | PreRequestHook | SetBatchSubRequestHandler |
|---------|---------------|---------------------------|
| Works for single requests | ✅ | ❌ (middleware only) |
| Works for batch sub-requests | ✅ | ✅ |
| Works for changeset operations | ✅ | ❌ |
| Requires middleware configuration | ❌ | ✅ |
| Can abort with error | ✅ | Depends on middleware |
| Can enrich context | ✅ | ✅ |

Use `SetBatchSubRequestHandler` only when you need batch sub-requests to pass through complex middleware chains that cannot be replicated in a hook.

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

By default the tracker caches change history in memory. For long-lived deployments you can opt into persistence so that delta
tokens continue to work after a process restart:

```go
service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{
    PersistentChangeTracking: true,
})
if err != nil {
    log.Fatalf("configure service: %v", err)
}
```

The persistent tracker stores events inside the reserved `_odata_change_log` table and reloads them during service startup.
Make sure your migrations or provisioning scripts allow the library to create and manage that table.

**Important notes:**

- Change tracking is disabled by default for every entity set.
- Singletons do not support change tracking. Attempting to enable it returns an error.
- If a client supplies `$deltatoken` for an entity set without change tracking enabled, the service returns `501 Not
  Implemented` with an explanatory error message.
- In-memory tracking (the default) loses history on restart. Enable `PersistentChangeTracking` to preserve delta tokens across
  restarts.
- When persistence is enabled, plan for database retention—`_odata_change_log` grows with each change event and should be
  purged according to your data lifecycle requirements.

### Available Hooks

The library supports the following hooks:

- `ODataBeforeCreate` - Called before creating a new entity
- `ODataAfterCreate` - Called after creating a new entity
- `ODataBeforeUpdate` - Called before updating an entity
- `ODataAfterUpdate` - Called after updating an entity
- `ODataBeforeDelete` - Called before deleting an entity
- `ODataAfterDelete` - Called after deleting an entity
- `ODataBeforeReadCollection` - Called before reading a collection (applies additional GORM scopes)
- `ODataAfterReadCollection` - Called after reading a collection (allows mutating/overriding the result)
- `ODataBeforeReadEntity` - Called before reading a single entity (applies additional GORM scopes)
- `ODataAfterReadEntity` - Called after reading a single entity (allows mutating/overriding the result)

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

// ODataBeforeCreate hook
func (p *Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
    p.CreatedAt = time.Now()
    if p.Price < 0 {
        return fmt.Errorf("price cannot be negative")
    }
    return nil
}

// ODataBeforeUpdate hook
func (p *Product) ODataBeforeUpdate(ctx context.Context, r *http.Request) error {
    p.UpdatedAt = time.Now()
    if p.Price < 0 {
        return fmt.Errorf("price cannot be negative")
    }
    return nil
}

// ODataAfterCreate hook
func (p *Product) ODataAfterCreate(ctx context.Context, r *http.Request) error {
    log.Printf("Product created: %s (ID: %d)", p.Name, p.ID)
    return nil
}
```

### Using the active transaction inside hooks

Entity and collection write handlers execute inside a shared GORM transaction. The active `*gorm.DB` is stored on the request
context so your hooks can participate in the same transaction by calling `odata.TransactionFromContext`:

```go
func (p *Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
    tx, ok := odata.TransactionFromContext(ctx)
    if !ok {
        return fmt.Errorf("transaction unavailable")
    }

    audit := CreationAudit{
        ProductID: p.ID,
        PerformedBy: r.Header.Get("X-User"),
    }
    if err := tx.Create(&audit).Error; err != nil {
        return err
    }
    return nil
}

func (p *Product) ODataBeforeUpdate(ctx context.Context, r *http.Request) error {
    tx, ok := odata.TransactionFromContext(ctx)
    if !ok {
        return fmt.Errorf("transaction unavailable")
    }

    if err := tx.Model(&Inventory{}).
        Where("product_id = ?", p.ID).
        Update("last_checked_at", time.Now()).Error; err != nil {
        return err
    }

    if p.IsRetired {
        return fmt.Errorf("retired products cannot be edited")
    }
    return nil
}
```

Any error returned by a hook still aborts the handler, rolling back changes performed via the shared transaction so partial updates
never escape to the database.

### Hook Use Cases

**Validation:**
```go
func (p *Product) ODataBeforeCreate(_ context.Context, _ *http.Request) error {
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
func (p *Product) ODataBeforeCreate(_ context.Context, _ *http.Request) error {
    p.CreatedAt = time.Now()
    p.UpdatedAt = time.Now()
    return nil
}

func (p *Product) ODataBeforeUpdate(_ context.Context, _ *http.Request) error {
    p.UpdatedAt = time.Now()
    return nil
}
```

**Audit Logging:**
```go
func (p *Product) ODataAfterUpdate(ctx context.Context, r *http.Request) error {
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
func (o *Order) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
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
func (p *Product) ODataBeforeCreate(_ context.Context, _ *http.Request) error {
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

- **Before hooks** (`ODataBeforeReadCollection` / `ODataBeforeReadEntity`) return additional [GORM scopes](https://gorm.io/docs/scopes.html). Each scope is applied to the underlying query *before* OData options like `$filter`, `$orderby`, `$top`, `$skip`, and `$count` execute. This is the preferred place for authorization filters, tenant scoping, or eager-loading navigation properties.
- **After hooks** (`ODataAfterReadCollection` / `ODataAfterReadEntity`) receive the fetched results after all query options and pagination have been applied. They can mutate or replace the response payload (e.g., redact fields, append computed properties) before it is sent to the client.

Hook signatures:

```go
func (Product) ODataBeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)
func (Product) ODataAfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error)
func (Product) ODataBeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)
func (Product) ODataAfterReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, error)
```

Each hook receives the active HTTP request, context, and parsed OData query options. Returning an error aborts the request and surfaces the error to the client.
Return `(nil, nil)` from an After hook to keep the original response body.

### Tenant Filtering Example

Apply multi-tenant filters centrally by returning scopes from `ODataBeforeReadCollection` and `ODataBeforeReadEntity` hooks:

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

func (Product) ODataBeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
    tenantID := r.Header.Get("X-Tenant-ID")
    if tenantID == "" {
        return nil, fmt.Errorf("missing tenant header")
    }
    return []func(*gorm.DB) *gorm.DB{Product{}.tenantScope(tenantID)}, nil
}

func (Product) ODataBeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
    tenantID := r.Header.Get("X-Tenant-ID")
    if tenantID == "" {
        return nil, fmt.Errorf("missing tenant header")
    }
    return []func(*gorm.DB) *gorm.DB{Product{}.tenantScope(tenantID)}, nil
}
```

By returning scopes instead of mutating the request, the same tenant filter is applied consistently across `$count`, pagination, `$expand`, and navigation reads.

### Redacting Sensitive Data

Use `ODataAfterReadEntity` or `ODataAfterReadCollection` to redact fields just before they leave the service:

```go
func (Product) ODataAfterReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, error) {
    product, ok := entity.(*Product)
    if !ok {
        return entity, nil
    }

    if !isPrivileged(r) {
        product.CostPrice = 0
    }
    return product, nil
}

func (Product) ODataAfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error) {
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

## Server-side Key Generation

Use server-side key generation when you need identifiers that are independent of the database’s auto-increment behaviour. go-odata exposes a registry of key generators that you can populate at service startup.

```go
service := odata.NewService(db)

service.RegisterKeyGenerator("timestamp", func(ctx context.Context) (interface{}, error) {
        return time.Now().UnixNano(), nil
})

type APIKey struct {
        KeyID string `json:"KeyID" gorm:"type:char(36);primaryKey" odata:"key,generate=uuid"`
        Owner string `json:"Owner" odata:"required"`
}
```

> Requires `import "context"` and `import "time"`.

Metadata analysis validates that `generate=uuid` (or any other name you choose) has a registered generator. During POST requests the handler zeroes auto-increment numeric keys and calls the requested generator for every key property that declares one. That behaviour holds for single entities, collections, and composite keys.

The development and performance sample servers ship with an `APIKeys` entity that uses `generate=uuid`. Run `go run ./cmd/devserver` and POST to `/APIKeys` without supplying a `KeyID` to see the feature in action.

## Asynchronous Processing

`go-odata` can run long-running requests asynchronously when clients send `Prefer: respond-async`. Enable it with `Service.EnableAsyncProcessing` and provide a monitor prefix (defaults to `/$async/jobs/`). The helper returns an error because the async manager now persists job state using GORM. The library writes to a reserved `_odata_async_jobs` table so application models remain untouched and finished jobs can be monitored even after a manager restart.

```go
if err := service.EnableAsyncProcessing(odata.AsyncConfig{
        MonitorPathPrefix:    "/$async/jobs/",
        DefaultRetryInterval: 5 * time.Second,
        JobRetention:         30 * time.Minute, // Overrides the default 24h retention window
}); err != nil {
        log.Fatalf("enable async processing: %v", err)
}

defer service.Close()
```

When `JobRetention` is left at zero the manager applies the 24-hour
`async.DefaultJobRetention` window automatically and prunes expired records in
the background. Environments that must keep audit data indefinitely can set
`DisableRetention: true` and perform their own archival or cleanup against the
`_odata_async_jobs` table.

Once enabled, the router reserves the monitor path and exposes the following behaviour:

- **Status monitors** live at `/{prefix}{jobID}` (for example, `/$async/jobs/6f92b3...`). Job IDs are limited to ASCII alphanumeric characters plus `_` and `-`. Requests that omit the job ID, add extra path segments, or include disallowed characters return `404 Not Found` or `400 Bad Request` before the async manager is invoked.
- **Polling** a pending job with `GET` or `HEAD` returns `202 Accepted` with `Preference-Applied: respond-async` and, when configured, a `Retry-After` header. The 202 response never includes a body.
- **Completion** replays the stored response exactly: the original status code, headers (including any `Preference-Applied` values set by the handler), and body are forwarded once the job finishes. Finished jobs remain queryable until the configured retention TTL (or the default window) deletes their rows from `_odata_async_jobs`.
- **Cancellation** is available via `DELETE` on the monitor URI. The router calls the async manager’s cancellation hook and returns `204 No Content`, even if the worker finishes later.

These guarantees let clients reliably poll job status without conflicting with regular entity routing.

`Service.Close` shuts down the async manager’s background goroutines and resets
the related configuration. You can call it from multiple cleanup hooks—each call
is a no-op once the manager is already stopped.

## Full-Text Search with Database FTS

The go-odata library automatically enhances `$search` query performance by utilizing database-native Full-Text Search (FTS) capabilities when available. This provides significant performance improvements for text search operations while maintaining backward compatibility.

### Supported Databases

- **SQLite**: FTS3, FTS4, and FTS5 virtual tables
- **PostgreSQL**: Built-in full-text search with `tsvector`, `tsquery`, and GIN indexes

### Overview

When you use the `$search` query parameter, the library:
1. Automatically detects if database FTS is available (SQLite FTS3/4/5 or PostgreSQL)
2. If available, creates FTS tables/indexes and applies search at the database level
3. If not available, falls back to the existing in-memory search implementation
4. Keeps FTS tables synchronized with your data using triggers

### How It Works

The FTS integration is completely automatic - no configuration required:

```go
// SQLite setup - FTS is automatically initialized
db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
service := odata.NewService(db)
service.RegisterEntity(&Product{})

// PostgreSQL setup - FTS is automatically initialized
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
service := odata.NewService(db)
service.RegisterEntity(&Product{})

// Clients can use $search as usual
// GET /Products?$search=laptop
// If FTS is available, search runs at database level
// Otherwise, it falls back to in-memory search
```

### Searchable Properties

Mark properties as searchable using the `searchable` tag:

```go
type Product struct {
    ID          int     `json:"ID" gorm:"primaryKey" odata:"key"`
    Name        string  `json:"Name" odata:"searchable"`
    Description string  `json:"Description" odata:"searchable"`
    Category    string  `json:"Category"`  // Not searchable
    Price       float64 `json:"Price"`
}
```

If no properties are marked as searchable, all string properties are automatically included in the search index.

### FTS Features

**Supported:**
- ✅ Full-text search with SQLite FTS3, FTS4, or FTS5
- ✅ Full-text search with PostgreSQL (tsvector/tsquery)
- ✅ Automatic FTS detection and initialization
- ✅ Automatic table/index synchronization via triggers
- ✅ Case-insensitive search
- ✅ Transparent fallback to in-memory search
- ✅ Configurable searchable properties
- ✅ Support for fuzziness and similarity settings

**Search Behavior:**
```bash
# Search for products containing "laptop"
GET /Products?$search=laptop

# Search works across all searchable fields
GET /Products?$search=gaming

# Case-insensitive by default
GET /Products?$search=LAPTOP  # Same as "laptop"
```

### Performance Benefits

FTS provides significant performance improvements for text search:

| Operation | In-Memory Search | FTS Search |
|-----------|-----------------|------------|
| Small datasets (<100 rows) | Fast | Fast |
| Medium datasets (100-10K rows) | Moderate | Fast |
| Large datasets (>10K rows) | Slow | Fast |

**Key Benefits:**
- Database-level filtering reduces memory usage
- Better performance on large datasets
- Utilizes database-optimized FTS indexes
- Reduces data transfer between database and application

### Database-Specific Implementation

#### SQLite FTS

The library automatically creates and manages FTS virtual tables:

```sql
-- Example: For Products entity, creates:
CREATE VIRTUAL TABLE products_fts USING fts4(
    id, name, description
);

-- With automatic triggers to keep data synchronized
CREATE TRIGGER products_fts_ai AFTER INSERT ON products BEGIN
    INSERT INTO products_fts(id, name, description) 
    VALUES (NEW.id, NEW.name, NEW.description);
END;
```

#### PostgreSQL FTS

For PostgreSQL, the library creates FTS tables with `tsvector` columns and GIN indexes:

```sql
-- Example: For Products entity, creates:
CREATE TABLE products_fts (
    id INTEGER PRIMARY KEY,
    search_vector tsvector
);

-- Create GIN index for fast full-text search
CREATE INDEX products_fts_search_idx 
ON products_fts USING GIN(search_vector);

-- Trigger function to maintain FTS table
CREATE OR REPLACE FUNCTION products_fts_sync() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO products_fts (id, search_vector)
        VALUES (NEW.id, 
            to_tsvector('english', coalesce(NEW.name, '')) ||
            to_tsvector('english', coalesce(NEW.description, ''))
        );
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        UPDATE products_fts 
        SET search_vector = 
            to_tsvector('english', coalesce(NEW.name, '')) ||
            to_tsvector('english', coalesce(NEW.description, ''))
        WHERE id = NEW.id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        DELETE FROM products_fts WHERE id = OLD.id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
```

**PostgreSQL FTS Features:**
- Uses `to_tsvector()` for text normalization and stemming
- Uses `plainto_tsquery()` for simple query parsing
- English language configuration by default
- Handles NULL values gracefully with `coalesce()`
- GIN indexes provide fast search performance

### Fallback Behavior

If FTS is not available (e.g., unsupported database or FTS not available):
- Search automatically falls back to in-memory implementation
- All existing fuzzy matching and similarity features work
- No changes needed in client code
- No errors or warnings

```go
// Works with any database backend
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
service := odata.NewService(db)
// $search falls back to in-memory search automatically
```

### Advanced Search Options

You can still use advanced search options with or without FTS:

```go
type Product struct {
    ID    int    `json:"ID" odata:"key"`
    Name  string `json:"Name" odata:"searchable,fuzziness=2"`
    Email string `json:"Email" odata:"searchable,similarity=0.9"`
}
```

When FTS is available:
- Basic search uses FTS for optimal performance
- Fuzzy matching and similarity scoring work at the application level

When FTS is not available:
- All features work via in-memory search
- Full fuzzy matching and similarity support

### FTS Versions

The library supports all SQLite FTS versions:

| Version | Features | Priority |
|---------|----------|----------|
| FTS5 | Latest, best performance | 1st choice |
| FTS4 | Good performance, widely available | 2nd choice |
| FTS3 | Basic FTS support | 3rd choice |

The library automatically detects and uses the best available version.

### Limitations

1. **Database Support**: Automatic FTS integration is available for SQLite (FTS3/4/5) and PostgreSQL only. All other database backends automatically fall back to the in-memory `$search` implementation.

2. **PostgreSQL Language Configuration**: PostgreSQL uses the built-in `english` text search configuration (`to_tsvector('english', ...)` and `plainto_tsquery('english', ...)`). If you need a different language or custom dictionary, you must modify the FTS implementation to change the configuration.

3. **PostgreSQL Trigger Requirements**: PostgreSQL FTS relies on helper tables, a GIN index, a `plpgsql` trigger function, and triggers to keep the search vectors in sync. The database user must have privileges to create tables, indexes, functions, and triggers; locked-down roles will prevent FTS initialization.

4. **SQLite Build Requirements**: SQLite FTS requires FTS3/4/5 support to be compiled into the SQLite build (some minimal builds disable it). If none of the FTS modules are available, the library falls back to in-memory search.

5. **Simple Queries**: FTS is optimized for simple full-text queries. Complex boolean expressions or advanced fuzzy matching may fall back to in-memory processing.

6. **Storage/Write Overhead**: FTS tables require additional disk space (approximately 30-50% of indexed text), and triggers add some overhead to INSERT/UPDATE/DELETE operations.

### Best Practices

1. **Mark Searchable Fields**: Explicitly mark fields as `searchable` to control which fields are indexed.

2. **Index Size**: Be mindful of text field sizes when marking as searchable - large text fields increase index size.

3. **Test Both Paths**: Test your search functionality with both FTS enabled and disabled to ensure consistent behavior.

4. **Monitor Performance**: Use SQLite's `EXPLAIN QUERY PLAN` to verify FTS usage in queries.

### Example: Complete Implementation

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/nlstn/go-odata"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type Article struct {
    ID      int    `json:"ID" gorm:"primaryKey" odata:"key"`
    Title   string `json:"Title" odata:"searchable"`
    Content string `json:"Content" odata:"searchable"`
    Author  string `json:"Author"`
    Tags    string `json:"Tags" odata:"searchable"`
}

func main() {
    db, _ := gorm.Open(sqlite.Open("articles.db"), &gorm.Config{})
    db.AutoMigrate(&Article{})
    
    service := odata.NewService(db)
    service.RegisterEntity(&Article{})
    
    http.Handle("/", service)
    log.Println("Server with FTS-enabled search on :8080")
    http.ListenAndServe(":8080", nil)
}
```

Clients can now use efficient full-text search:
```bash
# Search across title, content, and tags
GET /Articles?$search=golang

# Combine with other query options
GET /Articles?$search=database&$top=10&$orderby=Title
```
