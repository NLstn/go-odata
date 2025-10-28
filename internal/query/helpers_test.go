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

	query, args := buildComparisonCondition(filter, meta)

	if query != "shipping_city = ?" {
		t.Fatalf("expected SQL to use shipping_city column, got %s", query)
	}
	if len(args) != 1 || args[0] != "Seattle" {
		t.Fatalf("expected args to contain 'Seattle', got %#v", args)
	}
}
