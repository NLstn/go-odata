package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// CaseSensitivity creates the 11.2.17 Case Sensitivity test suite
func CaseSensitivity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.17 Case Sensitivity",
		"Tests that OData system query options are case-sensitive and must use the correct case according to OData specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptions",
	)

	suite.AddTest(
		"test_uppercase_filter",
		"Uppercase $FILTER is invalid and returns 400",
		func(ctx *framework.TestContext) error {
			// System query options are case-sensitive and must be lowercase with $ prefix
			resp, err := ctx.GET("/Products?$FILTER=Price gt 10")
			if err != nil {
				return err
			}

			// Should return 400 for incorrect case
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_uppercase_top",
		"Uppercase $TOP is invalid and returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$TOP=5")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_uppercase_skip",
		"Uppercase $SKIP is invalid and returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$SKIP=5")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_uppercase_orderby",
		"Uppercase $ORDERBY is invalid and returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$ORDERBY=Name")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_uppercase_select",
		"Uppercase $SELECT is invalid and returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$SELECT=Name")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_uppercase_expand",
		"Uppercase $EXPAND is invalid and returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$EXPAND=Category")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_mixed_case_filter",
		"Mixed case $Filter is invalid and returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$Filter=Price gt 10")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_filter",
		"Correct lowercase $filter works properly",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 10")
			if err != nil {
				return err
			}

			// Should return 200 for correct case
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_uppercase_count",
		"Uppercase $COUNT is invalid and returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$COUNT=true")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_uppercase_format",
		"Uppercase $FORMAT is invalid and returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$FORMAT=json")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
