# Server Configuration

This guide covers how to configure and integrate the go-odata service into your application.

## Table of Contents

- [Basic Setup](#basic-setup)
- [Customizing the Metadata Namespace](#customizing-the-metadata-namespace)
- [Default Max Top Configuration](#default-max-top-configuration)
- [Service as Handler](#service-as-handler)
- [Custom Path Mounting](#custom-path-mounting)
- [Adding Middleware](#adding-middleware)
- [Popular Router Integrations](#popular-router-integrations)
- [Multiple Handlers](#multiple-handlers)
- [Development Servers](#development-servers)

## Basic Setup

The minimal setup to get an OData service running:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/nlstn/go-odata"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type Product struct {
    ID          uint    `json:"ID" gorm:"primaryKey" odata:"key"`
    Name        string  `json:"Name" gorm:"not null" odata:"required"`
    Price       float64 `json:"Price" gorm:"not null"`
}

func main() {
    // Initialize database
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }
    
    // Auto-migrate
    db.AutoMigrate(&Product{})
    
    // Initialize OData service
    service, err := odata.NewService(db)
    if err != nil {
        log.Fatal(err)
    }
    
    // Register entity
    service.RegisterEntity(&Product{})
    
    // Create HTTP mux and register the OData service
    mux := http.NewServeMux()
    mux.Handle("/", service)
    
    // Start server
    log.Println("Starting OData service on :8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

## Customizing the Metadata Namespace

By default the service uses the `ODataService` namespace when generating metadata documents and `@odata.type` annotations. You can call `SetNamespace` before registering any entities to align the namespace with client expectations. As implemented in `odata.go`, the metadata handler and entity handlers reuse the namespace you configure, so a single call keeps every response consistent.

```go
// Initialize OData service
service, err := odata.NewService(db)
if err != nil {
    log.Fatal(err)
}

// Set a custom namespace before registering entities
if err := service.SetNamespace("Contoso.Sales"); err != nil {
    log.Fatal(err)
}

// Register entities after the namespace is set
service.RegisterEntity(&Product{})
```

## Default Max Top Configuration

You can configure default limits on the number of results returned when no explicit `$top` is provided by the client. This is useful to prevent clients from requesting large result sets that could impact performance.

### Service-Level Default

Set a default maximum number of results for all entity sets:

```go
service, err := odata.NewService(db)
if err != nil {
    log.Fatal(err)
}
service.RegisterEntity(&Product{})
service.RegisterEntity(&Order{})

// Set service-level default: all entity sets limited to 100 results by default
service.SetDefaultMaxTop(100)
```

### Entity-Level Default

Override the service-level default for specific entity sets:

```go
// Set service-level default
service.SetDefaultMaxTop(100)

// Override for specific entity sets
service.SetEntityDefaultMaxTop("Products", 50)  // Products limited to 50
service.SetEntityDefaultMaxTop("Orders", 200)   // Orders limited to 200
```

### Priority Order

The library applies limits in the following priority order (highest to lowest):

1. **Explicit `$top` in the request** - Always takes precedence
2. **`Prefer: odata.maxpagesize` header** - Client preference
3. **Entity-level default** - Set via `SetEntityDefaultMaxTop()`
4. **Service-level default** - Set via `SetDefaultMaxTop()`
5. **No limit** - If none of the above are set

### Removing Defaults

Pass `0` or a negative value to remove a default limit:

```go
// Remove service-level default
service.SetDefaultMaxTop(0)

// Remove entity-level default (falls back to service-level)
service.SetEntityDefaultMaxTop("Products", 0)
```

### Example

```go
service, err := odata.NewService(db)
if err != nil {
    log.Fatal(err)
}
service.RegisterEntity(&Product{})
service.RegisterEntity(&Order{})

// Set service-level default
service.SetDefaultMaxTop(100)

// Override for Products
service.SetEntityDefaultMaxTop("Products", 25)

// Request: GET /Products
// Returns: 25 results (entity-level default)

// Request: GET /Orders
// Returns: 100 results (service-level default)

// Request: GET /Products?$top=10
// Returns: 10 results (explicit $top overrides defaults)

// Request: GET /Products with Prefer: odata.maxpagesize=5
// Returns: 5 results (maxpagesize preference overrides defaults)
```

## Service as Handler

The `Service` implements `http.Handler`, so you can use it in multiple ways:

```go
mux := http.NewServeMux()

// Option 1: Use the Handler() method
mux.Handle("/", service.Handler())

// Option 2: Use the service directly (equivalent)
mux.Handle("/", service)

// Both are equivalent and provide the same functionality
```

## Custom Path Mounting

Mount the OData service at a custom path prefix:

```go
// Mount OData service at /api/odata/
mux.Handle("/api/odata/", http.StripPrefix("/api/odata", service))

// Now access entities via:
// http://localhost:8080/api/odata/Products
// http://localhost:8080/api/odata/$metadata
```

**Important:** When using a custom path:
- Use `http.StripPrefix` to remove the prefix before the service handles the request
- The trailing slash in the pattern is important: `/api/odata/`
- All OData URLs will be relative to this path

## Adding Middleware

Add custom middleware for authentication, logging, CORS, etc.

### Authentication Middleware

```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check for API key
        apiKey := r.Header.Get("X-API-Key")
        if apiKey == "" {
            http.Error(w, "Missing API key", http.StatusUnauthorized)
            return
        }
        
        // Validate API key
        if !isValidAPIKey(apiKey) {
            http.Error(w, "Invalid API key", http.StatusUnauthorized)
            return
        }
        
        // Call next handler
        next.ServeHTTP(w, r)
    })
}

// Apply middleware
mux.Handle("/", authMiddleware(service))
```

### Logging Middleware

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Create response writer wrapper to capture status code
        wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
        
        // Call next handler
        next.ServeHTTP(wrapper, r)
        
        // Log request details
        log.Printf(
            "%s %s %d %v",
            r.Method,
            r.URL.Path,
            wrapper.statusCode,
            time.Since(start),
        )
    })
}

type responseWriterWrapper struct {
    http.ResponseWriter
    statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
    w.statusCode = statusCode
    w.ResponseWriter.WriteHeader(statusCode)
}

// Apply middleware
mux.Handle("/", loggingMiddleware(service))
```

### CORS Middleware

```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Set CORS headers
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, OData-Version, OData-MaxVersion, If-Match, Prefer")
        
        // Handle preflight requests
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        
        // Call next handler
        next.ServeHTTP(w, r)
    })
}

// Apply middleware
mux.Handle("/", corsMiddleware(service))
```

### Chaining Multiple Middleware

```go
func chain(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
    for i := len(middleware) - 1; i >= 0; i-- {
        handler = middleware[i](handler)
    }
    return handler
}

// Apply multiple middleware in order
mux.Handle("/", chain(
    service,
    loggingMiddleware,
    corsMiddleware,
    authMiddleware,
))
```

## Popular Router Integrations

Use go-odata with popular routing frameworks by adapting the `Service` (which implements `http.Handler`) to the router's handler signature.

### Chi (`chi.Router`)

```go
import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/nlstn/go-odata"
)

func configureChiRouter(service *odata.Service) http.Handler {
    r := chi.NewRouter()

    // Register shared middleware first; it will run before the OData handler.
    r.Use(loggingMiddleware, authMiddleware)

    // Mount at a sub-route so Chi preserves the request context chain.
    r.Mount("/odata", service)

    return r
}
```

**Middleware ordering:** Chi executes router-level middleware in the order you call `Use` before delegating to mounted handlers, so register request-scoped middleware (tracing, auth) before `Mount`. Handler-specific middleware can be applied with `chi.Chain` if you need logic that runs only for the OData routes. Because Chi relies on `context.Context`, any values stored in the request context are preserved automatically when the service handles the request.

### Gin (`gin.Engine`)

```go
import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/nlstn/go-odata"
)

func configureGinEngine(service *odata.Service) *gin.Engine {
    router := gin.New()

    // Register global middleware before wrapping the OData handler.
    router.Use(gin.Logger(), gin.Recovery())

    // gin.WrapH adapts http.Handler to gin.HandlerFunc.
    router.Any("/odata/*path", gin.WrapH(http.StripPrefix("/odata", service)))

    return router
}
```

**Middleware ordering:** Gin executes handlers and middleware in the order registered. Call `router.Use(...)` before `router.Any` so global middleware runs first. Because `gin.WrapH` passes the underlying `*http.Request` through to go-odata, request-scoped context values and cancellations remain available within the service.

### Echo (`echo.Echo`)

```go
import (
    "net/http"

    "github.com/labstack/echo/v4"
    "github.com/nlstn/go-odata"
)

func configureEchoServer(service *odata.Service) *echo.Echo {
    e := echo.New()

    // Global middleware should be added before wrapping the handler.
    e.Use(requestIDMiddleware, loggingMiddleware)

    // echo.WrapHandler converts http.Handler to echo.HandlerFunc.
    e.Any("/odata/*", echo.WrapHandler(http.StripPrefix("/odata", service)))

    return e
}
```

**Middleware ordering:** Echo middleware wraps handlers in the order they are registered; add `e.Use` calls before `e.Any` so they execute before go-odata. Echo stores route-scoped data on `echo.Context`, but the wrapped handler still receives the original `*http.Request`, so ensure any context values you expect are attached to `c.Request().Context()` before invoking go-odata. For per-route middleware, wrap the handler with `e.Group("/odata")` and call `group.Use(...)` before registering the route.

## Multiple Handlers

Combine the OData service with other HTTP handlers:

```go
mux := http.NewServeMux()

// OData service at root
mux.Handle("/", service)

// Health check endpoint
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})

// Metrics endpoint
mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
    // Your metrics logic
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "requests": requestCount,
        "errors": errorCount,
    })
})

// Custom API endpoints
mux.HandleFunc("/api/custom", func(w http.ResponseWriter, r *http.Request) {
    // Custom logic
})

log.Fatal(http.ListenAndServe(":8080", mux))
```

**Note:** Since the OData service is mounted at `/`, it will handle all requests that don't match more specific patterns. Register specific handlers before the catch-all OData handler.

## Development Servers

The repository includes two example servers:

### Development Server (`cmd/devserver`)

A full-featured development server with demo functionality:

```bash
cd cmd/devserver
go run .
```

Features:
- Sample Product, Category, and User data
- Custom authentication middleware (demo only)
- Example actions and functions
- Lifecycle hooks (ODataBeforeCreate, ODataBeforeUpdate)
- Runs on `http://localhost:8080`

**Use for:** Testing features, exploring the library, development

### Compliance Server (`cmd/complianceserver`)

A minimal server for OData compliance testing:

```bash
cd cmd/complianceserver
go run .
```

Features:
- Minimal entities (Products, Categories, ProductDescriptions, Company singleton)
- No custom middleware or lifecycle hooks
- Standard actions and functions for compliance testing
- SQL query tracing support
- Runs on `http://localhost:9090`

**Use for:** Running compliance tests, validating OData v4 specification adherence

**Note:** The Go-based compliance test suite automatically starts and stops the compliance server.

See [cmd/complianceserver/README.md](../cmd/complianceserver/README.md) for more details.

## Database Configuration

The go-odata library works with GORM-compatible databases. Below are configuration examples for common databases.

### Supported Databases

- **SQLite** - Fully supported and tested in CI. Includes native FTS (FTS3/4/5) for `$search`.
- **PostgreSQL** - Fully supported and tested in CI (PostgreSQL 17). Includes native full-text search with `tsvector` and GIN indexes for `$search`.
- **MariaDB** - Fully supported and tested in CI (MariaDB 11). `$search` falls back to in-memory filtering.
- **MySQL** - Fully supported and tested in CI (MySQL 8). `$search` falls back to in-memory filtering.
- **Other databases** - Support is in progress and not covered by CI. `$search` falls back to in-memory filtering. [Open an issue](https://github.com/NLstn/go-odata/issues) if you need support for a specific database.

### SQLite (In-Memory)

```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
```

### SQLite (File-Based)

```go
db, err := gorm.Open(sqlite.Open("database.db"), &gorm.Config{})
```

### PostgreSQL

```go
import "gorm.io/driver/postgres"

dsn := "host=localhost user=myuser password=mypass dbname=mydb port=5432 sslmode=disable"
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
```

### MySQL

```go
import "gorm.io/driver/mysql"

dsn := "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
```

### SQL Server

```go
import "gorm.io/driver/sqlserver"

dsn := "sqlserver://user:pass@localhost:1433?database=dbname"
db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
```

## GORM Configuration

Customize GORM behavior:

```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
    // Disable foreign key constraints (useful for testing)
    DisableForeignKeyConstraintWhenMigrating: true,
    
    // Log all SQL queries
    Logger: logger.Default.LogMode(logger.Info),
    
    // Use prepared statements
    PrepareStmt: true,
    
    // Set connection pool settings
    ConnPool: &sql.DB{
        MaxIdleConns: 10,
        MaxOpenConns: 100,
    },
})
```

## Production Considerations

### Timeouts

Configure proper timeouts for production:

```go
server := &http.Server{
    Addr:         ":8080",
    Handler:      mux,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 15 * time.Second,
    IdleTimeout:  60 * time.Second,
}

log.Fatal(server.ListenAndServe())
```

### Graceful Shutdown

Implement graceful shutdown:

```go
server := &http.Server{
    Addr:    ":8080",
    Handler: mux,
}

// Run server in goroutine
go func() {
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("Server failed: %v", err)
    }
}()

// Wait for interrupt signal
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

log.Println("Shutting down server...")

// Shutdown with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := server.Shutdown(ctx); err != nil {
    log.Fatalf("Server forced to shutdown: %v", err)
}

log.Println("Server exited")
```

### Connection Pooling

Configure database connection pooling:

```go
sqlDB, err := db.DB()
if err != nil {
    log.Fatal(err)
}

// SetMaxIdleConns sets the maximum number of connections in the idle connection pool
sqlDB.SetMaxIdleConns(10)

// SetMaxOpenConns sets the maximum number of open connections to the database
sqlDB.SetMaxOpenConns(100)

// SetConnMaxLifetime sets the maximum amount of time a connection may be reused
sqlDB.SetConnMaxLifetime(time.Hour)
```
