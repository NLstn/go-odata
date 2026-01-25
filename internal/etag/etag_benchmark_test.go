package etag

import (
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
)

type benchmarkEntity struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int64     `json:"version"`
}

func BenchmarkETagGenerate_IntField(b *testing.B) {
	entity := &benchmarkEntity{
		ID:        12345,
		Name:      "Test Entity",
		UpdatedAt: time.Now(),
		Version:   100,
	}
	meta := &metadata.EntityMetadata{
		ETagProperty: &metadata.PropertyMetadata{
			FieldName: "Version",
			JsonName:  "version",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Generate(entity, meta)
	}
}

func BenchmarkETagGenerate_StringField(b *testing.B) {
	entity := &benchmarkEntity{
		ID:        12345,
		Name:      "Test Entity With Longer Name",
		UpdatedAt: time.Now(),
		Version:   100,
	}
	meta := &metadata.EntityMetadata{
		ETagProperty: &metadata.PropertyMetadata{
			FieldName: "Name",
			JsonName:  "name",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Generate(entity, meta)
	}
}

func BenchmarkETagGenerate_TimeField(b *testing.B) {
	entity := &benchmarkEntity{
		ID:        12345,
		Name:      "Test Entity",
		UpdatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Version:   100,
	}
	meta := &metadata.EntityMetadata{
		ETagProperty: &metadata.PropertyMetadata{
			FieldName: "UpdatedAt",
			JsonName:  "updated_at",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Generate(entity, meta)
	}
}

func BenchmarkETagGenerate_MapEntity(b *testing.B) {
	entity := map[string]interface{}{
		"id":         12345,
		"name":       "Test Entity",
		"updated_at": time.Now(),
		"version":    int64(100),
	}
	meta := &metadata.EntityMetadata{
		ETagProperty: &metadata.PropertyMetadata{
			FieldName: "version",
			JsonName:  "version",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = Generate(entity, meta)
	}
}

func BenchmarkETagGenerate_Parallel(b *testing.B) {
	entity := &benchmarkEntity{
		ID:        12345,
		Name:      "Test Entity",
		UpdatedAt: time.Now(),
		Version:   100,
	}
	meta := &metadata.EntityMetadata{
		ETagProperty: &metadata.PropertyMetadata{
			FieldName: "Version",
			JsonName:  "version",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = Generate(entity, meta)
		}
	})
}
