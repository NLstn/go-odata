package v4_0

import (
	"fmt"
	"net/url"
	"sort"

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
		"test_expand_with_filter_select_and_orderby",
		"Expand with $filter, $select, and $orderby applies shape and order constraints",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($filter=LanguageKey ne 'ZZ';$select=LanguageKey;$orderby=LanguageKey desc)")
			resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("Name eq 'Laptop'") + "&$expand=" + expand)
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

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected first product to be an object")
			}

			descriptionsRaw, ok := product["Descriptions"].([]interface{})
			if !ok || len(descriptionsRaw) == 0 {
				return fmt.Errorf("expected non-empty expanded Descriptions array")
			}

			languageKeys := make([]string, 0, len(descriptionsRaw))
			for i, raw := range descriptionsRaw {
				description, ok := raw.(map[string]interface{})
				if !ok {
					return fmt.Errorf("Descriptions[%d] is not an object", i)
				}

				lang, ok := description["LanguageKey"].(string)
				if !ok || lang == "" {
					return fmt.Errorf("Descriptions[%d] is missing LanguageKey", i)
				}
				languageKeys = append(languageKeys, lang)

				if _, hasDescription := description["Description"]; hasDescription {
					return fmt.Errorf("Descriptions[%d] should not include Description when $select=LanguageKey", i)
				}
			}

			sorted := append([]string(nil), languageKeys...)
			sort.Sort(sort.Reverse(sort.StringSlice(sorted)))
			for i := range languageKeys {
				if languageKeys[i] != sorted[i] {
					return fmt.Errorf("Descriptions are not ordered by LanguageKey desc: got %v", languageKeys)
				}
			}

			return nil
		},
	)

	// Test 6b: Expand with nested pagination options
	suite.AddTest(
		"test_expand_with_top_skip_and_orderby",
		"Expand with $top, $skip, and $orderby applies pagination semantics",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($top=1;$skip=1;$orderby=LanguageKey asc)")
			resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("Name eq 'Laptop'") + "&$expand=" + expand)
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

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected first product to be an object")
			}

			descriptionsRaw, ok := product["Descriptions"].([]interface{})
			if !ok {
				return fmt.Errorf("expected expanded Descriptions array")
			}
			if len(descriptionsRaw) != 1 {
				return fmt.Errorf("expected exactly 1 expanded description after $top/$skip, got %d", len(descriptionsRaw))
			}

			description, ok := descriptionsRaw[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("Descriptions[0] is not an object")
			}

			languageKey, ok := description["LanguageKey"].(string)
			if !ok {
				return fmt.Errorf("Descriptions[0] is missing LanguageKey")
			}

			if languageKey != "EN" {
				return fmt.Errorf("expected LanguageKey EN after ordering asc and skipping first result, got %q", languageKey)
			}

			return nil
		},
	)

	// Test 6c: Multi-level nested expand with nested options
	suite.AddTest(
		"test_expand_nested_inside_expand_with_options",
		"Expand supports nested $expand with multiple levels and options",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Products($expand=Descriptions($filter=LanguageKey eq 'EN';$orderby=LanguageKey desc))")
			resp, err := ctx.GET("/Categories?$filter=" + url.QueryEscape("Name eq 'Electronics'") + "&$expand=" + expand)
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

			category, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected first category to be an object")
			}

			productsRaw, ok := category["Products"].([]interface{})
			if !ok || len(productsRaw) == 0 {
				return fmt.Errorf("expected non-empty expanded Products array")
			}

			for i, rawProduct := range productsRaw {
				product, ok := rawProduct.(map[string]interface{})
				if !ok {
					return fmt.Errorf("Products[%d] is not an object", i)
				}

				descriptionsRaw, ok := product["Descriptions"].([]interface{})
				if !ok {
					return fmt.Errorf("Products[%d] is missing expanded Descriptions", i)
				}

				last := "ZZZZ"
				for j, rawDesc := range descriptionsRaw {
					description, ok := rawDesc.(map[string]interface{})
					if !ok {
						return fmt.Errorf("Products[%d].Descriptions[%d] is not an object", i, j)
					}

					languageKey, ok := description["LanguageKey"].(string)
					if !ok {
						return fmt.Errorf("Products[%d].Descriptions[%d] is missing LanguageKey", i, j)
					}
					if languageKey != "EN" {
						return fmt.Errorf("Products[%d].Descriptions[%d] expected only EN after filter, got %q", i, j, languageKey)
					}
					if languageKey > last {
						return fmt.Errorf("Products[%d].Descriptions are not ordered by LanguageKey desc", i)
					}
					last = languageKey
				}
			}

			return nil
		},
	)

	// Test 6d: Expand with malformed nested options syntax
	suite.AddTest(
		"test_expand_with_malformed_nested_options_syntax",
		"Malformed nested option separator/parenthesis returns 400",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($filter=LanguageKey eq 'EN';$orderby=LanguageKey desc")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
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
