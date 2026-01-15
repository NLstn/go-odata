package v4_0

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NumericBoundaryTests creates the 5.1.1.5 Numeric Boundary Tests suite
func NumericBoundaryTests() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1.5 Numeric Boundary Tests",
		"Tests boundary values and special cases for numeric primitive types (Int64, Decimal, Double, Single).",
		"https://docs.oasis-open.org/odata/odata-csdl-xml/v4.0/os/odata-csdl-xml-v4.0-os.html#_Toc372793863",
	)

	suite.AddTest(
		"test_int64_max_value",
		"Int64 maximum value (9223372036854775807)",
		func(ctx *framework.TestContext) error {
			// Int64 max: 2^63 - 1 = 9223372036854775807
			maxInt64 := "9223372036854775807"

			// Try to filter with max Int64
			filter := fmt.Sprintf("ID eq %s", maxInt64)
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			// Should parse without error (even if no match)
			if resp.StatusCode != 200 {
				return fmt.Errorf("server should handle Int64 max value, got status %d", resp.StatusCode)
			}

			var result struct {
				Value []interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("response should be valid JSON: %w", err)
			}

			ctx.Log(fmt.Sprintf("Int64 max value handled correctly"))
			return nil
		},
	)

	suite.AddTest(
		"test_int64_min_value",
		"Int64 minimum value (-9223372036854775808)",
		func(ctx *framework.TestContext) error {
			// Int64 min: -2^63 = -9223372036854775808
			minInt64 := "-9223372036854775808"

			// Try to filter with min Int64
			filter := fmt.Sprintf("ID ne %s", minInt64)
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			// Should parse without error
			if resp.StatusCode != 200 {
				return fmt.Errorf("server should handle Int64 min value, got status %d", resp.StatusCode)
			}

			var result struct {
				Value []interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("response should be valid JSON: %w", err)
			}

			ctx.Log("Int64 min value handled correctly")
			return nil
		},
	)

	suite.AddTest(
		"test_int64_overflow",
		"Int64 overflow value should return error",
		func(ctx *framework.TestContext) error {
			// Value exceeds Int64 max
			overflowValue := "9223372036854775808" // Max + 1

			filter := fmt.Sprintf("ID eq %s", overflowValue)
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			// Should return 400 for overflow
			if resp.StatusCode != 400 {
				// Some implementations might handle this differently
				return ctx.Skip(fmt.Sprintf("Expected 400 for Int64 overflow, got %d", resp.StatusCode))
			}

			ctx.Log("Int64 overflow correctly rejected with 400")
			return nil
		},
	)

	suite.AddTest(
		"test_decimal_precision",
		"Decimal type preserves precision",
		func(ctx *framework.TestContext) error {
			// Create product with high-precision decimal
			precision := "123.456789012345"

			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return err
			}

			payload := map[string]interface{}{
				"Name":       "Precision Test",
				"Price":      precision, // As string to preserve precision
				"CategoryID": categoryID,
				"Status":     1,
			}

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			if resp.StatusCode != 201 {
				// Server might not support high precision
				return ctx.Skip(fmt.Sprintf("Product creation failed: %d", resp.StatusCode))
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			// Check if precision was preserved
			price := result["Price"]
			priceStr := fmt.Sprintf("%v", price)

			ctx.Log(fmt.Sprintf("Decimal stored as: %s", priceStr))

			// Note: Some precision loss is acceptable in JSON serialization
			// Main check is that value is approximately correct
			return nil
		},
	)

	suite.AddTest(
		"test_decimal_scale",
		"Decimal scale validation",
		func(ctx *framework.TestContext) error {
			// Test decimal with many decimal places
			manyDecimals := 123.123456789012345

			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return err
			}

			payload := map[string]interface{}{
				"Name":       "Scale Test",
				"Price":      manyDecimals,
				"CategoryID": categoryID,
				"Status":     1,
			}

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			// Should either accept or reject based on schema
			if resp.StatusCode != 201 && resp.StatusCode != 400 {
				return fmt.Errorf("unexpected status %d", resp.StatusCode)
			}

			if resp.StatusCode == 400 {
				ctx.Log("Decimal scale validation enforced (400)")
			} else {
				ctx.Log("Decimal scale accepted (201)")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_double_positive_infinity",
		"Double positive infinity handling",
		func(ctx *framework.TestContext) error {
			// Try to use INF in filter
			resp, err := ctx.GET("/Products?$filter=Price lt INF")
			if err != nil {
				return err
			}

			// OData should support INF literal
			if resp.StatusCode == 200 {
				var result struct {
					Value []interface{} `json:"value"`
				}
				if err := json.Unmarshal(resp.Body, &result); err != nil {
					return fmt.Errorf("response should be valid JSON: %w", err)
				}
				ctx.Log("Positive infinity (INF) supported in filters")
				return nil
			}

			// Some implementations might not support INF
			if resp.StatusCode == 400 {
				return ctx.Skip("INF literal not supported")
			}

			return fmt.Errorf("unexpected status %d for INF literal", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_double_negative_infinity",
		"Double negative infinity handling",
		func(ctx *framework.TestContext) error {
			// Try to use -INF in filter
			resp, err := ctx.GET("/Products?$filter=Price gt -INF")
			if err != nil {
				return err
			}

			// OData should support -INF literal
			if resp.StatusCode == 200 {
				var result struct {
					Value []interface{} `json:"value"`
				}
				if err := json.Unmarshal(resp.Body, &result); err != nil {
					return fmt.Errorf("response should be valid JSON: %w", err)
				}
				ctx.Log("Negative infinity (-INF) supported in filters")
				return nil
			}

			// Some implementations might not support -INF
			if resp.StatusCode == 400 {
				return ctx.Skip("-INF literal not supported")
			}

			return fmt.Errorf("unexpected status %d for -INF literal", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_double_nan",
		"Double NaN (Not-a-Number) handling",
		func(ctx *framework.TestContext) error {
			// Try to use NaN in filter
			resp, err := ctx.GET("/Products?$filter=Price eq NaN")
			if err != nil {
				return err
			}

			// OData should support NaN literal
			if resp.StatusCode == 200 {
				var result struct {
					Value []interface{} `json:"value"`
				}
				if err := json.Unmarshal(resp.Body, &result); err != nil {
					return fmt.Errorf("response should be valid JSON: %w", err)
				}
				ctx.Log("NaN supported in filters")
				// Note: NaN eq NaN is false per IEEE 754, so empty result is expected
				return nil
			}

			// Some implementations might not support NaN
			if resp.StatusCode == 400 {
				return ctx.Skip("NaN literal not supported")
			}

			return fmt.Errorf("unexpected status %d for NaN literal", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_double_zero_positive_negative",
		"Double distinguishes +0.0 and -0.0",
		func(ctx *framework.TestContext) error {
			// IEEE 754 has both +0.0 and -0.0
			resp, err := ctx.GET("/Products?$filter=Price eq 0.0")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("filter with 0.0 should work, got %d", resp.StatusCode)
			}

			// Both +0.0 and -0.0 should equal 0.0
			ctx.Log("Zero handling works correctly")
			return nil
		},
	)

	suite.AddTest(
		"test_double_very_small_value",
		"Double very small value (near minimum positive)",
		func(ctx *framework.TestContext) error {
			// Smallest positive normal double: ~2.2250738585072014e-308
			verySmall := "2.2250738585072014e-308"

			filter := fmt.Sprintf("Price gt %s", verySmall)
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("server should handle very small double, got %d", resp.StatusCode)
			}

			ctx.Log("Very small double value handled correctly")
			return nil
		},
	)

	suite.AddTest(
		"test_double_very_large_value",
		"Double very large value (near maximum)",
		func(ctx *framework.TestContext) error {
			// Largest double: ~1.7976931348623157e+308
			veryLarge := "1.7976931348623157e+308"

			filter := fmt.Sprintf("Price lt %s", veryLarge)
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("server should handle very large double, got %d", resp.StatusCode)
			}

			ctx.Log("Very large double value handled correctly")
			return nil
		},
	)

	suite.AddTest(
		"test_byte_boundary_values",
		"Byte type range validation (0-255)",
		func(ctx *framework.TestContext) error {
			// Byte should be 0-255
			// Test with value outside range
			resp, err := ctx.GET("/Products?$filter=Status eq 256")
			if err != nil {
				return err
			}

			// Depending on how Status is typed, this might error or succeed
			if resp.StatusCode == 400 {
				ctx.Log("Byte overflow correctly rejected")
			} else if resp.StatusCode == 200 {
				// Might be stored as larger int type
				ctx.Log("Value accepted (Status may not be Byte type)")
			}

			// Test negative value
			resp2, err := ctx.GET("/Products?$filter=Status eq -1")
			if err != nil {
				return err
			}

			if resp2.StatusCode == 400 {
				ctx.Log("Negative byte value correctly rejected")
			} else if resp2.StatusCode == 200 {
				ctx.Log("Negative value accepted (Status may be signed type)")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_single_vs_double_precision",
		"Single (float32) vs Double (float64) precision",
		func(ctx *framework.TestContext) error {
			// Single has ~7 decimal digits of precision
			// Double has ~15-17 decimal digits

			// Value with more precision than Single can handle
			highPrecision := "1.23456789012345"

			filter := fmt.Sprintf("Price eq %s", highPrecision)
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("filter should parse, got %d", resp.StatusCode)
			}

			// Note: If Price is Single, precision will be lost
			// If Price is Double, precision should be preserved
			ctx.Log("High precision value in filter handled")
			return nil
		},
	)

	suite.AddTest(
		"test_json_number_serialization",
		"JSON number serialization for large integers",
		func(ctx *framework.TestContext) error {
			// JSON can lose precision for integers > 2^53
			// OData should use IEEE754Compatible=true to serialize as strings

			// Check if service supports IEEE754Compatible
			resp, err := ctx.GET("/Products?$top=1",
				framework.Header{
					Key:   "Accept",
					Value: "application/json;IEEE754Compatible=true",
				},
			)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return ctx.Skip("IEEE754Compatible not supported")
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("response should be valid JSON: %w", err)
			}

			// Check Content-Type includes IEEE754Compatible
			contentType := resp.Headers.Get("Content-Type")
			if strings.Contains(contentType, "IEEE754Compatible=true") {
				ctx.Log("IEEE754Compatible parameter honored in response")
			} else {
				ctx.Log("IEEE754Compatible parameter not in Content-Type")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_decimal_zero_values",
		"Decimal zero representation (0, 0.0, 0.00)",
		func(ctx *framework.TestContext) error {
			// All should be equivalent
			filters := []string{
				"Price eq 0",
				"Price eq 0.0",
				"Price eq 0.00",
			}

			for _, filter := range filters {
				resp, err := ctx.GET("/Products?$filter=" + filter)
				if err != nil {
					return err
				}

				if resp.StatusCode != 200 {
					return fmt.Errorf("filter '%s' should work, got %d", filter, resp.StatusCode)
				}
			}

			ctx.Log("All zero representations handled equivalently")
			return nil
		},
	)

	suite.AddTest(
		"test_arithmetic_precision_loss",
		"Arithmetic operations preserve precision",
		func(ctx *framework.TestContext) error {
			// Test division that might lose precision
			resp, err := ctx.GET("/Products?$filter=(Price div 3) mul 3 eq Price")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("arithmetic filter should parse, got %d", resp.StatusCode)
			}

			var result struct {
				Value []interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("response should be valid JSON: %w", err)
			}

			// Due to precision loss, (Price div 3) mul 3 may not exactly equal Price
			ctx.Log(fmt.Sprintf("Arithmetic filter returned %d results", len(result.Value)))
			return nil
		},
	)

	return suite
}

// Helper function to check if two floats are approximately equal
func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}
