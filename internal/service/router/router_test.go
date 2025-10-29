package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/async"
)

type stubEntityHandler struct {
	isSingleton     bool
	navigationProps map[string]bool
	streamProps     map[string]bool
	structuralProps map[string]bool
	complexProps    map[string]bool
	calls           []string
}

func newStubEntityHandler() *stubEntityHandler {
	return &stubEntityHandler{
		navigationProps: make(map[string]bool),
		streamProps:     make(map[string]bool),
		structuralProps: make(map[string]bool),
		complexProps:    make(map[string]bool),
	}
}

func (h *stubEntityHandler) record(call string) {
	h.calls = append(h.calls, call)
}

func (h *stubEntityHandler) IsSingleton() bool { return h.isSingleton }

func (h *stubEntityHandler) HandleCollection(http.ResponseWriter, *http.Request) {
	h.record("collection")
}

func (h *stubEntityHandler) HandleEntity(_ http.ResponseWriter, _ *http.Request, key string) {
	h.record("entity:" + key)
}

func (h *stubEntityHandler) HandleSingleton(http.ResponseWriter, *http.Request) {
	h.record("singleton")
}

func (h *stubEntityHandler) HandleCount(http.ResponseWriter, *http.Request) {
	h.record("count")
}

func (h *stubEntityHandler) HandleNavigationPropertyCount(_ http.ResponseWriter, _ *http.Request, key, prop string) {
	h.record("navcount:" + key + ":" + prop)
}

func (h *stubEntityHandler) HandleEntityRef(_ http.ResponseWriter, _ *http.Request, key string) {
	h.record("entityref:" + key)
}

func (h *stubEntityHandler) HandleCollectionRef(http.ResponseWriter, *http.Request) {
	h.record("collectionref")
}

func (h *stubEntityHandler) HandleNavigationProperty(_ http.ResponseWriter, _ *http.Request, key, prop string, isRef bool) {
	h.record("navigation:" + key + ":" + prop + ":" + boolToString(isRef))
}

func (h *stubEntityHandler) HandleStreamProperty(_ http.ResponseWriter, _ *http.Request, key, prop string, isValue bool) {
	h.record("stream:" + key + ":" + prop + ":" + boolToString(isValue))
}

func (h *stubEntityHandler) HandleStructuralProperty(_ http.ResponseWriter, _ *http.Request, key, prop string, isValue bool) {
	h.record("struct:" + key + ":" + prop + ":" + boolToString(isValue))
}

func (h *stubEntityHandler) HandleComplexTypeProperty(_ http.ResponseWriter, _ *http.Request, key string, segments []string, isValue bool) {
	h.record("complex:" + key + ":" + segments[len(segments)-1] + ":" + boolToString(isValue))
}

func (h *stubEntityHandler) HandleMediaEntityValue(_ http.ResponseWriter, _ *http.Request, key string) {
	h.record("media:" + key)
}

func (h *stubEntityHandler) IsNavigationProperty(name string) bool {
	return h.navigationProps[name]
}

func (h *stubEntityHandler) IsStreamProperty(name string) bool {
	return h.streamProps[name]
}

func (h *stubEntityHandler) IsStructuralProperty(name string) bool {
	return h.structuralProps[name]
}

func (h *stubEntityHandler) IsComplexTypeProperty(name string) bool {
	return h.complexProps[name]
}

func (h *stubEntityHandler) FetchEntity(string) (interface{}, error) { return nil, nil }

func boolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func newTestRouter(handler EntityHandler, actionDefs map[string][]*actions.ActionDefinition, functionDefs map[string][]*actions.FunctionDefinition, invoker ActionInvoker) *Router {
	if actionDefs == nil {
		actionDefs = make(map[string][]*actions.ActionDefinition)
	}
	if functionDefs == nil {
		functionDefs = make(map[string][]*actions.FunctionDefinition)
	}
	return NewRouter(
		func(string) (EntityHandler, bool) {
			if handler == nil {
				return nil, false
			}
			return handler, true
		},
		func(http.ResponseWriter, *http.Request) {},
		func(http.ResponseWriter, *http.Request) {},
		func(http.ResponseWriter, *http.Request) {},
		actionDefs,
		functionDefs,
		invoker,
	)
}

func TestRouter_AsyncMonitorGETFlow(t *testing.T) {
	mgr := async.NewManager(0)
	t.Cleanup(mgr.Close)

	router := newTestRouter(nil, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})
	router.SetAsyncMonitor("/$async/jobs", mgr)

	release := make(chan struct{})
	handler := func(ctx context.Context) (*async.StoredResponse, error) {
		<-release
		return &async.StoredResponse{
			StatusCode: http.StatusCreated,
			Header: http.Header{
				"Content-Type":       []string{"application/json"},
				"Preference-Applied": []string{"return=representation"},
			},
			Body: []byte(`{"status":"ok"}`),
		}, nil
	}

	job, err := mgr.StartJob(context.Background(), handler)
	if err != nil {
		t.Fatalf("failed to start job: %v", err)
	}
	job.SetMonitorURL("/$async/jobs/" + job.ID)
	job.SetRetryAfter(2 * time.Second)

	pendingReq := httptest.NewRequest(http.MethodGet, job.MonitorURL(), nil)
	pendingRec := httptest.NewRecorder()
	router.ServeHTTP(pendingRec, pendingReq)

	if pendingRec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for pending job, got %d", pendingRec.Code)
	}
	if got := pendingRec.Header().Get("Preference-Applied"); got != "respond-async" {
		t.Fatalf("expected pending monitor to apply respond-async, got %q", got)
	}
	if got := pendingRec.Header().Get("Retry-After"); got != "2" {
		t.Fatalf("expected Retry-After header of 2, got %q", got)
	}
	if body := pendingRec.Body.Len(); body != 0 {
		t.Fatalf("expected no body for 202 monitor response, got %d bytes", body)
	}

	close(release)
	job.Wait()

	completeReq := httptest.NewRequest(http.MethodGet, job.MonitorURL(), nil)
	completeRec := httptest.NewRecorder()
	router.ServeHTTP(completeRec, completeReq)

	if completeRec.Code != http.StatusCreated {
		t.Fatalf("expected final status code 201, got %d", completeRec.Code)
	}
	if got := completeRec.Header().Get("Preference-Applied"); got != "return=representation" {
		t.Fatalf("expected Preference-Applied header from stored response, got %q", got)
	}
	if got := completeRec.Header().Get("Retry-After"); got != "" {
		t.Fatalf("expected Retry-After to be cleared after completion, got %q", got)
	}
	if body := completeRec.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("unexpected response body %q", body)
	}
}

func TestRouter_AsyncMonitorDELETE(t *testing.T) {
	mgr := async.NewManager(0)
	t.Cleanup(mgr.Close)

	router := newTestRouter(nil, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})
	router.SetAsyncMonitor("/$async/jobs", mgr)

	handler := func(ctx context.Context) (*async.StoredResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	job, err := mgr.StartJob(context.Background(), handler)
	if err != nil {
		t.Fatalf("failed to start job: %v", err)
	}
	job.SetMonitorURL("/$async/jobs/" + job.ID)

	deleteReq := httptest.NewRequest(http.MethodDelete, job.MonitorURL(), nil)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 from delete, got %d", deleteRec.Code)
	}
	if deleteRec.Body.Len() != 0 {
		t.Fatalf("expected no body for DELETE monitor response, got %q", deleteRec.Body.String())
	}

	job.Wait()
}

func TestRouter_AsyncMonitorInvalidPaths(t *testing.T) {
	mgr := async.NewManager(0)
	t.Cleanup(mgr.Close)

	router := newTestRouter(nil, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})
	router.SetAsyncMonitor("/$async/jobs", mgr)

	cases := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{name: "missing id", target: "/$async/jobs/", wantStatus: http.StatusNotFound},
		{name: "nested path", target: "/$async/jobs/foo/bar", wantStatus: http.StatusNotFound},
		{name: "invalid chars", target: "/$async/jobs/has*star", wantStatus: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestRouter_ServiceDocument(t *testing.T) {
	called := false
	r := NewRouter(
		func(string) (EntityHandler, bool) { return nil, false },
		func(http.ResponseWriter, *http.Request) { called = true },
		func(http.ResponseWriter, *http.Request) {},
		func(http.ResponseWriter, *http.Request) {},
		nil,
		nil,
		func(http.ResponseWriter, *http.Request, string, string, bool, string) {},
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Fatalf("expected service document handler to be called")
	}
}

func TestRouter_Metadata(t *testing.T) {
	called := false
	r := NewRouter(
		func(string) (EntityHandler, bool) { return nil, false },
		func(http.ResponseWriter, *http.Request) {},
		func(http.ResponseWriter, *http.Request) { called = true },
		func(http.ResponseWriter, *http.Request) {},
		nil,
		nil,
		func(http.ResponseWriter, *http.Request, string, string, bool, string) {},
	)

	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Fatalf("expected metadata handler to be called")
	}
}

func TestRouter_Batch(t *testing.T) {
	called := false
	r := NewRouter(
		func(string) (EntityHandler, bool) { return nil, false },
		func(http.ResponseWriter, *http.Request) {},
		func(http.ResponseWriter, *http.Request) {},
		func(http.ResponseWriter, *http.Request) { called = true },
		nil,
		nil,
		func(http.ResponseWriter, *http.Request, string, string, bool, string) {},
	)

	req := httptest.NewRequest(http.MethodPost, "/$batch", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Fatalf("expected batch handler to be called")
	}
}

func TestRouter_CollectionRequest(t *testing.T) {
	handler := newStubEntityHandler()
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if len(handler.calls) != 1 || handler.calls[0] != "collection" {
		t.Fatalf("expected collection handler to be called, got %v", handler.calls)
	}
}

func TestRouter_EntityRequest(t *testing.T) {
	handler := newStubEntityHandler()
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products(1)", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if len(handler.calls) != 1 || handler.calls[0] != "entity:1" {
		t.Fatalf("expected entity handler to be called, got %v", handler.calls)
	}
}

func TestRouter_NavigationProperty(t *testing.T) {
	handler := newStubEntityHandler()
	handler.navigationProps["Orders"] = true
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products(1)/Orders", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if len(handler.calls) != 1 || handler.calls[0] != "navigation:1:Orders:false" {
		t.Fatalf("expected navigation handler to be called, got %v", handler.calls)
	}
}

func TestRouter_CountBranches(t *testing.T) {
	handler := newStubEntityHandler()
	handler.navigationProps["Orders"] = true
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products/$count", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if handler.calls[0] != "count" {
		t.Fatalf("expected count handler to be called, got %v", handler.calls)
	}

	handler.calls = nil
	req = httptest.NewRequest(http.MethodGet, "/Products(1)/Orders/$count", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if handler.calls[0] != "navcount:1:Orders" {
		t.Fatalf("expected navigation count handler, got %v", handler.calls)
	}
}

func TestRouter_RefBranches(t *testing.T) {
	handler := newStubEntityHandler()
	handler.navigationProps["Orders"] = true
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products/$ref", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if handler.calls[0] != "collectionref" {
		t.Fatalf("expected collection $ref handler, got %v", handler.calls)
	}

	handler.calls = nil
	req = httptest.NewRequest(http.MethodGet, "/Products(1)/$ref", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if handler.calls[0] != "entityref:1" {
		t.Fatalf("expected entity $ref handler, got %v", handler.calls)
	}
}

func TestRouter_ValueBranches(t *testing.T) {
	handler := newStubEntityHandler()
	handler.structuralProps["Name"] = true
	r := newTestRouter(handler, nil, nil, func(http.ResponseWriter, *http.Request, string, string, bool, string) {})

	req := httptest.NewRequest(http.MethodGet, "/Products(1)/Name/$value", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if handler.calls[0] != "struct:1:Name:true" {
		t.Fatalf("expected structural $value handler, got %v", handler.calls)
	}

	handler.calls = nil
	req = httptest.NewRequest(http.MethodGet, "/Products/$value", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for collection $value, got %d", w.Code)
	}
}

func TestRouter_UnboundFunctionInvocation(t *testing.T) {
	called := false
	r := newTestRouter(nil,
		nil,
		map[string][]*actions.FunctionDefinition{"TopProducts": nil},
		func(_ http.ResponseWriter, _ *http.Request, name, key string, isBound bool, entitySet string) {
			called = true
			if name != "TopProducts" || key != "" || isBound || entitySet != "" {
				t.Fatalf("unexpected invocation parameters: name=%s key=%s bound=%v set=%s", name, key, isBound, entitySet)
			}
		})

	req := httptest.NewRequest(http.MethodGet, "/TopProducts()", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Fatalf("expected action/function invoker to be called")
	}
}

func TestRouter_BoundActionInvocation(t *testing.T) {
	handler := newStubEntityHandler()
	handler.navigationProps["DoThing"] = false
	invoked := false
	r := newTestRouter(handler,
		map[string][]*actions.ActionDefinition{"DoThing": nil},
		nil,
		func(_ http.ResponseWriter, _ *http.Request, name, key string, isBound bool, entitySet string) {
			invoked = true
			if name != "DoThing" || key != "1" || !isBound || entitySet != "Products" {
				t.Fatalf("unexpected invocation parameters: name=%s key=%s bound=%v set=%s", name, key, isBound, entitySet)
			}
		})

	req := httptest.NewRequest(http.MethodPost, "/Products(1)/DoThing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !invoked {
		t.Fatalf("expected bound action to invoke action handler")
	}
}

func TestRouter_FunctionCompositionAfterNavigation(t *testing.T) {
	// Set up entity handler with navigation property
	handler := newStubEntityHandler()
	handler.navigationProps["Products"] = true

	// Create a target handler for the navigated entity set
	targetHandler := newStubEntityHandler()

	invoked := false
	r := &Router{
		resolveHandler: func(entitySet string) (EntityHandler, bool) {
			if entitySet == "Categories" {
				return handler, true
			}
			if entitySet == "Products" {
				return targetHandler, true
			}
			return nil, false
		},
		handleServiceDocument: func(http.ResponseWriter, *http.Request) {},
		handleMetadata:        func(http.ResponseWriter, *http.Request) {},
		handleBatch:           func(http.ResponseWriter, *http.Request) {},
		actions:               make(map[string][]*actions.ActionDefinition),
		functions:             map[string][]*actions.FunctionDefinition{"GetAveragePrice": nil},
		actionInvoker: func(_ http.ResponseWriter, _ *http.Request, name, key string, isBound bool, entitySet string) {
			invoked = true
			if name != "GetAveragePrice" || key != "" || !isBound || entitySet != "Products" {
				t.Fatalf("unexpected invocation parameters: name=%s key=%s bound=%v set=%s", name, key, isBound, entitySet)
			}
		},
	}

	// Test function composition: Categories(1)/Products/GetAveragePrice()
	req := httptest.NewRequest(http.MethodGet, "/Categories(1)/Products/GetAveragePrice()", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !invoked {
		t.Fatalf("expected function composition to invoke function on target entity set")
	}
}
