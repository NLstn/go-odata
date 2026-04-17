package v4_01

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FilterDivByOperator creates the OData 4.01 'divby' arithmetic operator test suite.
func FilterDivByOperator() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.5.1.1 divby Arithmetic Operator",
		"Validates the OData 4.01 'divby' decimal division operator in $filter expressions. "+
			"divby performs floating-point division, in contrast to 'div' which performs integer division.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_ArithmeticOperators",
	)

	// Test 1: divby returns 200 for a valid filter (OData 4.01 negotiated)
	suite.AddTest(
		"test_divby_operator_basic",
		"divby operator in $filter returns 200 when OData 4.01 is negotiated",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price divby 1.5 gt 0")
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			resp, err := ctx.GET("/Products?$filter="+filter, headers...)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	// Test 2: divby performs decimal (not integer) division
	suite.AddTest(
		"test_divby_performs_decimal_division",
		"divby performs decimal division (e.g. 3 divby 2 = 1.5, not 1)",
		func(ctx *framework.TestContext) error {
			// Price divby 2 ge 1 should match products with Price >= 2
			// (unlike integer div which would behave differently for odd prices)
			filter := url.QueryEscape("Price divby 2 ge 1")
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			resp, err := ctx.GET("/Products?$filter="+filter, headers...)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	// Test 3: divby combined with comparison and logical operators
	suite.AddTest(
		"test_divby_combined_with_and",
		"divby combined with 'and' logical operator",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price divby 2 gt 0 and Price divby 2 lt 1000")
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			resp, err := ctx.GET("/Products?$filter="+filter, headers...)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	// Test 4: divby is rejected (400) when OData 4.0 is negotiated
	suite.AddTest(
		"test_divby_version_negotiation_4_0_rejects",
		"divby operator is rejected with 400 when OData-MaxVersion 4.0 is negotiated",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("Price divby 1.5 gt 0")
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}
			resp, err := ctx.GET("/Products?$filter="+filter, headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusBadRequest); err != nil {
				return fmt.Errorf("4.0 negotiated request must reject 'divby' operator with 400: %v", err)
			}
			if err := ctx.AssertODataError(resp, http.StatusBadRequest, "not supported in OData 4.0"); err != nil {
				return fmt.Errorf("4.0 negotiated 'divby' rejection must include strict OData error payload: %v", err)
			}
			return nil
		},
	)

	return suite
}
