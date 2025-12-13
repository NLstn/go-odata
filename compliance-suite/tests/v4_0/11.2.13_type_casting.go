package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

var (
	derivedTypesChecked bool
	derivedTypesPresent bool
)

// TypeCasting creates the 11.2.13 Type Casting and Type Inheritance test suite
func TypeCasting() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.13 Type Casting and Type Inheritance",
		"Tests derived types, type casting in URLs, and polymorphic queries according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingDerivedTypes",
	)

	// Note: Type inheritance and casting are advanced OData features
	// Many implementations may not support them initially

	// Test 1: Filter by type using isof function
	suite.AddTest(
		"test_isof_function",
		"Filter by type using isof function",
		func(ctx *framework.TestContext) error {
			if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
				return err
			}
			resp, err := ctx.GET("/Products?$filter=isof('Namespace.SpecialProduct')")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("type casting failed but derived types exist in metadata (status: %d)", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 2: Type cast in URL path
	suite.AddTest(
		"test_type_cast_in_path",
		"Type cast in URL path",
		func(ctx *framework.TestContext) error {
			if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
				return err
			}
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Namespace.SpecialProduct")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return fmt.Errorf("type casting failed but derived types exist in metadata (status: %d)", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 3: Type cast in collection
	suite.AddTest(
		"test_type_cast_collection",
		"Type cast on collection",
		func(ctx *framework.TestContext) error {
			if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
				return err
			}
			resp, err := ctx.GET("/Products/Namespace.SpecialProduct")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return fmt.Errorf("type casting failed but derived types exist in metadata (status: %d)", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 4: Cast function in filter
	suite.AddTest(
		"test_cast_function",
		"Cast function in filter",
		func(ctx *framework.TestContext) error {
			if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
				return err
			}
			resp, err := ctx.GET("/Products?$filter=cast(ID,'Edm.String') eq '1'")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 400 || resp.StatusCode == 501 {
				return fmt.Errorf("type casting failed but derived types exist in metadata (status: %d)", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 5: Access derived type property
	suite.AddTest(
		"test_derived_property_access",
		"Access derived type property",
		func(ctx *framework.TestContext) error {
			if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
				return err
			}
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Namespace.SpecialProduct/SpecialProperty")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return fmt.Errorf("type casting failed but derived types exist in metadata (status: %d)", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 6: Filter with isof and property condition
	suite.AddTest(
		"test_isof_with_filter",
		"Filter with isof and other conditions",
		func(ctx *framework.TestContext) error {
			if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
				return err
			}
			resp, err := ctx.GET("/Products?$filter=isof('Namespace.SpecialProduct') and Price gt 100")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("type casting failed but derived types exist in metadata (status: %d)", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 7: Polymorphic query returns base and derived types
	suite.AddTest(
		"test_polymorphic_query",
		"Polymorphic query returns all types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("polymorphic entity set access failed (status: %d)", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 8: Type information in response (@odata.type)
	suite.AddTest(
		"test_type_annotation",
		"Type information in response",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				// Check for @odata.type annotation (optional in minimal metadata)
				bodyStr := string(resp.Body)
				if strings.Contains(bodyStr, "@odata.type") {
					return nil
				}
				// Pass - optional in minimal metadata
				ctx.Log("@odata.type not present (acceptable for minimal metadata)")
				return nil
			}

			if resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("entity retrieval failed (status: %d)", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 9: Derived types in metadata
	suite.AddTest(
		"test_derived_in_metadata",
		"Derived types in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				bodyStr := string(resp.Body)
				if strings.Contains(bodyStr, "BaseType") {
					return nil
				}
				// Pass - optional
				ctx.Log("No derived types in metadata (optional feature)")
				return nil
			}

			if resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("metadata retrieval failed (status: %d)", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 10: Create entity with derived type
	suite.AddTest(
		"test_create_derived_type",
		"Create entity with derived type",
		func(ctx *framework.TestContext) error {
			if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
				return err
			}
			resp, err := ctx.POST("/Products", map[string]interface{}{
				"@odata.type": "Namespace.SpecialProduct",
				"Name":        "Test",
				"Price":       100,
			})
			if err != nil {
				return err
			}

			if resp.StatusCode == 201 || resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 400 || resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("type casting failed but derived types exist in metadata (status: %d)", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 11: Type cast with navigation property
	suite.AddTest(
		"test_type_cast_navigation",
		"Type cast with navigation property",
		func(ctx *framework.TestContext) error {
			if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
				return err
			}
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Namespace.SpecialProduct/Category")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return fmt.Errorf("type casting failed but derived types exist in metadata (status: %d)", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	// Test 12: Invalid type cast returns error
	suite.AddTest(
		"test_invalid_type_cast",
		"Invalid type cast returns error",
		func(ctx *framework.TestContext) error {
			if err := skipIfDerivedTypesUnavailable(ctx); err != nil {
				return err
			}
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Namespace.InvalidType")
			if err != nil {
				return err
			}

			// Should return 404 or 400 for invalid type
			if resp.StatusCode == 404 || resp.StatusCode == 400 {
				return nil
			}

			if resp.StatusCode == 200 {
				return fmt.Errorf("invalid type cast should fail")
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	return suite
}

func skipIfDerivedTypesUnavailable(ctx *framework.TestContext) error {
	supported, err := ensureDerivedTypeSupport(ctx)
	if err != nil {
		return err
	}
	if !supported {
		return ctx.Skip("Service metadata does not declare derived type Namespace.SpecialProduct")
	}
	return nil
}

func ensureDerivedTypeSupport(ctx *framework.TestContext) (bool, error) {
	if derivedTypesChecked {
		return derivedTypesPresent, nil
	}
	resp, err := ctx.GET("/$metadata")
	if err != nil {
		return false, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return false, err
	}
	body := string(resp.Body)
	derivedTypesChecked = true
	derivedTypesPresent = strings.Contains(body, "Namespace.SpecialProduct")
	return derivedTypesPresent, nil
}
