package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/actions"
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
