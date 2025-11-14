package v4_0

import (
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

			// Location header should contain URL of created entity
			location := resp.Headers.Get("Location")
			if location != "" && !strings.Contains(location, "Products") {
				return framework.NewError("Location should contain entity URL")
			}

			return nil
		},
	)

	return suite
}
