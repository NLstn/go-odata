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
	suite.AddTest(
		"test_search_or_operator",
		"$search with OR operator",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$search=Laptop OR Phone")
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
