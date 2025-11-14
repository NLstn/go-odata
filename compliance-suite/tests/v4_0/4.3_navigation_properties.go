package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NavigationProperties creates the 4.3 Navigation Properties test suite
func NavigationProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"4.3 Navigation Properties",
		"Tests navigation property definitions and relationships in metadata including relationship types, multiplicity, and partner properties.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_NavigationProperty",
	)

	suite.AddTest(
		"test_navigation_properties_in_metadata",
		"Navigation properties declared in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "NavigationProperty") {
				return framework.NewError("Metadata should declare navigation properties for related entities")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_navigation_property_has_type",
		"Navigation properties specify target type",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="`) {
				return framework.NewError("Navigation properties must specify their target entity type")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_single_navigation_property",
		"Single-valued navigation property returns entity",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Category")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return nil // Feature not implemented, skip
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			// Should return an entity with @odata.context, not a collection
			if _, hasContext := result["@odata.context"]; !hasContext {
				return framework.NewError("Single-valued navigation should include @odata.context")
			}

			// Should NOT have a "value" array for single navigation
			if _, hasValue := result["value"]; hasValue {
				return framework.NewError("Single-valued navigation property should not have value array")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_collection_navigation_property",
		"Collection-valued navigation property returns collection",
		func(ctx *framework.TestContext) error {
			catPath, err := firstEntityPath(ctx, "Categories")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(catPath + "/Products")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return nil // Feature not implemented, skip
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			// Should return a collection with "value" array
			if _, hasValue := result["value"]; !hasValue {
				return framework.NewError("Collection-valued navigation property should return array with value wrapper")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_navigation_with_filter",
		"Navigation property supports $filter",
		func(ctx *framework.TestContext) error {
			catPath, err := firstEntityPath(ctx, "Categories")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(catPath + "/Products?$filter=Price gt 50")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return nil // Feature not implemented, skip
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	suite.AddTest(
		"test_navigation_count",
		"Navigation property supports $count",
		func(ctx *framework.TestContext) error {
			catPath, err := firstEntityPath(ctx, "Categories")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(catPath + "/Products/$count")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return nil // Feature not implemented, skip
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Should return a number
			body := strings.TrimSpace(string(resp.Body))
			if len(body) == 0 {
				return framework.NewError("Navigation property $count should return numeric value")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_invalid_navigation_property",
		"Invalid navigation property returns 404",
		func(ctx *framework.TestContext) error {
			path, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path + "/InvalidNavProperty")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 404); err != nil {
				return err
			}

			return nil
		},
	)

	return suite
}
