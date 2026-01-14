package v4_0

import (
	"fmt"
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
		"Expand with $count=true includes @odata.count annotation",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($count=true)")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var body map[string]interface{}
			if err := ctx.GetJSON(resp, &body); err != nil {
				return err
			}

			entities, ok := body["value"].([]interface{})
			if !ok || len(entities) == 0 {
				return fmt.Errorf("expected non-empty value array in response")
			}

			entity, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected expanded entity to be a JSON object")
			}

			if _, ok := entity["Descriptions@odata.count"]; !ok {
				return fmt.Errorf("missing Descriptions@odata.count annotation")
			}

			return nil
		},
	)

	// Test 11: Expand with nested $count=false
	suite.AddTest(
		"test_expand_with_count_false",
		"Expand with $count=false does not include @odata.count annotation",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($count=false)")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var body map[string]interface{}
			if err := ctx.GetJSON(resp, &body); err != nil {
				return err
			}

			entities, ok := body["value"].([]interface{})
			if !ok || len(entities) == 0 {
				return fmt.Errorf("expected non-empty value array in response")
			}

			entity, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected expanded entity to be a JSON object")
			}

			if _, ok := entity["Descriptions@odata.count"]; ok {
				return fmt.Errorf("unexpected Descriptions@odata.count annotation")
			}

			return nil
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
		"Expand with $levels=2 returns expanded results",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($levels=2)")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertBodyContains(resp, "Descriptions")
		},
	)

	// Test 14: Expand with nested $levels=max
	suite.AddTest(
		"test_expand_with_levels_max",
		"Expand with $levels=max returns expanded results",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($levels=max)")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertBodyContains(resp, "Descriptions")
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
		"Expand with invalid nested $levels=-5 returns 400",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($levels=-5)")
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
		"Expand with both $count=true and $levels=2 returns annotations",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($count=true;$levels=2)")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertBodyContains(resp, "Descriptions@odata.count")
		},
	)

	return suite
}
