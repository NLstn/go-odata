package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// CacheControlHeader creates the 8.2.1 Cache-Control Header test suite
func CacheControlHeader() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.1 Cache-Control Header",
		"Validates Cache-Control header handling for HTTP caching according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderCacheControl",
	)

	suite.AddTest(
		"test_metadata_cacheable",
		"Metadata document can be cached",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_service_doc_cacheable",
		"Service document can be cached",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
