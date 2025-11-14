package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NumericEdgeCases creates the 5.1.1.1 Numeric Edge Cases test suite
func NumericEdgeCases() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1.1 Numeric Edge Cases",
		"Tests handling of numeric edge cases including very large numbers, precision limits, and boundary conditions.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_large_integer",
		"Very large integer values in filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Status gt 999999")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_zero_value",
		"Zero value in numeric comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price eq 0")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_negative_numbers",
		"Negative numbers in filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt -1")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_decimal_precision",
		"Decimal precision with many places",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price eq 999.9999999")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_small_decimals",
		"Very small decimal values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt 0.001")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_integer_division",
		"Integer division behavior",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Status div 2 eq 1")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_modulo_operation",
		"Modulo operation",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Status mod 10 eq 0")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_numeric_null_comparison",
		"Numeric comparison with null values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price ne null and Price gt 0")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_complex_numeric_expression",
		"Complex numeric expressions",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=(Price mul 2) gt 1000 and (Price div 10) lt 200")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_int32_max_boundary",
		"Int32 maximum boundary value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Status lt 2147483647")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_arithmetic_precision",
		"Arithmetic precision maintained",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price add 0.01 gt Price")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_numeric_ordering",
		"Numeric ordering with edge values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Price desc")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
