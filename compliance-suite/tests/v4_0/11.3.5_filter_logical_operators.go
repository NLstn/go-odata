package v4_0

import (
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

func fetchLogicalOperatorItems(ctx *framework.TestContext, filterExpr string) ([]map[string]interface{}, error) {
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

func numericField(item map[string]interface{}, key string) (float64, error) {
	v, ok := item[key].(float64)
	if !ok {
		return 0, fmt.Errorf("item missing %s field or %s is not numeric", key, key)
	}
	return v, nil
}

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
			items, err := fetchLogicalOperatorItems(ctx, "Price gt 10 and Price lt 100")
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("AND filter returned no items: %w", err)
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price gt 10 and Price lt 100", func(item map[string]interface{}) (bool, string) {
				price, err := numericField(item, "Price")
				if err != nil {
					return false, err.Error()
				}
				if !(price > 10 && price < 100) {
					return false, fmt.Sprintf("Price=%.2f does not satisfy Price gt 10 and Price lt 100", price)
				}
				return true, ""
			})
		},
	)

	// Test 2: OR operator
	suite.AddTest(
		"test_or_operator",
		"OR operator works in filter expressions",
		func(ctx *framework.TestContext) error {
			items, err := fetchLogicalOperatorItems(ctx, "Price lt 10 or Price gt 100")
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("OR filter returned no items: %w", err)
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price lt 10 or Price gt 100", func(item map[string]interface{}) (bool, string) {
				price, err := numericField(item, "Price")
				if err != nil {
					return false, err.Error()
				}
				if !(price < 10 || price > 100) {
					return false, fmt.Sprintf("Price=%.2f does not satisfy Price lt 10 or Price gt 100", price)
				}
				return true, ""
			})
		},
	)

	// Test 3: NOT operator
	suite.AddTest(
		"test_not_operator",
		"NOT operator works in filter expressions",
		func(ctx *framework.TestContext) error {
			items, err := fetchLogicalOperatorItems(ctx, "not (Price gt 50)")
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("NOT filter returned no items: %w", err)
			}
			return ctx.AssertAllEntitiesSatisfy(items, "not (Price gt 50)", func(item map[string]interface{}) (bool, string) {
				price, err := numericField(item, "Price")
				if err != nil {
					return false, err.Error()
				}
				if price > 50 {
					return false, fmt.Sprintf("Price=%.2f violates not (Price gt 50)", price)
				}
				return true, ""
			})
		},
	)

	// Test 4: Complex expression with AND and OR
	suite.AddTest(
		"test_complex_and_or",
		"Complex expression with AND and OR",
		func(ctx *framework.TestContext) error {
			items, err := fetchLogicalOperatorItems(ctx, "(Price lt 10 or Price gt 100) and Status eq 9")
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("complex AND/OR filter returned no items: %w", err)
			}
			return ctx.AssertAllEntitiesSatisfy(items, "(Price lt 10 or Price gt 100) and Status eq 9", func(item map[string]interface{}) (bool, string) {
				price, err := numericField(item, "Price")
				if err != nil {
					return false, err.Error()
				}
				status, err := numericField(item, "Status")
				if err != nil {
					return false, err.Error()
				}
				if !((price < 10 || price > 100) && status == 9) {
					return false, fmt.Sprintf("Price=%.2f Status=%.0f does not satisfy expression", price, status)
				}
				return true, ""
			})
		},
	)

	// Test 5: Multiple AND operators
	suite.AddTest(
		"test_multiple_and",
		"Multiple AND operators chain correctly",
		func(ctx *framework.TestContext) error {
			items, err := fetchLogicalOperatorItems(ctx, "Price gt 10 and Price lt 100 and Status eq 1")
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("multiple AND filter returned no items: %w", err)
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price gt 10 and Price lt 100 and Status eq 1", func(item map[string]interface{}) (bool, string) {
				price, err := numericField(item, "Price")
				if err != nil {
					return false, err.Error()
				}
				status, err := numericField(item, "Status")
				if err != nil {
					return false, err.Error()
				}
				if !(price > 10 && price < 100 && status == 1) {
					return false, fmt.Sprintf("Price=%.2f Status=%.0f does not satisfy expression", price, status)
				}
				return true, ""
			})
		},
	)

	// Test 6: Multiple OR operators
	suite.AddTest(
		"test_multiple_or",
		"Multiple OR operators chain correctly",
		func(ctx *framework.TestContext) error {
			items, err := fetchLogicalOperatorItems(ctx, "Status eq 1 or Status eq 2 or Status eq 3")
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("multiple OR filter returned no items: %w", err)
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Status eq 1 or Status eq 2 or Status eq 3", func(item map[string]interface{}) (bool, string) {
				status, err := numericField(item, "Status")
				if err != nil {
					return false, err.Error()
				}
				if !(status == 1 || status == 2 || status == 3) {
					return false, fmt.Sprintf("Status=%.0f does not satisfy expression", status)
				}
				return true, ""
			})
		},
	)

	// Test 7: NOT with AND
	suite.AddTest(
		"test_not_with_and",
		"NOT with AND expression",
		func(ctx *framework.TestContext) error {
			items, err := fetchLogicalOperatorItems(ctx, "not (Price gt 50 and Status eq 1)")
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("NOT with AND filter returned no items: %w", err)
			}
			return ctx.AssertAllEntitiesSatisfy(items, "not (Price gt 50 and Status eq 1)", func(item map[string]interface{}) (bool, string) {
				price, err := numericField(item, "Price")
				if err != nil {
					return false, err.Error()
				}
				status, err := numericField(item, "Status")
				if err != nil {
					return false, err.Error()
				}
				if price > 50 && status == 1 {
					return false, fmt.Sprintf("Price=%.2f Status=%.0f violates expression", price, status)
				}
				return true, ""
			})
		},
	)

	// Test 8: Parentheses for precedence
	suite.AddTest(
		"test_parentheses_precedence",
		"Parentheses control operator precedence",
		func(ctx *framework.TestContext) error {
			items, err := fetchLogicalOperatorItems(ctx, "Price gt 10 and (Status eq 1 or Status eq 2)")
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("parentheses precedence filter returned no items: %w", err)
			}
			return ctx.AssertAllEntitiesSatisfy(items, "Price gt 10 and (Status eq 1 or Status eq 2)", func(item map[string]interface{}) (bool, string) {
				price, err := numericField(item, "Price")
				if err != nil {
					return false, err.Error()
				}
				status, err := numericField(item, "Status")
				if err != nil {
					return false, err.Error()
				}
				if !(price > 10 && (status == 1 || status == 2)) {
					return false, fmt.Sprintf("Price=%.2f Status=%.0f does not satisfy expression", price, status)
				}
				return true, ""
			})
		},
	)

	// Test 9: NOT with OR
	suite.AddTest(
		"test_not_with_or",
		"NOT with OR expression",
		func(ctx *framework.TestContext) error {
			items, err := fetchLogicalOperatorItems(ctx, "not (Price lt 10 or Price gt 100)")
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("NOT with OR filter returned no items: %w", err)
			}
			return ctx.AssertAllEntitiesSatisfy(items, "not (Price lt 10 or Price gt 100)", func(item map[string]interface{}) (bool, string) {
				price, err := numericField(item, "Price")
				if err != nil {
					return false, err.Error()
				}
				if price < 10 || price > 100 {
					return false, fmt.Sprintf("Price=%.2f violates expression", price)
				}
				return true, ""
			})
		},
	)

	return suite
}
