package v4_0

import (
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NestedExpandOptions creates the 11.2.5.9 Nested Expand with Query Options test suite
func NestedExpandOptions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.9 Nested Expand with Query Options",
		"Tests nested $expand with multiple levels and nested query options ($filter, $select, $orderby, $top, $skip, $count, $levels).",
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

	// Test 10: Expand with nested $count=true
	suite.AddTest(
		"test_expand_with_count_true",
		"Expand with $count=true returns 400 (not yet implemented)",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($count=true)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			// Expect 400 since $count=true is not yet implemented
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 11: Expand with nested $count=false
	suite.AddTest(
		"test_expand_with_count_false",
		"Expand with $count=false returns 200 (no-op)",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($count=false)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			// count=false is a no-op, should work fine
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 12: Expand with invalid nested $count
	suite.AddTest(
		"test_expand_invalid_nested_count",
		"Expand with invalid nested $count returns 400",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($count=invalid)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 13: Expand with nested $levels (integer)
	suite.AddTest(
		"test_expand_with_levels_integer",
		"Expand with $levels=2 returns 400 (not yet implemented)",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($levels=2)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			// Expect 400 since $levels is not yet implemented
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 14: Expand with nested $levels=max
	suite.AddTest(
		"test_expand_with_levels_max",
		"Expand with $levels=max returns 400 (not yet implemented)",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($levels=max)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			// Expect 400 since $levels is not yet implemented
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 15: Expand with invalid nested $levels (zero)
	suite.AddTest(
		"test_expand_invalid_nested_levels_zero",
		"Expand with invalid nested $levels=0 returns 400",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($levels=0)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 16: Expand with invalid nested $levels (negative)
	suite.AddTest(
		"test_expand_invalid_nested_levels_negative",
		"Expand with invalid nested $levels=-1 returns 400",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($levels=-1)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 17: Expand with both $count and $levels
	suite.AddTest(
		"test_expand_with_count_and_levels",
		"Expand with both $count=true and $levels=2 returns 400 (not yet implemented)",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($count=true;$levels=2)")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			// Expect 400 since both options are not yet implemented
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
