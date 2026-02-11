# Authorization

This guide explains how to implement authorization in go-odata, including role-based access control (RBAC), row-level security, and per-entity authorization patterns.

## Table of Contents

- [Overview](#overview)
- [Setting Up Authorization](#setting-up-authorization)
- [Implementing a Policy](#implementing-a-policy)
- [Populating Auth Context](#populating-auth-context)
- [Authorization Patterns](#authorization-patterns)
  - [Simple Role-Based Authorization](#simple-role-based-authorization)
  - [Row-Level Security with Query Filters](#row-level-security-with-query-filters)
  - [Per-Entity Authorization](#per-entity-authorization)
- [Operation Types](#operation-types)
- [Best Practices](#best-practices)

## Overview

go-odata provides a flexible authorization framework that allows you to control access to your OData resources. The framework is based on three main components:

1. **Policy Interface**: Implement the `Policy` interface to make authorization decisions
2. **Auth Context**: Contains authentication information (principal, roles, claims, scopes) and request metadata
3. **PreRequestHook**: Populates the auth context from incoming requests

The authorization framework automatically checks permissions for all operations including:
- CRUD operations (Create, Read, Update, Delete)
- Query operations (List, Filter, Sort)
- Metadata access
- Actions and Functions
- Navigation properties

## Setting Up Authorization

### Step 1: Create a Policy

Implement the `Policy` interface to make authorization decisions:

```go
type MyPolicy struct {
    // Add any dependencies like database access
    db *gorm.DB
}

func (p *MyPolicy) Authorize(
    ctx odata.AuthContext,
    resource odata.ResourceDescriptor,
    operation odata.Operation,
) odata.Decision {
    // Make authorization decision based on context and operation
    return odata.Allow()
}
```

### Step 2: Populate Auth Context

Use `PreRequestHook` to extract authentication information and store it in the request context using standard keys:

```go
service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
    // Extract token from header
    token := r.Header.Get("Authorization")
    
    // Validate token and extract claims (your JWT validation logic)
    claims, err := validateJWT(token)
    if err != nil {
        return nil, err
    }
    
    // Store auth data using standard context keys
    ctx := r.Context()
    ctx = context.WithValue(ctx, odata.PrincipalContextKey, claims.UserID)
    ctx = context.WithValue(ctx, odata.RolesContextKey, claims.Roles)
    ctx = context.WithValue(ctx, odata.ClaimsContextKey, claims.Data)
    ctx = context.WithValue(ctx, odata.ScopesContextKey, claims.Scopes)
    
    return ctx, nil
})
```

The framework automatically extracts these values from the context and includes them in the `AuthContext` passed to your policy.

### Step 3: Register the Policy

```go
policy := &MyPolicy{db: db}
if err := service.SetPolicy(policy); err != nil {
    log.Fatal(err)
}
```

## Implementing a Policy

### AuthContext Structure

The `AuthContext` contains:

```go
type AuthContext struct {
    Principal interface{}            // User identifier (e.g., user ID, email)
    Roles     []string                // User's roles (e.g., ["admin", "user"])
    Claims    map[string]interface{}  // Additional JWT claims
    Scopes    []string                // OAuth scopes
    Request   RequestMetadata         // HTTP request details
}
```

### ResourceDescriptor Structure

The `ResourceDescriptor` describes the resource being accessed:

```go
type ResourceDescriptor struct {
    EntitySetName string                  // e.g., "Products"
    EntityType    string                  // e.g., "Product"
    KeyValues     map[string]interface{}  // Entity key values
    PropertyPath  []string                // Property being accessed
    Entity        interface{}             // The actual entity (for updates/deletes)
}
```

### Decision Types

Return one of two decision types:

```go
// Allow access
return odata.Allow()

// Deny access with optional reason
return odata.Deny("User does not have required role")
```

## Authorization Patterns

### Simple Role-Based Authorization

Check user roles to make authorization decisions:

```go
type RoleBasedPolicy struct{}

func (p *RoleBasedPolicy) Authorize(
    ctx odata.AuthContext,
    resource odata.ResourceDescriptor,
    op odata.Operation,
) odata.Decision {
    // Allow metadata operations
    if op == odata.OperationMetadata {
        return odata.Allow()
    }
    
    // Check if user has required role
    for _, role := range ctx.Roles {
        if role == "admin" {
            return odata.Allow() // Admins can do anything
        }
    }
    
    // Regular users can only read
    if op == odata.OperationRead || op == odata.OperationQuery {
        return odata.Allow()
    }
    
    return odata.Deny("Operation not permitted")
}
```

### Row-Level Security with Query Filters

Use `QueryFilterProvider` to automatically filter query results based on user permissions:

```go
type TenantPolicy struct{}

func (p *TenantPolicy) Authorize(
    ctx odata.AuthContext,
    resource odata.ResourceDescriptor,
    op odata.Operation,
) odata.Decision {
    // Basic authorization
    if ctx.Principal == nil {
        return odata.Deny("Authentication required")
    }
    return odata.Allow()
}

// QueryFilter adds a filter to restrict results to the user's tenant
func (p *TenantPolicy) QueryFilter(
    ctx odata.AuthContext,
    resource odata.ResourceDescriptor,
    op odata.Operation,
) (*odata.FilterExpression, error) {
    // Extract tenant ID from claims
    tenantID, ok := ctx.Claims["tenant_id"]
    if !ok {
        return nil, nil
    }
    
    // Return filter: tenantId eq '{tenant_id}'
    return &odata.FilterExpression{
        Property: "tenantId",
        Operator: "eq",
        Value:    tenantID,
    }, nil
}
```

The query filter is automatically applied to all collection queries and combined with user-specified filters using AND logic.

### Per-Entity Authorization

For scenarios where users have different roles per entity (e.g., club ownership, project membership):

```go
type PerEntityPolicy struct {
    db *gorm.DB
}

// ClubMembership represents user roles per club
type ClubMembership struct {
    UserID string
    ClubID int
    Role   string // "owner", "admin", "member", "viewer"
}

func (p *PerEntityPolicy) Authorize(
    ctx odata.AuthContext,
    resource odata.ResourceDescriptor,
    op odata.Operation,
) odata.Decision {
    // Allow metadata
    if op == odata.OperationMetadata {
        return odata.Allow()
    }
    
    userID, ok := ctx.Principal.(string)
    if !ok {
        return odata.Deny("Authentication required")
    }
    
    // Check for global admin role
    for _, role := range ctx.Roles {
        if role == "global-admin" {
            return odata.Allow()
        }
    }
    
    // For club operations, check per-club role
    if resource.EntitySetName == "Clubs" && len(resource.KeyValues) > 0 {
        clubID := resource.KeyValues["id"].(int)
        role := p.getUserRoleForClub(userID, clubID)
        
        switch op {
        case odata.OperationRead:
            // Any member can read
            return p.allowIfRole(role, "viewer", "member", "admin", "owner")
        case odata.OperationUpdate:
            // Only admins and owners can update
            return p.allowIfRole(role, "admin", "owner")
        case odata.OperationDelete:
            // Only owners can delete
            return p.allowIfRole(role, "owner")
        }
    }
    
    return odata.Deny("Access denied")
}

func (p *PerEntityPolicy) getUserRoleForClub(userID string, clubID int) string {
    var membership ClubMembership
    p.db.Where("user_id = ? AND club_id = ?", userID, clubID).First(&membership)
    return membership.Role
}

func (p *PerEntityPolicy) allowIfRole(userRole string, allowedRoles ...string) odata.Decision {
    for _, allowed := range allowedRoles {
        if userRole == allowed {
            return odata.Allow()
        }
    }
    return odata.Deny("Insufficient permissions")
}

// QueryFilter restricts clubs to those where user has membership
func (p *PerEntityPolicy) QueryFilter(
    ctx odata.AuthContext,
    resource odata.ResourceDescriptor,
    op odata.Operation,
) (*odata.FilterExpression, error) {
    if resource.EntitySetName != "Clubs" {
        return nil, nil
    }
    
    userID, _ := ctx.Principal.(string)
    
    // Get user's club memberships
    var memberships []ClubMembership
    p.db.Where("user_id = ?", userID).Find(&memberships)
    
    if len(memberships) == 0 {
        // No memberships - return filter that matches nothing
        return &odata.FilterExpression{
            Property: "id",
            Operator: "eq",
            Value:    -1,
        }, nil
    }
    
    // Build OR filter for all accessible clubs
    var filters []*odata.FilterExpression
    for _, m := range memberships {
        filters = append(filters, &odata.FilterExpression{
            Property: "id",
            Operator: "eq",
            Value:    m.ClubID,
        })
    }
    
    // Combine with OR
    result := filters[0]
    for i := 1; i < len(filters); i++ {
        result = &odata.FilterExpression{
            Logical: "or",
            Left:    result,
            Right:   filters[i],
        }
    }
    
    return result, nil
}
```

## Operation Types

Your policy receives one of these operation types:

| Operation | Description | Example |
|-----------|-------------|---------|
| `OperationRead` | Reading a single entity | `GET /Products(1)` |
| `OperationQuery` | Querying a collection | `GET /Products` |
| `OperationCreate` | Creating a new entity | `POST /Products` |
| `OperationUpdate` | Updating an entity | `PATCH /Products(1)` |
| `OperationDelete` | Deleting an entity | `DELETE /Products(1)` |
| `OperationMetadata` | Accessing metadata | `GET /$metadata` |
| `OperationAction` | Executing an action | `POST /Products(1)/Discontinue` |
| `OperationFunction` | Executing a function | `GET /Products(1)/GetPrice()` |

## Standard Context Keys

Use these keys in `PreRequestHook` to store auth data that will be automatically extracted:

```go
// Store principal/user identifier
ctx = context.WithValue(ctx, odata.PrincipalContextKey, userID)

// Store user roles
ctx = context.WithValue(ctx, odata.RolesContextKey, []string{"admin", "user"})

// Store additional claims
ctx = context.WithValue(ctx, odata.ClaimsContextKey, map[string]interface{}{
    "tenant_id": "123",
    "email": "user@example.com",
})

// Store OAuth scopes
ctx = context.WithValue(ctx, odata.ScopesContextKey, []string{"read", "write"})
```

## Best Practices

### 1. Always Allow Metadata Operations

Metadata is generally not sensitive and clients need it to discover the API:

```go
if op == odata.OperationMetadata {
    return odata.Allow()
}
```

### 2. Return 401 vs 403 Correctly

The framework automatically returns:
- `401 Unauthorized` when no `Authorization` header is present
- `403 Forbidden` when authentication is present but authorization fails

### 3. Use Query Filters for Performance

For row-level security, implement `QueryFilterProvider` instead of checking each entity individually. This pushes filtering to the database level:

```go
type MyPolicy struct{}

func (p *MyPolicy) QueryFilter(...) (*odata.FilterExpression, error) {
    // Return a filter expression that restricts visible rows
}
```

### 4. Keep Policy Lightweight

For per-entity authorization with database access:
- Cache frequently accessed data
- Use database indexes on foreign keys
- Consider denormalizing role information for better performance

### 5. Validate at Multiple Levels

Consider implementing authorization at multiple levels:
- **Policy**: For request-level authorization
- **Query Filters**: For row-level filtering
- **After Read Hooks**: For field-level redaction

Example field-level redaction:

```go
func (u User) ODataAfterReadEntity(
    ctx context.Context,
    r *http.Request,
    opts *odata.QueryOptions,
    entity interface{},
) (interface{}, error) {
    user := entity.(*User)
    
    // Extract auth data from context
    principal, ok := ctx.Value(odata.PrincipalContextKey).(string)
    if !ok {
        // Redact sensitive fields
        user.SSN = "REDACTED"
        user.Salary = 0
    }
    
    return user, nil
}
```

### 6. Test Authorization Thoroughly

Test all operation types and edge cases:
- Anonymous access
- Missing credentials
- Invalid credentials
- Different user roles
- Resource ownership
- Cross-tenant access attempts

### 7. Log Authorization Failures

Consider logging authorization failures for security monitoring:

```go
func (p *MyPolicy) Authorize(...) odata.Decision {
    decision := p.makeDecision(...)
    if !decision.Allowed {
        log.Warnf("Authorization denied: user=%v, resource=%v, operation=%v, reason=%v",
            ctx.Principal, resource.EntitySetName, operation, decision.Reason)
    }
    return decision
}
```

## Complete Example

See [documentation/examples/auth_examples.go](examples/auth_examples.go) for complete, runnable examples including:

1. Simple role-based authorization
2. Tenant-based authorization with row-level security
3. Resource-level authorization
4. Scope-based authorization
5. Per-entity authorization (per-club roles)
6. Populating AuthContext from JWT tokens
7. Field-level redaction

## Related Documentation

- [Pre-Request Hook](advanced-features.md#pre-request-hook) - Setting up request preprocessing
- [Lifecycle Hooks](advanced-features.md#lifecycle-hooks) - Entity-level hooks for additional authorization
- [Actions and Functions](actions-and-functions.md) - Authorization for custom operations
- [Testing](testing.md) - Testing authorization policies
