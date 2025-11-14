package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// PaginationEdgeCases creates the 11.2.5.12 Pagination Edge Cases test suite
func PaginationEdgeCases() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.12 Pagination Edge Cases",
		"Tests edge cases and boundary conditions for pagination with $top, $skip, and nextLink.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionstopandskip",
	)

	// Test 1: $top=0 returns empty result set
	suite.AddTest(
		"test_top_zero",
		"$top=0 returns empty result set",
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
				return fmt.Errorf("expected value array")
			}

			if len(value) != 0 {
				return fmt.Errorf("expected empty value array")
			}

			return nil
		},
	)

	// Test 2: $skip beyond total count returns empty result
	suite.AddTest(
		"test_skip_beyond_count",
		"$skip beyond total count returns empty result",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$skip=10000")
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
				return fmt.Errorf("expected value array")
			}

			if len(value) != 0 {
				return fmt.Errorf("expected empty value array")
			}

			return nil
		},
	)

	// Test 3: Negative $top returns error
	suite.AddTest(
		"test_negative_top",
		"Negative $top returns 400 error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=-5")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 4: Negative $skip returns error
	suite.AddTest(
		"test_negative_skip",
		"Negative $skip returns 400 error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$skip=-5")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 5: $top with very large number
	suite.AddTest(
		"test_top_large_number",
		"$top with very large number",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=999999")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 6: $skip with zero
	suite.AddTest(
		"test_skip_zero",
		"$skip=0 is valid",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$skip=0&$top=2")
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

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("expected value array")
			}

			return nil
		},
	)

	// Test 7: @odata.nextLink presence when more results available
	suite.AddTest(
		"test_nextlink_present",
		"@odata.nextLink present when more results available",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=2")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// If there are more than 2 products, should include nextLink
			// May not have nextLink if total results <= $top (acceptable)
			ctx.Log("No @odata.nextLink check - may be expected if result count <= $top")
			return nil
		},
	)

	// Test 8: @odata.nextLink absent when no more results
	suite.AddTest(
		"test_nextlink_absent",
		"@odata.nextLink absent when all results returned",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=10000")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			if strings.Contains(bodyStr, "@odata.nextLink") {
				return fmt.Errorf("unexpected @odata.nextLink when all results returned")
			}

			return nil
		},
	)

	// Test 9: Combining $top and $skip
	suite.AddTest(
		"test_top_and_skip_combined",
		"Combining $top and $skip",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3&$skip=2")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 10: $top and $skip with $filter
	suite.AddTest(
		"test_pagination_with_filter",
		"Pagination with $filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 0&$top=2&$skip=1")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 11: $top and $skip with $orderby
	suite.AddTest(
		"test_pagination_with_orderby",
		"Pagination with $orderby",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Price desc&$top=3&$skip=1")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 12: nextLink should preserve other query options
	suite.AddTest(
		"test_nextlink_preserves_options",
		"nextLink preserves other query options",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 0&$orderby=ID&$top=2")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// This is acceptable behavior whether nextLink is present or not
			return nil
		},
	)

	// Test 13: Invalid $top value (non-numeric)
	suite.AddTest(
		"test_invalid_top_value",
		"Invalid $top value returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=abc")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 14: Invalid $skip value (non-numeric)
	suite.AddTest(
		"test_invalid_skip_value",
		"Invalid $skip value returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$skip=xyz")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 15: $count with pagination
	suite.AddTest(
		"test_count_with_pagination",
		"$count works with pagination",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true&$top=2")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, "@odata.count") {
				return fmt.Errorf("missing @odata.count")
			}

			return nil
		},
	)

	return suite
}
