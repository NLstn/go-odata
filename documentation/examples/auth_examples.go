//go:build example

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

// Example 8: Per-Entity Authorization (e.g., Per-Club Roles)
// ===========================================================
//
// This example demonstrates how to implement authorization where users have
// different roles per entity (e.g., a user can be an owner of Club A but just
// a member of Club B). This requires database access in the Policy to resolve
// entity-specific roles.

// Club entity
type Club struct {
	ID          int    `json:"id" odata:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     string `json:"ownerId"`
}

// ClubMembership represents a user's membership in a club
type ClubMembership struct {
	ID     int    `gorm:"primarykey" json:"id" odata:"key"`
	UserID string `json:"userId"`
	ClubID int    `json:"clubId"`
	Role   string `json:"role"` // "owner", "admin", "member", "viewer"
}

// PerEntityPolicy implements authorization that depends on per-entity roles.
// It requires database access to resolve a user's role for specific entities.
type PerEntityPolicy struct {
	db *gorm.DB
}

func NewPerEntityPolicy(db *gorm.DB) *PerEntityPolicy {
	return &PerEntityPolicy{db: db}
}

func (p *PerEntityPolicy) Authorize(ctx odata.AuthContext, resource odata.ResourceDescriptor, op odata.Operation) odata.Decision {
	// Allow metadata operations
	if op == odata.OperationMetadata {
		return odata.Allow()
	}

	// Extract user ID from auth context
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

	// For Club entity operations, check per-club roles
	if resource.EntitySetName == "Clubs" {
		// For collection operations (query, list), allow authenticated users to see all clubs
		// but use QueryFilter to restrict based on membership
		if op == odata.OperationQuery {
			return odata.Allow()
		}

		// For specific club operations, check the user's role for that club
		if len(resource.KeyValues) > 0 {
			clubID, ok := resource.KeyValues["id"].(int)
			if !ok {
				return odata.Deny("Invalid club ID")
			}

			role := p.getUserRoleForClub(userID, clubID)

			// Determine if operation is allowed based on role
			switch op {
			case odata.OperationRead:
				// Anyone with any role can read
				if role != "" {
					return odata.Allow()
				}
			case odata.OperationUpdate:
				// Only owners and admins can update
				if role == "owner" || role == "admin" {
					return odata.Allow()
				}
			case odata.OperationDelete:
				// Only owners can delete
				if role == "owner" {
					return odata.Allow()
				}
			case odata.OperationAction, odata.OperationFunction:
				// Check action/function name from PropertyPath
				if len(resource.PropertyPath) > 0 {
					actionName := resource.PropertyPath[0]
					switch actionName {
					case "PromoteMember":
						// Only owners can promote members
						if role == "owner" {
							return odata.Allow()
						}
					case "PostMessage":
						// Members and above can post messages
						if role == "owner" || role == "admin" || role == "member" {
							return odata.Allow()
						}
					}
				}
			}

			return odata.Deny(fmt.Sprintf("User does not have required role for this club (current role: %s)", role))
		}

		// For creation, allow authenticated users (they become the owner)
		if op == odata.OperationCreate {
			return odata.Allow()
		}
	}

	return odata.Deny("Access denied")
}

// QueryFilter implements row-level security by filtering clubs to only show
// those where the user has membership.
func (p *PerEntityPolicy) QueryFilter(ctx odata.AuthContext, resource odata.ResourceDescriptor, op odata.Operation) (*odata.FilterExpression, error) {
	// Only filter Clubs collection
	if resource.EntitySetName != "Clubs" {
		return nil, nil
	}

	// Extract user ID
	userID, ok := ctx.Principal.(string)
	if !ok {
		return nil, nil
	}

	// Check for global admin
	for _, role := range ctx.Roles {
		if role == "global-admin" {
			return nil, nil // No filter needed for global admins
		}
	}

	// Get all club IDs where user has membership
	var memberships []ClubMembership
	if err := p.db.Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
		return nil, fmt.Errorf("failed to query memberships: %w", err)
	}

	if len(memberships) == 0 {
		// User has no memberships, return filter that matches nothing
		// This is represented as "id eq -1" (assuming -1 is never a valid ID)
		return &odata.FilterExpression{
			Property: "id",
			Operator: "eq",
			Value:    -1,
		}, nil
	}

	// Build OR filter for all club IDs
	var filters []*odata.FilterExpression
	for _, membership := range memberships {
		filters = append(filters, &odata.FilterExpression{
			Property: "id",
			Operator: "eq",
			Value:    membership.ClubID,
		})
	}

	// If only one club, return single filter
	if len(filters) == 1 {
		return filters[0], nil
	}

	// Build OR chain for multiple clubs
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

// getUserRoleForClub queries the database to get the user's role for a specific club
func (p *PerEntityPolicy) getUserRoleForClub(userID string, clubID int) string {
	var membership ClubMembership
	if err := p.db.Where("user_id = ? AND club_id = ?", userID, clubID).First(&membership).Error; err != nil {
		return "" // No membership found
	}
	return membership.Role
}

// Example 9: Setup with Per-Entity Authorization
// ==============================================

func setupPerEntityAuthorization() {
	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Migrate tables
	if err := db.AutoMigrate(&Club{}, &ClubMembership{}); err != nil {
		log.Fatal(err)
	}

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil {
		log.Fatal(err)
	}

	// Register entities
	if err := service.RegisterEntity(&Club{}); err != nil {
		log.Fatal(err)
	}

	// Set up PreRequestHook to populate auth context with standard keys
	// This extracts user authentication from the request and stores it in
	// the request context using the standard keys that the framework will
	// automatically extract in buildAuthContext.
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		// Extract and validate JWT token (simplified example)
		token := r.Header.Get("Authorization")
		if token == "" {
			// Allow anonymous access - let policy decide
			return r.Context(), nil
		}

		// In production: validate JWT and extract claims
		// For example: claims := validateJWT(token)

		// Simulate extracting user info from token
		userID := "user-123"            // Extract from JWT claims
		globalRoles := []string{"user"} // Extract from JWT claims

		// Store auth data in context using the standard keys
		ctx := r.Context()
		ctx = context.WithValue(ctx, odata.PrincipalContextKey, userID)
		ctx = context.WithValue(ctx, odata.RolesContextKey, globalRoles)
		ctx = context.WithValue(ctx, odata.ClaimsContextKey, map[string]interface{}{
			"email": "user@example.com",
		})

		return ctx, nil
	}); err != nil {
		log.Fatal(err)
	}

	// Set the per-entity authorization policy
	policy := NewPerEntityPolicy(db)
	if err := service.SetPolicy(policy); err != nil {
		log.Fatal(err)
	}

	// Start server
	log.Println("Server with per-entity authorization running on :8080")
	log.Fatal(http.ListenAndServe(":8080", service))
}

// Key Takeaways for Per-Entity Authorization:
// ============================================
//
// 1. Store Entity Relationships in Database
//    - Create a membership/role table linking users to entities
//    - Store per-entity roles (owner, admin, member, etc.)
//
// 2. Policy Needs Database Access
//    - Pass *gorm.DB to policy constructor
//    - Query database in Authorize() to resolve entity-specific roles
//
// 3. Use QueryFilterProvider for Collection Queries
//    - Implement QueryFilter() to restrict visible entities
//    - Build filter expressions that match user's accessible entities
//
// 4. Use Standard Context Keys in PreRequestHook
//    - Store Principal using odata.PrincipalContextKey
//    - Store global Roles using odata.RolesContextKey
//    - Store Claims using odata.ClaimsContextKey
//    - The framework automatically extracts these in buildAuthContext
//
// 5. Check Both Global and Entity-Specific Roles
//    - Check ctx.Roles for global roles (e.g., "global-admin")
//    - Query database for entity-specific roles when needed
//    - Combine both to make authorization decisions
//
// 6. Handle Different Operations Appropriately
//    - Read: Allow if user has any role for entity
//    - Update: Require elevated role (admin, owner)
//    - Delete: Require highest role (owner)
//    - Actions/Functions: Check operation name and apply custom logic
