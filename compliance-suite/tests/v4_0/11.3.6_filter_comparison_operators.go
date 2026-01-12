package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FilterComparisonOperators creates the 11.3.6 Comparison Operators test suite
func FilterComparisonOperators() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.6 Comparison Operators in $filter",
		"Tests comparison operators (eq, ne, gt, ge, lt, le) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_ComparisonOperators",
	)

	// Test 1: eq (equals) operator
	suite.AddTest(
		"test_eq_operator",
		"eq (equals) operator works",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Status eq 1")
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

	// Test 2: ne (not equals) operator
	suite.AddTest(
		"test_ne_operator",
		"ne (not equals) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Status ne 0")
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

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("missing 'value' field in response or not an array")
			}

			// Strictly validate: all returned entities must have Status != 0
			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				if status, ok := entity["Status"]; ok {
					statusVal := int(status.(float64))
					if statusVal == 0 {
						return fmt.Errorf("filter validation failed: found entity with Status=0, but filter was 'Status ne 0'")
					}
				}
			}

			return nil
		},
	)

	// Test 3: gt (greater than) operator
	suite.AddTest(
		"test_gt_operator",
		"gt (greater than) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price gt 50")
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

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("missing 'value' field in response or not an array")
			}

			// Strictly validate: all returned entities must have Price > 50
			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				if price, ok := entity["Price"]; ok {
					priceVal := price.(float64)
					if priceVal <= 50 {
						return fmt.Errorf("filter validation failed: found entity with Price=%v, but filter was 'Price gt 50'", priceVal)
					}
				}
			}

			return nil
		},
	)

	// Test 4: ge (greater than or equal) operator
	suite.AddTest(
		"test_ge_operator",
		"ge (greater than or equal) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price ge 50")
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

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("missing 'value' field in response or not an array")
			}

			// Strictly validate: all returned entities must have Price >= 50
			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				if price, ok := entity["Price"]; ok {
					priceVal := price.(float64)
					if priceVal < 50 {
						return fmt.Errorf("filter validation failed: found entity with Price=%v, but filter was 'Price ge 50'", priceVal)
					}
				}
			}

			return nil
		},
	)

	// Test 5: lt (less than) operator
	suite.AddTest(
		"test_lt_operator",
		"lt (less than) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price lt 100")
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

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("missing 'value' field in response or not an array")
			}

			// Strictly validate: all returned entities must have Price < 100
			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				if price, ok := entity["Price"]; ok {
					priceVal := price.(float64)
					if priceVal >= 100 {
						return fmt.Errorf("filter validation failed: found entity with Price=%v, but filter was 'Price lt 100'", priceVal)
					}
				}
			}

			return nil
		},
	)

	// Test 6: le (less than or equal) operator
	suite.AddTest(
		"test_le_operator",
		"le (less than or equal) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price le 100")
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

			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("missing 'value' field in response or not an array")
			}

			// Strictly validate: all returned entities must have Price <= 100
			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				if price, ok := entity["Price"]; ok {
					priceVal := price.(float64)
					if priceVal > 100 {
						return fmt.Errorf("filter validation failed: found entity with Price=%v, but filter was 'Price le 100'", priceVal)
					}
				}
			}

			return nil
		},
	)

	// Test 7: eq with string
	suite.AddTest(
		"test_eq_string",
		"eq operator works with strings",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Name eq 'Laptop'")
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

	// Test 8: ne with string
	suite.AddTest(
		"test_ne_string",
		"ne operator works with strings",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Name ne 'Laptop'")
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

	// Test 9: Comparison with decimal numbers
	suite.AddTest(
		"test_decimal_comparison",
		"Comparison operators work with decimal numbers",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price eq 99.99")
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

	// Test 10: Comparison with null
	suite.AddTest(
		"test_null_comparison",
		"Comparison with null value",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("CategoryID eq null")
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

	// Test 11: Multiple comparisons combined
	suite.AddTest(
		"test_multiple_comparisons",
		"Multiple comparison operators combined",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price ge 10 and Price le 100")
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

	// Test 12: Invalid comparison operator returns error
	suite.AddTest(
		"test_invalid_operator",
		"Invalid comparison operator returns 400",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price equals 50")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	return suite
}
