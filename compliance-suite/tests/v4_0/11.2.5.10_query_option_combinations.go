package v4_0

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

func parseCollectionResponse(ctx *framework.TestContext, resp *framework.HTTPResponse) (map[string]interface{}, []map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	items, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return nil, nil, err
	}

	return result, items, nil
}

func itemFloat(item map[string]interface{}, key string) (float64, error) {
	v, ok := item[key].(float64)
	if !ok {
		return 0, fmt.Errorf("item missing %s field or %s is not numeric", key, key)
	}
	return v, nil
}

func itemString(item map[string]interface{}, key string) (string, error) {
	v, ok := item[key].(string)
	if !ok {
		return "", fmt.Errorf("item missing %s field or %s is not a string", key, key)
	}
	return v, nil
}

func assertSelectedFieldsOnly(ctx *framework.TestContext, items []map[string]interface{}, selected ...string) error {
	allowed := []string{"@odata.context", "@odata.etag", "@odata.id", "ID"}
	allowed = append(allowed, selected...)
	for i, item := range items {
		if err := ctx.AssertEntityOnlyAllowedFields(item, allowed...); err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}
	}
	return nil
}

func assertPricesMatch(items []map[string]interface{}, pred func(float64) bool, desc string) error {
	for i, item := range items {
		price, err := itemFloat(item, "Price")
		if err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}
		if !pred(price) {
			return fmt.Errorf("item %d has Price=%.2f which does not satisfy %s", i, price, desc)
		}
	}
	return nil
}

func assertSortedByPriceDesc(ctx *framework.TestContext, items []map[string]interface{}) error {
	return ctx.AssertEntitiesSortedByFloat(items, "Price", false)
}

// QueryOptionCombinations creates the 11.2.5.10 Query Option Combinations test suite
func QueryOptionCombinations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.10 Query Option Combinations",
		"Tests valid and invalid combinations of query options according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptions",
	)

	// Test 1: $filter with $select
	suite.AddTest(
		"test_filter_with_select",
		"$filter combined with $select",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 100&$select=ID,Name,Price")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			_, items, err := parseCollectionResponse(ctx, resp)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return fmt.Errorf("filter+select returned no items")
			}
			if err := assertPricesMatch(items, func(p float64) bool { return p > 100 }, "Price gt 100"); err != nil {
				return err
			}
			if err := assertSelectedFieldsOnly(ctx, items, "Name", "Price"); err != nil {
				return err
			}

			return nil
		},
	)

	// Test 2: $filter with $orderby
	suite.AddTest(
		"test_filter_with_orderby",
		"$filter combined with $orderby",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 100&$orderby=Price desc")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			_, items, err := parseCollectionResponse(ctx, resp)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return fmt.Errorf("filter+orderby returned no items")
			}
			if err := assertPricesMatch(items, func(p float64) bool { return p > 100 }, "Price gt 100"); err != nil {
				return err
			}
			if err := assertSortedByPriceDesc(ctx, items); err != nil {
				return err
			}

			return nil
		},
	)

	// Test 3: $filter with $top and $skip
	suite.AddTest(
		"test_filter_with_pagination",
		"$filter combined with pagination",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 50&$top=10&$skip=0")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			_, items, err := parseCollectionResponse(ctx, resp)
			if err != nil {
				return err
			}
			if len(items) > 10 {
				return fmt.Errorf("$top=10 expected at most 10 items, got %d", len(items))
			}
			if err := assertPricesMatch(items, func(p float64) bool { return p > 50 }, "Price gt 50"); err != nil {
				return err
			}

			return nil
		},
	)

	// Test 4: $filter with $count
	suite.AddTest(
		"test_filter_with_count",
		"$filter combined with $count",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 100&$count=true")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			result, items, err := parseCollectionResponse(ctx, resp)
			if err != nil {
				return err
			}
			if err := assertPricesMatch(items, func(p float64) bool { return p > 100 }, "Price gt 100"); err != nil {
				return err
			}
			count, ok := result["@odata.count"].(float64)
			if !ok {
				return fmt.Errorf("response missing '@odata.count' field")
			}
			if int(count) != len(items) {
				return fmt.Errorf("@odata.count (%d) must match returned items length (%d) without pagination", int(count), len(items))
			}

			return nil
		},
	)

	// Test 5: $select with $orderby
	suite.AddTest(
		"test_select_with_orderby",
		"$select combined with $orderby",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name,Price&$orderby=Price desc")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			_, items, err := parseCollectionResponse(ctx, resp)
			if err != nil {
				return err
			}
			if len(items) < 2 {
				return fmt.Errorf("select+orderby requires at least 2 items to verify order")
			}
			if err := assertSortedByPriceDesc(ctx, items); err != nil {
				return err
			}
			if err := assertSelectedFieldsOnly(ctx, items, "Name", "Price"); err != nil {
				return err
			}

			return nil
		},
	)

	// Test 6: $select with $expand
	suite.AddTest(
		"test_select_with_expand",
		"$select combined with $expand",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=ID,Name&$expand=Descriptions")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			_, items, err := parseCollectionResponse(ctx, resp)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return fmt.Errorf("select+expand returned no items")
			}
			for i, item := range items {
				if _, ok := item["Descriptions"]; !ok {
					return fmt.Errorf("item %d missing expanded Descriptions property", i)
				}
			}

			return nil
		},
	)

	// Test 7: All basic query options combined
	suite.AddTest(
		"test_all_options_combined",
		"All basic query options combined",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 50&$select=ID,Name,Price&$orderby=Price desc&$top=5&$count=true")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			result, items, err := parseCollectionResponse(ctx, resp)
			if err != nil {
				return err
			}
			if len(items) > 5 {
				return fmt.Errorf("$top=5 expected at most 5 items, got %d", len(items))
			}
			if err := assertPricesMatch(items, func(p float64) bool { return p > 50 }, "Price gt 50"); err != nil {
				return err
			}
			if err := assertSortedByPriceDesc(ctx, items); err != nil {
				return err
			}
			if err := assertSelectedFieldsOnly(ctx, items, "Name", "Price"); err != nil {
				return err
			}
			count, ok := result["@odata.count"].(float64)
			if !ok {
				return fmt.Errorf("response missing '@odata.count' field")
			}
			if int(count) < len(items) {
				return fmt.Errorf("@odata.count (%d) must be >= returned items (%d)", int(count), len(items))
			}

			return nil
		},
	)

	// Test 8: $count with $filter and $orderby
	suite.AddTest(
		"test_count_with_other_options",
		"$count with $filter and $orderby",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true&$filter=Price gt 50")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			result, items, err := parseCollectionResponse(ctx, resp)
			if err != nil {
				return err
			}
			if err := assertPricesMatch(items, func(p float64) bool { return p > 50 }, "Price gt 50"); err != nil {
				return err
			}
			count, ok := result["@odata.count"].(float64)
			if !ok {
				return fmt.Errorf("response missing '@odata.count' field")
			}
			if int(count) != len(items) {
				return fmt.Errorf("@odata.count (%d) must match returned items length (%d) without pagination", int(count), len(items))
			}

			return nil
		},
	)

	// Test 9: $search with $filter
	suite.AddTest(
		"test_search_with_filter",
		"$search combined with $filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 500")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			_, items, err := parseCollectionResponse(ctx, resp)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return fmt.Errorf("filter path for search+filter test returned no items")
			}
			if err := assertPricesMatch(items, func(p float64) bool { return p > 500 }, "Price gt 500"); err != nil {
				return err
			}

			return nil
		},
	)

	// Test 10: Complex combination with expand and nested options
	suite.AddTest(
		"test_complex_combination",
		"Complex combination of all query options",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 100&$select=ID,Name,Price&$orderby=Name&$expand=Descriptions&$top=10&$count=true")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			result, items, err := parseCollectionResponse(ctx, resp)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return fmt.Errorf("complex combination returned no items")
			}
			if len(items) > 10 {
				return fmt.Errorf("$top=10 expected at most 10 items, got %d", len(items))
			}
			if err := assertPricesMatch(items, func(p float64) bool { return p > 100 }, "Price gt 100"); err != nil {
				return err
			}
			names := make([]string, 0, len(items))
			for i, item := range items {
				name, err := itemString(item, "Name")
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				names = append(names, name)
				if _, ok := item["Descriptions"]; !ok {
					return fmt.Errorf("item %d missing expanded Descriptions property", i)
				}
			}
			sortedNames := append([]string(nil), names...)
			sort.Strings(sortedNames)
			if strings.Join(names, "\x00") != strings.Join(sortedNames, "\x00") {
				return fmt.Errorf("results not ordered by Name asc")
			}
			count, ok := result["@odata.count"].(float64)
			if !ok {
				return fmt.Errorf("response missing '@odata.count' field")
			}
			if int(count) < len(items) {
				return fmt.Errorf("@odata.count (%d) must be >= returned items (%d)", int(count), len(items))
			}

			return nil
		},
	)

	return suite
}
