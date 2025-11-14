package v4_01

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QueryCompute creates the 11.2.5.8 System Query Option $compute test suite
func QueryCompute() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.8 System Query Option $compute",
		"Validates $compute query option for adding computed properties to query results according to OData v4.01 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptioncompute",
	)

	// Test 1: Simple $compute with arithmetic
	suite.AddTest(
		"test_compute_arithmetic",
		"Simple $compute with arithmetic (OData v4.01)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.1 as PriceWithTax")
			if err != nil {
				return err
			}

			// Accept 200 (supported) or 400/501 (not supported yet)
			if resp.StatusCode == 200 {
				return nil
			}
			if resp.StatusCode == 400 || resp.StatusCode == 501 {
				ctx.Log("$compute not supported (optional OData v4.01 feature)")
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 2: $compute with string function
	suite.AddTest(
		"test_compute_string_function",
		"$compute with string function",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=toupper(Name) as UpperName")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 3: $compute with $select
	suite.AddTest(
		"test_compute_with_select",
		"$compute combined with $select",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 2 as DoublePrice&$select=Name,DoublePrice")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 4: $compute with $filter
	suite.AddTest(
		"test_compute_with_filter",
		"$compute combined with $filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.1 as PriceWithTax&$filter=PriceWithTax gt 100")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 5: $compute with $orderby
	suite.AddTest(
		"test_compute_with_orderby",
		"$compute combined with $orderby",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price div 2 as HalfPrice&$orderby=HalfPrice")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 6: Multiple computed properties
	suite.AddTest(
		"test_multiple_computed",
		"Multiple computed properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.1 as WithTax,Price mul 0.9 as Discounted")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 7: $compute with date functions
	suite.AddTest(
		"test_compute_date_functions",
		"$compute with date functions",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=year(CreatedAt) as CreatedYear")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 8: Invalid $compute syntax
	suite.AddTest(
		"test_invalid_compute_syntax",
		"Invalid $compute syntax returns error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=InvalidSyntax")
			if err != nil {
				return err
			}

			// Should return 400 when syntax is invalid, or 501 if not supported
			if resp.StatusCode == 400 {
				return nil
			}
			if resp.StatusCode == 501 {
				ctx.Log("$compute not supported (optional)")
				return nil
			}
			if resp.StatusCode == 200 {
				return framework.NewError("Invalid syntax accepted")
			}

			return framework.NewError(fmt.Sprintf("Expected 400 or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 9: $compute with nested properties
	suite.AddTest(
		"test_compute_nested_properties",
		"$compute with nested properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Address/City as Location")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 10: $compute in $expand
	suite.AddTest(
		"test_compute_in_expand",
		"$compute within $expand (advanced)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$expand=Category($compute=ID mul 2 as DoubleID)")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	return suite
}
