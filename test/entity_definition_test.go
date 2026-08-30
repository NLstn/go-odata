package odata_test

import (
	"testing"

	odata "github.com/nlstn/go-odata"
)

func TestPublicEntityDefinition(t *testing.T) {
	t.Parallel()

	definition := odata.EntityDefinition{
		Name:          "Product",
		EntitySetName: "Products",
		Properties: []odata.PropertyDefinition{
			{Name: "ID", Type: odata.EdmInt64},
			{Name: "Name", Type: odata.EdmString},
		},
		Keys: []string{"ID"},
	}

	if err := definition.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
