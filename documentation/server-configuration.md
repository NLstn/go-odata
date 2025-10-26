# Server Configuration

This guide covers how to configure and integrate the go-odata service into your application.

## Table of Contents

- [Basic Setup](#basic-setup)
- [Service as Handler](#service-as-handler)
- [Custom Path Mounting](#custom-path-mounting)
- [Adding Middleware](#adding-middleware)
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
    service := odata.NewService(db)
    
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
- Lifecycle hooks (BeforeCreate, BeforeUpdate)
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

**Note:** The compliance test script automatically starts and stops the compliance server.

See [cmd/complianceserver/README.md](../cmd/complianceserver/README.md) for more details.

## Database Configuration

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
