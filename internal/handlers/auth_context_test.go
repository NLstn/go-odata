package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/auth"
)

type authContextCapturePolicy struct {
	lastAuthContext auth.AuthContext
}

func (p *authContextCapturePolicy) Authorize(ctx auth.AuthContext, _ auth.ResourceDescriptor, _ auth.Operation) auth.Decision {
	p.lastAuthContext = ctx
	return auth.Allow()
}

func TestBuildAuthContext_ExtractsFromRequestContext(t *testing.T) {
	tests := []struct {
		name              string
		setupContext      func(*http.Request) *http.Request
		expectedPrincipal interface{}
		expectedRoles     []string
		expectedClaims    map[string]interface{}
		expectedScopes    []string
	}{
		{
			name: "All auth data present",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), auth.PrincipalContextKey, "user@example.com")
				ctx = context.WithValue(ctx, auth.RolesContextKey, []string{"admin", "user"})
				ctx = context.WithValue(ctx, auth.ClaimsContextKey, map[string]interface{}{
					"tenant_id": "123",
					"user_id":   "456",
				})
				ctx = context.WithValue(ctx, auth.ScopesContextKey, []string{"read", "write"})
				return r.WithContext(ctx)
			},
			expectedPrincipal: "user@example.com",
			expectedRoles:     []string{"admin", "user"},
			expectedClaims:    map[string]interface{}{"tenant_id": "123", "user_id": "456"},
			expectedScopes:    []string{"read", "write"},
		},
		{
			name: "Only principal present",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), auth.PrincipalContextKey, "user123")
				return r.WithContext(ctx)
			},
			expectedPrincipal: "user123",
			expectedRoles:     nil,
			expectedClaims:    nil,
			expectedScopes:    nil,
		},
		{
			name: "Only roles present",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), auth.RolesContextKey, []string{"editor"})
				return r.WithContext(ctx)
			},
			expectedPrincipal: nil,
			expectedRoles:     []string{"editor"},
			expectedClaims:    nil,
			expectedScopes:    nil,
		},
		{
			name: "No auth data",
			setupContext: func(r *http.Request) *http.Request {
				return r
			},
			expectedPrincipal: nil,
			expectedRoles:     nil,
			expectedClaims:    nil,
			expectedScopes:    nil,
		},
		{
			name: "Invalid types ignored",
			setupContext: func(r *http.Request) *http.Request {
				ctx := context.WithValue(r.Context(), auth.RolesContextKey, "not-a-slice")
				ctx = context.WithValue(ctx, auth.ClaimsContextKey, "not-a-map")
				ctx = context.WithValue(ctx, auth.ScopesContextKey, 123)
				return r.WithContext(ctx)
			},
			expectedPrincipal: nil,
			expectedRoles:     nil,
			expectedClaims:    nil,
			expectedScopes:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req = tt.setupContext(req)

			authCtx := buildAuthContext(req)

			if authCtx.Principal != tt.expectedPrincipal {
				t.Errorf("Principal = %v, want %v", authCtx.Principal, tt.expectedPrincipal)
			}

			if !slicesEqual(authCtx.Roles, tt.expectedRoles) {
				t.Errorf("Roles = %v, want %v", authCtx.Roles, tt.expectedRoles)
			}

			if !mapsEqual(authCtx.Claims, tt.expectedClaims) {
				t.Errorf("Claims = %v, want %v", authCtx.Claims, tt.expectedClaims)
			}

			if !slicesEqual(authCtx.Scopes, tt.expectedScopes) {
				t.Errorf("Scopes = %v, want %v", authCtx.Scopes, tt.expectedScopes)
			}

			// Verify request metadata is still populated
			if authCtx.Request.Method != "GET" {
				t.Errorf("Request.Method = %v, want GET", authCtx.Request.Method)
			}
			if authCtx.Request.Path != "/test" {
				t.Errorf("Request.Path = %v, want /test", authCtx.Request.Path)
			}
		})
	}
}

func TestAuthorizeRequest_WithAuthContext(t *testing.T) {
	handler, _ := setupTestHandler(t)
	policy := &authContextCapturePolicy{}
	handler.SetPolicy(policy)

	// Create request with auth context
	req := httptest.NewRequest(http.MethodGet, "/TestEntities", nil)
	ctx := context.WithValue(req.Context(), auth.PrincipalContextKey, "test-user")
	ctx = context.WithValue(ctx, auth.RolesContextKey, []string{"viewer"})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.HandleCollection(w, req)

	// Verify auth context was passed to policy
	if policy.lastAuthContext.Principal != "test-user" {
		t.Errorf("Principal = %v, want test-user", policy.lastAuthContext.Principal)
	}
	if len(policy.lastAuthContext.Roles) != 1 || policy.lastAuthContext.Roles[0] != "viewer" {
		t.Errorf("Roles = %v, want [viewer]", policy.lastAuthContext.Roles)
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
