package odata_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// PreRequestHookProduct is a test entity for pre-request hook tests
type PreRequestHookProduct struct {
	ID    uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name  string  `json:"Name"`
	Price float64 `json:"Price"`
}

// contextKey is used for storing user info in context
type contextKey string

const userContextKey contextKey = "user"

func setupPreRequestHookTest(t *testing.T) (*odata.Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&PreRequestHookProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(&PreRequestHookProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return service, db
}

func TestPreRequestHook_SingleRequest_ContextEnrichment(t *testing.T) {
	service, db := setupPreRequestHookTest(t)

	// Insert test data
	product := PreRequestHookProduct{ID: 1, Name: "Test Product", Price: 99.99}
	db.Create(&product)

	// Track whether hook was called and what token was seen
	var hookCalled bool
	var tokenSeen string

	// Set the pre-request hook to capture authorization header
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		hookCalled = true
		tokenSeen = r.Header.Get("Authorization")
		if tokenSeen != "" {
			// Simulate loading user from token and adding to context
			return context.WithValue(r.Context(), userContextKey, "test-user"), nil
		}
		return nil, nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	// Make request with Authorization header
	req := httptest.NewRequest(http.MethodGet, "/PreRequestHookProducts(1)", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	if !hookCalled {
		t.Error("PreRequestHook was not called")
	}

	if tokenSeen != "Bearer test-token" {
		t.Errorf("PreRequestHook did not see Authorization header. Got: %s", tokenSeen)
	}
}

func TestPreRequestHook_SingleRequest_AuthenticationFailure(t *testing.T) {
	service, _ := setupPreRequestHookTest(t)

	// Set the pre-request hook to reject invalid tokens
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		token := r.Header.Get("Authorization")
		if token == "Bearer invalid-token" {
			return nil, fmt.Errorf("authentication failed: invalid token")
		}
		return nil, nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	// Make request with invalid token
	req := httptest.NewRequest(http.MethodGet, "/PreRequestHookProducts(1)", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestPreRequestHook_BatchSubRequest_NonChangeset(t *testing.T) {
	service, db := setupPreRequestHookTest(t)

	// Insert test data
	product := PreRequestHookProduct{ID: 1, Name: "Test Product", Price: 99.99}
	db.Create(&product)

	// Track hook calls
	hookCallCount := 0
	var authHeadersFromHook []string

	// Set the pre-request hook
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		hookCallCount++
		auth := r.Header.Get("Authorization")
		if auth != "" {
			authHeadersFromHook = append(authHeadersFromHook, auth)
		}
		return nil, nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	// Create batch request with Authorization header in sub-request
	boundary := "batch_pre_request_hook"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /PreRequestHookProducts(1) HTTP/1.1
Host: localhost
Accept: application/json
Authorization: Bearer sub-request-token


--%s--
`, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	req.Header.Set("Authorization", "Bearer outer-request-token")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Hook should be called at least twice:
	// 1. For the outer batch request
	// 2. For the sub-request (when it goes through ServeHTTP)
	if hookCallCount < 2 {
		t.Errorf("Expected hook to be called at least 2 times, got %d", hookCallCount)
	}

	// Verify we saw both authorization headers
	outerFound := false
	subFound := false
	for _, auth := range authHeadersFromHook {
		if auth == "Bearer outer-request-token" {
			outerFound = true
		}
		if auth == "Bearer sub-request-token" {
			subFound = true
		}
	}
	if !outerFound {
		t.Error("Hook did not see outer request Authorization header")
	}
	if !subFound {
		t.Error("Hook did not see sub-request Authorization header")
	}
}

func TestPreRequestHook_BatchChangeset_ContextEnrichment(t *testing.T) {
	service, _ := setupPreRequestHookTest(t)

	// Track hook calls and context availability
	hookCallCount := 0
	var authHeadersSeen []string

	// Set the pre-request hook
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		hookCallCount++
		auth := r.Header.Get("Authorization")
		if auth != "" {
			authHeadersSeen = append(authHeadersSeen, auth)
		}
		return context.WithValue(r.Context(), userContextKey, "changeset-user"), nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	// Create batch request with changeset
	batchBoundary := "batch_changeset_hook"
	changesetBoundary := "changeset_hook"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /PreRequestHookProducts HTTP/1.1
Host: localhost
Content-Type: application/json
Authorization: Bearer changeset-token

{"Name":"New Product","Price":50.00}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	req.Header.Set("Authorization", "Bearer outer-batch-token")
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify hook was called for changeset sub-request
	// The hook should be called for:
	// 1. The outer batch request
	// 2. The changeset sub-request (POST)
	if hookCallCount < 2 {
		t.Errorf("Expected hook to be called at least 2 times, got %d", hookCallCount)
	}

	// Verify we saw the changeset token
	changesetTokenFound := false
	for _, auth := range authHeadersSeen {
		if auth == "Bearer changeset-token" {
			changesetTokenFound = true
			break
		}
	}
	if !changesetTokenFound {
		t.Errorf("Hook did not see changeset sub-request Authorization header. Seen: %v", authHeadersSeen)
	}

	// Verify product was created
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "HTTP/1.1 201") {
		t.Errorf("Expected 201 Created response in batch. Body: %s", responseBody)
	}
}

func TestPreRequestHook_BatchChangeset_AuthenticationFailure(t *testing.T) {
	service, _ := setupPreRequestHookTest(t)

	// Set the pre-request hook to reject invalid tokens
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		token := r.Header.Get("Authorization")
		if token == "Bearer invalid-changeset-token" {
			return nil, fmt.Errorf("changeset authentication failed")
		}
		return nil, nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	// Create batch request with changeset containing invalid token
	batchBoundary := "batch_changeset_auth_fail"
	changesetBoundary := "changeset_auth_fail"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /PreRequestHookProducts HTTP/1.1
Host: localhost
Content-Type: application/json
Authorization: Bearer invalid-changeset-token

{"Name":"Should Fail","Price":100.00}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	// Batch request itself should succeed
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// But the changeset sub-request should fail with 403
	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "403") {
		t.Errorf("Expected 403 error in batch response. Body: %s", responseBody)
	}
}

func TestPreRequestHook_NilHook(t *testing.T) {
	service, db := setupPreRequestHookTest(t)

	// Insert test data
	product := PreRequestHookProduct{ID: 1, Name: "Test Product", Price: 99.99}
	db.Create(&product)

	// Don't set any hook - should work without errors

	req := httptest.NewRequest(http.MethodGet, "/PreRequestHookProducts(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestPreRequestHook_NilContextReturn(t *testing.T) {
	service, db := setupPreRequestHookTest(t)

	// Insert test data
	product := PreRequestHookProduct{ID: 1, Name: "Test Product", Price: 99.99}
	db.Create(&product)

	// Set hook that returns nil context (no enrichment)
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/PreRequestHookProducts(1)", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestPreRequestHook_MultipleSubRequests(t *testing.T) {
	service, db := setupPreRequestHookTest(t)

	// Insert test data
	products := []PreRequestHookProduct{
		{ID: 1, Name: "Product 1", Price: 10.00},
		{ID: 2, Name: "Product 2", Price: 20.00},
	}
	for _, p := range products {
		db.Create(&p)
	}

	// Track hook calls
	hookCallCount := 0

	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		hookCallCount++
		return nil, nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	// Create batch request with multiple GET requests
	boundary := "batch_multiple_sub"
	body := fmt.Sprintf(`--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /PreRequestHookProducts(1) HTTP/1.1
Host: localhost
Accept: application/json


--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /PreRequestHookProducts(2) HTTP/1.1
Host: localhost
Accept: application/json


--%s--
`, boundary, boundary, boundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Hook should be called for:
	// 1. The outer batch request
	// 2. First sub-request (GET Products(1))
	// 3. Second sub-request (GET Products(2))
	if hookCallCount < 3 {
		t.Errorf("Expected hook to be called at least 3 times, got %d", hookCallCount)
	}
}

func TestPreRequestHook_MultipleHookUpdates(t *testing.T) {
	service, db := setupPreRequestHookTest(t)

	// Insert test data
	product := PreRequestHookProduct{ID: 1, Name: "Test Product", Price: 99.99}
	db.Create(&product)

	// Track which hook was called
	var hook1Called, hook2Called bool

	// Set the first hook
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		hook1Called = true
		return nil, nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	// Make a request to verify hook 1 is called
	req1 := httptest.NewRequest(http.MethodGet, "/PreRequestHookProducts(1)", nil)
	w1 := httptest.NewRecorder()
	service.ServeHTTP(w1, req1)

	if !hook1Called {
		t.Error("First hook was not called")
	}
	if hook2Called {
		t.Error("Second hook was called before it was set")
	}

	// Reset tracking and set the second hook
	hook1Called = false
	hook2Called = false
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		hook2Called = true
		return nil, nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	// Make another request to verify hook 2 is now called
	req2 := httptest.NewRequest(http.MethodGet, "/PreRequestHookProducts(1)", nil)
	w2 := httptest.NewRecorder()
	service.ServeHTTP(w2, req2)

	if hook1Called {
		t.Error("First hook was called after second hook was set")
	}
	if !hook2Called {
		t.Error("Second hook was not called")
	}
}

func TestPreRequestHook_MultipleHookUpdates_BatchChangeset(t *testing.T) {
	service, _ := setupPreRequestHookTest(t)

	// Track which hook was called
	var hook1Called, hook2Called bool

	// Set the first hook
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		hook1Called = true
		return nil, nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	// Create a batch request with changeset
	batchBoundary := "batch_multi_hook"
	changesetBoundary := "changeset_multi_hook"
	body1 := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /PreRequestHookProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 1","Price":10.00}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req1 := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body1))
	req1.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w1 := httptest.NewRecorder()
	service.ServeHTTP(w1, req1)

	if !hook1Called {
		t.Error("First hook was not called for batch changeset")
	}
	if hook2Called {
		t.Error("Second hook was called before it was set")
	}

	// Reset tracking and set the second hook
	hook1Called = false
	hook2Called = false
	if err := service.SetPreRequestHook(func(r *http.Request) (context.Context, error) {
		hook2Called = true
		return nil, nil
	}); err != nil {
		t.Fatalf("Failed to set pre-request hook: %v", err)
	}

	// Make another batch request with changeset
	body2 := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /PreRequestHookProducts HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Product 2","Price":20.00}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req2 := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body2))
	req2.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w2 := httptest.NewRecorder()
	service.ServeHTTP(w2, req2)

	if hook1Called {
		t.Error("First hook was called after second hook was set for batch changeset")
	}
	if !hook2Called {
		t.Error("Second hook was not called for batch changeset")
	}
}
