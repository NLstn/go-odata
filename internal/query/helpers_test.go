package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

type helperTestAddress struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

type helperTestDimensions struct {
	Length float64 `json:"Length"`
}

type helperTestProduct struct {
	ID              int                   `json:"ID" odata:"key"`
	ShippingAddress *helperTestAddress    `json:"shippingAddress,omitempty" gorm:"embedded;embeddedPrefix:shipping_" odata:"nullable"`
	Dimensions      *helperTestDimensions `json:"Dimensions,omitempty" gorm:"embedded;embeddedPrefix:dim_" odata:"nullable"`
}

func getHelperTestMetadata(t *testing.T) *metadata.EntityMetadata {
	t.Helper()
	meta, err := metadata.AnalyzeEntity(helperTestProduct{})
	if err != nil {
		t.Fatalf("AnalyzeEntity returned error: %v", err)
	}
	return meta
}

func TestGetColumnNameComplexPath(t *testing.T) {
	meta := getHelperTestMetadata(t)

	if got := GetColumnName("ShippingAddress/City", meta); got != "shipping_city" {
		t.Fatalf("expected shipping_city, got %s", got)
	}

	if got := GetColumnName("shippingAddress/city", meta); got != "shipping_city" {
		t.Fatalf("expected shipping_city for json path, got %s", got)
	}

	if got := GetColumnName("Dimensions/Length", meta); got != "dim_length" {
		t.Fatalf("expected dim_length, got %s", got)
	}
}

func TestPropertyExistsWithComplexPath(t *testing.T) {
	meta := getHelperTestMetadata(t)

	if !propertyExists("ShippingAddress/City", meta) {
		t.Fatal("expected propertyExists to return true for ShippingAddress/City")
	}

	if !propertyExists("shippingAddress/city", meta) {
		t.Fatal("expected propertyExists to return true for json path")
	}

	if propertyExists("ShippingAddress/Unknown", meta) {
		t.Fatal("expected propertyExists to return false for unknown nested property")
	}
}

func TestBuildComparisonConditionComplexPath(t *testing.T) {
	meta := getHelperTestMetadata(t)

	filter := &FilterExpression{
		Property: "ShippingAddress/City",
		Operator: OpEqual,
		Value:    "Seattle",
	}

	query, args := buildComparisonCondition("sqlite", filter, meta)

	if query != "shipping_city = ?" {
		t.Fatalf("expected SQL to use shipping_city column, got %s", query)
	}
	if len(args) != 1 || args[0] != "Seattle" {
		t.Fatalf("expected args to contain 'Seattle', got %#v", args)
	}
}

func TestGetPropertyFieldName(t *testing.T) {
	meta := getHelperTestMetadata(t)

	if got := GetPropertyFieldName("ID", meta); got != "ID" {
		t.Errorf("expected 'ID', got %s", got)
	}

	if got := GetPropertyFieldName("shippingAddress", meta); got != "ShippingAddress" {
		t.Errorf("expected 'ShippingAddress', got %s", got)
	}

	if got := GetPropertyFieldName("nonExistent", meta); got != "nonExistent" {
		t.Errorf("expected 'nonExistent', got %s", got)
	}
}

func TestMergeFilterExpressions(t *testing.T) {
	left := &FilterExpression{
		Property: "name",
		Operator: OpEqual,
		Value:    "John",
	}
	right := &FilterExpression{
		Property: "age",
		Operator: OpGreaterThan,
		Value:    18,
	}

	t.Run("both non-nil", func(t *testing.T) {
		result := MergeFilterExpressions(left, right)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Logical != LogicalAnd {
			t.Errorf("expected LogicalAnd, got %v", result.Logical)
		}
		if result.Left != left {
			t.Error("expected left to be preserved")
		}
		if result.Right != right {
			t.Error("expected right to be preserved")
		}
	})

	t.Run("left nil", func(t *testing.T) {
		result := MergeFilterExpressions(nil, right)
		if result != right {
			t.Error("expected right to be returned")
		}
	})

	t.Run("right nil", func(t *testing.T) {
		result := MergeFilterExpressions(left, nil)
		if result != left {
			t.Error("expected left to be returned")
		}
	})

	t.Run("both nil", func(t *testing.T) {
		result := MergeFilterExpressions(nil, nil)
		if result != nil {
			t.Error("expected nil result")
		}
	})
}

func TestParseFilterExpression(t *testing.T) {
	meta := getHelperTestMetadata(t)

	t.Run("valid filter", func(t *testing.T) {
		result, err := ParseFilterExpression("ID eq 1", meta)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("invalid filter", func(t *testing.T) {
		_, err := ParseFilterExpression("invalid filter syntax", meta)
		if err == nil {
			t.Error("expected error for invalid filter")
		}
	})
}

func TestParseFilterExpressionWithConfig(t *testing.T) {
	meta := getHelperTestMetadata(t)

	t.Run("valid filter with custom limit", func(t *testing.T) {
		result, err := ParseFilterExpressionWithConfig("ID eq 1", meta, 500)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})
}
