# Advanced Features

This guide covers advanced features of go-odata including singletons, ETags, and lifecycle hooks.

## Table of Contents

- [Singletons](#singletons)
- [ETags (Optimistic Concurrency Control)](#etags-optimistic-concurrency-control)
- [Lifecycle Hooks](#lifecycle-hooks)

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

### Available Hooks

The library supports the following hooks:

- `BeforeCreate` - Called before creating a new entity
- `AfterCreate` - Called after creating a new entity
- `BeforeUpdate` - Called before updating an entity
- `AfterUpdate` - Called after updating an entity
- `BeforeDelete` - Called before deleting an entity
- `AfterDelete` - Called after deleting an entity

### Implementing Hooks

Define hooks by implementing the corresponding interface:

```go
type Product struct {
    ID        uint      `json:"ID" gorm:"primaryKey" odata:"key"`
    Name      string    `json:"Name" odata:"required"`
    Price     float64   `json:"Price"`
    CreatedAt time.Time `json:"CreatedAt"`
    UpdatedAt time.Time `json:"UpdatedAt"`
}

// BeforeCreate hook
func (p *Product) BeforeCreate() error {
    p.CreatedAt = time.Now()
    if p.Price < 0 {
        return fmt.Errorf("price cannot be negative")
    }
    return nil
}

// BeforeUpdate hook
func (p *Product) BeforeUpdate() error {
    p.UpdatedAt = time.Now()
    if p.Price < 0 {
        return fmt.Errorf("price cannot be negative")
    }
    return nil
}

// AfterCreate hook
func (p *Product) AfterCreate() error {
    log.Printf("Product created: %s (ID: %d)", p.Name, p.ID)
    return nil
}
```

### Hook Use Cases

**Validation:**
```go
func (p *Product) BeforeCreate() error {
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
func (p *Product) BeforeCreate() error {
    p.CreatedAt = time.Now()
    p.UpdatedAt = time.Now()
    return nil
}

func (p *Product) BeforeUpdate() error {
    p.UpdatedAt = time.Now()
    return nil
}
```

**Audit Logging:**
```go
func (p *Product) AfterUpdate() error {
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
func (o *Order) BeforeCreate() error {
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
func (p *Product) BeforeCreate() error {
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
