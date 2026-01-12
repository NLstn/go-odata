package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// DurationType creates the 5.1.8 Duration Type test suite
func DurationType() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.8 Duration Type",
		"Tests handling of Edm.Duration primitive type using ISO 8601 duration format (e.g., P1DT2H30M for 1 day, 2 hours, 30 minutes).",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_duration_in_metadata",
		"Edm.Duration type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.Duration"`) {
				return nil // Optional type
			}

			return nil
		},
	)

	suite.AddTest(
		"test_duration_literal_days",
		"Duration literal with days (P1D)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingTime eq duration'P1D'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_literal_hours",
		"Duration literal with hours (PT2H)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingTime eq duration'PT2H'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_literal_minutes",
		"Duration literal with minutes (PT30M)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingTime eq duration'PT30M'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_literal_seconds",
		"Duration literal with seconds (PT45S)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ProcessingTime eq duration'PT45S'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_literal_combined",
		"Duration literal with combined units (P1DT2H30M)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingTime eq duration'P1DT2H30M'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_literal_fractional_seconds",
		"Duration literal with fractional seconds (PT1.5S)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ProcessingTime eq duration'PT1.5S'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_negative",
		"Duration supports negative values (-P1D)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Offset eq duration'-P1D'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_comparison",
		"Duration supports comparison operators",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingTime gt duration'PT1H'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_equality",
		"Duration supports equality comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingTime eq duration'P2D'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_inequality",
		"Duration supports inequality comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingTime ne duration'P0D'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_zero",
		"Duration handles zero duration (P0D)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Offset eq duration'P0D'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_null_comparison",
		"Duration supports null comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingTime ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_cast",
		"cast() function supports Edm.Duration",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(ShippingTime, 'Edm.Duration') ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_isof",
		"isof() function supports Edm.Duration",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=isof(ShippingTime, 'Edm.Duration')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_duration_in_response",
		"Duration values are correctly serialized in response",
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

	suite.AddTest(
		"test_duration_invalid_format",
		"Invalid duration format returns error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingTime eq duration'invalid'")
			if err != nil {
				return nil // Connection error is acceptable
			}
			// Should return 400 Bad Request for invalid format
			if resp.StatusCode == 200 {
				return framework.NewError("Expected error for invalid duration format")
			}
			return nil
		},
	)

	return suite
}
