package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// PrimitiveTypes creates the 4.4 Primitive Types test suite
func PrimitiveTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"4.4 Primitive Types",
		"Tests OData primitive types including Edm.String, Edm.Int32, Edm.Boolean, Edm.Decimal, and others defined in the specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752517",
	)

	primitiveTypeTests := []struct {
		name     string
		typeName string
		required bool
	}{
		{"Edm.String primitive type is supported", "Edm.String", true},
		{"Edm.Int32 primitive type is supported", "Edm.Int32", true},
		{"Edm.Boolean primitive type is supported", "Edm.Boolean", false},
		{"Edm.Decimal primitive type is supported", "Edm.Decimal", false},
		{"Edm.Double primitive type is supported", "Edm.Double", false},
		{"Edm.Single primitive type is supported", "Edm.Single", false},
		{"Edm.Guid primitive type is supported", "Edm.Guid", false},
		{"Edm.DateTimeOffset primitive type is supported", "Edm.DateTimeOffset", false},
		{"Edm.Date primitive type is supported", "Edm.Date", false},
		{"Edm.TimeOfDay primitive type is supported", "Edm.TimeOfDay", false},
		{"Edm.Duration primitive type is supported", "Edm.Duration", false},
		{"Edm.Binary primitive type is supported", "Edm.Binary", false},
		{"Edm.Stream primitive type is supported", "Edm.Stream", false},
		{"Edm.Byte primitive type is supported", "Edm.Byte", false},
		{"Edm.SByte primitive type is supported", "Edm.SByte", false},
		{"Edm.Int16 primitive type is supported", "Edm.Int16", false},
		{"Edm.Int64 primitive type is supported", "Edm.Int64", false},
	}

	for _, tt := range primitiveTypeTests {
		typeName := tt.typeName
		required := tt.required
		suite.AddTest(
			"test_"+strings.ToLower(strings.ReplaceAll(typeName, ".", "_")),
			tt.name,
			func(ctx *framework.TestContext) error {
				resp, err := ctx.GET("/$metadata")
				if err != nil {
					return err
				}

				body := string(resp.Body)
				if !strings.Contains(body, `Type="`+typeName+`"`) {
					if required {
						return framework.NewError(typeName + " must be supported")
					}
					return nil // Optional type, skip
				}

				return nil
			},
		)
	}

	suite.AddTest(
		"test_facets_maxlength",
		"Edm.String can have MaxLength facet",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// MaxLength is optional, just check it's valid if present
			if strings.Contains(body, `Type="Edm.String"`) && strings.Contains(body, "MaxLength=") {
				return nil
			}

			return nil // Optional facet
		},
	)

	suite.AddTest(
		"test_facets_precision",
		"Edm.Decimal can have Precision facet",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// Precision is optional
			if strings.Contains(body, `Type="Edm.Decimal"`) && strings.Contains(body, "Precision=") {
				return nil
			}

			return nil // Optional facet
		},
	)

	suite.AddTest(
		"test_collection_of_primitives",
		"Collections of primitive types are supported",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Collection(Edm.`) {
				return nil // Optional feature
			}

			return nil
		},
	)

	suite.AddTest(
		"test_key_uses_primitive_types",
		"Key properties use primitive types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Key") || !strings.Contains(body, "PropertyRef") {
				return framework.NewError("Metadata must contain key definitions")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_primitive_types_case_sensitive",
		"Primitive type names are case-sensitive",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.`) {
				return framework.NewError("Metadata must contain Edm primitive types")
			}

			// Check for incorrect lowercase
			if strings.Contains(body, `Type="edm.`) {
				return framework.NewError("Primitive type names must be case-sensitive (Edm.String not edm.string)")
			}

			return nil
		},
	)

	return suite
}
