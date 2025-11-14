package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// IncludeAnnotationsElement creates the 3.5 Element edmx:IncludeAnnotations test suite
func IncludeAnnotationsElement() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"3.5 Element edmx:IncludeAnnotations",
		"Validates edmx:IncludeAnnotations elements for including annotations from references according to the OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752505",
	)

	// Test: Validate edmx:IncludeAnnotations element structure if present
	suite.AddTest(
		"test_includeannotations_element_structure",
		"edmx:IncludeAnnotations elements have correct structure if present",
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

			// Check 2: edmx:IncludeAnnotations is optional - test passes if not present
			if !strings.Contains(body, "<edmx:IncludeAnnotations") {
				return nil
			}

			// IncludeAnnotations exist - validate their structure

			// Check 3: MUST have TermNamespace attribute
			if !strings.Contains(body, "<edmx:IncludeAnnotations") || !strings.Contains(body, "TermNamespace=") {
				return framework.NewError("edmx:IncludeAnnotations elements must have TermNamespace attribute")
			}

			// Check 4: TermNamespace attribute must not be empty
			if strings.Contains(body, `TermNamespace=""`) {
				return framework.NewError("TermNamespace attribute must not be empty")
			}

			// Check 5: If Qualifier is present, it must not be empty
			if strings.Contains(body, `<edmx:IncludeAnnotations`) && strings.Contains(body, `Qualifier=""`) {
				return framework.NewError("Qualifier attribute must not be empty if specified")
			}

			// Check 6: If TargetNamespace is present, it must not be empty
			if strings.Contains(body, `<edmx:IncludeAnnotations`) && strings.Contains(body, `TargetNamespace=""`) {
				return framework.NewError("TargetNamespace attribute must not be empty if specified")
			}

			return nil
		},
	)

	return suite
}
