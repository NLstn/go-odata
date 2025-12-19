# Security Policy and Best Practices

## ⚠️ Important Security Notice

The `cmd/devserver` and example code in this repository **MUST NOT** be used in production environments without significant security enhancements. These examples are designed for development and testing purposes only and contain intentionally simplified authentication/authorization mechanisms that are **NOT SECURE**.

## Reporting Security Vulnerabilities

If you discover a security vulnerability in go-odata, please report it privately to the maintainers:

- **DO NOT** open a public GitHub issue for security vulnerabilities
- Email: [Add security contact email]
- Include detailed steps to reproduce the vulnerability
- Allow reasonable time for a fix before public disclosure

## Security Considerations for Production Use

### 🔴 CRITICAL: Authentication & Authorization

#### What's Insecure in Examples

The `cmd/devserver` contains a dummy authentication middleware that is **completely insecure**:

```go
// ❌ NEVER use this in production!
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // This accepts ANY value as a valid user ID!
        userID := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
        // ...
    })
}
```

The Product entity checks authorization using a client-controlled header:

```go
// ❌ NEVER trust client-provided headers for authorization!
isAdmin := r.Header.Get("X-User-Role") == "admin"
```

#### Secure Implementation

For production, implement proper authentication:

##### 1. JWT-Based Authentication (Recommended)

```go
import (
    "github.com/golang-jwt/jwt/v5"
    "crypto/rsa"
)

type Claims struct {
    UserID  string `json:"user_id"`
    IsAdmin bool   `json:"is_admin"`
    jwt.RegisteredClaims
}

func secureAuthMiddleware(publicKey *rsa.PublicKey) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            tokenString := strings.TrimPrefix(authHeader, "Bearer ")
            
            token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
                // Verify signing method
                if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
                    return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
                }
                return publicKey, nil
            })

            if err != nil || !token.Valid {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }

            claims, ok := token.Claims.(*Claims)
            if !ok {
                http.Error(w, "Invalid claims", http.StatusUnauthorized)
                return
            }

            // Store validated claims in context
            ctx := context.WithValue(r.Context(), "userID", claims.UserID)
            ctx = context.WithValue(ctx, "isAdmin", claims.IsAdmin)
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

##### 2. Entity Hooks with Secure Authorization

```go
func (p Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
    // ✓ Get role from validated context (NOT from headers!)
    isAdmin, ok := r.Context().Value("isAdmin").(bool)
    if !ok || !isAdmin {
        return fmt.Errorf("only administrators are allowed to create products")
    }
    return nil
}
```

##### 3. OAuth2 Integration

```go
import "golang.org/x/oauth2"

// Use OAuth2 providers like Auth0, Okta, Azure AD
func setupOAuth2(provider oauth2.Config) http.Handler {
    // Configure OAuth2 flow
    // Validate tokens with provider
    // Store validated claims in context
}
```

### 🔴 CRITICAL: Password and Secret Management

#### Secure Password Storage

**Never** store passwords in plaintext. Always use strong hashing algorithms:

```go
import "golang.org/x/crypto/bcrypt"

type User struct {
    UserID       uint   `json:"UserID" gorm:"primaryKey" odata:"key"`
    Name         string `json:"Name" gorm:"not null" odata:"required"`
    PasswordHash string `json:"-" gorm:"not null"` // Never expose via JSON/OData!
    IsAdmin      bool   `json:"IsAdmin"`
}

// Hash password before storing
func (u *User) SetPassword(password string) error {
    if len(password) < 12 {
        return errors.New("password must be at least 12 characters")
    }
    
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    
    u.PasswordHash = string(hash)
    return nil
}

// Verify password
func (u *User) CheckPassword(password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
    return err == nil
}
```

#### API Key Security

**Never** store API keys in plaintext:

```go
import (
    "crypto/rand"
    "encoding/base64"
    "golang.org/x/crypto/bcrypt"
)

type APIKey struct {
    KeyID       string    `json:"KeyID" gorm:"primaryKey"`
    KeyHash     string    `json:"-" gorm:"not null"` // Store hash, not actual key
    Owner       string    `json:"Owner"`
    Description string    `json:"Description"`
    CreatedAt   time.Time `json:"CreatedAt"`
    ExpiresAt   *time.Time `json:"ExpiresAt"`
}

// Generate new API key
func GenerateAPIKey() (keyID string, actualKey string, err error) {
    // Generate random key
    keyBytes := make([]byte, 32)
    if _, err := rand.Read(keyBytes); err != nil {
        return "", "", err
    }
    
    actualKey = base64.URLEncoding.EncodeToString(keyBytes)
    keyID = "key_" + actualKey[:16] // Prefix for identification
    
    return keyID, actualKey, nil
}

// Hash API key before storage
func (k *APIKey) SetKey(actualKey string) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(actualKey), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    k.KeyHash = string(hash)
    return nil
}

// Verify API key
func (k *APIKey) VerifyKey(providedKey string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(providedKey))
    return err == nil
}
```

### 🟡 HIGH: Transport Layer Security

Always use HTTPS in production:

```go
// Production server with TLS
func main() {
    mux := http.NewServeMux()
    mux.Handle("/", service)
    
    // Use TLS
    err := http.ListenAndServeTLS(
        ":443",
        "/path/to/cert.pem",
        "/path/to/key.pem",
        mux,
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

**Recommended TLS configuration:**
- Use TLS 1.2 or higher
- Use strong cipher suites
- Enable HSTS (HTTP Strict Transport Security)
- Consider using Let's Encrypt for free certificates

### 🟡 HIGH: Rate Limiting

Implement rate limiting to prevent abuse:

```go
import (
    "golang.org/x/time/rate"
    "sync"
)

type visitor struct {
    limiter  *rate.Limiter
    lastSeen time.Time
}

var visitors = make(map[string]*visitor)
var mu sync.Mutex

func rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := r.RemoteAddr
        
        mu.Lock()
        v, exists := visitors[ip]
        if !exists {
            limiter := rate.NewLimiter(rate.Every(time.Minute), 100) // 100 requests per minute
            visitors[ip] = &visitor{limiter, time.Now()}
            v = visitors[ip]
        }
        v.lastSeen = time.Now()
        mu.Unlock()
        
        if !v.limiter.Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### 🟢 MEDIUM: Input Validation

The library handles OData query validation, but validate business logic inputs:

```go
func (p *Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
    // Validate business rules
    if p.Price < 0 {
        return errors.New("price cannot be negative")
    }
    
    if len(p.Name) < 3 {
        return errors.New("product name must be at least 3 characters")
    }
    
    // Sanitize HTML in description
    if p.Description != nil {
        sanitized := html.EscapeString(*p.Description)
        p.Description = &sanitized
    }
    
    return nil
}
```

### 🟢 MEDIUM: Logging and Monitoring

Implement comprehensive security logging:

```go
import "log/slog"

func securityAuditMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Get user from context
        userID, _ := r.Context().Value("userID").(string)
        
        // Log request
        slog.Info("API request",
            "method", r.Method,
            "path", r.URL.Path,
            "user_id", userID,
            "ip", r.RemoteAddr,
        )
        
        // Wrap response writer to capture status
        wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        next.ServeHTTP(wrapped, r)
        
        duration := time.Since(start)
        
        // Log response
        slog.Info("API response",
            "method", r.Method,
            "path", r.URL.Path,
            "user_id", userID,
            "status", wrapped.statusCode,
            "duration_ms", duration.Milliseconds(),
        )
        
        // Alert on suspicious activity
        if wrapped.statusCode == http.StatusUnauthorized {
            slog.Warn("Unauthorized access attempt",
                "ip", r.RemoteAddr,
                "path", r.URL.Path,
            )
        }
    })
}
```

### 🟢 MEDIUM: Database Security

#### Connection String Security

Never hardcode database credentials:

```go
// ❌ Never do this
db, _ := gorm.Open(postgres.Open("postgresql://user:password@localhost/db"))

// ✓ Use environment variables
dsn := os.Getenv("DATABASE_URL")
if dsn == "" {
    log.Fatal("DATABASE_URL environment variable not set")
}
db, err := gorm.Open(postgres.Open(dsn))
```

#### SQL Injection Prevention

The library uses GORM which provides SQL injection protection, but always:

1. Use parameterized queries (GORM does this automatically)
2. Never concatenate user input into SQL strings
3. Validate and sanitize all inputs

#### Database Permissions

Use least privilege principle:

```sql
-- Create read-only user for queries
CREATE USER odata_readonly WITH PASSWORD 'strong_password';
GRANT SELECT ON ALL TABLES IN SCHEMA public TO odata_readonly;

-- Create limited user for writes
CREATE USER odata_writer WITH PASSWORD 'strong_password';
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO odata_writer;
```

## Security Checklist for Production

Before deploying to production, ensure:

- [ ] Authentication uses secure tokens (JWT, OAuth2, etc.)
- [ ] Authorization checks use server-side validated data (not client headers)
- [ ] Passwords are hashed with bcrypt or argon2
- [ ] API keys are hashed before storage
- [ ] HTTPS/TLS is enforced for all connections
- [ ] TLS certificates are valid and properly configured
- [ ] Rate limiting is implemented
- [ ] Input validation is comprehensive
- [ ] Security logging and monitoring is active
- [ ] Database credentials are stored securely (env variables, secrets manager)
- [ ] Database uses least-privilege permissions
- [ ] Error messages don't leak sensitive information
- [ ] CORS is properly configured
- [ ] Security headers are set (HSTS, CSP, X-Frame-Options, etc.)
- [ ] Regular security updates are applied
- [ ] Secrets are not committed to version control

## Recommended Security Headers

```go
func securityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Prevent clickjacking
        w.Header().Set("X-Frame-Options", "DENY")
        
        // Prevent XSS
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // Content Security Policy
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        
        // HSTS (only over HTTPS)
        if r.TLS != nil {
            w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        }
        
        next.ServeHTTP(w, r)
    })
}
```

## Additional Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [Go Security Best Practices](https://golang.org/doc/security)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)

## Support

For security questions or concerns, please contact the maintainers through the appropriate channels listed at the top of this document.

---

**Remember:** Security is not a feature you add at the end—it must be built into your application from the start. The examples in this repository are intentionally simplified for learning purposes and must be significantly enhanced for production use.
