package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// TypeDefinitions creates the 5.4 Type Definitions test suite
func TypeDefinitions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.4 Type Definitions",
		"Validates custom type definitions in metadata document and their proper usage.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_TypeDefinition",
	)

	suite.AddTest(
		"test_typedef_in_metadata",
		"Metadata contains valid schema",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<Schema") && !strings.Contains(body, `"$Version"`) {
				return framework.NewError("Metadata does not contain valid schema")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_typedef_underlying_type",
		"Type definitions have UnderlyingType",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "TypeDefinition") {
				return nil // No type definitions, optional
			}

			if !strings.Contains(body, "UnderlyingType") {
				return framework.NewError("TypeDefinition found but missing UnderlyingType")
			}

			return nil
		},
	)

	return suite
}
