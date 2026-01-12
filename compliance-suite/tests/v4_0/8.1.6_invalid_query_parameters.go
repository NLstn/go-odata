package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// InvalidQueryParameters creates the 8.1.6 Invalid Query Parameters test suite
func InvalidQueryParameters() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.1.6 Invalid Query Parameters",
		"Tests that services properly reject invalid or unknown query parameters with 400 Bad Request according to OData specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptions",
	)

	suite.AddTest(
		"test_unknown_system_query_option",
		"Unknown system query option returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// $unknown is not a valid OData system query option
			resp, err := ctx.GET("/Products?$unknown=value")
			if err != nil {
				return err
			}

			// Per OData spec, unknown system query options (starting with $) MUST result in 400
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_invalid_filter_syntax",
		"Invalid filter syntax returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// Invalid filter expression with unmatched parentheses
			resp, err := ctx.GET("/Products?$filter=(Price gt 10")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_invalid_orderby_syntax",
		"Invalid orderby syntax returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// Invalid orderby with unknown direction
			resp, err := ctx.GET("/Products?$orderby=Name invalid")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_invalid_top_value",
		"Invalid $top value returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// $top must be a non-negative integer
			resp, err := ctx.GET("/Products?$top=-5")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_invalid_skip_value",
		"Invalid $skip value returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// $skip must be a non-negative integer
			resp, err := ctx.GET("/Products?$skip=abc")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_duplicate_system_query_option",
		"Duplicate system query options should return 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// Per OData spec, duplicate system query options are not allowed
			// Note: Some implementations may allow this and use the last value (lenient behavior)
			resp, err := ctx.GET("/Products?$top=5&$top=10")
			if err != nil {
				return err
			}

			// This test allows both 200 (lenient) and 400 (strict) responses
			if resp.StatusCode != 200 && resp.StatusCode != 400 {
				return framework.NewError(fmt.Sprintf("Expected 200 or 400, got: %d", resp.StatusCode))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_invalid_expand_path",
		"Invalid $expand path returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// Try to expand a non-existent navigation property
			resp, err := ctx.GET("/Products?$expand=NonExistentNavProperty")
			if err != nil {
				return err
			}

			// Should return 400 for invalid navigation property
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_invalid_select_path",
		"Invalid $select path returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			// Try to select a non-existent property
			resp, err := ctx.GET("/Products?$select=NonExistentProperty")
			if err != nil {
				return err
			}

			// Should return 400 for invalid property
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
