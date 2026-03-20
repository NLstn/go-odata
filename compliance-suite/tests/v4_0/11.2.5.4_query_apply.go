package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

func parseApplyItems(resp *framework.HTTPResponse) ([]map[string]interface{}, error) {
	var body struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(body.Value) == 0 {
		return nil, framework.NewError("response should contain at least one result")
	}
	return body.Value, nil
}

func firstPresent(item map[string]interface{}, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		if v, ok := item[key]; ok {
			return v, true
		}
	}

	for k, v := range item {
		for _, key := range keys {
			if strings.EqualFold(k, key) {
				return v, true
			}
		}
	}

	return nil, false
}

// QueryApply creates the 11.2.5.4 System Query Option $apply test suite
func QueryApply() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.4 System Query Option $apply",
		"Tests $apply query option for data aggregation according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html",
	)

	// Test 1: Basic aggregate transformation
	suite.AddTest(
		"test_apply_aggregate_count",
		"$apply with aggregate (count)",
		func(ctx *framework.TestContext) error {
			baselineResp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(baselineResp, 200); err != nil {
				return err
			}
			var baseline map[string]interface{}
			if err := json.Unmarshal(baselineResp.Body, &baseline); err != nil {
				return fmt.Errorf("failed to parse baseline response: %w", err)
			}
			expectedCount, ok := baseline["@odata.count"].(float64)
			if !ok {
				return framework.NewError("baseline response missing @odata.count")
			}

			filter := url.QueryEscape("aggregate($count as Total)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}

			rawTotal, ok := firstPresent(items[0], "Total", "total")
			if !ok {
				return framework.NewError("Aggregate response must include Total field")
			}
			total, ok := rawTotal.(float64)
			if !ok {
				return framework.NewError("Aggregate response Total field is not numeric")
			}
			if int(total) != int(expectedCount) {
				return framework.NewError(fmt.Sprintf("aggregate count mismatch: expected %.0f products, got %.0f", expectedCount, total))
			}

			return nil
		},
	)

	// Test 2: groupby transformation
	suite.AddTest(
		"test_apply_groupby",
		"$apply with groupby",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("groupby((CategoryID))")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}

			categoryIDs := make(map[string]struct{})
			for i, item := range items {
				rawCategoryID, ok := firstPresent(item, "CategoryID", "category_id")
				if !ok {
					return framework.NewError(fmt.Sprintf("group %d missing CategoryID", i))
				}
				categoryIDs[fmt.Sprintf("%v", rawCategoryID)] = struct{}{}
			}
			if len(categoryIDs) != 3 {
				return framework.NewError(fmt.Sprintf("expected 3 category groups, got %d", len(categoryIDs)))
			}

			return nil
		},
	)

	// Test 3: groupby with aggregate
	suite.AddTest(
		"test_apply_groupby_aggregate",
		"$apply with groupby and aggregate",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("groupby((CategoryID),aggregate($count as Count))")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}

			totalBaselineResp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(totalBaselineResp, 200); err != nil {
				return err
			}
			var baseline map[string]interface{}
			if err := json.Unmarshal(totalBaselineResp.Body, &baseline); err != nil {
				return fmt.Errorf("failed to parse baseline response: %w", err)
			}
			expectedCount, ok := baseline["@odata.count"].(float64)
			if !ok {
				return framework.NewError("baseline response missing @odata.count")
			}

			total := 0
			for i, item := range items {
				if _, ok := firstPresent(item, "CategoryID", "category_id"); !ok {
					return framework.NewError(fmt.Sprintf("group %d missing CategoryID", i))
				}
				rawCount, ok := firstPresent(item, "Count", "count")
				if !ok {
					return framework.NewError(fmt.Sprintf("group %d missing Count", i))
				}
				count, ok := rawCount.(float64)
				if !ok {
					return framework.NewError(fmt.Sprintf("group %d Count is not numeric", i))
				}
				if count <= 0 {
					return framework.NewError(fmt.Sprintf("group %d has non-positive Count %.0f", i, count))
				}
				total += int(count)
			}
			if total != int(expectedCount) {
				return framework.NewError(fmt.Sprintf("grouped aggregate count mismatch: expected %.0f, got %d", expectedCount, total))
			}

			return nil
		},
	)

	// Test 4: filter transformation
	suite.AddTest(
		"test_apply_filter",
		"$apply with filter transformation",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("filter(Price gt 10)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}
			for i, item := range items {
				rawPrice, ok := firstPresent(item, "Price", "price")
				if !ok {
					return framework.NewError(fmt.Sprintf("item %d missing Price", i))
				}
				price, ok := rawPrice.(float64)
				if !ok {
					return framework.NewError(fmt.Sprintf("item %d Price is not numeric", i))
				}
				if price <= 10 {
					return framework.NewError(fmt.Sprintf("item %d has Price=%.2f which violates filter(Price gt 10)", i, price))
				}
			}

			return nil
		},
	)

	// Test 5: Invalid $apply expression should return 400
	suite.AddTest(
		"test_apply_invalid",
		"Invalid $apply expression returns 400",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("invalid(syntax)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 400); err != nil {
				return err
			}

			return nil
		},
	)

	return suite
}
