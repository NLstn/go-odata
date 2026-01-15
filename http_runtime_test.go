package odata

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/nlstn/go-odata/internal/actions"
	servrouter "github.com/nlstn/go-odata/internal/service/router"
	servruntime "github.com/nlstn/go-odata/internal/service/runtime"
)

type pathRecorder struct {
	mu    sync.Mutex
	paths []string
}

func (r *pathRecorder) record(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paths = append(r.paths, req.URL.Path)
}

func (r *pathRecorder) last() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.paths) == 0 {
		return ""
	}
	return r.paths[len(r.paths)-1]
}

func (r *pathRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.paths)
}

type mockEntityHandler struct {
	record func(*http.Request)
}

func (h *mockEntityHandler) IsSingleton() bool {
	return false
}

func (h *mockEntityHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleEntity(w http.ResponseWriter, r *http.Request, _ string) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleSingleton(w http.ResponseWriter, r *http.Request) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleCount(w http.ResponseWriter, r *http.Request) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleNavigationPropertyCount(w http.ResponseWriter, r *http.Request, _, _ string) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleEntityRef(w http.ResponseWriter, r *http.Request, _ string) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleCollectionRef(w http.ResponseWriter, r *http.Request) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleNavigationProperty(w http.ResponseWriter, r *http.Request, _, _ string, _ bool) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleStreamProperty(w http.ResponseWriter, r *http.Request, _, _ string, _ bool) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleStructuralProperty(w http.ResponseWriter, r *http.Request, _, _ string, _ bool) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleComplexTypeProperty(w http.ResponseWriter, r *http.Request, _ string, _ []string, _ bool) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) HandleMediaEntityValue(w http.ResponseWriter, r *http.Request, _ string) {
	h.record(r)
	w.WriteHeader(http.StatusOK)
}

func (h *mockEntityHandler) IsNavigationProperty(_ string) bool {
	return false
}

func (h *mockEntityHandler) IsStreamProperty(_ string) bool {
	return false
}

func (h *mockEntityHandler) IsStructuralProperty(_ string) bool {
	return false
}

func (h *mockEntityHandler) IsComplexTypeProperty(_ string) bool {
	return false
}

func (h *mockEntityHandler) NavigationTargetSet(_ string) (string, bool) {
	return "", false
}

func (h *mockEntityHandler) FetchEntity(_ string) (interface{}, error) {
	return nil, nil
}

func newRuntimeService(recorder *pathRecorder) *Service {
	handler := &mockEntityHandler{record: recorder.record}
	resolver := func(name string) (servrouter.EntityHandler, bool) {
		switch name {
		case "Products", "odatax":
			return handler, true
		default:
			return nil, false
		}
	}
	serviceDocHandler := func(w http.ResponseWriter, r *http.Request) {
		recorder.record(r)
		w.WriteHeader(http.StatusOK)
	}
	noopHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	actionInvoker := func(w http.ResponseWriter, r *http.Request, _ string, _ string, _ bool, _ string) {
		recorder.record(r)
		w.WriteHeader(http.StatusOK)
	}
	actionsMap := map[string][]*actions.ActionDefinition{
		"Products": {
			{Name: "Products"},
		},
	}
	router := servrouter.NewRouter(
		resolver,
		serviceDocHandler,
		noopHandler,
		noopHandler,
		actionsMap,
		map[string][]*actions.FunctionDefinition{},
		actionInvoker,
		nil,
	)
	runtime := servruntime.New(router, nil)
	return &Service{
		logger:  slog.Default(),
		runtime: runtime,
	}
}

func TestServiceServeHTTP_BasePathRuntimeNormalization(t *testing.T) {
	recorder := &pathRecorder{}
	service := newRuntimeService(recorder)
	if err := service.SetBasePath("/odata"); err != nil {
		t.Fatalf("SetBasePath: %v", err)
	}

	tests := []struct {
		name        string
		requestPath string
		wantPath    string
	}{
		{
			name:        "exact base path",
			requestPath: "/odata",
			wantPath:    "/",
		},
		{
			name:        "base path prefix stripped",
			requestPath: "/odata/Products",
			wantPath:    "/Products",
		},
		{
			name:        "base path prefix not stripped",
			requestPath: "/odatax/Products",
			wantPath:    "/odatax/Products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder.mu.Lock()
			recorder.paths = nil
			recorder.mu.Unlock()

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			w := httptest.NewRecorder()

			service.ServeHTTP(w, req)

			if got := recorder.last(); got != tt.wantPath {
				t.Fatalf("runtime saw path %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestServiceServeHTTP_PreRequestHookErrorSkipsRuntime(t *testing.T) {
	recorder := &pathRecorder{}
	service := newRuntimeService(recorder)

	called := false
	service.preRequestHook = func(_ *http.Request) (context.Context, error) {
		called = true
		return nil, errors.New("blocked")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if !called {
		t.Fatal("expected preRequestHook to be called")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	if recorder.count() != 0 {
		t.Fatalf("expected runtime not to be called")
	}
}
