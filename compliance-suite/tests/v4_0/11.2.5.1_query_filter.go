package v4_0

import (
	"fmt"
	"net/url"
	"strconv"
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

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			// Verify response structure
			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return err
			}

			value, ok := data["value"].([]interface{})
			if !ok {
				return framework.NewError("value must be an array")
			}

			// Verify the filter actually worked - should return at least 1 entity with Name='Laptop'
			if len(value) < 1 {
				return framework.NewError(fmt.Sprintf("Expected at least 1 entity, got %d entities", len(value)))
			}

			// Verify all returned entities have Name='Laptop'
			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					return framework.NewError("Entity must be an object")
				}

				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError("Entity must have Name field as string")
				}

				if name != "Laptop" {
					return framework.NewError(fmt.Sprintf("Expected Name='Laptop', got Name='%s'", name))
				}
			}

			return nil
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

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			// Verify response structure
			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return err
			}

			value, ok := data["value"].([]interface{})
			if !ok {
				return framework.NewError("value must be an array")
			}

			// Verify all returned entities have Price > 100
			if len(value) == 0 {
				return framework.NewError("No entities returned or no Price field found")
			}

			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				price, ok := entity["Price"]
				if !ok {
					return framework.NewError("Entity must have Price field")
				}

				var priceValue float64
				switch v := price.(type) {
				case float64:
					priceValue = v
				case int:
					priceValue = float64(v)
				case string:
					var err error
					priceValue, err = strconv.ParseFloat(v, 64)
					if err != nil {
						return framework.NewError(fmt.Sprintf("Failed to parse Price as float: %v", err))
					}
				}

				if priceValue <= 100 {
					return framework.NewError(fmt.Sprintf("Found entity with Price=%v which is not > 100", priceValue))
				}
			}

			return nil
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

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			// Verify response structure
			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return err
			}

			value, ok := data["value"].([]interface{})
			if !ok {
				return framework.NewError("value must be an array")
			}

			// Verify all returned entities have "Laptop" in their Name
			if len(value) == 0 {
				return framework.NewError("No entities returned - expected at least one product with 'Laptop' in name")
			}

			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					return framework.NewError("Entity must be an object")
				}

				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError("Entity must have Name field as string")
				}

				if len(name) == 0 {
					return framework.NewError("Name field is empty")
				}

				// Strictly verify that the filter actually worked - all returned names must contain "Laptop"
				if !strings.Contains(name, "Laptop") {
					return framework.NewError(fmt.Sprintf("Filter failed: found entity with Name='%s' which does not contain 'Laptop'", name))
				}
			}

			return nil
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

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			// Verify response structure
			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return err
			}

			value, ok := data["value"].([]interface{})
			if !ok {
				return framework.NewError("value must be an array")
			}

			// Verify all returned entities have 10 < Price < 1000
			if len(value) == 0 {
				return framework.NewError("No entities returned or no Price field found")
			}

			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				price, ok := entity["Price"]
				if !ok {
					return framework.NewError("Entity must have Price field")
				}

				var priceValue float64
				switch v := price.(type) {
				case float64:
					priceValue = v
				case int:
					priceValue = float64(v)
				case string:
					var err error
					priceValue, err = strconv.ParseFloat(v, 64)
					if err != nil {
						return framework.NewError(fmt.Sprintf("Failed to parse Price as float: %v", err))
					}
				}

				if priceValue <= 10 || priceValue >= 1000 {
					return framework.NewError(fmt.Sprintf("Found entity with Price=%v which is not in range (10, 1000)", priceValue))
				}
			}

			return nil
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

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			// Verify response structure
			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return err
			}

			value, ok := data["value"].([]interface{})
			if !ok {
				return framework.NewError("value must be an array")
			}

			// Verify we got at least 1 entity
			if len(value) < 1 {
				return framework.NewError(fmt.Sprintf("Expected at least 1 entity, got %d", len(value)))
			}

			// Verify all returned entities have Name='Laptop' or Name='Wireless Mouse'
			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError("Entity must have Name field")
				}

				if name != "Laptop" && name != "Wireless Mouse" {
					return framework.NewError(fmt.Sprintf("Found entity with Name='%s' which is not 'Laptop' or 'Wireless Mouse'", name))
				}
			}

			return nil
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

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			// Verify response structure
			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return err
			}

			value, ok := data["value"].([]interface{})
			if !ok {
				return framework.NewError("value must be an array")
			}

			// If no results, that's valid (no products match the criteria)
			if len(value) == 0 {
				return nil
			}

			// Verify all returned entities have 100 < Price < 1000
			for _, item := range value {
				entity, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				price, ok := entity["Price"]
				if !ok {
					return framework.NewError("Entity must have Price field")
				}

				var priceValue float64
				switch v := price.(type) {
				case float64:
					priceValue = v
				case int:
					priceValue = float64(v)
				case string:
					var err error
					priceValue, err = strconv.ParseFloat(v, 64)
					if err != nil {
						return framework.NewError(fmt.Sprintf("Failed to parse Price as float: %v", err))
					}
				}

				if priceValue <= 100 || priceValue >= 1000 {
					return framework.NewError(fmt.Sprintf("Found entity with Price=%v which is not in range (100, 1000)", priceValue))
				}
			}

			return nil
		},
	)

	return suite
}
