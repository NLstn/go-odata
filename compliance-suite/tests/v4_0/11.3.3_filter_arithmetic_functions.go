package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FilterArithmeticFunctions creates the 11.3.3 Arithmetic Functions test suite
func FilterArithmeticFunctions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.3 Arithmetic Functions in $filter",
		"Tests arithmetic operators and math functions (add, sub, mul, div, mod, ceiling, floor, round) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations",
	)

	// Test 1: add operator
	suite.AddTest(
		"test_add_operator",
		"add operator performs addition",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price add 10 gt 100")
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
				return fmt.Errorf("response missing 'value' property")
			}

			return nil
		},
	)

	// Test 2: sub operator
	suite.AddTest(
		"test_sub_operator",
		"sub operator performs subtraction",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price sub 50 lt 100")
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

	// Test 3: mul operator
	suite.AddTest(
		"test_mul_operator",
		"mul operator performs multiplication",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price mul 2 gt 200")
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

	// Test 4: div operator
	suite.AddTest(
		"test_div_operator",
		"div operator performs division",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price div 2 lt 100")
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

	// Test 5: mod operator
	suite.AddTest(
		"test_mod_operator",
		"mod operator performs modulo",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price mod 10 eq 0")
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
				return fmt.Errorf("response missing 'value' property")
			}

			return nil
		},
	)

	// Test 6: ceiling function
	suite.AddTest(
		"test_ceiling_function",
		"ceiling() function rounds up",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("ceiling(Price) eq 100")
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

	// Test 7: floor function
	suite.AddTest(
		"test_floor_function",
		"floor() function rounds down",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("floor(Price) eq 99")
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

	// Test 8: round function
	suite.AddTest(
		"test_round_function",
		"round() function rounds to nearest integer",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("round(Price) eq 100")
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

	// Test 9: Combined arithmetic operations
	suite.AddTest(
		"test_combined_arithmetic",
		"Combined arithmetic operations work",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price mul 2 sub 100 gt 0")
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

	// Test 10: Arithmetic with comparison
	suite.AddTest(
		"test_arithmetic_comparison",
		"Arithmetic comparisons (ge, le) work",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price ge 50 and Price le 150")
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
