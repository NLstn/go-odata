package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderContentType creates the 8.1.1 Header Content-Type test suite
func HeaderContentType() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.1.1 Header Content-Type",
		"Tests that Content-Type header is properly set according to OData v4 specification, including media type and odata.metadata parameter.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderContentType",
	)

	// Helper function to get product path for each test
	// Note: Must refetch on each call because database is reseeded between tests
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		return firstEntityPath(ctx, "Products")
	}

	// Test 1: Service Document should return application/json with odata.metadata=minimal
	suite.AddTest(
		"test_service_doc_content_type",
		"Service Document returns application/json with odata.metadata=minimal",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")

			if contentType == "" {
				return framework.NewError("Content-Type header is missing")
			}

			// Strictly validate Content-Type format per OData spec
			// Must be application/json with odata.metadata parameter
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return framework.NewError(fmt.Sprintf("Expected application/json, got: %s", contentType))
			}

			if !strings.Contains(contentType, "odata.metadata=") {
				return framework.NewError(fmt.Sprintf("Missing odata.metadata parameter. Got: %s", contentType))
			}

			// Validate that odata.metadata has a valid value (minimal, full, or none)
			if !strings.Contains(contentType, "odata.metadata=minimal") &&
				!strings.Contains(contentType, "odata.metadata=full") &&
				!strings.Contains(contentType, "odata.metadata=none") {
				return framework.NewError(fmt.Sprintf("Invalid odata.metadata value. Got: %s. Must be minimal, full, or none", contentType))
			}

			return nil
		},
	)

	// Test 2: Metadata Document should return application/xml
	suite.AddTest(
		"test_metadata_xml_content_type",
		"Metadata Document returns application/xml",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")

			if !strings.Contains(contentType, "application/xml") {
				return framework.NewError(fmt.Sprintf("Expected application/xml, got: %s", contentType))
			}

			return nil
		},
	)

	// Test 3: Metadata Document with $format=json should return application/json
	suite.AddTest(
		"test_metadata_json_content_type",
		"Metadata Document with $format=json returns application/json",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata?$format=json")
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")

			if !strings.Contains(contentType, "application/json") {
				return framework.NewError(fmt.Sprintf("Expected application/json, got: %s", contentType))
			}

			return nil
		},
	)

	// Test 4: Entity Collection should return application/json with odata.metadata
	suite.AddTest(
		"test_entity_collection_content_type",
		"Entity Collection returns application/json with odata.metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")

			if !strings.Contains(contentType, "application/json") {
				return framework.NewError(fmt.Sprintf("Expected application/json, got: %s", contentType))
			}

			if !strings.Contains(contentType, "odata.metadata") {
				return framework.NewError(fmt.Sprintf("Missing odata.metadata parameter. Got: %s", contentType))
			}

			return nil
		},
	)

	// Test 5: Single Entity should return application/json with odata.metadata
	suite.AddTest(
		"test_single_entity_content_type",
		"Single Entity returns application/json with odata.metadata",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")

			if !strings.Contains(contentType, "application/json") {
				return framework.NewError(fmt.Sprintf("Expected application/json, got: %s", contentType))
			}

			if !strings.Contains(contentType, "odata.metadata") {
				return framework.NewError(fmt.Sprintf("Missing odata.metadata parameter. Got: %s", contentType))
			}

			return nil
		},
	)

	return suite
}
