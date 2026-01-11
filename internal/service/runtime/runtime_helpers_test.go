package runtime

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractEntitySetFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "Root path",
			path:     "/",
			expected: "",
		},
		{
			name:     "Metadata path",
			path:     "/$metadata",
			expected: "",
		},
		{
			name:     "Batch path",
			path:     "/$batch",
			expected: "",
		},
		{
			name:     "Simple entity set",
			path:     "/Products",
			expected: "Products",
		},
		{
			name:     "Entity set with key",
			path:     "/Products(1)",
			expected: "Products",
		},
		{
			name:     "Entity set with string key",
			path:     "/Customers('ALFKI')",
			expected: "Customers",
		},
		{
			name:     "Entity set with navigation",
			path:     "/Products(1)/Category",
			expected: "Products",
		},
		{
			name:     "Entity set without leading slash",
			path:     "Products",
			expected: "Products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEntitySetFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("extractEntitySetFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExtractOperationType(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		method   string
		expected string
	}{
		{
			name:     "Metadata GET",
			path:     "/$metadata",
			method:   http.MethodGet,
			expected: "metadata",
		},
		{
			name:     "Service document GET",
			path:     "/",
			method:   http.MethodGet,
			expected: "service_document",
		},
		{
			name:     "Batch POST",
			path:     "/$batch",
			method:   http.MethodPost,
			expected: "batch",
		},
		{
			name:     "Count GET",
			path:     "/Products/$count",
			method:   http.MethodGet,
			expected: "count",
		},
		{
			name:     "Ref GET",
			path:     "/Products(1)/Category/$ref",
			method:   http.MethodGet,
			expected: "ref",
		},
		{
			name:     "Entity GET",
			path:     "/Products(1)",
			method:   http.MethodGet,
			expected: "read_entity",
		},
		{
			name:     "Collection GET",
			path:     "/Products",
			method:   http.MethodGet,
			expected: "read_collection",
		},
		{
			name:     "Entity HEAD",
			path:     "/Products(1)",
			method:   http.MethodHead,
			expected: "read_entity",
		},
		{
			name:     "Collection HEAD",
			path:     "/Products",
			method:   http.MethodHead,
			expected: "read_collection",
		},
		{
			name:     "Create POST",
			path:     "/Products",
			method:   http.MethodPost,
			expected: "create",
		},
		{
			name:     "Update PATCH",
			path:     "/Products(1)",
			method:   http.MethodPatch,
			expected: "patch",
		},
		{
			name:     "Update PUT",
			path:     "/Products(1)",
			method:   http.MethodPut,
			expected: "update",
		},
		{
			name:     "Delete DELETE",
			path:     "/Products(1)",
			method:   http.MethodDelete,
			expected: "delete",
		},
		{
			name:     "Unknown method",
			path:     "/Products",
			method:   http.MethodOptions,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			result := extractOperationType(req)
			if result != tt.expected {
				t.Errorf("extractOperationType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsAsyncEligiblePath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"", false},
		{"$metadata", false},
		{"$batch", false},
		{"Products", true},
		{"Products(1)", true},
		{"Customers", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isAsyncEligiblePath(tt.path)
			if result != tt.expected {
				t.Errorf("isAsyncEligiblePath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestStatusRecorder_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

	// First WriteHeader should set status
	sr.WriteHeader(http.StatusNotFound)
	if sr.statusCode != http.StatusNotFound {
		t.Errorf("statusRecorder.statusCode = %d, want %d", sr.statusCode, http.StatusNotFound)
	}
	if !sr.written {
		t.Error("statusRecorder.written should be true after WriteHeader")
	}

	// Second WriteHeader should be ignored
	sr.WriteHeader(http.StatusOK)
	if sr.statusCode != http.StatusNotFound {
		t.Errorf("statusRecorder.statusCode should not change, got %d", sr.statusCode)
	}
}

func TestStatusRecorder_Write(t *testing.T) {
	w := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

	// Write should set written flag
	n, err := sr.Write([]byte("test"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != 4 {
		t.Errorf("Write() n = %d, want 4", n)
	}
	if !sr.written {
		t.Error("statusRecorder.written should be true after Write")
	}
}

func TestQueueToken_Release(t *testing.T) {
	// Test nil token
	var nilToken *queueToken
	nilToken.release() // Should not panic

	// Test token with nil channel
	emptyToken := &queueToken{}
	emptyToken.release() // Should not panic

	// Test token with channel
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	token := &queueToken{ch: ch}
	token.release()

	// Channel should have one less item
	select {
	case <-ch:
		t.Error("Channel should be empty after release")
	default:
		// Expected - channel is empty
	}
}

func TestCloneHeader(t *testing.T) {
	original := http.Header{
		"Content-Type": []string{"application/json"},
		"X-Custom":     []string{"value1", "value2"},
	}

	cloned := cloneHeader(original)

	// Verify cloned values match
	if cloned.Get("Content-Type") != "application/json" {
		t.Errorf("cloned Content-Type = %q, want %q", cloned.Get("Content-Type"), "application/json")
	}

	// Verify it's a deep copy
	original.Set("Content-Type", "text/plain")
	if cloned.Get("Content-Type") != "application/json" {
		t.Error("Modifying original should not affect clone")
	}

	// Verify multiple values
	if len(cloned["X-Custom"]) != 2 {
		t.Errorf("Expected 2 X-Custom values, got %d", len(cloned["X-Custom"]))
	}
}

func TestBufferRequestBody(t *testing.T) {
	t.Run("Nil body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Body = nil

		body, err := bufferRequestBody(req)
		if err != nil {
			t.Errorf("bufferRequestBody() error = %v", err)
		}
		if body != nil {
			t.Errorf("bufferRequestBody() body = %v, want nil", body)
		}
	})

	t.Run("With body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Body = http.NoBody

		body, err := bufferRequestBody(req)
		if err != nil {
			t.Errorf("bufferRequestBody() error = %v", err)
		}
		if body == nil || len(body) != 0 {
			t.Errorf("bufferRequestBody() unexpected body = %v", body)
		}
	})
}

func TestRestoreRequestBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Body = nil

	body := []byte("test body content")
	restoreRequestBody(req, body)

	if req.Body == nil {
		t.Fatal("Request body should not be nil after restore")
	}
	if req.ContentLength != int64(len(body)) {
		t.Errorf("ContentLength = %d, want %d", req.ContentLength, len(body))
	}
	if req.GetBody == nil {
		t.Error("GetBody should be set")
	}
}

func TestRuntime_New(t *testing.T) {
	rt := New(nil, nil)
	if rt == nil {
		t.Fatal("New() should return non-nil Runtime")
	}
}

func TestRuntime_SetRouter(t *testing.T) {
	rt := New(nil, nil)
	rt.SetRouter(nil) // Should not panic
}

func TestRuntime_SetLogger(t *testing.T) {
	rt := New(nil, nil)
	rt.SetLogger(nil) // Should not panic
}

func TestRuntime_SetObservability(t *testing.T) {
	rt := New(nil, nil)
	rt.SetObservability(nil) // Should not panic
}

func TestRuntime_ConfigureAsync_Disable(t *testing.T) {
	rt := New(nil, nil)

	// Configure async with nil manager should disable
	rt.ConfigureAsync(nil, nil, "", 0)

	if rt.asyncEnabled {
		t.Error("asyncEnabled should be false after ConfigureAsync(nil, ...)")
	}
	if rt.asyncManager != nil {
		t.Error("asyncManager should be nil")
	}
}

func TestRuntime_ServeHTTP_NoRouter(t *testing.T) {
	rt := New(nil, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	rt.ServeHTTP(w, req, false)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("ServeHTTP() status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
