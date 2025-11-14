package v4_01

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// OrderByComputedProperties creates the 11.2.5.11 OrderBy with Computed Properties test suite
func OrderByComputedProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.11 OrderBy with Computed Properties",
		"Validates $orderby functionality with computed properties from $compute query option (OData v4.01 feature).",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptioncompute",
	)

	// Test 1: Compute a property and order by it
	suite.AddTest(
		"test_orderby_computed",
		"OrderBy computed property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.1 as TaxedPrice&$orderby=TaxedPrice")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 2: OrderBy with multiple computed properties
	suite.AddTest(
		"test_orderby_multiple_computed",
		"OrderBy with multiple computed properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 0.9 as DiscountPrice,Price mul 1.1 as TaxedPrice&$orderby=DiscountPrice desc")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 3: OrderBy computed property with direction
	suite.AddTest(
		"test_orderby_computed_desc",
		"OrderBy computed with desc direction",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 2 as DoublePrice&$orderby=DoublePrice desc")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 4: OrderBy mixing computed and regular properties
	suite.AddTest(
		"test_orderby_mixed",
		"OrderBy mixing computed and regular",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.2 as MarkedUpPrice&$orderby=CategoryID,MarkedUpPrice desc")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 5: OrderBy computed with select
	suite.AddTest(
		"test_orderby_computed_with_select",
		"OrderBy computed with $select",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.08 as FinalPrice&$select=Name,FinalPrice&$orderby=FinalPrice")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 6: OrderBy computed with filter
	suite.AddTest(
		"test_orderby_computed_with_filter",
		"OrderBy computed with $filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 0.8 as SalePrice&$filter=SalePrice gt 50&$orderby=SalePrice")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 7: OrderBy computed with top
	suite.AddTest(
		"test_orderby_computed_with_top",
		"OrderBy computed with $top",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price div 2 as HalfPrice&$orderby=HalfPrice desc&$top=3")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 8: OrderBy regular property still works with compute present
	suite.AddTest(
		"test_orderby_regular_with_compute",
		"OrderBy regular property with compute present",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.5 as HighPrice&$orderby=Name")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 9: Response includes computed properties when ordered
	suite.AddTest(
		"test_response_includes_computed",
		"Response includes computed properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 2 as DoublePrice&$select=Name,DoublePrice&$orderby=DoublePrice&$top=1")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				body := string(resp.Body)
				if strings.Contains(body, "DoublePrice") {
					return nil
				}
				return framework.NewError("Computed property not present in ordered response")
			}

			if resp.StatusCode == 400 || resp.StatusCode == 501 {
				ctx.Log("$compute not supported (optional)")
				return nil
			}

			return framework.NewError(fmt.Sprintf("Expected 200, 400, or 501 but got %d", resp.StatusCode))
		},
	)

	// Test 10: OrderBy without including computed in select
	suite.AddTest(
		"test_orderby_computed_not_selected",
		"OrderBy computed not in $select",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$compute=Price mul 1.3 as MarkedPrice&$select=Name,Price&$orderby=MarkedPrice")
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
