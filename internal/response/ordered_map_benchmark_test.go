package response

import (
	"encoding/json"
	"testing"
)

// BenchmarkOrderedMapCreation benchmarks OrderedMap creation
func BenchmarkOrderedMapCreation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		om := NewOrderedMapWithCapacity(20)
		_ = om
	}
}

// BenchmarkOrderedMapPooled benchmarks pooled OrderedMap acquisition
func BenchmarkOrderedMapPooled(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		om := AcquireOrderedMapWithCapacity(20)
		om.Release()
	}
}

// BenchmarkOrderedMapSetAndMarshal benchmarks the typical usage pattern
func BenchmarkOrderedMapSetAndMarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		om := NewOrderedMapWithCapacity(15)
		// Simulate typical entity with 10 fields
		om.Set("@odata.id", "http://localhost/Products(1)")
		om.Set("ID", 1)
		om.Set("Name", "Product Name Here")
		om.Set("Price", 99.99)
		om.Set("Description", "A longer description for the product that might be a bit verbose")
		om.Set("CategoryID", 5)
		om.Set("InStock", true)
		om.Set("Quantity", int64(100))
		om.Set("Rating", 4.5)
		om.Set("CreatedAt", "2024-01-15T10:30:00Z")

		_, err := om.MarshalJSON()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOrderedMapPooledSetAndMarshal benchmarks pooled OrderedMap with typical usage
func BenchmarkOrderedMapPooledSetAndMarshal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		om := AcquireOrderedMapWithCapacity(15)
		// Simulate typical entity with 10 fields
		om.Set("@odata.id", "http://localhost/Products(1)")
		om.Set("ID", 1)
		om.Set("Name", "Product Name Here")
		om.Set("Price", 99.99)
		om.Set("Description", "A longer description for the product that might be a bit verbose")
		om.Set("CategoryID", 5)
		om.Set("InStock", true)
		om.Set("Quantity", int64(100))
		om.Set("Rating", 4.5)
		om.Set("CreatedAt", "2024-01-15T10:30:00Z")

		_, err := om.MarshalJSON()
		if err != nil {
			b.Fatal(err)
		}
		om.Release()
	}
}

// BenchmarkOrderedMapMarshalJSONSimple benchmarks JSON marshaling with simple types
func BenchmarkOrderedMapMarshalJSONSimple(b *testing.B) {
	om := NewOrderedMapWithCapacity(10)
	om.Set("ID", 1)
	om.Set("Name", "Simple Product")
	om.Set("Price", 19.99)
	om.Set("Active", true)
	om.Set("Count", int64(42))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := om.MarshalJSON()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOrderedMapMarshalJSONComplex benchmarks JSON marshaling with nested structures
func BenchmarkOrderedMapMarshalJSONComplex(b *testing.B) {
	om := NewOrderedMapWithCapacity(15)
	om.Set("@odata.context", "http://localhost/$metadata#Products")
	om.Set("@odata.id", "http://localhost/Products(1)")
	om.Set("@odata.etag", "W/\"12345\"")
	om.Set("ID", 1)
	om.Set("Name", "Complex Product with a longer name")
	om.Set("Price", 299.99)
	om.Set("Description", "A very detailed description that contains multiple sentences. It describes the product thoroughly.")
	om.Set("CategoryID", 10)
	om.Set("InStock", true)
	om.Set("Tags", []string{"electronics", "gadgets", "popular"})

	// Nested OrderedMap for Category
	category := NewOrderedMap()
	category.Set("ID", 10)
	category.Set("Name", "Electronics")
	om.Set("Category", category)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := om.MarshalJSON()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOrderedMapMarshalJSONLarge benchmarks marshaling with many fields
func BenchmarkOrderedMapMarshalJSONLarge(b *testing.B) {
	om := NewOrderedMapWithCapacity(30)
	for i := 0; i < 25; i++ {
		switch i % 5 {
		case 0:
			om.Set("field_int_"+string(rune('a'+i)), i*100)
		case 1:
			om.Set("field_str_"+string(rune('a'+i)), "value_"+string(rune('a'+i)))
		case 2:
			om.Set("field_float_"+string(rune('a'+i)), float64(i)*1.5)
		case 3:
			om.Set("field_bool_"+string(rune('a'+i)), i%2 == 0)
		case 4:
			om.Set("field_int64_"+string(rune('a'+i)), int64(i*1000))
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := om.MarshalJSON()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStdJSONMarshal compares against standard json.Marshal for reference
func BenchmarkStdJSONMarshal(b *testing.B) {
	data := map[string]interface{}{
		"ID":          1,
		"Name":        "Simple Product",
		"Price":       19.99,
		"Active":      true,
		"Count":       int64(42),
		"Description": "A longer description for the product that might be a bit verbose",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCollectionMarshal simulates marshaling a collection of entities
func BenchmarkCollectionMarshal_10Entities(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		entities := make([]*OrderedMap, 10)
		for j := 0; j < 10; j++ {
			om := AcquireOrderedMapWithCapacity(10)
			om.Set("ID", j)
			om.Set("Name", "Product Name")
			om.Set("Price", 99.99)
			om.Set("CategoryID", j%5)
			om.Set("InStock", true)
			entities[j] = om
		}

		// Marshal all
		for _, om := range entities {
			_, err := om.MarshalJSON()
			if err != nil {
				b.Fatal(err)
			}
			om.Release()
		}
	}
}

// BenchmarkCollectionMarshal_100Entities simulates marshaling a larger collection
func BenchmarkCollectionMarshal_100Entities(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		entities := make([]*OrderedMap, 100)
		for j := 0; j < 100; j++ {
			om := AcquireOrderedMapWithCapacity(10)
			om.Set("ID", j)
			om.Set("Name", "Product Name")
			om.Set("Price", 99.99)
			om.Set("CategoryID", j%5)
			om.Set("InStock", true)
			entities[j] = om
		}

		// Marshal all
		for _, om := range entities {
			_, err := om.MarshalJSON()
			if err != nil {
				b.Fatal(err)
			}
			om.Release()
		}
	}
}

// BenchmarkFormatKeyValue benchmarks the key value formatting
func BenchmarkFormatKeyValue_Int(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = formatInterfaceValue(12345)
	}
}

// BenchmarkFormatKeyValue_String benchmarks string formatting
func BenchmarkFormatKeyValue_String(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = formatInterfaceValue("test-string-value")
	}
}

// BenchmarkFormatKeyValue_Int64 benchmarks int64 formatting
func BenchmarkFormatKeyValue_Int64(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = formatInterfaceValue(int64(9223372036854775807))
	}
}
