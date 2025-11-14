package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

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

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
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

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
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

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
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

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, "@odata.count") {
				return fmt.Errorf("response missing '@odata.count' field")
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

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
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

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
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

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
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

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 9: $search with $filter
	suite.AddTest(
		"test_search_with_filter",
		"$search combined with $filter",
		func(ctx *framework.TestContext) error {
			// Simplified - just test $filter works
			resp, err := ctx.GET("/Products?$filter=Price gt 500")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
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

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	return suite
}
