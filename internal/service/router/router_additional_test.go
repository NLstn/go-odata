package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/response"
)

func TestRouter_ODataMaxVersionRejected(t *testing.T) {
	handler := newStubEntityHandler()
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	req.Header.Set("OData-MaxVersion", "3.0")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotAcceptable {
		t.Fatalf("expected status %d, got %d", http.StatusNotAcceptable, rec.Code)
	}
	if len(handler.calls) != 0 {
		t.Fatalf("expected no handler calls, got %v", handler.calls)
	}
}

func TestRouter_ODataMaxVersionInvalidIgnored(t *testing.T) {
	called := false
	r := NewRouter(
		func(string) (EntityHandler, bool) { return nil, false },
		func(http.ResponseWriter, *http.Request) { called = true },
		func(http.ResponseWriter, *http.Request) {},
		func(http.ResponseWriter, *http.Request) {},
		nil,
		nil,
		func(http.ResponseWriter, *http.Request, string, string, bool, string) {},
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("OData-MaxVersion", "invalid")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !called {
		t.Fatalf("expected service document handler to be called")
	}
}

func TestRouter_ActionOrFunctionMethodNotAllowed(t *testing.T) {
	r := newTestRouter(nil, nil, map[string][]*actions.FunctionDefinition{
		"TopProducts": nil,
	}, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodPut, "/TopProducts()", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestRouter_SerializeKeyMap(t *testing.T) {
	r := newTestRouter(nil, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	keyString := r.serializeKeyMap(map[string]string{
		"ID":   "123",
		"Name": "alpha",
	})
	parts := strings.Split(keyString, ",")
	if len(parts) != 2 {
		t.Fatalf("expected 2 key parts, got %q", keyString)
	}

	got := map[string]bool{}
	for _, part := range parts {
		got[part] = true
	}
	if !got["ID=123"] || !got["Name='alpha'"] {
		t.Fatalf("unexpected serialized keys %q", keyString)
	}
}

func TestRouter_SetAsyncMonitorPrefix(t *testing.T) {
	r := newTestRouter(nil, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	r.SetAsyncMonitor("async/jobs", nil)

	if r.asyncMonitorPrefix != "/async/jobs/" {
		t.Fatalf("expected async monitor prefix to be normalized, got %q", r.asyncMonitorPrefix)
	}
}

func TestIsValidAsyncJobID(t *testing.T) {
	cases := []struct {
		id    string
		valid bool
	}{
		{id: "abc-123_DEF", valid: true},
		{id: "", valid: false},
		{id: "has space", valid: false},
		{id: "bad*char", valid: false},
		{id: "unicode-ß", valid: false},
	}

	for _, tc := range cases {
		if got := isValidAsyncJobID(tc.id); got != tc.valid {
			t.Fatalf("isValidAsyncJobID(%q) = %v, want %v", tc.id, got, tc.valid)
		}
	}
}

func TestRouter_HeadUnboundFunction_InvokesActionInvoker(t *testing.T) {
	invoked := false
	var invokedMethod string
	invoker := func(w http.ResponseWriter, req *http.Request, name, key string, isBound bool, entitySet string) {
		invoked = true
		invokedMethod = req.Method
	}

	r := newTestRouter(nil, nil, map[string][]*actions.FunctionDefinition{
		"TopProducts": nil,
	}, invoker)

	req := httptest.NewRequest(http.MethodHead, "/TopProducts()", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if !invoked {
		t.Fatal("expected action invoker to be called for HEAD request on unbound function")
	}
	if invokedMethod != http.MethodHead {
		t.Fatalf("expected method HEAD forwarded to invoker, got %q", invokedMethod)
	}
}

func TestRouter_StreamPropertyRefRejected(t *testing.T) {
	handler := newStubEntityHandler()
	handler.streamProps["Photo"] = true
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products(1)/Photo/$ref", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// TestRouter_KeyAsSegments_Entity verifies that /Products/1 routes to HandleEntity under OData 4.01.
func TestRouter_KeyAsSegments_Entity(t *testing.T) {
	handler := newStubEntityHandler()
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products/1", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if len(handler.calls) != 1 || handler.calls[0] != "entity:1" {
		t.Fatalf("expected entity:1 call, got %v", handler.calls)
	}
}

// TestRouter_KeyAsSegments_NavigationProperty verifies that /Products/1/Category routes to
// HandleNavigationProperty when "Category" is a known navigation property.
func TestRouter_KeyAsSegments_NavigationProperty(t *testing.T) {
	handler := newStubEntityHandler()
	handler.navigationProps["Category"] = true
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products/1/Category", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if len(handler.calls) != 1 || handler.calls[0] != "navigation:1:Category:false" {
		t.Fatalf("expected navigation:1:Category:false call, got %v", handler.calls)
	}
}

// TestRouter_KeyAsSegments_StructuralProperty verifies that /Products/1/Name routes to
// HandleStructuralProperty when "Name" is a known structural property.
func TestRouter_KeyAsSegments_StructuralProperty(t *testing.T) {
	handler := newStubEntityHandler()
	handler.structuralProps["Name"] = true
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products/1/Name", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if len(handler.calls) != 1 || handler.calls[0] != "struct:1:Name:false" {
		t.Fatalf("expected struct:1:Name:false call, got %v", handler.calls)
	}
}

// TestRouter_KeyAsSegments_Ref verifies that /Products/1/$ref routes to HandleEntityRef.
func TestRouter_KeyAsSegments_Ref(t *testing.T) {
	handler := newStubEntityHandler()
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products/1/$ref", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if len(handler.calls) != 1 || handler.calls[0] != "entityref:1" {
		t.Fatalf("expected entityref:1 call, got %v", handler.calls)
	}
}

// TestRouter_KeyAsSegments_NotActive40 verifies that /Products/1 with OData-MaxVersion: 4.0
// does NOT apply key-as-segments and instead returns 404 (since "1" is not a property).
func TestRouter_KeyAsSegments_NotActive40(t *testing.T) {
	handler := newStubEntityHandler()
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products/1", nil)
	req.Header.Set("OData-MaxVersion", "4.0")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// "1" is not a known property for OData 4.0, so should return 404
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d under OData 4.0 (key-as-segments not active), got %d", http.StatusNotFound, rec.Code)
	}
}

// TestRouter_KeyAsSegments_KnownPropertyNotTreatedAsKey verifies that a segment matching a
// navigation property name is never treated as a key (even in 4.01 mode).
func TestRouter_KeyAsSegments_KnownPropertyNotTreatedAsKey(t *testing.T) {
	handler := newStubEntityHandler()
	handler.navigationProps["Details"] = true
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	// /Products/Details should be treated as collection operation (action/function or property error),
	// not as an entity with key "Details"
	req := httptest.NewRequest(http.MethodGet, "/Products/Details", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Without a key, attempting a navigation property lookup on a collection should yield an error
	if rec.Code == http.StatusOK {
		t.Fatalf("expected error status, got 200 – known property should not be treated as key")
	}
	if len(handler.calls) != 0 {
		t.Fatalf("expected no handler calls (property on collection is invalid), got %v", handler.calls)
	}
}

// TestRouter_KeyAsSegments_Singleton verifies that singletons are unaffected by key-as-segments.
func TestRouter_KeyAsSegments_Singleton(t *testing.T) {
	handler := newStubEntityHandler()
	handler.isSingleton = true
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	// /Me/1 should not treat "1" as a key for a singleton
	req := httptest.NewRequest(http.MethodGet, "/Me/1", nil)
	req.Header.Set("OData-MaxVersion", "4.01")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	// Singletons should not apply key-as-segments; "1" is not a property → 404
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for singleton with unknown segment, got %d", rec.Code)
	}
}

// TestResolveKeyAsSegment_NoRemainingSegments checks that the key is extracted
// and all property fields are cleared when no segments follow the key.
func TestResolveKeyAsSegment_NoRemainingSegments(t *testing.T) {
	from := &response.ODataURLComponents{
		EntitySet:          "Products",
		NavigationProperty: "42",
		PropertySegments:   []string{"42"},
		PropertyPath:       "42",
		EntityKeyMap:       map[string]string{},
	}
	got := resolveKeyAsSegment(from)

	if got.EntityKey != "42" {
		t.Errorf("EntityKey = %q, want %q", got.EntityKey, "42")
	}
	if got.NavigationProperty != "" {
		t.Errorf("NavigationProperty = %q, want empty", got.NavigationProperty)
	}
	if len(got.PropertySegments) != 0 {
		t.Errorf("PropertySegments = %v, want nil/empty", got.PropertySegments)
	}
}

// TestResolveKeyAsSegment_WithPropertySegment checks that segments after the key are preserved.
func TestResolveKeyAsSegment_WithPropertySegment(t *testing.T) {
	from := &response.ODataURLComponents{
		EntitySet:          "Products",
		NavigationProperty: "42",
		PropertySegments:   []string{"42", "Name"},
		PropertyPath:       "42/Name",
		EntityKeyMap:       map[string]string{},
	}
	got := resolveKeyAsSegment(from)

	if got.EntityKey != "42" {
		t.Errorf("EntityKey = %q, want %q", got.EntityKey, "42")
	}
	if got.NavigationProperty != "Name" {
		t.Errorf("NavigationProperty = %q, want %q", got.NavigationProperty, "Name")
	}
	if len(got.PropertySegments) != 1 || got.PropertySegments[0] != "Name" {
		t.Errorf("PropertySegments = %v, want [Name]", got.PropertySegments)
	}
	if got.PropertyPath != "Name" {
		t.Errorf("PropertyPath = %q, want %q", got.PropertyPath, "Name")
	}
}
