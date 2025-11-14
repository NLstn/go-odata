package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ResponseHeaders creates the 8.1.3 Response Headers test suite
func ResponseHeaders() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.1.3 Response Headers",
		"Tests that OData services return proper response headers including Content-Type, OData-Version, and other required headers.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ResponseHeaders",
	)

	suite.AddTest(
		"test_content_type_present",
		"Response includes Content-Type header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")
			if contentType == "" {
				return framework.NewError("Response must include Content-Type header")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_odata_version_present",
		"Response includes OData-Version header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return framework.NewError("Response should include OData-Version header")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_odata_version_value",
		"OData-Version is 4.0 or 4.01",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return framework.NewError("OData-Version header missing")
			}

			if !strings.HasPrefix(odataVersion, "4.0") && !strings.HasPrefix(odataVersion, "4.01") {
				return framework.NewError("OData-Version should be 4.0 or 4.01")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_content_type_charset",
		"Content-Type includes appropriate format",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")
			if contentType == "" {
				return framework.NewError("Content-Type header missing")
			}

			// Should contain application/json
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return framework.NewError("Content-Type should be application/json")
			}

			return nil
		},
	)

	return suite
}
