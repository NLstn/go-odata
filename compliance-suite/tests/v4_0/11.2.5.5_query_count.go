package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QueryCount creates the 11.2.5.5 System Query Option $count test suite
func QueryCount() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.5 System Query Option $count",
		"Tests $count query option according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptioncount",
	)

	// Test 1: $count=true includes @odata.count in response
	suite.AddTest(
		"test_count_true",
		"$count=true includes @odata.count in response",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Verify @odata.count is present
			if _, ok := result["@odata.count"]; !ok {
				return fmt.Errorf("@odata.count field is missing")
			}

			return nil
		},
	)

	// Test 2: $count=false does not include @odata.count
	suite.AddTest(
		"test_count_false",
		"$count=false excludes @odata.count",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=false")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Verify @odata.count is NOT present
			if _, ok := result["@odata.count"]; ok {
				return fmt.Errorf("@odata.count should not be present when $count=false")
			}

			return nil
		},
	)

	// Test 3: $count with $filter returns filtered count
	suite.AddTest(
		"test_count_with_filter",
		"$count with $filter returns filtered count",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true&$filter=Price%20gt%20100")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Verify @odata.count is present
			countVal, ok := result["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count field is missing")
			}

			count, ok := countVal.(float64)
			if !ok {
				return fmt.Errorf("@odata.count is not a number")
			}

			// Get the items
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			items := len(value)

			// The count should equal the number of items returned (when no $top is used)
			if int(count) != items {
				return fmt.Errorf("count=%d but response contains %d items", int(count), items)
			}

			// Verify all items actually match the filter (Price > 100)
			for i, v := range value {
				item, ok := v.(map[string]interface{})
				if !ok {
					return fmt.Errorf("item %d is not an object", i)
				}
				price, ok := item["Price"].(float64)
				if !ok {
					return fmt.Errorf("item %d missing Price field or not a number", i)
				}
				if price <= 100 {
					return fmt.Errorf("found item with Price=%.2f which is not > 100", price)
				}
			}

			return nil
		},
	)

	// Test 4: $count with $top still returns total count
	suite.AddTest(
		"test_count_with_top",
		"$count with $top returns total count, not page count",
		func(ctx *framework.TestContext) error {
			// First get total count without $top
			totalResp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}
			if totalResp.StatusCode != 200 {
				return fmt.Errorf("failed to get total count: status %d", totalResp.StatusCode)
			}

			var totalResult map[string]interface{}
			if err := json.Unmarshal(totalResp.Body, &totalResult); err != nil {
				return fmt.Errorf("failed to parse total count JSON: %w", err)
			}

			totalCountVal, ok := totalResult["@odata.count"]
			if !ok {
				return fmt.Errorf("total response missing @odata.count")
			}
			totalCountFloat, ok := totalCountVal.(float64)
			if !ok {
				return fmt.Errorf("@odata.count is not a number")
			}
			totalCount := int(totalCountFloat)

			// Now get with $top=1
			resp, err := ctx.GET("/Products?$count=true&$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Verify @odata.count is present
			countVal, ok := result["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count field is missing")
			}

			countFloat, ok := countVal.(float64)
			if !ok {
				return fmt.Errorf("@odata.count is not a number")
			}
			count := int(countFloat)

			// Get the items
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			items := len(value)

			// The @odata.count should be the TOTAL count, not the page count
			if count != totalCount {
				return fmt.Errorf("count=%d but total count is %d (should match total, not page size)", count, totalCount)
			}

			// The number of items in the response should be 1 (due to $top=1)
			if items != 1 {
				return fmt.Errorf("expected 1 item in response (due to $top=1), got %d", items)
			}

			// The count should be greater than or equal to the items
			if count < items {
				return fmt.Errorf("count=%d should be >= items=%d", count, items)
			}

			return nil
		},
	)

	return suite
}
