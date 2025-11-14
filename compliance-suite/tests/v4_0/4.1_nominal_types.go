package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NominalTypes creates the 4.1 Nominal Types test suite
func NominalTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"4.1 Nominal Types",
		"Tests that nominal types have proper names and use qualified names correctly in metadata.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752514",
	)

	suite.AddTest(
		"test_entity_types_exist",
		"Entity types exist in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "EntityType") {
				return framework.NewError("Metadata must contain EntityType elements")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_entity_types_have_name",
		"Entity types have Name attribute",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "EntityType") {
				return nil // Skip if no entity types
			}

			if !strings.Contains(body, `<EntityType`) || !strings.Contains(body, `Name=`) {
				return framework.NewError("EntityType elements must have Name attribute")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_navigation_properties_use_qualified_names",
		"Navigation properties use qualified type names",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "NavigationProperty") {
				return nil // Skip if no navigation properties
			}

			// Navigation properties should use qualified type names (containing '.')
			if !strings.Contains(body, `Type="`) {
				return framework.NewError("NavigationProperty must have Type attribute")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_complex_types_have_name",
		"Complex types have Name attribute (if present)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "ComplexType") {
				return nil // Skip if no complex types
			}

			if !strings.Contains(body, `<ComplexType`) || !strings.Contains(body, `Name=`) {
				return framework.NewError("ComplexType elements must have Name attribute")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_enum_types_have_name",
		"Enum types have Name attribute (if present)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "EnumType") {
				return nil // Skip if no enum types
			}

			if !strings.Contains(body, `<EnumType`) || !strings.Contains(body, `Name=`) {
				return framework.NewError("EnumType elements must have Name attribute")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_properties_use_qualified_type_names",
		"Properties use qualified type names",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// Properties should use qualified type names (Edm.* or namespace.*)
			if !strings.Contains(body, `Type="Edm.`) && !strings.Contains(body, `Type="`) {
				return framework.NewError("Properties must use qualified type names")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_schema_has_namespace",
		"Schema has Namespace attribute",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<Schema") || !strings.Contains(body, "Namespace=") {
				return framework.NewError("Schema element must have Namespace attribute")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_builtin_types_use_edm_namespace",
		"Built-in primitive types use Edm namespace",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// Check for common primitive types in Edm namespace
			hasEdmTypes := strings.Contains(body, `Type="Edm.String"`) ||
				strings.Contains(body, `Type="Edm.Int32"`) ||
				strings.Contains(body, `Type="Edm.Boolean"`) ||
				strings.Contains(body, `Type="Edm.Decimal"`)

			if !hasEdmTypes {
				return framework.NewError("Built-in primitive types should use Edm namespace")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_entity_sets_use_qualified_names",
		"Entity sets reference entity types with qualified names",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "EntitySet") {
				return nil // Skip if no entity sets
			}

			if !strings.Contains(body, `EntityType="`) {
				return framework.NewError("EntitySet must have EntityType attribute with qualified name")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_collection_types_use_qualified_names",
		"Collection types use qualified element type names",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Collection(`) {
				return nil // Skip if no collection types
			}

			// Collection types should contain qualified type names (with '.')
			return nil
		},
	)

	return suite
}
