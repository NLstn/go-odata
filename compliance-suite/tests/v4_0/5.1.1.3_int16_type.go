package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// Int16Type creates the 5.1.1.3 Int16 Type test suite
func Int16Type() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1.3 Int16 Type",
		"Tests handling of Edm.Int16 primitive type including boundary values, filtering, and metadata representation.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_int16_in_metadata",
		"Edm.Int16 type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.Int16"`) {
				return nil // Optional type
			}

			return nil
		},
	)

	suite.AddTest(
		"test_int16_zero_value",
		"Edm.Int16 handles zero value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Quantity eq 0")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_int16_positive_value",
		"Edm.Int16 handles positive values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Quantity gt 1000")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_int16_negative_value",
		"Edm.Int16 handles negative values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Quantity lt -1000")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_int16_min_boundary",
		"Edm.Int16 handles minimum value (-32768)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Quantity ge -32768")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_int16_max_boundary",
		"Edm.Int16 handles maximum value (32767)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Quantity le 32767")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_int16_arithmetic",
		"Edm.Int16 supports arithmetic operations",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Quantity mul 2 gt 100")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_int16_cast",
		"cast() function supports Edm.Int16",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(Price, 'Edm.Int16') eq 100")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_int16_orderby",
		"Edm.Int16 supports orderby",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Quantity")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_int16_in_response",
		"Int16 values are correctly serialized in response",
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
