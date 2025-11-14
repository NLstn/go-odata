package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderODataVersion creates the 8.2.6 Header OData-Version test suite
func HeaderODataVersion() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.6 Header OData-Version",
		"Tests that OData-Version header is properly set and version negotiation works according to OData v4 specification, including OData-MaxVersion handling.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataVersion",
	)

	// Test 1: Service should return OData-Version: 4.0 header
	suite.AddTest(
		"test_odata_version_header",
		"Service returns OData-Version header with value 4.0 or 4.01",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return framework.NewError("Header not found")
			}

			odataVersion = strings.TrimSpace(odataVersion)
			if odataVersion != "4.0" && odataVersion != "4.01" {
				return framework.NewError(fmt.Sprintf("Got version: %s", odataVersion))
			}

			return nil
		},
	)

	// Test 2: Service should accept request with OData-MaxVersion: 4.0
	suite.AddTest(
		"test_maxversion_40",
		"Service accepts OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"},
			)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 3: Service should accept request with OData-MaxVersion: 4.01
	suite.AddTest(
		"test_maxversion_401",
		"Service accepts OData-MaxVersion: 4.01",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 4: Service should reject request with OData-MaxVersion: 3.0
	suite.AddTest(
		"test_maxversion_30",
		"Service rejects OData-MaxVersion: 3.0 with 406 Not Acceptable",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "OData-MaxVersion", Value: "3.0"},
			)
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 406)
		},
	)

	// Test 5: OData-Version header should be present in all responses
	suite.AddTest(
		"test_entity_collection_header",
		"Entity collection response includes OData-Version header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return framework.NewError("Header not found")
			}

			return nil
		},
	)

	// Test 6: OData-Version header should be present in error responses
	suite.AddTest(
		"test_error_response_header",
		"Error response includes OData-Version header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "OData-MaxVersion", Value: "3.0"},
			)
			if err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return framework.NewError("No OData-Version header")
			}

			return nil
		},
	)

	return suite
}
