package handlers

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// BenchmarkMetadataXML benchmarks XML metadata generation with caching
func BenchmarkMetadataXML(b *testing.B) {
	// Create test entity metadata
	entities := map[string]*metadata.EntityMetadata{
		"Products": {
			EntitySetName: "Products",
			EntityName:    "Product",
			EntityType:    reflect.TypeOf(struct{ ID string }{}),
			Properties: []metadata.PropertyMetadata{
				{
					JsonName:   "ID",
					FieldName:  "ID",
					Type:       reflect.TypeOf(""),
					IsKey:      true,
					IsRequired: true,
				},
				{
					JsonName:  "Name",
					FieldName: "Name",
					Type:      reflect.TypeOf(""),
				},
			},
			KeyProperties: []metadata.PropertyMetadata{
				{
					JsonName:   "ID",
					FieldName:  "ID",
					Type:       reflect.TypeOf(""),
					IsKey:      true,
					IsRequired: true,
				},
			},
		},
	}

	handler := NewMetadataHandler(entities)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/$metadata", nil)
		req.Header.Set("Accept", "application/xml")
		w := httptest.NewRecorder()
		handler.HandleMetadata(w, req)
	}
}

// BenchmarkMetadataJSON benchmarks JSON metadata generation with caching
func BenchmarkMetadataJSON(b *testing.B) {
	// Create test entity metadata
	entities := map[string]*metadata.EntityMetadata{
		"Products": {
			EntitySetName: "Products",
			EntityName:    "Product",
			EntityType:    reflect.TypeOf(struct{ ID string }{}),
			Properties: []metadata.PropertyMetadata{
				{
					JsonName:   "ID",
					FieldName:  "ID",
					Type:       reflect.TypeOf(""),
					IsKey:      true,
					IsRequired: true,
				},
				{
					JsonName:  "Name",
					FieldName: "Name",
					Type:      reflect.TypeOf(""),
				},
			},
			KeyProperties: []metadata.PropertyMetadata{
				{
					JsonName:   "ID",
					FieldName:  "ID",
					Type:       reflect.TypeOf(""),
					IsKey:      true,
					IsRequired: true,
				},
			},
		},
	}

	handler := NewMetadataHandler(entities)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/$metadata", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		handler.HandleMetadata(w, req)
	}
}

// BenchmarkMetadataXML_ConcurrentCacheHit benchmarks concurrent cache hits (measures lock contention)
func BenchmarkMetadataXML_ConcurrentCacheHit(b *testing.B) {
	entities := map[string]*metadata.EntityMetadata{
		"Products": {
			EntitySetName: "Products",
			EntityName:    "Product",
			EntityType:    reflect.TypeOf(struct{ ID string }{}),
			Properties: []metadata.PropertyMetadata{
				{
					JsonName:   "ID",
					FieldName:  "ID",
					Type:       reflect.TypeOf(""),
					IsKey:      true,
					IsRequired: true,
				},
			},
			KeyProperties: []metadata.PropertyMetadata{
				{
					JsonName:   "ID",
					FieldName:  "ID",
					Type:       reflect.TypeOf(""),
					IsKey:      true,
					IsRequired: true,
				},
			},
		},
	}

	handler := NewMetadataHandler(entities)

	// Prime the cache
	req := httptest.NewRequest("GET", "/$metadata", nil)
	req.Header.Set("Accept", "application/xml")
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	b.ResetTimer()
	b.ReportAllocs()

	// Run parallel benchmark (tests lock contention on cache reads)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/$metadata", nil)
			req.Header.Set("Accept", "application/xml")
			w := httptest.NewRecorder()
			handler.HandleMetadata(w, req)
		}
	})
}

// BenchmarkMetadataJSON_ConcurrentCacheHit benchmarks concurrent JSON cache hits
func BenchmarkMetadataJSON_ConcurrentCacheHit(b *testing.B) {
	entities := map[string]*metadata.EntityMetadata{
		"Products": {
			EntitySetName: "Products",
			EntityName:    "Product",
			EntityType:    reflect.TypeOf(struct{ ID string }{}),
			Properties: []metadata.PropertyMetadata{
				{
					JsonName:   "ID",
					FieldName:  "ID",
					Type:       reflect.TypeOf(""),
					IsKey:      true,
					IsRequired: true,
				},
			},
			KeyProperties: []metadata.PropertyMetadata{
				{
					JsonName:   "ID",
					FieldName:  "ID",
					Type:       reflect.TypeOf(""),
					IsKey:      true,
					IsRequired: true,
				},
			},
		},
	}

	handler := NewMetadataHandler(entities)

	// Prime the cache
	req := httptest.NewRequest("GET", "/$metadata", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	handler.HandleMetadata(w, req)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/$metadata", nil)
			req.Header.Set("Accept", "application/json")
			w := httptest.NewRecorder()
			handler.HandleMetadata(w, req)
		}
	})
}

// BenchmarkMetadata_WithNamespaceChanges benchmarks with concurrent namespace changes
func BenchmarkMetadata_WithNamespaceChanges(b *testing.B) {
	entities := map[string]*metadata.EntityMetadata{
		"Products": {
			EntitySetName: "Products",
			EntityName:    "Product",
			EntityType:    reflect.TypeOf(struct{ ID string }{}),
			Properties: []metadata.PropertyMetadata{
				{
					JsonName:   "ID",
					FieldName:  "ID",
					Type:       reflect.TypeOf(""),
					IsKey:      true,
					IsRequired: true,
				},
			},
			KeyProperties: []metadata.PropertyMetadata{
				{
					JsonName:   "ID",
					FieldName:  "ID",
					Type:       reflect.TypeOf(""),
					IsKey:      true,
					IsRequired: true,
				},
			},
		},
	}

	handler := NewMetadataHandler(entities)

	// Start goroutine that periodically changes namespace
	done := make(chan bool)
	go func() {
		namespaces := []string{"Namespace1", "Namespace2", "Namespace3"}
		idx := 0
		for {
			select {
			case <-done:
				return
			default:
				handler.SetNamespace(namespaces[idx%len(namespaces)])
				idx++
			}
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/$metadata", nil)
			req.Header.Set("Accept", "application/xml")
			w := httptest.NewRecorder()
			handler.HandleMetadata(w, req)
		}
	})

	close(done)
}
