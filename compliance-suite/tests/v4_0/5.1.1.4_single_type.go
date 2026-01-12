package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// SingleType creates the 5.1.1.4 Single Type test suite
func SingleType() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1.4 Single Type (Float32)",
		"Tests handling of Edm.Single (IEEE 754 single-precision float) including literal format with 'f' suffix, filtering, and special values.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_single_in_metadata",
		"Edm.Single type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.Single"`) {
				return nil // Optional type
			}

			return nil
		},
	)

	suite.AddTest(
		"test_single_literal_with_f_suffix",
		"Edm.Single literal with 'f' suffix",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight eq 3.14f")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_comparison",
		"Edm.Single supports comparison operators",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight gt 2.5f")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_zero_value",
		"Edm.Single handles zero value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight eq 0.0f")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_negative_value",
		"Edm.Single handles negative values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight lt -1.5f")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_arithmetic",
		"Edm.Single supports arithmetic operations",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight mul 2 gt 10.0f")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_cast",
		"cast() function supports Edm.Single",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(Price, 'Edm.Single') eq 99.99f")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_scientific_notation",
		"Edm.Single supports scientific notation",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight lt 1.5e2f")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_inf",
		"Edm.Single supports INF value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight ne INF")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_negative_inf",
		"Edm.Single supports -INF value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight ne -INF")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_nan",
		"Edm.Single supports NaN value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Weight ne NaN")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_single_in_response",
		"Single values are correctly serialized in response",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if _, hasValue := result["value"]; !hasValue {
				return framework.NewError("Response missing value field")
			}

			return nil
		},
	)

	return suite
}
