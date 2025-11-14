package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QueryTopSkip creates the 11.2.5.3 and 11.2.5.4 System Query Options $top and $skip test suite
func QueryTopSkip() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.3 System Query Options $top and $skip",
		"Tests $top and $skip query options for paging according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionstopandskip",
	)

	// Test 1: $top limits the number of items returned
	suite.AddTest(
		"test_top_limit",
		"$top limits number of items",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=2")
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

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			count := len(value)
			// $top=2 should return at most 2 items
			if count > 2 {
				return fmt.Errorf("returned %d items, expected max 2", count)
			}

			// Verify we got at least 1 item (assuming Products collection is not empty)
			if count < 1 {
				return fmt.Errorf("returned %d items, expected at least 1", count)
			}

			return nil
		},
	)

	// Test 2: $skip skips the specified number of items
	suite.AddTest(
		"test_skip_items",
		"$skip skips items",
		func(ctx *framework.TestContext) error {
			// First get all products to know what to expect
			allResp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if allResp.StatusCode != 200 {
				return fmt.Errorf("failed to get all products: status %d", allResp.StatusCode)
			}

			var allResult map[string]interface{}
			if err := json.Unmarshal(allResp.Body, &allResult); err != nil {
				return fmt.Errorf("failed to parse all products JSON: %w", err)
			}

			allValue, ok := allResult["value"].([]interface{})
			if !ok || len(allValue) < 2 {
				// Not enough products to test $skip
				return nil
			}

			// Get the second item's ID from the full list
			secondItem, ok := allValue[1].(map[string]interface{})
			if !ok {
				return fmt.Errorf("second item is not an object")
			}
			secondID := secondItem["ID"]

			// Now test $skip=1&$top=1 - should return the second item
			resp, err := ctx.GET("/Products?$skip=1&$top=1")
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

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			// Verify we got exactly 1 item
			if len(value) != 1 {
				return fmt.Errorf("expected exactly 1 item, got %d", len(value))
			}

			// Verify it's the second item (ID should match)
			item, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("returned item is not an object")
			}
			returnedID := item["ID"]

			if fmt.Sprint(returnedID) != fmt.Sprint(secondID) {
				return fmt.Errorf("expected ID=%v (2nd item), got ID=%v", secondID, returnedID)
			}

			return nil
		},
	)

	// Test 3: $top=0 returns empty collection
	suite.AddTest(
		"test_top_zero",
		"$top=0 returns empty collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=0")
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

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) != 0 {
				return fmt.Errorf("expected 0 items, got %d", len(value))
			}

			return nil
		},
	)

	// Test 4: Combine $skip and $top for paging
	suite.AddTest(
		"test_skip_top_paging",
		"Combine $skip and $top for paging",
		func(ctx *framework.TestContext) error {
			// First get all products to understand the collection
			allResp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if allResp.StatusCode != 200 {
				return fmt.Errorf("failed to get all products: status %d", allResp.StatusCode)
			}

			var allResult map[string]interface{}
			if err := json.Unmarshal(allResp.Body, &allResult); err != nil {
				return fmt.Errorf("failed to parse all products JSON: %w", err)
			}

			allValue, ok := allResult["value"].([]interface{})
			if !ok {
				return fmt.Errorf("all products response missing 'value' array")
			}
			totalCount := len(allValue)

			// Test $skip=2&$top=3
			resp, err := ctx.GET("/Products?$skip=2&$top=3")
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

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			count := len(value)

			// Should return at most 3 items
			if count > 3 {
				return fmt.Errorf("returned %d items, expected max 3", count)
			}

			// Calculate expected max: min(3, total - 2)
			expectedMax := totalCount - 2
			if expectedMax > 3 {
				expectedMax = 3
			}
			if expectedMax < 0 {
				expectedMax = 0
			}

			// Verify count is reasonable
			if count > expectedMax {
				return fmt.Errorf("returned %d items, expected max %d (total=%d, skip=2, top=3)", count, expectedMax, totalCount)
			}

			return nil
		},
	)

	return suite
}
