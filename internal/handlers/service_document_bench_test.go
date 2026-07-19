package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// BenchmarkServiceDocument measures the per-request cost of serving the service
// document. It uses only the public handler API so it compiles against both the
// cached and uncached implementations for before/after comparison.
func BenchmarkServiceDocument(b *testing.B) {
	meta1, _ := metadata.AnalyzeEntity(ServiceDocumentTestEntity{})
	meta2, _ := metadata.AnalyzeEntity(TestEntity{})
	h := NewServiceDocumentHandler(map[string]*metadata.EntityMetadata{
		"ServiceDocumentTestEntities": meta1,
		"TestEntities":                meta2,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.HandleServiceDocument(w, req)
	}
}
