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

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			return ensureComputedProperties(entities, "PriceWithTax")
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

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			return ensureComputedProperties(entities, "UpperName")
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

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			return ensureComputedProperties(entities, "DoublePrice")
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

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			return ensureComputedProperties(entities, "PriceWithTax")
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

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			return ensureComputedProperties(entities, "HalfPrice")
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

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			return ensureComputedProperties(entities, "WithTax", "Discounted")
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

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			return ensureComputedProperties(entities, "CreatedYear")
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

			if resp.StatusCode == 400 {
				return nil
			}
			if resp.StatusCode == 200 {
				return framework.NewError("Invalid syntax accepted")
			}

			return framework.NewError(fmt.Sprintf("Expected 400 but got %d", resp.StatusCode))
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

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			return ensureComputedProperties(entities, "Location")
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

			if err := requireStatusOK(resp); err != nil {
				return err
			}

			entities, err := decodeCollection(resp)
			if err != nil {
				return err
			}

			for i, entity := range entities {
				categoryRaw, ok := entity["Category"]
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing expanded Category", i))
				}

				category, ok := categoryRaw.(map[string]interface{})
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d has invalid Category payload", i))
				}

				if _, ok := category["DoubleID"]; !ok {
					return framework.NewError(fmt.Sprintf("entity %d expanded Category missing computed property \"DoubleID\"", i))
				}
			}

			return nil
		},
	)

	return suite
}
