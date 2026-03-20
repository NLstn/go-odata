package v4_0

import (
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// CaseSensitivity creates the 11.2.17 Case Sensitivity test suite.
// These tests explicitly negotiate OData 4.0 to verify the strict 4.0 behavior for
// system query option names, even when the server also supports OData 4.01.
func CaseSensitivity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.17 Case Sensitivity",
		"Tests that OData 4.0-negotiated requests enforce strict system query option casing and $-prefix behavior.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptions",
	)

	v4Headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}

	suite.AddTest(
		"test_correct_lowercase_filter",
		"Correct lowercase $filter works under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 10", v4Headers...)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			return ctx.AssertHeader(resp, "OData-Version", "4.0")
		},
	)

	suite.AddTest(
		"test_correct_lowercase_top",
		"Correct lowercase $top works under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=5", v4Headers...)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_select",
		"Correct lowercase $select works under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name", v4Headers...)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_orderby",
		"Correct lowercase $orderby works under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Name", v4Headers...)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	suite.AddTest(
		"test_correct_lowercase_count",
		"Correct lowercase $count works under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true", v4Headers...)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	suite.AddTest(
		"test_unknown_dollar_option_rejected",
		"Unknown $-prefixed options are rejected with 400 under OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$unknownOption=value", v4Headers...)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_mixed_case_system_option_rejected_in_v4_0",
		"Mixed-case system query option names are rejected when negotiated to OData 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$FILTER=Price gt 10", v4Headers...)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusBadRequest); err != nil {
				return framework.NewError(fmt.Sprintf("expected mixed-case system query option to be rejected under OData 4.0: %v", err))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_dollarless_known_option_treated_as_custom_in_v4_0",
		"Known query options without a $ prefix are treated as custom options when negotiated to OData 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?filter=Price gt 100000&$top=1", v4Headers...)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			if len(items) != 1 {
				return framework.NewError(fmt.Sprintf("expected custom 'filter' parameter to be ignored so only $top applies, got %d item(s)", len(items)))
			}

			return nil
		},
	)

	return suite
}
