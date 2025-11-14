package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// EDMXElement creates the 3.1 Element edmx:Edmx test suite
func EDMXElement() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"3.1 Element edmx:Edmx",
		"Tests the root edmx:Edmx element of the CSDL XML document, validating its structure, attributes, and required namespace declarations according to the OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752501",
	)

	// Test: Validate complete edmx:Edmx element structure
	suite.AddTest(
		"test_edmx_element_structure",
		"edmx:Edmx element has correct structure and attributes",
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

			// Check 2: edmx:Edmx root element is present
			if !strings.Contains(body, "<edmx:Edmx") {
				return framework.NewError("Missing edmx:Edmx root element")
			}

			// Check 3: Proper EDMX namespace declaration
			expectedNamespace := `xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx"`
			if !strings.Contains(body, expectedNamespace) {
				return framework.NewError(fmt.Sprintf(
					"Missing or invalid edmx namespace declaration. Expected: %s",
					expectedNamespace,
				))
			}

			// Check 4: Version attribute is present
			if !strings.Contains(body, "Version=") {
				return framework.NewError("edmx:Edmx element must have Version attribute")
			}

			// Check 5: Element is properly closed
			if !strings.Contains(body, "</edmx:Edmx>") {
				return framework.NewError("edmx:Edmx element must be properly closed with </edmx:Edmx>")
			}

			// Check 6: Contains exactly one edmx:DataServices element
			count := strings.Count(body, "<edmx:DataServices")
			if count != 1 {
				return framework.NewError(fmt.Sprintf(
					"edmx:Edmx must contain exactly one edmx:DataServices element (found: %d)",
					count,
				))
			}

			return nil
		},
	)

	return suite
}
