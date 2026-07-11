package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

// openTypeTestEntity implements IsOpenType() returning true
type openTypeTestEntity struct {
	ID   int    `json:"ID" odata:"key"`
	Name string `json:"Name"`
}

func (openTypeTestEntity) IsOpenType() bool {
	return true
}

func getOpenTypeTestMetadata(t *testing.T) *metadata.EntityMetadata {
	t.Helper()
	meta, err := metadata.AnalyzeEntity(openTypeTestEntity{})
	if err != nil {
		t.Fatalf("Failed to analyze openTypeTestEntity: %v", err)
	}
	return meta
}

// TestOpenType_FilterAllowsDynamicProperty verifies that filtering on an undeclared property
// is allowed for open types (where IsOpenType() returns true).
func TestOpenType_FilterAllowsDynamicProperty(t *testing.T) {
	meta := getOpenTypeTestMetadata(t)

	// Filter references a property "DynamicProp" that is not declared on the entity.
	// For an open type this must succeed.
	filter := "DynamicProp eq 'hello'"
	tokenizer := NewTokenizer(filter)
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	parser := NewASTParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parsing failed: %v", err)
	}
	defer ReleaseASTNode(ast)

	_, err = ASTToFilterExpression(ast, meta)
	if err != nil {
		t.Errorf("Expected no error for dynamic property on open type, got: %v", err)
	}
}

// TestClosedType_FilterRejectsDynamicProperty verifies that filtering on an undeclared property
// returns an error for a closed (non-open) entity type.
func TestClosedType_FilterRejectsDynamicProperty(t *testing.T) {
	meta := getTestMetadata(t)

	// Filter references "DynamicProp" which does not exist on the closed TestEntity.
	filter := "DynamicProp eq 'hello'"
	tokenizer := NewTokenizer(filter)
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	parser := NewASTParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parsing failed: %v", err)
	}
	defer ReleaseASTNode(ast)

	_, err = ASTToFilterExpression(ast, meta)
	if err == nil {
		t.Error("Expected error for unknown property on closed type, got nil")
	}
}

// TestOpenType_SelectAllowsDynamicProperty verifies that $select with an undeclared property
// is allowed for open types.
func TestOpenType_SelectAllowsDynamicProperty(t *testing.T) {
	meta := getOpenTypeTestMetadata(t)

	err := validateExpandSelect([]string{"DynamicProp"}, meta, nil)
	if err != nil {
		t.Errorf("Expected no error for dynamic property in $select on open type, got: %v", err)
	}
}

// TestClosedType_SelectRejectsDynamicProperty verifies that $select with an undeclared property
// returns an error for a closed entity type.
func TestClosedType_SelectRejectsDynamicProperty(t *testing.T) {
	meta := getTestMetadata(t)

	err := validateExpandSelect([]string{"DynamicProp"}, meta, nil)
	if err == nil {
		t.Error("Expected error for unknown property in $select on closed type, got nil")
	}
}
