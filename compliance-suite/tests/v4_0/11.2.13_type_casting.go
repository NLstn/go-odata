package v4_0

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

var (
	derivedTypesChecked   bool
	derivedTypesPresent   bool
	detectedNamespace     string
	specialProductTypeRef string // Full qualified type name e.g., "ComplianceService.SpecialProduct"
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
			resp, err := ctx.GET("/Products?$filter=isof('" + specialProductTypeRef + "')")
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
			productPath, err := firstSpecialProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/" + specialProductTypeRef)
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
			resp, err := ctx.GET("/Products/" + specialProductTypeRef)
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
			productPath, err := firstSpecialProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/" + specialProductTypeRef + "/SpecialProperty")
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
			resp, err := ctx.GET("/Products?$filter=isof('" + specialProductTypeRef + "') and Price gt 100")
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
				"@odata.type":     "#" + specialProductTypeRef,
				"Name":            "Test Special Product",
				"Price":           100,
				"ProductType":     "SpecialProduct",
				"SpecialProperty": "Test special property",
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
			productPath, err := firstSpecialProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/" + specialProductTypeRef + "/Category")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			// 204 No Content is acceptable if the navigation property is null
			if resp.StatusCode == 204 {
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
			resp, err := ctx.GET(productPath + "/" + detectedNamespace + ".InvalidType")
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
		return ctx.Skip("Service metadata does not declare derived type with ProductType discriminator")
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

	// Extract the namespace from metadata (e.g., ComplianceService)
	// Look for <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="...">
	namespaceRegex := regexp.MustCompile(`Namespace="([^"]+)"`)
	matches := namespaceRegex.FindStringSubmatch(body)
	if len(matches) > 1 {
		detectedNamespace = matches[1]
	} else {
		detectedNamespace = "ODataService" // default fallback
	}

	// Check if the Products entity has a ProductType property (discriminator)
	// which indicates type inheritance is being used
	hasProductType := strings.Contains(body, `Name="ProductType"`)
	hasProduct := strings.Contains(body, `EntityType Name="Product"`)

	// The service supports derived types if it has both the Product entity and ProductType discriminator
	derivedTypesPresent = hasProductType && hasProduct

	if derivedTypesPresent {
		// Set the qualified type reference for SpecialProduct
		specialProductTypeRef = detectedNamespace + ".SpecialProduct"
	}

	return derivedTypesPresent, nil
}

// firstSpecialProductPath returns the path to a product with ProductType="SpecialProduct"
func firstSpecialProductPath(ctx *framework.TestContext) (string, error) {
	// Query for products with ProductType = SpecialProduct to find a valid special product
	resp, err := ctx.GET("/Products?$filter=ProductType eq 'SpecialProduct'&$top=1")
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		// Fallback to getting any product if filtering fails
		return firstEntityPath(ctx, "Products")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return firstEntityPath(ctx, "Products")
	}

	values, ok := data["value"].([]interface{})
	if !ok || len(values) == 0 {
		return firstEntityPath(ctx, "Products")
	}

	first := values[0].(map[string]interface{})
	id := first["ID"]
	return fmt.Sprintf("/Products(%v)", formatKeyValue(id)), nil
}

func formatKeyValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		// The server accepts raw UUID values without the guid prefix
		// Check if it matches UUID format (8-4-4-4-12 hex characters)
		if isUUID(val) {
			return val // Return raw UUID without prefix
		}
		return fmt.Sprintf("'%s'", val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// isUUID checks if a string matches the UUID format (8-4-4-4-12 hex characters)
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	// Check format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	// Check that all other characters are hex digits
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // Skip dashes
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
