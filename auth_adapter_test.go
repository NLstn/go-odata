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

func TestSetPolicyAppliesToHandlers(t *testing.T) {
	db := setupTestDB(t)
	service, err := NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }

	if err := service.RegisterEntity(Product{}); err != nil {
		t.Fatalf("RegisterEntity error: %v", err)
	}

	service.SetPolicy(denyAllPolicy{reason: "blocked"})

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

	service.SetPolicy(allowAllPolicy{})

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
