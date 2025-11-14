package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// DataServicesElement creates the 3.2 Element edmx:DataServices test suite
func DataServicesElement() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"3.2 Element edmx:DataServices",
		"Validates the edmx:DataServices element contains one or more edm:Schema elements according to the OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752502",
	)

	// Test: Validate complete edmx:DataServices element structure
	suite.AddTest(
		"test_dataservices_element_structure",
		"edmx:DataServices has correct structure with Schema elements",
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

			// Check 2: edmx:DataServices element is present
			if !strings.Contains(body, "<edmx:DataServices") {
				return framework.NewError("Missing edmx:DataServices element")
			}

			// Check 3: edmx:DataServices is properly closed
			if !strings.Contains(body, "</edmx:DataServices>") {
				return framework.NewError("edmx:DataServices must be properly closed")
			}

			// Check 4: MUST contain at least one Schema element
			if !strings.Contains(body, "<Schema") {
				return framework.NewError("edmx:DataServices must contain at least one Schema element")
			}

			// Check 5: Schema elements have proper EDM namespace
			if !strings.Contains(body, `xmlns="http://docs.oasis-open.org/odata/ns/edm"`) {
				return framework.NewError(
					`Schema elements must use EDM namespace: xmlns="http://docs.oasis-open.org/odata/ns/edm"`,
				)
			}

			// Check 6: Schema elements have Namespace attribute
			if !strings.Contains(body, "<Schema") || !strings.Contains(body, "Namespace=") {
				return framework.NewError("Schema elements must have Namespace attribute")
			}

			// Check 7: Schemas contain entity model elements
			hasElements := strings.Contains(body, "<EntityType") ||
				strings.Contains(body, "<ComplexType") ||
				strings.Contains(body, "<EntityContainer") ||
				strings.Contains(body, "<EnumType") ||
				strings.Contains(body, "<Action") ||
				strings.Contains(body, "<Function")

			if !hasElements {
				return framework.NewError("Schema should contain entity model elements")
			}

			return nil
		},
	)

	return suite
}
