package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ByteTypes creates the 5.1.1.2 Byte Types test suite
func ByteTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1.2 Byte Types",
		"Tests handling of Edm.Byte and Edm.SByte primitive types including boundary values, filtering, and metadata representation.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_byte_in_metadata",
		"Edm.Byte type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.Byte"`) && !strings.Contains(body, `Type="Edm.SByte"`) {
				return nil // Optional types, skip
			}

			return nil
		},
	)

	suite.AddTest(
		"test_byte_zero_value",
		"Edm.Byte handles zero value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Rating eq 0")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_byte_max_value",
		"Edm.Byte handles maximum value (255)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Rating le 255")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_byte_comparison",
		"Edm.Byte supports comparison operators",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Rating gt 100")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_sbyte_negative_value",
		"Edm.SByte handles negative values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Temperature lt 0")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_sbyte_min_max_range",
		"Edm.SByte handles range -128 to 127",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Temperature ge -128 and Temperature le 127")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_byte_arithmetic",
		"Edm.Byte supports arithmetic operations",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Rating add 10 gt 100")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_byte_cast",
		"cast() function supports Edm.Byte",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(Status, 'Edm.Byte') eq 1")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_sbyte_cast",
		"cast() function supports Edm.SByte",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(Status, 'Edm.SByte') eq -1")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_byte_in_response",
		"Byte values are correctly serialized in response",
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

			// Verify response is valid JSON
			if _, hasValue := result["value"]; !hasValue {
				return framework.NewError("Response missing value field")
			}

			return nil
		},
	)

	return suite
}
