package query

import (
	"testing"
)

// BenchmarkASTParserPooling_Simple benchmarks simple AST parsing with pooling
func BenchmarkASTParserPooling_Simple(b *testing.B) {
	input := "Price gt 100"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		tokens, _ := t.TokenizeAll()
		parser := NewASTParser(tokens)
		node, _ := parser.Parse()
		ReleaseASTNode(node)
	}
}

// BenchmarkASTParserPooling_Complex benchmarks complex AST parsing with pooling
func BenchmarkASTParserPooling_Complex(b *testing.B) {
	input := "contains(Name, 'test') and Price gt 100 and Rating ge 4 or (InStock eq true and Category eq 'Electronics')"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		tokens, _ := t.TokenizeAll()
		parser := NewASTParser(tokens)
		node, _ := parser.Parse()
		ReleaseASTNode(node)
	}
}

// BenchmarkASTParserPooling_WithoutRelease benchmarks AST parsing without releasing nodes
func BenchmarkASTParserPooling_WithoutRelease(b *testing.B) {
	input := "contains(Name, 'test') and Price gt 100 and Rating ge 4 or (InStock eq true and Category eq 'Electronics')"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		tokens, _ := t.TokenizeAll()
		parser := NewASTParser(tokens)
		_, _ = parser.Parse()
	}
}

// BenchmarkASTParserPooling_ManyLiterals benchmarks AST parsing with many literal nodes
func BenchmarkASTParserPooling_ManyLiterals(b *testing.B) {
	input := "Category in ('Electronics', 'Computers', 'Phones', 'Tablets', 'Accessories', 'Software', 'Games', 'Movies', 'Music', 'Books')"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		tokens, _ := t.TokenizeAll()
		parser := NewASTParser(tokens)
		node, _ := parser.Parse()
		ReleaseASTNode(node)
	}
}

// BenchmarkASTParserPooling_ArithmeticExpression benchmarks arithmetic expression parsing
func BenchmarkASTParserPooling_ArithmeticExpression(b *testing.B) {
	input := "Price add 10 mul 2 sub 5 div 2 mod 3 gt 100"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewTokenizer(input)
		tokens, _ := t.TokenizeAll()
		parser := NewASTParser(tokens)
		node, _ := parser.Parse()
		ReleaseASTNode(node)
	}
}

// TestASTNodePooling verifies that pooled nodes can be correctly used
func TestASTNodePooling(t *testing.T) {
	// Test that acquiring and releasing nodes works correctly
	tests := []struct {
		name  string
		input string
	}{
		{"Simple comparison", "Price gt 100"},
		{"Logical AND", "Price gt 100 and Name eq 'test'"},
		{"Logical OR", "Price gt 100 or Name eq 'test'"},
		{"Function call", "contains(Name, 'test')"},
		{"Grouped expression", "(Price gt 100 and Name eq 'test') or Rating ge 4"},
		{"IN operator", "Category in ('A', 'B', 'C')"},
		{"Complex expression", "contains(Name, 'test') and Price gt 100 and Rating ge 4 or (InStock eq true and Category eq 'Electronics')"},
		{"Arithmetic", "Price add 10 gt 100"},
		{"NOT operator", "not Active eq true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.input)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			parser := NewASTParser(tokens)
			node, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			// Release and re-acquire multiple times to test pooling
			for i := 0; i < 3; i++ {
				ReleaseASTNode(node)

				// Parse again to get new nodes from the pool
				parser2 := NewASTParser(tokens)
				node, err = parser2.Parse()
				if err != nil {
					t.Fatalf("Parsing failed on iteration %d: %v", i+1, err)
				}
			}

			// Final release
			ReleaseASTNode(node)
		})
	}
}
