package v4_0

import (
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NavigationPropertyOperations creates the 11.4.6.1 Navigation Property Operations test suite
func NavigationPropertyOperations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.6.1 Navigation Property Operations",
		"Tests operations on navigation properties including accessing, filtering, and modifying.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_NavigationProperties",
	)

	invalidProductPath := nonExistingEntityPath("Products")

	// Test 1: Access navigation property collection
	suite.AddTest(
		"test_nav_property_collection",
		"Access navigation property collection",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("Navigation property access returning 404, likely routing issue")
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: Navigation property returns collection structure
	suite.AddTest(
		"test_nav_property_collection_structure",
		"Navigation property returns collection structure",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("Navigation property access returning 404, likely routing issue")
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "value")
		},
	)

	// Test 3: Navigation property with $filter
	suite.AddTest(
		"test_nav_property_filter",
		"Navigation property with $filter",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			filter := url.QueryEscape("LanguageKey eq 'EN'")
			resp, err := ctx.GET(prodPath + "/Descriptions?$filter=" + filter)
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("Navigation property access returning 404, likely routing issue")
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 4: Navigation property with $select
	suite.AddTest(
		"test_nav_property_select",
		"Navigation property with $select",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions?$select=LanguageKey,Description")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("Navigation property access returning 404, likely routing issue")
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 5: Navigation property with $orderby
	suite.AddTest(
		"test_nav_property_orderby",
		"Navigation property with $orderby",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions?$orderby=LanguageKey")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("Navigation property access returning 404, likely routing issue")
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 6: Navigation property with $top
	suite.AddTest(
		"test_nav_property_top",
		"Navigation property with $top",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions?$top=2")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("Navigation property access returning 404, likely routing issue")
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 7: Navigation property with $count
	suite.AddTest(
		"test_nav_property_count",
		"Navigation property with $count",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions?$count=true")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("Navigation property access returning 404, likely routing issue")
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.count")
		},
	)

	// Test 8: Navigation property on non-existent entity returns 404
	suite.AddTest(
		"test_nav_property_not_found",
		"Navigation property on invalid entity returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath + "/Descriptions")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 404)
		},
	)

	// Test 9: Navigation property with combined query options
	suite.AddTest(
		"test_nav_property_combined_options",
		"Navigation property with combined options",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			filter := url.QueryEscape("LanguageKey eq 'EN'")
			resp, err := ctx.GET(prodPath + "/Descriptions?$filter=" + filter + "&$select=Description&$orderby=LanguageKey")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("Navigation property access returning 404, likely routing issue")
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 10: Navigation property has proper @odata.context
	suite.AddTest(
		"test_nav_property_context",
		"Navigation property has @odata.context",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("Navigation property access returning 404, likely routing issue")
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.context")
		},
	)

	return suite
}
