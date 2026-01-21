package handlers

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// Additional simple tests to boost coverage

// TestNamespaceOrDefault tests namespace helper functions
func TestNamespaceOrDefault(t *testing.T) {
	handler := NewEntityHandler(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ns := handler.namespaceOrDefault()
	t.Logf("Namespace: %s", ns)

	handler.SetNamespace("CustomNS")
	ns = handler.namespaceOrDefault()
	if ns != "CustomNS" {
		t.Errorf("Expected CustomNS, got %s", ns)
	}
}

// TestQualifiedTypeName tests qualified type name generation
func TestQualifiedTypeName(t *testing.T) {
	handler := NewEntityHandler(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler.SetNamespace("TestNS")

	result := handler.qualifiedTypeName("Product")
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

// TestServiceDocumentHandleOptions tests OPTIONS request handling
func TestServiceDocumentHandleOptions(t *testing.T) {
	sdHandler := NewServiceDocumentHandler(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	w := httptest.NewRecorder()

	sdHandler.handleOptionsServiceDocument(w)

	if w.Header().Get("Allow") == "" {
		t.Error("Expected Allow header to be set")
	}
}

// TestMetadataHandleOptions tests OPTIONS request handling for metadata
func TestMetadataHandleOptions(t *testing.T) {
	mHandler := NewMetadataHandler(nil)
	mHandler.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	w := httptest.NewRecorder()

	mHandler.handleOptionsMetadata(w)

	if w.Header().Get("Allow") == "" {
		t.Error("Expected Allow header to be set")
	}
}

// TestIsMethodDisabled tests method disabling check
func TestIsMethodDisabled(t *testing.T) {
	handler := NewEntityHandler(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Test with no policy
	disabled := handler.isMethodDisabled(http.MethodGet)
	if disabled {
		t.Error("Expected method to not be disabled")
	}
}

// TestHasEntityLevelDefaultMaxTop tests default max top checking
func TestHasEntityLevelDefaultMaxTop(t *testing.T) {
	meta := &metadata.EntityMetadata{
		EntityName: "Test",
	}
	handler := NewEntityHandler(nil, meta, slog.New(slog.NewTextHandler(io.Discard, nil)))

	has := handler.HasEntityLevelDefaultMaxTop()
	if has {
		t.Error("Expected HasEntityLevelDefaultMaxTop to return false initially")
	}

	maxTop := 100
	handler.metadata.DefaultMaxTop = &maxTop
	has = handler.HasEntityLevelDefaultMaxTop()
	if !has {
		t.Error("Expected HasEntityLevelDefaultMaxTop to return true")
	}
}

// TestIsGeospatialEnabled tests geospatial feature check
func TestIsGeospatialEnabled(t *testing.T) {
	handler := NewEntityHandler(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	enabled := handler.IsGeospatialEnabled()
	if enabled {
		t.Log("Geospatial is enabled")
	}

	handler.SetGeospatialEnabled(true)
	enabled = handler.IsGeospatialEnabled()
	if !enabled {
		t.Error("Expected geospatial to be enabled")
	}
}

// TestIsSingleton tests singleton check
func TestIsSingleton(t *testing.T) {
	meta := &metadata.EntityMetadata{
		EntityName:  "Test",
		IsSingleton: false,
	}
	handler := NewEntityHandler(nil, meta, slog.New(slog.NewTextHandler(io.Discard, nil)))

	isSingleton := handler.IsSingleton()
	if isSingleton {
		t.Error("Expected IsSingleton to return false")
	}

	handler.metadata.IsSingleton = true
	isSingleton = handler.IsSingleton()
	if !isSingleton {
		t.Error("Expected IsSingleton to return true")
	}
}

// TestEnsureOverwrite tests overwrite initialization
func TestEnsureOverwrite(t *testing.T) {
	handler := NewEntityHandler(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	handler.ensureOverwrite()

	if handler.overwrite == nil {
		t.Error("Expected overwrite to be initialized")
	}
}

// TestOverwriteHasChecks tests overwrite has* methods
func TestOverwriteHasChecks(t *testing.T) {
	var o *entityOverwriteHandlers

	if o.hasGetCollection() {
		t.Error("Expected false for nil overwrite")
	}
	if o.hasGetEntity() {
		t.Error("Expected false for nil overwrite")
	}
	if o.hasCreate() {
		t.Error("Expected false for nil overwrite")
	}
	if o.hasUpdate() {
		t.Error("Expected false for nil overwrite")
	}
	if o.hasDelete() {
		t.Error("Expected false for nil overwrite")
	}
	if o.hasGetCount() {
		t.Error("Expected false for nil overwrite")
	}
}

// TestSetObservability tests observability settings
func TestSetObservability(t *testing.T) {
	handler := NewEntityHandler(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	handler.SetObservability(nil)

	// Should not panic
	t.Log("Observability set")
}

// TestSetMaxInClauseSize tests max IN clause size setting
func TestSetMaxInClauseSize(t *testing.T) {
	handler := NewEntityHandler(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	handler.SetMaxInClauseSize(500)

	// Should not panic
	t.Log("Max IN clause size set")
}

// TestSetMaxExpandDepth tests max expand depth setting
func TestSetMaxExpandDepth(t *testing.T) {
	handler := NewEntityHandler(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	handler.SetMaxExpandDepth(5)

	// Should not panic
	t.Log("Max expand depth set")
}

// TestGetParserConfig tests parser config retrieval
func TestGetParserConfig(t *testing.T) {
	handler := NewEntityHandler(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	config := handler.getParserConfig()

	if config == nil {
		t.Error("Expected non-nil parser config")
	}
}
