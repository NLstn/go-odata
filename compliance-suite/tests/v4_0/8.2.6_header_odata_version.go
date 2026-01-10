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

	// Test 1: Service should return OData-Version: 4.01 header by default
	suite.AddTest(
		"test_odata_version_header",
		"Service returns OData-Version header with value 4.01 by default",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return framework.NewError("Header not found")
			}

			// Without OData-MaxVersion, service should return its highest supported version
			odataVersion = strings.TrimSpace(odataVersion)
			if odataVersion != "4.01" {
				return framework.NewError(fmt.Sprintf("Expected version 4.01, got: %s", odataVersion))
			}

			return nil
		},
	)

	// Test 2: Service should respond with OData-Version: 4.0 when OData-MaxVersion: 4.0
	suite.AddTest(
		"test_maxversion_40_response",
		"Service responds with OData-Version: 4.0 when OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"},
			)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			odataVersion = strings.TrimSpace(odataVersion)
			if odataVersion != "4.0" {
				return framework.NewError(fmt.Sprintf("Expected OData-Version: 4.0, got: %s (spec requires response version <= OData-MaxVersion)", odataVersion))
			}

			return nil
		},
	)

	// Test 3: Service should respond with OData-Version: 4.01 when OData-MaxVersion: 4.01
	suite.AddTest(
		"test_maxversion_401_response",
		"Service responds with OData-Version: 4.01 when OData-MaxVersion: 4.01",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
			)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			odataVersion = strings.TrimSpace(odataVersion)
			if odataVersion != "4.01" {
				return framework.NewError(fmt.Sprintf("Expected OData-Version: 4.01, got: %s", odataVersion))
			}

			return nil
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

	// Test 7: Entity collection respects version negotiation
	suite.AddTest(
		"test_entity_collection_version_negotiation",
		"Entity collection response respects OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products",
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"},
			)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			odataVersion = strings.TrimSpace(odataVersion)
			if odataVersion != "4.0" {
				return framework.NewError(fmt.Sprintf("Expected OData-Version: 4.0, got: %s", odataVersion))
			}

			return nil
		},
	)

	return suite
}
