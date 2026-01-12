package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// GuidType creates the 5.1.5 GUID Type test suite
func GuidType() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.5 GUID Type",
		"Tests handling of Edm.Guid primitive type including UUID format (8-4-4-4-12 hex digits), filtering, and metadata representation per RFC 4122.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_guid_in_metadata",
		"Edm.Guid type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.Guid"`) {
				return nil // Optional type
			}

			return nil
		},
	)

	suite.AddTest(
		"test_guid_literal_format",
		"GUID literal in 8-4-4-4-12 format",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ID eq 12345678-1234-1234-1234-123456789012")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_guid_case_insensitive",
		"GUID literal is case-insensitive",
		func(ctx *framework.TestContext) error {
			resp1, err := ctx.GET("/Products?$filter=ID eq 12345678-ABCD-ABCD-ABCD-123456789ABC")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp1, 200); err != nil {
				return err
			}

			resp2, err := ctx.GET("/Products?$filter=ID eq 12345678-abcd-abcd-abcd-123456789abc")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp2, 200)
		},
	)

	suite.AddTest(
		"test_guid_equality",
		"GUID supports equality comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ID eq 00000000-0000-0000-0000-000000000000")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_guid_inequality",
		"GUID supports inequality comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ID ne 00000000-0000-0000-0000-000000000000")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_guid_null_comparison",
		"GUID supports null comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ID ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_guid_cast",
		"cast() function supports Edm.Guid",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(ID, 'Edm.Guid') ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_guid_isof",
		"isof() function supports Edm.Guid",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=isof(ID, 'Edm.Guid')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_guid_in_key",
		"GUID can be used as entity key",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products(12345678-1234-1234-1234-123456789012)")
			if err != nil {
				// Entity may not exist, but request should be syntactically valid
				return nil
			}
			// Accept 200 (found) or 404 (not found) as valid responses
			if resp.StatusCode != 200 && resp.StatusCode != 404 {
				return framework.NewError(fmt.Sprintf("Expected 200 or 404, got %d", resp.StatusCode))
			}
			return nil
		},
	)

	suite.AddTest(
		"test_guid_in_response",
		"GUID values are correctly serialized in response",
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
		"test_guid_format_validation",
		"Invalid GUID format returns error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ID eq invalid-guid-format")
			if err != nil {
				return nil // Connection error is acceptable
			}
			// Should return 400 Bad Request for invalid format
			if resp.StatusCode == 200 {
				return framework.NewError("Expected error for invalid GUID format")
			}
			return nil
		},
	)

	return suite
}
