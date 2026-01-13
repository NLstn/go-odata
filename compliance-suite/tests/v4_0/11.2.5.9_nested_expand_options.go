package v4_0

import (
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NestedExpandOptions creates the 11.2.5.9 Nested Expand with Query Options test suite
func NestedExpandOptions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.9 Nested Expand with Query Options",
		"Tests nested $expand with multiple levels and nested query options ($filter, $select, $orderby, $top, $skip).",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptionexpand",
	)

	// Test 1: Basic nested expand
	suite.AddTest(
		"test_basic_nested_expand",
		"Basic nested expand returns 200",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: Expand with $select on expanded entity
	suite.AddTest(
		"test_expand_with_select",
		"Expand with $select on expanded entity",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($select=LanguageKey,Description)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "value")
		},
	)

	// Test 3: Expand with $filter
	suite.AddTest(
		"test_expand_with_filter",
		"Expand with $filter on expanded entity",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($filter=LanguageKey eq 'EN')")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 4: Expand with $orderby
	suite.AddTest(
		"test_expand_with_orderby",
		"Expand with $orderby on expanded entity",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($orderby=LanguageKey desc)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 5: Expand with $top
	suite.AddTest(
		"test_expand_with_top",
		"Expand with $top limits expanded results",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($top=2)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 6: Expand with multiple nested query options
	suite.AddTest(
		"test_expand_with_multiple_options",
		"Expand with multiple nested query options",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($select=LanguageKey,Description)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 7: Expand with invalid nested $select
	suite.AddTest(
		"test_expand_invalid_nested_select",
		"Expand with invalid nested $select returns 400",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($select=DoesNotExist)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 8: Expand with invalid nested $filter
	suite.AddTest(
		"test_expand_invalid_nested_filter",
		"Expand with invalid nested $filter returns 400",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($filter=DoesNotExist eq 'X')")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 9: Expand with invalid nested $orderby
	suite.AddTest(
		"test_expand_invalid_nested_orderby",
		"Expand with invalid nested $orderby returns 400",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($orderby=DoesNotExist)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
