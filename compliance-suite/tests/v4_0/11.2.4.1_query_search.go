package v4_0

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QuerySearch creates the 11.2.4.1 System Query Option $search test suite
func QuerySearch() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.4.1 System Query Option $search",
		"Tests $search query option for free-text search according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionsearch",
	)

	// Test 1: Basic $search query
	suite.AddTest(
		"test_basic_search",
		"Basic $search query with single term",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$search=Laptop")
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected status 200 but received %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			return nil
		},
	)

	// Test 2: $search with multiple terms (AND)
	suite.AddTest(
		"test_search_multiple_terms",
		"$search with multiple terms (implicit AND)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$search=Laptop Pro")
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected status 200 but received %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			return nil
		},
	)

	// Test 3: $search with OR operator
	// This test verifies that OR is treated as a boolean operator, not as a literal search term.
	// Strategy: search for each term individually, then for "A OR B", and assert that the
	// OR result is the union of the two individual results (count ≥ max of the two, and
	// count = count(A) + count(B) when the sets are disjoint).
	suite.AddTest(
		"test_search_or_operator",
		"$search with OR operator returns results matching either term",
		func(ctx *framework.TestContext) error {
			countResults := func(resp *framework.HTTPResponse) (int, error) {
				var result map[string]interface{}
				if err := json.Unmarshal(resp.Body, &result); err != nil {
					return 0, fmt.Errorf("failed to parse JSON: %w", err)
				}
				arr, ok := result["value"].([]interface{})
				if !ok {
					return 0, fmt.Errorf("response missing 'value' array")
				}
				return len(arr), nil
			}

			// "Laptop" matches "Laptop" and "Premium Laptop Pro" → 2 results
			respLaptop, err := ctx.GET("/Products?$search=Laptop")
			if err != nil {
				return fmt.Errorf("GET /Products?$search=Laptop: %w", err)
			}
			if respLaptop.StatusCode != http.StatusOK {
				return fmt.Errorf("GET /Products?$search=Laptop: expected 200, got %d", respLaptop.StatusCode)
			}
			countLaptop, err := countResults(respLaptop)
			if err != nil {
				return err
			}
			if countLaptop == 0 {
				return fmt.Errorf("$search=Laptop returned 0 results; test data may be missing")
			}

			// "Mouse" matches "Wireless Mouse" and "Gaming Mouse Ultra" → 2 results
			respMouse, err := ctx.GET("/Products?$search=Mouse")
			if err != nil {
				return fmt.Errorf("GET /Products?$search=Mouse: %w", err)
			}
			if respMouse.StatusCode != http.StatusOK {
				return fmt.Errorf("GET /Products?$search=Mouse: expected 200, got %d", respMouse.StatusCode)
			}
			countMouse, err := countResults(respMouse)
			if err != nil {
				return err
			}
			if countMouse == 0 {
				return fmt.Errorf("$search=Mouse returned 0 results; test data may be missing")
			}

			// "Laptop OR Mouse" must return both sets (these terms are disjoint in the seed data)
			respOr, err := ctx.GET("/Products?$search=Laptop OR Mouse")
			if err != nil {
				return fmt.Errorf("GET /Products?$search=Laptop OR Mouse: %w", err)
			}
			if respOr.StatusCode != http.StatusOK {
				return fmt.Errorf("GET /Products?$search=Laptop OR Mouse: expected 200, got %d", respOr.StatusCode)
			}
			countOr, err := countResults(respOr)
			if err != nil {
				return err
			}

			// The OR result must be at least as large as either individual result
			if countOr < countLaptop {
				return fmt.Errorf("OR result (%d) is smaller than Laptop-only result (%d); OR is not working correctly", countOr, countLaptop)
			}
			if countOr < countMouse {
				return fmt.Errorf("OR result (%d) is smaller than Mouse-only result (%d); OR is not working correctly", countOr, countMouse)
			}
			// Since the two terms are disjoint in the seed data, the OR count must equal the sum
			expectedOr := countLaptop + countMouse
			if countOr != expectedOr {
				return fmt.Errorf("OR result count = %d, expected %d (Laptop=%d + Mouse=%d); OR is not returning the union of results", countOr, expectedOr, countLaptop, countMouse)
			}

			return nil
		},
	)

	// Test 4: Combine $search with $filter
	suite.AddTest(
		"test_search_with_filter",
		"Combine $search with $filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$search=Laptop&$filter=Price gt 100")
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("expected status 200 but received %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			return nil
		},
	)

	return suite
}
