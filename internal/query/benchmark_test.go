package query

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// getTestEntityMetadata returns test entity metadata for benchmarks
func getTestEntityMetadata() *metadata.EntityMetadata {
	return &metadata.EntityMetadata{
		EntitySetName: "Products",
		EntityName:    "Product",
		EntityType: reflect.TypeOf(struct {
			ID          string
			Name        string
			Price       float64
			Description string
			Category    string
			Rating      int
			InStock     bool
			CreatedAt   string
		}{}),
		Properties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Type: reflect.TypeOf(""), IsKey: true, IsRequired: true},
			{JsonName: "Name", FieldName: "Name", Type: reflect.TypeOf("")},
			{JsonName: "Price", FieldName: "Price", Type: reflect.TypeOf(0.0)},
			{JsonName: "Description", FieldName: "Description", Type: reflect.TypeOf("")},
			{JsonName: "Category", FieldName: "Category", Type: reflect.TypeOf("")},
			{JsonName: "Rating", FieldName: "Rating", Type: reflect.TypeOf(0)},
			{JsonName: "InStock", FieldName: "InStock", Type: reflect.TypeOf(false)},
			{JsonName: "CreatedAt", FieldName: "CreatedAt", Type: reflect.TypeOf("")},
		},
		KeyProperties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Type: reflect.TypeOf(""), IsKey: true, IsRequired: true},
		},
	}
}

// BenchmarkTokenizer_Simple benchmarks simple tokenization
func BenchmarkTokenizer_Simple(b *testing.B) {
	input := "Price gt 100"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		_, _ = t.TokenizeAll()
	}
}

// BenchmarkTokenizer_Complex benchmarks complex tokenization with functions
func BenchmarkTokenizer_Complex(b *testing.B) {
	input := "contains(Name, 'test') and Price gt 100 and Rating ge 4 or (InStock eq true and Category eq 'Electronics')"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		_, _ = t.TokenizeAll()
	}
}

// BenchmarkTokenizer_ManyTokens benchmarks tokenization with many tokens
func BenchmarkTokenizer_ManyTokens(b *testing.B) {
	input := "Price gt 100 and Price lt 500 and Rating ge 3 and Rating le 5 and Name ne 'test' and Category eq 'Electronics' and InStock eq true"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		_, _ = t.TokenizeAll()
	}
}

// BenchmarkTokenizer_DateTimeLiteral benchmarks date/time literal tokenization
func BenchmarkTokenizer_DateTimeLiteral(b *testing.B) {
	input := "CreatedAt gt 2024-01-01T00:00:00Z and CreatedAt lt 2024-12-31T23:59:59Z"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		_, _ = t.TokenizeAll()
	}
}

// BenchmarkParseQueryOptions_Simple benchmarks simple query parsing
func BenchmarkParseQueryOptions_Simple(b *testing.B) {
	entityMeta := getTestEntityMetadata()
	params := url.Values{
		"$filter": []string{"Price gt 100"},
		"$top":    []string{"10"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseQueryOptions(params, entityMeta)
	}
}

// BenchmarkParseQueryOptions_Complex benchmarks complex query parsing
func BenchmarkParseQueryOptions_Complex(b *testing.B) {
	entityMeta := getTestEntityMetadata()
	params := url.Values{
		"$filter":  []string{"contains(Name, 'test') and Price gt 100 and Rating ge 4"},
		"$select":  []string{"Name,Price,Rating,InStock"},
		"$orderby": []string{"Price desc,Rating asc"},
		"$top":     []string{"20"},
		"$skip":    []string{"10"},
		"$count":   []string{"true"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseQueryOptions(params, entityMeta)
	}
}

// BenchmarkParseQueryOptions_ManyConditions benchmarks parsing with many filter conditions
func BenchmarkParseQueryOptions_ManyConditions(b *testing.B) {
	entityMeta := getTestEntityMetadata()
	params := url.Values{
		"$filter": []string{"Price gt 100 and Price lt 500 and Rating ge 3 and Rating le 5 and Name ne 'test' and Category eq 'Electronics' and InStock eq true"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseQueryOptions(params, entityMeta)
	}
}

// BenchmarkTokenizer_String benchmarks string literal tokenization
func BenchmarkTokenizer_String(b *testing.B) {
	input := "Name eq 'This is a longer test string with some special characters: !@#$%'"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		_, _ = t.TokenizeAll()
	}
}

// BenchmarkTokenizer_GUID benchmarks GUID literal tokenization
func BenchmarkTokenizer_GUID(b *testing.B) {
	input := "ID eq 12345678-1234-1234-1234-123456789012"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		_, _ = t.TokenizeAll()
	}
}

// BenchmarkTokenizer_NextToken benchmarks individual token extraction
func BenchmarkTokenizer_NextToken(b *testing.B) {
	input := "Price gt 100 and Name eq 'test'"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		for {
			tok, err := t.NextToken()
			if err != nil || tok.Type == TokenEOF {
				break
			}
		}
	}
}

// getTestEntityMetadataWithNavigationProperties returns test entity metadata with navigation properties for benchmarks
func getTestEntityMetadataWithNavigationProperties() *metadata.EntityMetadata {
	// Define related entities first
	categoryMeta := &metadata.EntityMetadata{
		EntitySetName: "Categories",
		EntityName:    "Category",
		EntityType: reflect.TypeOf(struct {
			ID   string
			Name string
		}{}),
		Properties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Name: "ID", Type: reflect.TypeOf(""), IsKey: true, IsRequired: true, ColumnName: "id"},
			{JsonName: "Name", FieldName: "Name", Name: "Name", Type: reflect.TypeOf(""), ColumnName: "name"},
		},
		KeyProperties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Name: "ID", Type: reflect.TypeOf(""), IsKey: true, IsRequired: true, ColumnName: "id"},
		},
	}

	supplierMeta := &metadata.EntityMetadata{
		EntitySetName: "Suppliers",
		EntityName:    "Supplier",
		EntityType: reflect.TypeOf(struct {
			ID      string
			Name    string
			Country string
		}{}),
		Properties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Name: "ID", Type: reflect.TypeOf(""), IsKey: true, IsRequired: true, ColumnName: "id"},
			{JsonName: "Name", FieldName: "Name", Name: "Name", Type: reflect.TypeOf(""), ColumnName: "name"},
			{JsonName: "Country", FieldName: "Country", Name: "Country", Type: reflect.TypeOf(""), ColumnName: "country"},
		},
		KeyProperties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Name: "ID", Type: reflect.TypeOf(""), IsKey: true, IsRequired: true, ColumnName: "id"},
		},
	}

	productMeta := &metadata.EntityMetadata{
		EntitySetName: "Products",
		EntityName:    "Product",
		EntityType: reflect.TypeOf(struct {
			ID          string
			Name        string
			Price       float64
			CategoryID  string
			SupplierID  string
			Description string
		}{}),
		Properties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Name: "ID", Type: reflect.TypeOf(""), IsKey: true, IsRequired: true, ColumnName: "id"},
			{JsonName: "Name", FieldName: "Name", Name: "Name", Type: reflect.TypeOf(""), ColumnName: "name"},
			{JsonName: "Price", FieldName: "Price", Name: "Price", Type: reflect.TypeOf(0.0), ColumnName: "price"},
			{JsonName: "CategoryID", FieldName: "CategoryID", Name: "CategoryID", Type: reflect.TypeOf(""), ColumnName: "category_id"},
			{JsonName: "SupplierID", FieldName: "SupplierID", Name: "SupplierID", Type: reflect.TypeOf(""), ColumnName: "supplier_id"},
			{JsonName: "Description", FieldName: "Description", Name: "Description", Type: reflect.TypeOf(""), ColumnName: "description"},
			{JsonName: "Category", FieldName: "Category", Name: "Category", Type: reflect.TypeOf(categoryMeta), IsNavigationProp: true, NavigationIsArray: false, NavigationTarget: "Category", ColumnName: "category"},
			{JsonName: "Supplier", FieldName: "Supplier", Name: "Supplier", Type: reflect.TypeOf(supplierMeta), IsNavigationProp: true, NavigationIsArray: false, NavigationTarget: "Supplier", ColumnName: "supplier"},
		},
		KeyProperties: []metadata.PropertyMetadata{
			{JsonName: "ID", FieldName: "ID", Name: "ID", Type: reflect.TypeOf(""), IsKey: true, IsRequired: true, ColumnName: "id"},
		},
	}

	// Set up registry for navigation resolution
	registry := map[string]*metadata.EntityMetadata{
		"Category": categoryMeta,
		"Supplier": supplierMeta,
		"Product":  productMeta,
	}
	productMeta.SetEntitiesRegistry(registry)

	return productMeta
}

// BenchmarkParseQueryOptions_WithNavigationPaths benchmarks parsing with navigation property paths
func BenchmarkParseQueryOptions_WithNavigationPaths(b *testing.B) {
	entityMeta := getTestEntityMetadataWithNavigationProperties()
	params := url.Values{
		"$filter": []string{"Category/Name eq 'Electronics' and Supplier/Country eq 'USA' and Price gt 100"},
		"$select": []string{"Name,Price,Category/Name,Supplier/Name,Supplier/Country"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseQueryOptions(params, entityMeta)
	}
}

// BenchmarkParseQueryOptions_ComplexNavigationPaths benchmarks complex queries with many repeated navigation paths
func BenchmarkParseQueryOptions_ComplexNavigationPaths(b *testing.B) {
	entityMeta := getTestEntityMetadataWithNavigationProperties()
	// This filter uses the same navigation paths multiple times to test cache effectiveness
	params := url.Values{
		"$filter": []string{
			"(Category/Name eq 'Electronics' or Category/Name eq 'Computers') and " +
				"(Supplier/Country eq 'USA' or Supplier/Country eq 'Canada') and " +
				"(Price gt 100 and Price lt 1000)",
		},
		"$select":  []string{"Name,Price,Category/Name,Supplier/Name"},
		"$orderby": []string{"Category/Name,Supplier/Country,Price"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseQueryOptions(params, entityMeta)
	}
}
