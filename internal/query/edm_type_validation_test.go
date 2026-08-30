package query

import (
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

func TestValidateValueAgainstPropertyTypeWithoutGoType(t *testing.T) {
	t.Parallel()

	entityMetadata := &metadata.EntityMetadata{
		Properties: []metadata.PropertyMetadata{
			{Name: "Name", JsonName: "Name", EdmType: metadata.PrimitiveTypeString},
			{Name: "Amount", JsonName: "Amount", EdmType: metadata.PrimitiveTypeDecimal},
		},
	}

	if err := validateValueAgainstPropertyType("Amount", float64(42), "number", entityMetadata); err != nil {
		t.Fatalf("numeric value for Edm.Decimal returned error: %v", err)
	}

	err := validateValueAgainstPropertyType("Name", float64(42), "number", entityMetadata)
	if err == nil || !strings.Contains(err.Error(), "string property") {
		t.Fatalf("numeric value for Edm.String error = %v, want type mismatch", err)
	}

	err = validateValueAgainstPropertyType("Amount", "forty-two", "string", entityMetadata)
	if err == nil || !strings.Contains(err.Error(), "numeric property") {
		t.Fatalf("string value for Edm.Decimal error = %v, want type mismatch", err)
	}
}
