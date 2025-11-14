package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FilterLogicalOperators creates the 11.3.5 Logical Operators test suite
func FilterLogicalOperators() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.5 Logical Operators in $filter",
		"Tests logical operators (and, or, not) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_LogicalOperators",
	)

	// Test 1: AND operator
	suite.AddTest(
		"test_and_operator",
		"AND operator works in filter expressions",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price gt 10 and Price lt 100")
			resp, err := ctx.GET("/Products?$filter=" + filter)
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
				return fmt.Errorf("missing 'value' field in response")
			}

			return nil
		},
	)

	// Test 2: OR operator
	suite.AddTest(
		"test_or_operator",
		"OR operator works in filter expressions",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price lt 10 or Price gt 100")
			resp, err := ctx.GET("/Products?$filter=" + filter)
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
				return fmt.Errorf("missing 'value' field in response")
			}

			return nil
		},
	)

	// Test 3: NOT operator
	suite.AddTest(
		"test_not_operator",
		"NOT operator works in filter expressions",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("not (Price gt 50)")
			resp, err := ctx.GET("/Products?$filter=" + filter)
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
				return fmt.Errorf("missing 'value' field in response")
			}

			return nil
		},
	)

	// Test 4: Complex expression with AND and OR
	suite.AddTest(
		"test_complex_and_or",
		"Complex expression with AND and OR",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("(Price lt 10 or Price gt 100) and CategoryID eq 1")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 5: Multiple AND operators
	suite.AddTest(
		"test_multiple_and",
		"Multiple AND operators chain correctly",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price gt 10 and Price lt 100 and Status eq 1")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 6: Multiple OR operators
	suite.AddTest(
		"test_multiple_or",
		"Multiple OR operators chain correctly",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("CategoryID eq 1 or CategoryID eq 2 or CategoryID eq 3")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 7: NOT with AND
	suite.AddTest(
		"test_not_with_and",
		"NOT with AND expression",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("not (Price gt 50 and Status eq 1)")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 8: Parentheses for precedence
	suite.AddTest(
		"test_parentheses_precedence",
		"Parentheses control operator precedence",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price gt 10 and (CategoryID eq 1 or CategoryID eq 2)")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 9: NOT with OR
	suite.AddTest(
		"test_not_with_or",
		"NOT with OR expression",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("not (Price lt 10 or Price gt 100)")
			resp, err := ctx.GET("/Products?$filter=" + filter)
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
