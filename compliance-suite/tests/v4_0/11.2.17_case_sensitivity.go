package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// CaseSensitivity creates the 11.2.17 Case Sensitivity test suite.
// Note: OData 4.01 supersedes the 4.0 case-sensitivity requirement: in 4.01 system query
// option names are case-insensitive and the $ prefix is optional. A server that implements
// OData 4.01 therefore accepts uppercase/mixed-case options from all clients. These tests
// validate that the canonical lowercase forms still work correctly.
func CaseSensitivity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.17 Case Sensitivity",
		"Tests that OData system query options work correctly. Note: OData 4.01 makes option names case-insensitive, so a 4.01-capable server accepts any casing.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptions",
	)

	suite.AddTest(
		"test_correct_lowercase_filter",
		"Correct lowercase $filter works properly",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 10")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_top",
		"Correct lowercase $top works properly",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=5")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_select",
		"Correct lowercase $select works properly",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_orderby",
		"Correct lowercase $orderby works properly",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Name")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_count",
		"Correct lowercase $count works properly",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_unknown_dollar_option_rejected",
		"Truly unknown $-prefixed options are rejected with 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$unknownOption=value")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
