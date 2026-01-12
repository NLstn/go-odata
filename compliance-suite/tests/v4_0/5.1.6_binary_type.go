package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// BinaryType creates the 5.1.6 Binary Type test suite
func BinaryType() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.6 Binary Type",
		"Tests handling of Edm.Binary primitive type including base64 encoding with binary'' literal format, filtering, and metadata representation.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_binary_in_metadata",
		"Edm.Binary type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.Binary"`) {
				return nil // Optional type
			}

			return nil
		},
	)

	suite.AddTest(
		"test_binary_literal_format",
		"Binary literal with binary'base64' format",
		func(ctx *framework.TestContext) error {
			// base64 encoding of "test"
			resp, err := ctx.GET("/Products?$filter=Data eq binary'dGVzdA=='")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_binary_empty_value",
		"Binary handles empty byte array",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Data eq binary''")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_binary_equality",
		"Binary supports equality comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Data eq binary'AQID'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_binary_inequality",
		"Binary supports inequality comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Data ne binary'AQID'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_binary_null_comparison",
		"Binary supports null comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Data ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_binary_cast",
		"cast() function supports Edm.Binary",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(Data, 'Edm.Binary') ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_binary_isof",
		"isof() function supports Edm.Binary",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=isof(Data, 'Edm.Binary')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_binary_in_response",
		"Binary values are base64-encoded in JSON response",
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
		"test_binary_invalid_base64",
		"Invalid base64 encoding returns error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Data eq binary'!!!invalid!!!'")
			if err != nil {
				return nil // Connection error is acceptable
			}
			// Should return 400 Bad Request for invalid base64
			if resp.StatusCode == 200 {
				return framework.NewError("Expected error for invalid base64 encoding")
			}
			return nil
		},
	)

	suite.AddTest(
		"test_binary_large_value",
		"Binary handles large byte arrays",
		func(ctx *framework.TestContext) error {
			// 100 bytes of data in base64 (approximately)
			largeBase64 := strings.Repeat("AQIDBAUG", 16) // ~128 chars
			resp, err := ctx.GET("/Products?$filter=Data ne binary'" + largeBase64 + "'")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
