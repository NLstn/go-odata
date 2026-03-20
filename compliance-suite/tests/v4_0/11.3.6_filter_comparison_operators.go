package v4_0

import (
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

func fetchComparisonItems(ctx *framework.TestContext, filterExpr string) ([]map[string]interface{}, error) {
	filter := url.QueryEscape(filterExpr)
	resp, err := ctx.GET("/Products?$filter=" + filter)
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}

	return ctx.ParseEntityCollection(resp)
}

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
			items, err := fetchComparisonItems(ctx, "Status eq 1")
			if err != nil {
				return err
			}
			return ctx.AssertMinCollectionSize(items, 1)
		},
	)

	// Test 2: ne (not equals) operator
	suite.AddTest(
		"test_ne_operator",
		"ne (not equals) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			items, err := fetchComparisonItems(ctx, "Status ne 0")
			if err != nil {
				return err
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Status ne 0", func(entity map[string]interface{}) (bool, string) {
				statusFloat, ok := entity["Status"].(float64)
				if !ok {
					return false, "Status field is missing or non-numeric"
				}
				if int(statusFloat) == 0 {
					return false, "found Status=0"
				}
				return true, ""
			})
		},
	)

	// Test 3: gt (greater than) operator
	suite.AddTest(
		"test_gt_operator",
		"gt (greater than) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			items, err := fetchComparisonItems(ctx, "Price gt 50")
			if err != nil {
				return err
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price gt 50", func(entity map[string]interface{}) (bool, string) {
				price, ok := entity["Price"].(float64)
				if !ok {
					return false, "Price field is missing or non-numeric"
				}
				if price <= 50 {
					return false, fmt.Sprintf("found Price=%v", price)
				}
				return true, ""
			})
		},
	)

	// Test 4: ge (greater than or equal) operator
	suite.AddTest(
		"test_ge_operator",
		"ge (greater than or equal) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			items, err := fetchComparisonItems(ctx, "Price ge 50")
			if err != nil {
				return err
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price ge 50", func(entity map[string]interface{}) (bool, string) {
				price, ok := entity["Price"].(float64)
				if !ok {
					return false, "Price field is missing or non-numeric"
				}
				if price < 50 {
					return false, fmt.Sprintf("found Price=%v", price)
				}
				return true, ""
			})
		},
	)

	// Test 5: lt (less than) operator
	suite.AddTest(
		"test_lt_operator",
		"lt (less than) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			items, err := fetchComparisonItems(ctx, "Price lt 100")
			if err != nil {
				return err
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price lt 100", func(entity map[string]interface{}) (bool, string) {
				price, ok := entity["Price"].(float64)
				if !ok {
					return false, "Price field is missing or non-numeric"
				}
				if price >= 100 {
					return false, fmt.Sprintf("found Price=%v", price)
				}
				return true, ""
			})
		},
	)

	// Test 6: le (less than or equal) operator
	suite.AddTest(
		"test_le_operator",
		"le (less than or equal) operator works and returns only matching entities",
		func(ctx *framework.TestContext) error {
			items, err := fetchComparisonItems(ctx, "Price le 100")
			if err != nil {
				return err
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price le 100", func(entity map[string]interface{}) (bool, string) {
				price, ok := entity["Price"].(float64)
				if !ok {
					return false, "Price field is missing or non-numeric"
				}
				if price > 100 {
					return false, fmt.Sprintf("found Price=%v", price)
				}
				return true, ""
			})
		},
	)

	// Test 7: eq with string
	suite.AddTest(
		"test_eq_string",
		"eq operator works with strings",
		func(ctx *framework.TestContext) error {
			items, err := fetchComparisonItems(ctx, "Name eq 'Laptop'")
			if err != nil {
				return err
			}
			return ctx.AssertMinCollectionSize(items, 1)
		},
	)

	// Test 8: ne with string
	suite.AddTest(
		"test_ne_string",
		"ne operator works with strings",
		func(ctx *framework.TestContext) error {
			items, err := fetchComparisonItems(ctx, "Name ne 'Laptop'")
			if err != nil {
				return err
			}
			return ctx.AssertMinCollectionSize(items, 1)
		},
	)

	// Test 9: Comparison with decimal numbers
	suite.AddTest(
		"test_decimal_comparison",
		"Comparison operators work with decimal numbers",
		func(ctx *framework.TestContext) error {
			_, err := fetchComparisonItems(ctx, "Price eq 99.99")
			if err != nil {
				return err
			}
			return nil
		},
	)

	// Test 10: Comparison with null
	suite.AddTest(
		"test_null_comparison",
		"Comparison with null value",
		func(ctx *framework.TestContext) error {
			_, err := fetchComparisonItems(ctx, "CategoryID eq null")
			if err != nil {
				return err
			}
			return nil
		},
	)

	// Test 11: Multiple comparisons combined
	suite.AddTest(
		"test_multiple_comparisons",
		"Multiple comparison operators combined",
		func(ctx *framework.TestContext) error {
			items, err := fetchComparisonItems(ctx, "Price ge 10 and Price le 100")
			if err != nil {
				return err
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price ge 10 and Price le 100", func(entity map[string]interface{}) (bool, string) {
				price, ok := entity["Price"].(float64)
				if !ok {
					return false, "Price field is missing or non-numeric"
				}
				if price < 10 || price > 100 {
					return false, fmt.Sprintf("found Price=%v", price)
				}
				return true, ""
			})
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
