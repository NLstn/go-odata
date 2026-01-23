package odata

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type denyAllPolicy struct {
	reason string
}

func (p denyAllPolicy) Authorize(ctx AuthContext, resource ResourceDescriptor, operation Operation) Decision {
	return Deny(p.reason)
}

type allowAllPolicy struct{}

func (p allowAllPolicy) Authorize(ctx AuthContext, resource ResourceDescriptor, operation Operation) Decision {
	return Allow()
}

// tenantFilterPolicy implements QueryFilterProvider for row-level security
type tenantFilterPolicy struct {
	tenantID string
}

func (p tenantFilterPolicy) Authorize(ctx AuthContext, resource ResourceDescriptor, operation Operation) Decision {
	return Allow()
}

func (p tenantFilterPolicy) QueryFilter(ctx AuthContext, resource ResourceDescriptor, operation Operation) (*FilterExpression, error) {
	// This demonstrates the row-level security use case - filtering entities by tenant
	return &FilterExpression{
		Property: "TenantID",
		Operator: FilterOperator("eq"),
		Value:    p.tenantID,
	}, nil
}

func TestQueryFilterProviderExported(t *testing.T) {
	// Verify QueryFilterProvider interface is accessible and can be implemented
	var policy QueryFilterProvider = tenantFilterPolicy{tenantID: "tenant-123"}

	// Verify it implements Policy interface
	var _ Policy = policy

	// Verify it can be used with SetPolicy
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.SetPolicy(policy); err != nil {
		t.Fatalf("SetPolicy error: %v", err)
	}
}

func TestSetPolicyAppliesToHandlers(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(Product{}); err != nil {
		t.Fatalf("RegisterEntity error: %v", err)
	}

	if err := service.SetPolicy(denyAllPolicy{reason: "blocked"}); err != nil {
		t.Fatalf("SetPolicy error: %v", err)
	}

	requests := []struct {
		name string
		path string
	}{
		{name: "service document", path: "/"},
		{name: "metadata", path: "/$metadata"},
		{name: "collection", path: "/Products"},
	}

	for _, reqCase := range requests {
		t.Run(reqCase.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, reqCase.path, nil)
			req.Header.Set("Authorization", "Bearer test")
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
			}
		})
	}
}

func TestAllowPolicyPermitsAccess(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}

	if err := service.RegisterEntity(Product{}); err != nil {
		t.Fatalf("RegisterEntity error: %v", err)
	}

	if err := service.SetPolicy(allowAllPolicy{}); err != nil {
		t.Fatalf("SetPolicy error: %v", err)
	}

	requests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{name: "service document", path: "/", expectedStatus: http.StatusOK},
		{name: "metadata", path: "/$metadata", expectedStatus: http.StatusOK},
		{name: "collection", path: "/Products", expectedStatus: http.StatusOK},
	}

	for _, reqCase := range requests {
		t.Run(reqCase.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, reqCase.path, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if w.Code != reqCase.expectedStatus {
				t.Fatalf("expected status %d, got %d", reqCase.expectedStatus, w.Code)
			}
		})
	}
}
