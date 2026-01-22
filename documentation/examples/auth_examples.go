// +build example

// Package main demonstrates authorization patterns in go-odata.
//
// This example shows how to:
// 1. Implement a custom Policy for authorization decisions
// 2. Use QueryFilterProvider for row-level security
// 3. Populate AuthContext from request data
// 4. Combine policies with custom business logic
//
// Note: This is a standalone example file that demonstrates authorization concepts.
// It cannot be run directly with other example files due to package conflicts.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Example 1: Basic Policy Implementation
// ======================================

// SimplePolicy implements basic role-based authorization.
type SimplePolicy struct{}

func (p *SimplePolicy) Authorize(ctx odata.AuthContext, resource odata.ResourceDescriptor, op odata.Operation) odata.Decision {
	// Allow all metadata operations
	if op == odata.OperationMetadata {
		return odata.Allow()
	}

	// Check if user has required role
	hasRole := false
	for _, role := range ctx.Roles {
		if role == "admin" || role == "user" {
			hasRole = true
			break
		}
	}

	if !hasRole {
		return odata.Deny("User does not have required role")
	}

	// Admins can do anything
	for _, role := range ctx.Roles {
		if role == "admin" {
			return odata.Allow()
		}
	}

	// Regular users can only read
	if op == odata.OperationRead || op == odata.OperationQuery {
		return odata.Allow()
	}

	return odata.Deny("Operation not permitted for user role")
}

// Example 2: Tenant-Based Policy with Row-Level Security
// =======================================================

// TenantPolicy implements multi-tenant authorization with row-level filtering.
type TenantPolicy struct{}

func (p *TenantPolicy) Authorize(ctx odata.AuthContext, resource odata.ResourceDescriptor, op odata.Operation) odata.Decision {
	// Allow metadata operations
	if op == odata.OperationMetadata {
		return odata.Allow()
	}

	// Ensure user has a tenant ID in their claims
	tenantID, ok := ctx.Claims["tenant_id"]
	if !ok {
		return odata.Deny("User is not associated with a tenant")
	}

	// For single entity operations, verify the entity belongs to the user's tenant
	if resource.Entity != nil {
		// Assuming entities have a TenantID field
		if entityWithTenant, ok := resource.Entity.(interface{ GetTenantID() string }); ok {
			if entityWithTenant.GetTenantID() != tenantID {
				return odata.Deny("Access denied: entity belongs to different tenant")
			}
		}
	}

	return odata.Allow()
}

// QueryFilter implements QueryFilterProvider to add tenant filtering to all queries.
func (p *TenantPolicy) QueryFilter(ctx odata.AuthContext, resource odata.ResourceDescriptor, op odata.Operation) (*odata.FilterExpression, error) {
	// Don't filter metadata operations
	if op == odata.OperationMetadata {
		return nil, nil
	}

	// Extract tenant ID from claims
	tenantID, ok := ctx.Claims["tenant_id"]
	if !ok {
		return nil, fmt.Errorf("user is not associated with a tenant")
	}

	// Create a filter expression: TenantID eq 'user-tenant-id'
	filter := &odata.FilterExpression{
		Property: "TenantID",
		Operator: odata.OpEqual,
		Value:    tenantID,
	}

	return filter, nil
}

// Example 3: Resource-Level Policy
// =================================

// ResourcePolicy implements fine-grained resource-level authorization.
type ResourcePolicy struct{}

func (p *ResourcePolicy) Authorize(ctx odata.AuthContext, resource odata.ResourceDescriptor, op odata.Operation) odata.Decision {
	// Allow metadata operations
	if op == odata.OperationMetadata {
		return odata.Allow()
	}

	// Check permissions based on entity set
	switch resource.EntitySetName {
	case "Employees":
		// Only HR can modify employees
		if op == odata.OperationCreate || op == odata.OperationUpdate || op == odata.OperationDelete {
			for _, role := range ctx.Roles {
				if role == "hr" {
					return odata.Allow()
				}
			}
			return odata.Deny("Only HR can modify employee records")
		}
		return odata.Allow()

	case "SalaryRecords":
		// Only HR and Finance can access salary records
		for _, role := range ctx.Roles {
			if role == "hr" || role == "finance" {
				return odata.Allow()
			}
		}
		return odata.Deny("Insufficient permissions to access salary records")

	case "Products":
		// Everyone can read products, only admins can modify
		if op == odata.OperationRead || op == odata.OperationQuery {
			return odata.Allow()
		}
		for _, role := range ctx.Roles {
			if role == "admin" {
				return odata.Allow()
			}
		}
		return odata.Deny("Only administrators can modify products")
	}

	// Default: allow
	return odata.Allow()
}

// Example 4: Scope-Based Policy (OAuth/JWT)
// ==========================================

// ScopePolicy implements OAuth2 scope-based authorization.
type ScopePolicy struct{}

func (p *ScopePolicy) Authorize(ctx odata.AuthContext, resource odata.ResourceDescriptor, op odata.Operation) odata.Decision {
	// Map operations to required scopes
	requiredScope := ""
	switch op {
	case odata.OperationRead, odata.OperationQuery:
		requiredScope = fmt.Sprintf("%s.read", resource.EntitySetName)
	case odata.OperationCreate:
		requiredScope = fmt.Sprintf("%s.write", resource.EntitySetName)
	case odata.OperationUpdate:
		requiredScope = fmt.Sprintf("%s.write", resource.EntitySetName)
	case odata.OperationDelete:
		requiredScope = fmt.Sprintf("%s.delete", resource.EntitySetName)
	case odata.OperationMetadata:
		return odata.Allow() // Always allow metadata
	}

	// Check if user has the required scope
	for _, scope := range ctx.Scopes {
		if scope == requiredScope || scope == "*" {
			return odata.Allow()
		}
	}

	return odata.Deny(fmt.Sprintf("Missing required scope: %s", requiredScope))
}

// Example 5: Populating AuthContext in PreRequestHook
// ====================================================

// This example shows how to extract authentication information from the HTTP request
// and populate the AuthContext that will be used by authorization policies.

func createAuthContextFromRequest(r *http.Request) (context.Context, error) {
	// Extract authentication token from header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Anonymous access - create empty auth context
		return context.WithValue(r.Context(), "auth", odata.AuthContext{}), nil
	}

	// Parse Bearer token (simplified - in production, validate JWT/token properly)
	_ = strings.TrimPrefix(authHeader, "Bearer ")

	// In production, validate the token and extract claims
	// For this example, we'll simulate extracting user info from a token
	authCtx := odata.AuthContext{
		Principal: "user@example.com",
		Roles:     []string{"user", "developer"},
		Claims: map[string]interface{}{
			"user_id":   "12345",
			"tenant_id": "tenant-abc",
			"email":     "user@example.com",
		},
		Scopes: []string{"Products.read", "Products.write", "Orders.read"},
	}

	// You could also extract information from the token:
	// claims, err := validateJWT(token)
	// if err != nil {
	//     return nil, odata.ErrUnauthorized
	// }
	//
	// authCtx.Principal = claims["sub"]
	// authCtx.Roles = claims["roles"].([]string)
	// authCtx.Claims = claims
	// authCtx.Scopes = strings.Split(claims["scope"].(string), " ")

	// Add auth context to request context
	ctx := context.WithValue(r.Context(), "auth", authCtx)
	return ctx, nil
}

// Example 6: Complete Setup with Policy and Auth Context
// =======================================================

func main() {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil {
		log.Fatal(err)
	}

	// Set up PreRequestHook to populate auth context
	service.SetPreRequestHook(createAuthContextFromRequest)

	// Choose and configure authorization policy
	// Option 1: Simple role-based policy
	// policy := &SimplePolicy{}

	// Option 2: Tenant-based policy with row-level security
	policy := &TenantPolicy{}

	// Option 3: Resource-level policy
	// policy := &ResourcePolicy{}

	// Option 4: Scope-based policy
	// policy := &ScopePolicy{}

	// Set the policy on the service
	err = service.SetPolicy(policy)
	if err != nil {
		log.Fatal(err)
	}

	// Register entities
	// ... (register your entities here)

	// Start server
	log.Println("OData service with authorization running on :8080")
	log.Fatal(http.ListenAndServe(":8080", service))
}

// Example 7: Field-Level Authorization with After Read Hook
// ==========================================================

// User entity with sensitive fields
type User struct {
	ID       int    `json:"id" odata:"key"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Salary   int    `json:"salary"` // Sensitive field
	SSN      string `json:"ssn"`    // Sensitive field
	TenantID string `json:"tenantId"`
}

func (u *User) GetTenantID() string {
	return u.TenantID
}

// ODataAfterReadEntity redacts sensitive fields based on user permissions
func (u User) ODataAfterReadEntity(ctx context.Context, r *http.Request, opts *odata.QueryOptions, entity interface{}) (interface{}, error) {
	user := entity.(*User)

	// Extract auth context from request context
	authCtx, ok := ctx.Value("auth").(odata.AuthContext)
	if !ok {
		// No auth context - redact all sensitive fields
		user.Salary = 0
		user.SSN = "REDACTED"
		return user, nil
	}

	// Check if user has appropriate role to view sensitive data
	isAuthorized := false
	for _, role := range authCtx.Roles {
		if role == "admin" || role == "hr" {
			isAuthorized = true
			break
		}
	}

	if !isAuthorized {
		// Redact sensitive fields for unauthorized users
		user.Salary = 0
		user.SSN = "REDACTED"
	}

	return user, nil
}
