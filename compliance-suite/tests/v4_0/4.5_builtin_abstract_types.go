package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// BuiltInAbstractTypes creates the 4.5 Built-In Abstract Types test suite
func BuiltInAbstractTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"4.5 Built-In Abstract Types",
		"Tests built-in abstract types (Edm.PrimitiveType, Edm.ComplexType, Edm.EntityType) and ensures they are not used where concrete types are required.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752518",
	)

	suite.AddTest(
		"test_edm_entitytype_not_as_singleton",
		"Edm.EntityType cannot be used as type of singleton",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, `Singleton`) && strings.Contains(body, `Type="Edm.EntityType"`) {
				return framework.NewError("Edm.EntityType (abstract) cannot be used as singleton type")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_edm_entitytype_not_as_entityset",
		"Edm.EntityType cannot be used as type of entity set",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, `EntitySet`) && strings.Contains(body, `EntityType="Edm.EntityType"`) {
				return framework.NewError("Edm.EntityType (abstract) cannot be used as entity set type")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_edm_complextype_not_as_basetype",
		"Edm.ComplexType cannot be used as base type",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, `BaseType="Edm.ComplexType"`) {
				return framework.NewError("Edm.ComplexType (abstract) cannot be used as base type")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_edm_entitytype_not_as_basetype",
		"Edm.EntityType cannot be used as base type",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, `BaseType="Edm.EntityType"`) {
				return framework.NewError("Edm.EntityType (abstract) cannot be used as base type")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_concrete_primitive_types_used",
		"Concrete primitive types can be used in properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Property") {
				return framework.NewError("Metadata must have properties")
			}

			hasConcretePrimitives := strings.Contains(body, `Type="Edm.String"`) ||
				strings.Contains(body, `Type="Edm.Int32"`)

			if !hasConcretePrimitives {
				return framework.NewError("Properties should use concrete primitive types")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_concrete_entity_types_in_entitysets",
		"Concrete entity types can be used in entity sets",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "EntitySet") {
				return framework.NewError("Metadata must have entity sets")
			}

			// Verify entity sets don't use abstract Edm.EntityType
			if strings.Contains(body, `EntityType="Edm.EntityType"`) {
				return framework.NewError("Entity sets must use concrete entity types")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_collection_edm_primitivetype_not_used",
		"Collection(Edm.PrimitiveType) should not be used",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, `Type="Collection(Edm.PrimitiveType)"`) {
				return framework.NewError("Collection(Edm.PrimitiveType) should use concrete types")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_navigation_uses_concrete_types",
		"Navigation properties reference concrete entity types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "NavigationProperty") {
				return nil // No navigation properties
			}

			// Verify navigation properties don't use abstract Edm.EntityType
			if strings.Contains(body, `NavigationProperty`) && strings.Contains(body, `Type="Edm.EntityType"`) {
				return framework.NewError("Navigation properties must reference concrete entity types")
			}

			return nil
		},
	)

	return suite
}
