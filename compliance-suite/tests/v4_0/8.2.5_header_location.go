package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderLocation creates the 8.2.5 Location Header test suite
func HeaderLocation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.5 Location Header",
		"Tests Location header in responses for created entities.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderLocation",
	)

	suite.AddTest(
		"test_location_header",
		"Location header for created entity",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Products", map[string]interface{}{
				"Name":       "Location Test",
				"Price":      99.99,
				"CategoryID": 1,
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 201 && resp.StatusCode != 200 {
				// Skip if creation fails due to validation
				if resp.StatusCode == 400 {
					return ctx.Skip("Entity creation validation error (likely schema mismatch)")
				}
				return ctx.Skip("Entity creation not supported or failed")
			}

			// OData v4 requires Location header in 201 responses
			location := resp.Headers.Get("Location")
			if location == "" {
				return fmt.Errorf("Location header is required for 201 responses per OData v4 spec")
			}

			if !strings.Contains(location, "Products") {
				return fmt.Errorf("Location header should contain entity URL, got: %s", location)
			}

			return nil
		},
	)

	return suite
}
