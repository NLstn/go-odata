package response

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"

	internalMetadata "github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
)

// These benchmarks quantify the #861 win: serializing a collection of entity
// structs via the direct writer (writeFastCollection) versus the legacy
// per-entity OrderedMap path (addNavigationLinks + envelope.marshalTo). Both
// produce identical JSON (see TestFastWriterMatchesLegacyPath); the direct writer
// eliminates the per-field interface{} boxing and hashed map inserts, so it should
// show markedly lower allocs/op and ns/op.
//
// Run:
//   go test ./internal/response/ -run x -bench 'EntityCollection' -benchmem

func benchEntities(n int) []ewEntity {
	desc := "a moderately sized product description string"
	cat := ewCat{ID: 7, Name: "Category Seven"}
	out := make([]ewEntity, n)
	for i := 0; i < n; i++ {
		out[i] = ewEntity{
			ID:          uint(i + 1),
			Name:        "Product Name Here",
			Description: &desc,
			Price:       19.99,
			Status:      1 | 4,
			Version:     i + 1,
			CreatedAt:   time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			Address:     &ewAddress{Street: "123 Some Street", City: "Metropolis"},
			CatID:       uintPtr(7),
		}
		_ = cat
	}
	return out
}

// benchSetup builds the metadata + provider used by all collection benchmarks.
func benchSetup(b *testing.B) (*internalMetadata.EntityMetadata, *ewProvider) {
	b.Helper()
	fullMD, err := internalMetadata.AnalyzeEntity(&ewEntity{})
	if err != nil {
		b.Fatalf("AnalyzeEntity: %v", err)
	}
	fullMD.EntitySetName = "EwEntities"
	return fullMD, newEwProvider(fullMD)
}

func benchFast(b *testing.B, data []ewEntity, provider *ewProvider, fullMD *internalMetadata.EntityMetadata, selectedProps []string) {
	slice, ok := canFastWriteCollection(data, fullMD)
	if !ok {
		b.Fatal("fast path not eligible")
	}
	ctx := &fastEntityContext{
		baseURL:       "http://example.com",
		entitySetName: "EwEntities",
		metadataLevel: MetadataMinimal,
		metadata:      provider,
		fullMetadata:  fullMD,
		selectedSet:   buildSelectedSet(selectedProps),
		keySet:        buildKeySet(provider),
	}
	var buf bytes.Buffer
	buf.Grow(64 * 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := writeFastCollection(&buf, slice, ctx, "http://example.com/$metadata#EwEntities", nil, nil, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func benchLegacy(b *testing.B, data interface{}, provider *ewProvider, fullMD *internalMetadata.EntityMetadata) {
	r := httptest.NewRequest("GET", "http://example.com/EwEntities", nil)
	var buf bytes.Buffer
	buf.Grow(64 * 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		transformed := addNavigationLinks(data, provider, nil, nil, r, "EwEntities", MetadataMinimal, fullMD)
		envelope := AcquireOrderedMapWithCapacity(2)
		envelope.Set("@odata.context", "http://example.com/$metadata#EwEntities")
		envelope.Set("value", transformed)
		if err := envelope.marshalTo(&buf); err != nil {
			b.Fatal(err)
		}
		envelope.Release()
		releaseOrderedMaps(transformed)
	}
}

func BenchmarkEntityCollectionFast100(b *testing.B) {
	fullMD, provider := benchSetup(b)
	benchFast(b, benchEntities(100), provider, fullMD, nil)
}

func BenchmarkEntityCollectionLegacy100(b *testing.B) {
	fullMD, provider := benchSetup(b)
	benchLegacy(b, benchEntities(100), provider, fullMD)
}

func BenchmarkEntityCollectionFast500(b *testing.B) {
	fullMD, provider := benchSetup(b)
	benchFast(b, benchEntities(500), provider, fullMD, nil)
}

func BenchmarkEntityCollectionLegacy500(b *testing.B) {
	fullMD, provider := benchSetup(b)
	benchLegacy(b, benchEntities(500), provider, fullMD)
}

// $select: the direct writer projects from the struct; the legacy path first
// materialized []map[string]interface{} via ApplySelect and serialized that.
func BenchmarkEntityCollectionSelectFast100(b *testing.B) {
	fullMD, provider := benchSetup(b)
	benchFast(b, benchEntities(100), provider, fullMD, []string{"Name", "Price"})
}

func BenchmarkEntityCollectionSelectLegacy100(b *testing.B) {
	fullMD, provider := benchSetup(b)
	data := query.ApplySelect(benchEntities(100), []string{"Name", "Price"}, fullMD, nil)
	benchLegacy(b, data, provider, fullMD)
}

// benchSetupExpand builds metadata with the navigation target registered so
// ResolveNavigationTarget works during $expand serialization.
func benchSetupExpand(b *testing.B) (*internalMetadata.EntityMetadata, *ewProvider) {
	b.Helper()
	fullMD, provider := benchSetup(b)
	catMD, err := internalMetadata.AnalyzeEntity(&ewCat{})
	if err != nil {
		b.Fatalf("AnalyzeEntity ewCat: %v", err)
	}
	catMD.EntitySetName = "EwCats"
	fullMD.SetEntitiesRegistry(map[string]*internalMetadata.EntityMetadata{
		fullMD.EntityName: fullMD,
		catMD.EntityName:  catMD,
	})
	return fullMD, provider
}

func benchEntitiesExpanded(n int) []ewEntity {
	out := benchEntities(n)
	c := ewCat{ID: 7, Name: "Category Seven"}
	for i := range out {
		out[i].Category = &c
	}
	return out
}

// $expand: the parent entity stays on the direct writer; the expanded value is
// serialized through the shared JSON path (as the legacy path also does).
func BenchmarkEntityCollectionExpandFast100(b *testing.B) {
	fullMD, provider := benchSetupExpand(b)
	data := benchEntitiesExpanded(100)
	expand := []query.ExpandOption{{NavigationProperty: "Category"}}
	slice, ok := canFastWriteCollection(data, fullMD)
	if !ok {
		b.Fatal("fast path not eligible")
	}
	ctx := &fastEntityContext{
		baseURL:       "http://example.com",
		entitySetName: "EwEntities",
		metadataLevel: MetadataMinimal,
		metadata:      provider,
		fullMetadata:  fullMD,
		expandOptions: expand,
		keySet:        buildKeySet(provider),
	}
	var buf bytes.Buffer
	buf.Grow(64 * 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := writeFastCollection(&buf, slice, ctx, "http://example.com/$metadata#EwEntities", nil, nil, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEntityCollectionExpandLegacy100(b *testing.B) {
	fullMD, provider := benchSetupExpand(b)
	data := benchEntitiesExpanded(100)
	expand := []query.ExpandOption{{NavigationProperty: "Category"}}
	r := httptest.NewRequest("GET", "http://example.com/EwEntities", nil)
	var buf bytes.Buffer
	buf.Grow(64 * 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		transformed := addNavigationLinks(data, provider, expand, nil, r, "EwEntities", MetadataMinimal, fullMD)
		envelope := AcquireOrderedMapWithCapacity(2)
		envelope.Set("@odata.context", "http://example.com/$metadata#EwEntities")
		envelope.Set("value", transformed)
		if err := envelope.marshalTo(&buf); err != nil {
			b.Fatal(err)
		}
		envelope.Release()
		releaseOrderedMaps(transformed)
	}
}
