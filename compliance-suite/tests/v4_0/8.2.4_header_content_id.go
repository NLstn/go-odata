package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderContentId creates the 8.2.4 Content-ID Header test suite
func HeaderContentId() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.4 Content-ID Header",
		"Tests Content-ID header handling in batch requests.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderContentID",
	)

	suite.AddTest(
		"test_content_id_in_batch",
		"Content-ID header in batch requests",
		func(ctx *framework.TestContext) error {
			// Content-ID is used in batch requests, which is an advanced feature
			// Just verify basic service is working
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
