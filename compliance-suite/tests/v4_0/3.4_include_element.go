package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// IncludeElement creates the 3.4 Element edmx:Include test suite
func IncludeElement() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"3.4 Element edmx:Include",
		"Validates edmx:Include elements that include schemas from referenced documents according to the OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752504",
	)

	// Test: Validate edmx:Include element structure if present
	suite.AddTest(
		"test_include_element_structure",
		"edmx:Include elements have correct structure if present",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			// Check 1: Status code must be 200
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)

			// Check 2: edmx:Include is optional - test passes if not present
			if !strings.Contains(body, "<edmx:Include") {
				return nil
			}

			// Includes exist - validate their structure

			// Check 3: Includes MUST have Namespace attribute
			if !strings.Contains(body, "<edmx:Include") || !strings.Contains(body, "Namespace=") {
				return framework.NewError("edmx:Include elements must have Namespace attribute")
			}

			// Check 4: Namespace attributes must not be empty
			if strings.Contains(body, `Namespace=""`) {
				return framework.NewError("Namespace attribute must not be empty")
			}

			// Check 5: If Alias is present, it must not be empty
			if strings.Contains(body, `<edmx:Include`) && strings.Contains(body, `Alias=""`) {
				return framework.NewError("Alias attribute must not be empty if specified")
			}

			return nil
		},
	)

	return suite
}
