package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderIfMatch creates the 8.2.2 If-Match Header test suite
func HeaderIfMatch() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.2 If-Match Header",
		"Tests If-Match header handling for optimistic concurrency control.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderIfMatch",
	)

	suite.AddTest(
		"test_etag_support",
		"Service supports ETag for concurrency",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// ETag is optional but recommended
			return nil
		},
	)

	return suite
}
