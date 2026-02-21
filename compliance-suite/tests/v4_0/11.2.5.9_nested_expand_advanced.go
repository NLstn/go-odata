package v4_0

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NestedExpandAdvanced creates a test suite for complex nested expand scenarios not covered
// by the basic 11.2.5.9 suite: multiple simultaneous expands, single-entity nav select,
// top-level select + nested expand interactions, edge cases ($top=0, $skip overflow),
// string functions in nested filter, all options combined, and circular navigation.
func NestedExpandAdvanced() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.9 Advanced Nested Expand Combinations",
		"Tests complex nested $expand combinations: multiple simultaneous expands, single-entity navigation "+
			"select field exclusion, top-level $select + nested $expand, edge cases ($top=0, $skip overflow), "+
			"string functions in nested $filter, all options combined, and circular navigation expand.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptionexpand",
	)

	// Test 1: Multiple simultaneous $expand with different nested options on each
	suite.AddTest(
		"test_multiple_simultaneous_expand_with_nested_options",
		"Multiple simultaneous $expand with different nested options on each",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Category($select=Name),Descriptions($select=LanguageKey;$filter=LanguageKey eq 'EN')")
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
				return fmt.Errorf("expected non-empty value array")
			}

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}

			// Category must be present and be an object (single entity nav)
			categoryRaw, ok := product["Category"]
			if !ok {
				return fmt.Errorf("expected Category to be expanded")
			}
			if categoryRaw == nil {
				return fmt.Errorf("expected Category to be a non-null object")
			}
			category, ok := categoryRaw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected Category to be an object, not array")
			}
			if _, hasName := category["Name"]; !hasName {
				return fmt.Errorf("expected Category.Name to be present after $select=Name")
			}

			// Descriptions must be present and be an array filtered to EN only
			descriptionsRaw, ok := product["Descriptions"].([]interface{})
			if !ok {
				return fmt.Errorf("expected Descriptions to be an array")
			}
			for i, raw := range descriptionsRaw {
				desc, ok := raw.(map[string]interface{})
				if !ok {
					return fmt.Errorf("Descriptions[%d] is not an object", i)
				}
				lang, ok := desc["LanguageKey"].(string)
				if !ok || lang == "" {
					return fmt.Errorf("Descriptions[%d] missing LanguageKey", i)
				}
				if lang != "EN" {
					return fmt.Errorf("Descriptions[%d] has LanguageKey %q but filter should return only EN", i, lang)
				}
			}

			return nil
		},
	)

	// Test 2: Single-entity navigation with nested $select — field exclusion validated
	suite.AddTest(
		"test_single_entity_nav_nested_select_field_exclusion",
		"Single-entity navigation $expand with nested $select excludes non-selected fields",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Category($select=Name)")
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
				return fmt.Errorf("expected non-empty value array")
			}

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}

			categoryRaw, ok := product["Category"]
			if !ok {
				return fmt.Errorf("expected Category to be expanded")
			}
			if categoryRaw == nil {
				return fmt.Errorf("Laptop's Category must not be null (Laptop belongs to Electronics)")
			}

			category, ok := categoryRaw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected Category to be an object, not an array")
			}

			if _, hasName := category["Name"]; !hasName {
				return fmt.Errorf("Category.Name must be present after $select=Name")
			}

			if _, hasDesc := category["Description"]; hasDesc {
				return fmt.Errorf("Category.Description must NOT be present when $select=Name (field excluded by $select)")
			}

			return nil
		},
	)

	// Test 3: Top-level $select combined with nested $expand
	suite.AddTest(
		"test_top_level_select_with_nested_expand",
		"Top-level $select combined with nested $expand preserves expand and excludes unselected fields",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($select=LanguageKey)")
			resp, err := ctx.GET("/Products?$select=Name,Price&$expand=" + expand + "&$filter=" + url.QueryEscape("Name eq 'Laptop'"))
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
				return fmt.Errorf("expected non-empty value array")
			}

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}

			// Name and Price must be present
			if _, hasName := product["Name"]; !hasName {
				return fmt.Errorf("product.Name must be present (in $select)")
			}
			if _, hasPrice := product["Price"]; !hasPrice {
				return fmt.Errorf("product.Price must be present (in $select)")
			}

			// Status must NOT be present (excluded by top-level $select)
			if _, hasStatus := product["Status"]; hasStatus {
				return fmt.Errorf("product.Status must NOT be present when $select=Name,Price")
			}

			// Descriptions must be present as expanded navigation property
			descriptionsRaw, ok := product["Descriptions"].([]interface{})
			if !ok {
				return fmt.Errorf("Descriptions must be present and be an array")
			}
			if len(descriptionsRaw) == 0 {
				return fmt.Errorf("expected at least one Description for Laptop")
			}

			// Each description should have LanguageKey (from nested $select)
			desc0, ok := descriptionsRaw[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("Descriptions[0] is not an object")
			}
			if _, hasLang := desc0["LanguageKey"]; !hasLang {
				return fmt.Errorf("Descriptions[0].LanguageKey must be present")
			}

			return nil
		},
	)

	// Test 4: $expand with $top=0 returns empty array (not 400)
	suite.AddTest(
		"test_expand_with_top_zero_returns_empty_array",
		"$expand with $top=0 returns empty collection, not an error",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($top=0)")
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
				return fmt.Errorf("expected non-empty value array")
			}

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}

			descriptionsRaw, ok := product["Descriptions"].([]interface{})
			if !ok {
				return fmt.Errorf("expected Descriptions to be an array")
			}

			if len(descriptionsRaw) != 0 {
				return fmt.Errorf("expected Descriptions to be empty with $top=0, got %d items", len(descriptionsRaw))
			}

			return nil
		},
	)

	// Test 5: $expand with $skip exceeding collection size returns empty array
	suite.AddTest(
		"test_expand_with_skip_overflow_returns_empty_array",
		"$expand with $skip exceeding collection size returns empty collection, not an error",
		func(ctx *framework.TestContext) error {
			// Coffee Mug has exactly 1 description; skip=999 should yield empty result
			expand := url.QueryEscape("Descriptions($skip=999)")
			resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("Name eq 'Coffee Mug'") + "&$expand=" + expand)
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
				return fmt.Errorf("expected non-empty value array (Coffee Mug should be found)")
			}

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}

			descriptionsRaw, ok := product["Descriptions"].([]interface{})
			if !ok {
				return fmt.Errorf("expected Descriptions to be an array")
			}

			if len(descriptionsRaw) != 0 {
				return fmt.Errorf("expected empty Descriptions after $skip=999 (Coffee Mug has only 1 description), got %d items", len(descriptionsRaw))
			}

			return nil
		},
	)

	// Test 6: $expand on non-existent navigation property returns 400
	suite.AddTest(
		"test_expand_nonexistent_nav_property_returns_400",
		"$expand on a non-existent navigation property returns 400",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("NonExistentNavProperty")
			resp, err := ctx.GET("/Products?$expand=" + expand)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 7: Self-referencing $expand (RelatedProducts) returns valid array
	suite.AddTest(
		"test_self_referencing_expand_related_products",
		"Self-referencing $expand on RelatedProducts returns a valid array without server error",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("RelatedProducts($select=Name,Price)")
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
				return fmt.Errorf("expected non-empty value array")
			}

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}

			// RelatedProducts must be present and be an array (may be empty since no sample data)
			relatedRaw, ok := product["RelatedProducts"]
			if !ok {
				return fmt.Errorf("expected RelatedProducts to be expanded")
			}

			if _, ok := relatedRaw.([]interface{}); !ok {
				return fmt.Errorf("expected RelatedProducts to be an array, got %T", relatedRaw)
			}

			return nil
		},
	)

	// Test 8: Three-level deep expand with content validation
	suite.AddTest(
		"test_three_level_deep_expand_with_content_validation",
		"Three-level deep $expand (Categories→Products→Descriptions) with combined nested options and content validation",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Products($select=Name,Price;$filter=Price gt 100;$expand=Descriptions($filter=LanguageKey eq 'EN';$select=LanguageKey,Description))")
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
				return fmt.Errorf("expected non-empty value array for Electronics category")
			}

			category, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected category to be an object")
			}

			productsRaw, ok := category["Products"].([]interface{})
			if !ok || len(productsRaw) == 0 {
				return fmt.Errorf("expected non-empty Products in Electronics category (filtered to Price > 100)")
			}

			for i, rawProduct := range productsRaw {
				product, ok := rawProduct.(map[string]interface{})
				if !ok {
					return fmt.Errorf("Products[%d] is not an object", i)
				}

				// Validate $select=Name,Price — Status should not be present
				if _, hasStatus := product["Status"]; hasStatus {
					return fmt.Errorf("Products[%d].Status should not be present (excluded by $select=Name,Price)", i)
				}
				if _, hasName := product["Name"]; !hasName {
					return fmt.Errorf("Products[%d].Name must be present", i)
				}

				// Validate $filter=Price gt 100
				price, ok := product["Price"].(float64)
				if !ok {
					return fmt.Errorf("Products[%d].Price is missing or not a number", i)
				}
				if price <= 100 {
					return fmt.Errorf("Products[%d].Price=%v violates $filter=Price gt 100", i, price)
				}

				// Validate nested $expand=Descriptions with $filter=LanguageKey eq 'EN'
				descriptionsRaw, ok := product["Descriptions"].([]interface{})
				if !ok {
					return fmt.Errorf("Products[%d] is missing expanded Descriptions", i)
				}

				for j, rawDesc := range descriptionsRaw {
					desc, ok := rawDesc.(map[string]interface{})
					if !ok {
						return fmt.Errorf("Products[%d].Descriptions[%d] is not an object", i, j)
					}

					lang, ok := desc["LanguageKey"].(string)
					if !ok || lang == "" {
						return fmt.Errorf("Products[%d].Descriptions[%d].LanguageKey is missing", i, j)
					}

					if lang != "EN" {
						return fmt.Errorf("Products[%d].Descriptions[%d].LanguageKey=%q violates $filter=LanguageKey eq 'EN'", i, j, lang)
					}

					// $select=LanguageKey,Description — LongText should not be present
					if _, hasLongText := desc["LongText"]; hasLongText {
						return fmt.Errorf("Products[%d].Descriptions[%d].LongText should not be present (excluded by $select=LanguageKey,Description)", i, j)
					}
				}
			}

			return nil
		},
	)

	// Test 9: Nested $filter using string function (startswith)
	suite.AddTest(
		"test_nested_filter_with_string_function_startswith",
		"Nested $filter using startswith() string function filters expanded results correctly",
		func(ctx *framework.TestContext) error {
			// Laptop has EN and DE descriptions; startswith(LanguageKey,'E') should return only EN
			expand := url.QueryEscape("Descriptions($filter=startswith(LanguageKey,'E'))")
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
				return fmt.Errorf("expected non-empty value array")
			}

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}

			descriptionsRaw, ok := product["Descriptions"].([]interface{})
			if !ok {
				return fmt.Errorf("expected Descriptions to be an array")
			}

			if len(descriptionsRaw) == 0 {
				return fmt.Errorf("expected at least one Description starting with 'E' (EN) for Laptop")
			}

			for i, raw := range descriptionsRaw {
				desc, ok := raw.(map[string]interface{})
				if !ok {
					return fmt.Errorf("Descriptions[%d] is not an object", i)
				}
				lang, ok := desc["LanguageKey"].(string)
				if !ok {
					return fmt.Errorf("Descriptions[%d] is missing LanguageKey", i)
				}
				if !strings.HasPrefix(lang, "E") {
					return fmt.Errorf("Descriptions[%d].LanguageKey=%q does not start with 'E' (startswith filter failed)", i, lang)
				}
			}

			return nil
		},
	)

	// Test 10: All nested options combined at once
	suite.AddTest(
		"test_all_nested_options_combined",
		"All nested expand options combined ($filter, $select, $orderby, $top, $skip, $count=true) in a single query",
		func(ctx *framework.TestContext) error {
			// Laptop has EN and DE; filter excludes ZZ, orderby asc, top=2, skip=0, count=true
			expand := url.QueryEscape("Descriptions($filter=LanguageKey ne 'ZZ';$select=LanguageKey,Description;$orderby=LanguageKey asc;$top=2;$skip=0;$count=true)")
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
				return fmt.Errorf("expected non-empty value array")
			}

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}

			// $count=true must produce Descriptions@odata.count annotation
			if _, hasCount := product["Descriptions@odata.count"]; !hasCount {
				return fmt.Errorf("expected Descriptions@odata.count annotation from $count=true")
			}

			descriptionsRaw, ok := product["Descriptions"].([]interface{})
			if !ok {
				return fmt.Errorf("expected Descriptions to be an array")
			}

			// $top=2: at most 2 results
			if len(descriptionsRaw) > 2 {
				return fmt.Errorf("expected at most 2 Descriptions (from $top=2), got %d", len(descriptionsRaw))
			}

			// Validate $select=LanguageKey,Description — LongText must not be present
			// Validate $orderby=LanguageKey asc — results must be ascending
			lastKey := ""
			for i, raw := range descriptionsRaw {
				desc, ok := raw.(map[string]interface{})
				if !ok {
					return fmt.Errorf("Descriptions[%d] is not an object", i)
				}

				lang, ok := desc["LanguageKey"].(string)
				if !ok {
					return fmt.Errorf("Descriptions[%d].LanguageKey is missing", i)
				}

				if _, hasLongText := desc["LongText"]; hasLongText {
					return fmt.Errorf("Descriptions[%d].LongText should not be present (excluded by $select=LanguageKey,Description)", i)
				}

				if _, hasCustomName := desc["CustomName"]; hasCustomName {
					return fmt.Errorf("Descriptions[%d].CustomName should not be present (excluded by $select)", i)
				}

				if lastKey != "" && lang < lastKey {
					return fmt.Errorf("Descriptions are not ordered by LanguageKey asc: %q came after %q", lang, lastKey)
				}
				lastKey = lang
			}

			return nil
		},
	)

	// Test 11: Circular navigation expand (Product→Category→Products)
	suite.AddTest(
		"test_circular_navigation_expand",
		"Circular navigation expand (Products→Category→Products) returns valid nested structure without infinite loop",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Category($expand=Products($select=Name,Price))")
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
				return fmt.Errorf("expected non-empty value array")
			}

			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}

			categoryRaw, ok := product["Category"]
			if !ok {
				return fmt.Errorf("expected Category to be expanded")
			}
			if categoryRaw == nil {
				return fmt.Errorf("Laptop's Category must not be null")
			}

			category, ok := categoryRaw.(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected Category to be an object")
			}

			// Category must contain Products (back-reference)
			nestedProductsRaw, ok := category["Products"].([]interface{})
			if !ok {
				return fmt.Errorf("expected Category.Products to be an array")
			}

			if len(nestedProductsRaw) == 0 {
				return fmt.Errorf("expected at least one Product in Category.Products (Electronics has products)")
			}

			// Validate $select=Name,Price on nested Products
			for i, rawNested := range nestedProductsRaw {
				nestedProduct, ok := rawNested.(map[string]interface{})
				if !ok {
					return fmt.Errorf("Category.Products[%d] is not an object", i)
				}
				if _, hasName := nestedProduct["Name"]; !hasName {
					return fmt.Errorf("Category.Products[%d].Name must be present", i)
				}
				if _, hasPrice := nestedProduct["Price"]; !hasPrice {
					return fmt.Errorf("Category.Products[%d].Price must be present", i)
				}
				// Status must not be present (excluded by $select=Name,Price)
				if _, hasStatus := nestedProduct["Status"]; hasStatus {
					return fmt.Errorf("Category.Products[%d].Status must NOT be present (excluded by $select=Name,Price)", i)
				}
			}

			return nil
		},
	)

	return suite
}
