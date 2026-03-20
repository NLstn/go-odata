package v4_0

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QueryFilter creates the 11.2.5.1 System Query Option $filter test suite
func QueryFilter() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.1 System Query Option $filter",
		"Tests $filter query option according to OData v4 specification, including equality, comparison, and logical operators.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionfilter",
	)

	parsePrice := func(value interface{}) (float64, error) {
		switch v := value.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		default:
			return 0, fmt.Errorf("Price must be numeric, got %T", value)
		}
	}

	// Test 1: Basic eq (equals) operator with string
	suite.AddTest(
		"test_filter_eq",
		"$filter with eq operator",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Name eq 'Laptop'")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			// Verify the filter actually worked - should return at least 1 entity with Name='Laptop'
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return framework.NewError(fmt.Sprintf("Expected at least 1 entity, got %d entities", len(items)))
			}

			// Verify all returned entities have Name='Laptop'
			return ctx.AssertAllEntitiesSatisfy(items, "Name eq 'Laptop'", func(entity map[string]interface{}) (bool, string) {
				name, ok := entity["Name"].(string)
				if !ok {
					return false, "Name field is missing or not a string"
				}
				if name != "Laptop" {
					return false, fmt.Sprintf("Expected Name='Laptop', got Name='%s'", name)
				}
				return true, ""
			})
		},
	)

	// Test 2: gt (greater than) operator
	suite.AddTest(
		"test_filter_gt",
		"$filter with gt operator",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price gt 100")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			// Verify all returned entities have Price > 100
			if len(items) == 0 {
				return framework.NewError("No entities returned or no Price field found")
			}

			return ctx.AssertAllEntitiesSatisfy(items, "Price gt 100", func(entity map[string]interface{}) (bool, string) {
				priceRaw, ok := entity["Price"]
				if !ok {
					return false, "Entity must have Price field"
				}
				priceValue, err := parsePrice(priceRaw)
				if err != nil {
					return false, err.Error()
				}
				if priceValue <= 100 {
					return false, fmt.Sprintf("Found entity with Price=%v which is not > 100", priceValue)
				}
				return true, ""
			})
		},
	)

	// Test 3: String contains function
	suite.AddTest(
		"test_filter_contains",
		"$filter with contains() function",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("contains(Name,'Laptop')")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			// Verify all returned entities have "Laptop" in their Name
			if len(items) == 0 {
				return framework.NewError("No entities returned - expected at least one product with 'Laptop' in name")
			}

			return ctx.AssertAllEntitiesSatisfy(items, "contains(Name,'Laptop')", func(entity map[string]interface{}) (bool, string) {
				name, ok := entity["Name"].(string)
				if !ok {
					return false, "Entity must have Name field as string"
				}
				if len(name) == 0 {
					return false, "Name field is empty"
				}
				if !strings.Contains(name, "Laptop") {
					return false, fmt.Sprintf("Filter failed: found entity with Name='%s' which does not contain 'Laptop'", name)
				}
				return true, ""
			})
		},
	)

	// Test 4: Boolean operators (and)
	suite.AddTest(
		"test_filter_and",
		"$filter with 'and' operator",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price gt 10 and Price lt 1000")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			// Verify all returned entities have 10 < Price < 1000
			if len(items) == 0 {
				return framework.NewError("No entities returned or no Price field found")
			}

			return ctx.AssertAllEntitiesSatisfy(items, "Price gt 10 and Price lt 1000", func(entity map[string]interface{}) (bool, string) {
				priceRaw, ok := entity["Price"]
				if !ok {
					return false, "Entity must have Price field"
				}
				priceValue, err := parsePrice(priceRaw)
				if err != nil {
					return false, err.Error()
				}
				if priceValue <= 10 || priceValue >= 1000 {
					return false, fmt.Sprintf("Found entity with Price=%v which is not in range (10, 1000)", priceValue)
				}
				return true, ""
			})
		},
	)

	// Test 5: Boolean operators (or)
	suite.AddTest(
		"test_filter_or",
		"$filter with 'or' operator",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Name eq 'Laptop' or Name eq 'Wireless Mouse'")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			// Verify we got at least 1 entity
			if len(items) < 1 {
				return framework.NewError(fmt.Sprintf("Expected at least 1 entity, got %d", len(items)))
			}

			// Verify all returned entities have Name='Laptop' or Name='Wireless Mouse'
			return ctx.AssertAllEntitiesSatisfy(items, "Name eq 'Laptop' or Name eq 'Wireless Mouse'", func(entity map[string]interface{}) (bool, string) {
				name, ok := entity["Name"].(string)
				if !ok {
					return false, "Entity must have Name field"
				}
				if name != "Laptop" && name != "Wireless Mouse" {
					return false, fmt.Sprintf("Found entity with Name='%s' which is not 'Laptop' or 'Wireless Mouse'", name)
				}
				return true, ""
			})
		},
	)

	// Test 6: Parentheses for grouping
	suite.AddTest(
		"test_filter_parentheses",
		"$filter with parentheses",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("(Price gt 100) and (Price lt 1000)")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			// If no results, that's valid (no products match the criteria)
			if len(items) == 0 {
				return nil
			}

			// Verify all returned entities have 100 < Price < 1000
			return ctx.AssertAllEntitiesSatisfy(items, "(Price gt 100) and (Price lt 1000)", func(entity map[string]interface{}) (bool, string) {
				priceRaw, ok := entity["Price"]
				if !ok {
					return false, "Entity must have Price field"
				}
				priceValue, err := parsePrice(priceRaw)
				if err != nil {
					return false, err.Error()
				}
				if priceValue <= 100 || priceValue >= 1000 {
					return false, fmt.Sprintf("Found entity with Price=%v which is not in range (100, 1000)", priceValue)
				}
				return true, ""
			})
		},
	)

	return suite
}
