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
