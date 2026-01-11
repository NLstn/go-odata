package auth

import (
	"net/http"
	"testing"

	"github.com/nlstn/go-odata/internal/query"
)

func TestAllow(t *testing.T) {
	decision := Allow()
	if !decision.Allowed {
		t.Error("Allow() should return Allowed: true")
	}
	if decision.Reason != "" {
		t.Errorf("Allow() should have empty Reason, got %q", decision.Reason)
	}
}

func TestDeny(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{"Empty reason", ""},
		{"With reason", "Access denied"},
		{"Detailed reason", "User lacks required permissions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Deny(tt.reason)
			if decision.Allowed {
				t.Error("Deny() should return Allowed: false")
			}
			if decision.Reason != tt.reason {
				t.Errorf("Deny() reason = %q, want %q", decision.Reason, tt.reason)
			}
		})
	}
}

func TestAuthContext(t *testing.T) {
	// Test creation and field access
	ctx := AuthContext{
		Principal: "user123",
		Roles:     []string{"admin", "user"},
		Claims:    map[string]interface{}{"email": "test@example.com"},
		Scopes:    []string{"read", "write"},
		Request: RequestMetadata{
			Method:     "GET",
			Path:       "/api/users",
			Headers:    http.Header{"Authorization": []string{"Bearer token"}},
			Query:      map[string][]string{"filter": {"name eq 'test'"}},
			RemoteAddr: "192.168.1.1",
		},
	}

	if ctx.Principal != "user123" {
		t.Errorf("Principal = %v, want user123", ctx.Principal)
	}
	if len(ctx.Roles) != 2 {
		t.Errorf("len(Roles) = %d, want 2", len(ctx.Roles))
	}
	if len(ctx.Claims) != 1 {
		t.Errorf("len(Claims) = %d, want 1", len(ctx.Claims))
	}
	if len(ctx.Scopes) != 2 {
		t.Errorf("len(Scopes) = %d, want 2", len(ctx.Scopes))
	}
	if ctx.Request.Method != "GET" {
		t.Errorf("Request.Method = %q, want GET", ctx.Request.Method)
	}
}

func TestRequestMetadata(t *testing.T) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")

	metadata := RequestMetadata{
		Method:     "POST",
		Path:       "/api/products",
		Headers:    headers,
		Query:      map[string][]string{"$top": {"10"}, "$skip": {"5"}},
		RemoteAddr: "10.0.0.1:8080",
	}

	if metadata.Method != "POST" {
		t.Errorf("Method = %q, want POST", metadata.Method)
	}
	if metadata.Path != "/api/products" {
		t.Errorf("Path = %q, want /api/products", metadata.Path)
	}
	if len(metadata.Headers) != 2 {
		t.Errorf("len(Headers) = %d, want 2", len(metadata.Headers))
	}
	if len(metadata.Query) != 2 {
		t.Errorf("len(Query) = %d, want 2", len(metadata.Query))
	}
	if metadata.RemoteAddr != "10.0.0.1:8080" {
		t.Errorf("RemoteAddr = %q, want 10.0.0.1:8080", metadata.RemoteAddr)
	}
}

func TestResourceDescriptor(t *testing.T) {
	entity := struct {
		ID   int
		Name string
	}{ID: 1, Name: "Test"}

	descriptor := ResourceDescriptor{
		EntitySetName: "Products",
		EntityType:    "Product",
		KeyValues:     map[string]interface{}{"ID": 1},
		PropertyPath:  []string{"Name", "Description"},
		Entity:        entity,
	}

	if descriptor.EntitySetName != "Products" {
		t.Errorf("EntitySetName = %q, want Products", descriptor.EntitySetName)
	}
	if descriptor.EntityType != "Product" {
		t.Errorf("EntityType = %q, want Product", descriptor.EntityType)
	}
	if len(descriptor.KeyValues) != 1 {
		t.Errorf("len(KeyValues) = %d, want 1", len(descriptor.KeyValues))
	}
	if len(descriptor.PropertyPath) != 2 {
		t.Errorf("len(PropertyPath) = %d, want 2", len(descriptor.PropertyPath))
	}
	if descriptor.Entity == nil {
		t.Error("Entity should not be nil")
	}
}

func TestOperationConstants(t *testing.T) {
	// Test that operation constants are distinct
	operations := []Operation{
		OperationRead,
		OperationCreate,
		OperationUpdate,
		OperationDelete,
		OperationQuery,
		OperationMetadata,
	}

	seen := make(map[Operation]bool)
	for _, op := range operations {
		if seen[op] {
			t.Errorf("Operation %v is not unique", op)
		}
		seen[op] = true
	}

	if len(seen) != 6 {
		t.Errorf("Expected 6 unique operations, got %d", len(seen))
	}
}

func TestDecision(t *testing.T) {
	t.Run("Allowed decision", func(t *testing.T) {
		decision := Decision{Allowed: true, Reason: ""}
		if !decision.Allowed {
			t.Error("Expected Allowed to be true")
		}
	})

	t.Run("Denied decision with reason", func(t *testing.T) {
		decision := Decision{Allowed: false, Reason: "Insufficient permissions"}
		if decision.Allowed {
			t.Error("Expected Allowed to be false")
		}
		if decision.Reason != "Insufficient permissions" {
			t.Errorf("Reason = %q, want 'Insufficient permissions'", decision.Reason)
		}
	})
}

// MockPolicy implements the Policy interface for testing
type MockPolicy struct {
	allowDecision bool
	denyReason    string
}

func (m *MockPolicy) Authorize(ctx AuthContext, resource ResourceDescriptor, operation Operation) Decision {
	if m.allowDecision {
		return Allow()
	}
	return Deny(m.denyReason)
}

func TestPolicyInterface(t *testing.T) {
	t.Run("Allow policy", func(t *testing.T) {
		policy := &MockPolicy{allowDecision: true}
		ctx := AuthContext{}
		resource := ResourceDescriptor{}
		decision := policy.Authorize(ctx, resource, OperationRead)

		if !decision.Allowed {
			t.Error("Expected Allow decision")
		}
	})

	t.Run("Deny policy", func(t *testing.T) {
		policy := &MockPolicy{allowDecision: false, denyReason: "Test deny"}
		ctx := AuthContext{}
		resource := ResourceDescriptor{}
		decision := policy.Authorize(ctx, resource, OperationRead)

		if decision.Allowed {
			t.Error("Expected Deny decision")
		}
		if decision.Reason != "Test deny" {
			t.Errorf("Reason = %q, want 'Test deny'", decision.Reason)
		}
	})
}

// MockQueryFilterProvider implements the QueryFilterProvider interface for testing
type MockQueryFilterProvider struct {
	MockPolicy
	filter *query.FilterExpression
	err    error
}

func (m *MockQueryFilterProvider) QueryFilter(ctx AuthContext, resource ResourceDescriptor, operation Operation) (*query.FilterExpression, error) {
	return m.filter, m.err
}

func TestQueryFilterProviderInterface(t *testing.T) {
	t.Run("Returns filter", func(t *testing.T) {
		expectedFilter := &query.FilterExpression{Property: "UserID"}
		provider := &MockQueryFilterProvider{
			MockPolicy: MockPolicy{allowDecision: true},
			filter:     expectedFilter,
		}

		ctx := AuthContext{}
		resource := ResourceDescriptor{}
		filter, err := provider.QueryFilter(ctx, resource, OperationQuery)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if filter != expectedFilter {
			t.Error("Filter mismatch")
		}
	})

	t.Run("Returns nil filter", func(t *testing.T) {
		provider := &MockQueryFilterProvider{
			MockPolicy: MockPolicy{allowDecision: true},
			filter:     nil,
		}

		ctx := AuthContext{}
		resource := ResourceDescriptor{}
		filter, err := provider.QueryFilter(ctx, resource, OperationQuery)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if filter != nil {
			t.Error("Expected nil filter")
		}
	})
}
