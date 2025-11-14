package v4_0

import (
	"encoding/json"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// RequestHeaders creates the 8.1.2 Request Headers test suite
func RequestHeaders() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.1.2 Request Headers",
		"Tests proper handling of OData request headers including Accept, Content-Type, OData-MaxVersion, and other standard HTTP request headers.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_RequestHeaders",
	)

	suite.AddTest(
		"test_no_accept_header",
		"Service accepts requests without Accept header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_accept_json",
		"Service respects Accept: application/json",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{
				Key:   "Accept",
				Value: "application/json",
			})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Response should be valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return framework.NewError("Response is not valid JSON")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_accept_xml_metadata",
		"Service handles Accept: application/xml for metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{
				Key:   "Accept",
				Value: "application/xml",
			})
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_odata_maxversion_header",
		"OData-MaxVersion header is respected",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{
				Key:   "OData-MaxVersion",
				Value: "4.0",
			})
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_odata_version_request",
		"OData-Version header in request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{
				Key:   "OData-Version",
				Value: "4.0",
			})
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
