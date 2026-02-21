package v4_01

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NestedExpandOptions creates the 11.2.5.9 Nested Expand with Query Options test suite for OData v4.01.
func NestedExpandOptions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.9 Nested Expand with Query Options",
		"Validates nested $expand options including multi-option combinations, nested multi-level expands, $count, and $levels in OData v4.01.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptionexpand",
	)

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
