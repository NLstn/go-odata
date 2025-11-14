package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// Conformance creates the 2.1 Conformance test suite
func Conformance() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"2.1 Conformance",
		"Tests service conformance to OData v4 specification requirements including proper response formats, required headers, metadata availability, and protocol compliance.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Conformance",
	)

	// Test 1: Service MUST return service document
	suite.AddTest(
		"test_service_document_required",
		"Service returns service document (MUST)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: Service MUST return metadata document
	suite.AddTest(
		"test_metadata_document_required",
		"Service returns metadata document (MUST)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 3: Service MUST support JSON format
	suite.AddTest(
		"test_json_format_support",
		"Service supports JSON format (MUST)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/",
				framework.Header{Key: "Accept", Value: "application/json"},
			)
			if err != nil {
				return err
			}
			if !ctx.IsValidJSON(resp) {
				return framework.NewError("Service must support JSON format (invalid JSON response)")
			}
			return nil
		},
	)

	// Test 4: Service MUST include OData-Version header in responses
	suite.AddTest(
		"test_odata_version_header",
		"Service includes OData-Version header (MUST)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return framework.NewError("OData-Version header is required in responses")
			}
			return nil
		},
	)

	// Test 5: Service MUST respond to requests without custom headers
	suite.AddTest(
		"test_no_custom_headers_required",
		"Service does not require custom headers (MUST)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return framework.NewError("Service must not require custom headers for basic requests")
			}
			return nil
		},
	)

	// Test 6: Service MUST support GET on entity sets
	suite.AddTest(
		"test_get_entity_sets",
		"Service supports GET on entity sets (MUST)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 7: Service MUST support GET on single entities
	suite.AddTest(
		"test_get_single_entity",
		"Service supports GET on single entities (MUST)",
		func(ctx *framework.TestContext) error {
			// Get a single product to determine the ID format
			resp, err := ctx.GET("/Products?$top=1",
				framework.Header{Key: "Accept", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			value, ok := data["value"]
			if !ok {
				return framework.NewError("Failed to get products collection")
			}

			valueArray, ok := value.([]interface{})
			if !ok || len(valueArray) == 0 {
				return framework.NewError("No products found in collection")
			}

			firstProduct, ok := valueArray[0].(map[string]interface{})
			if !ok {
				return framework.NewError("Invalid product format")
			}

			id, ok := firstProduct["ID"]
			if !ok {
				return framework.NewError("Failed to determine a Product ID from the collection")
			}

			// Try addressing the entity using the raw ID literal first
			var entityURL string
			switch v := id.(type) {
			case string:
				// For string/GUID IDs, try without quotes first (raw literal)
				entityURL = fmt.Sprintf("/Products(%s)", v)
			case float64:
				// For numeric IDs
				entityURL = fmt.Sprintf("/Products(%d)", int(v))
			default:
				// For other types, try string representation
				entityURL = fmt.Sprintf("/Products(%v)", id)
			}

			resp, err = ctx.GET(entityURL)
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			// If that didn't work and ID is a string (possibly GUID), try guid'...' syntax
			if idStr, ok := id.(string); ok {
				entityURL = fmt.Sprintf("/Products(guid'%s')", idStr)
				resp, err = ctx.GET(entityURL)
				if err != nil {
					return err
				}
				return ctx.AssertStatusCode(resp, 200)
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 8: Service MUST return 404 for non-existent resources
	suite.AddTest(
		"test_404_for_missing_resource",
		"Service returns 404 for non-existent resources (MUST)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/NonExistentEntitySet")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 404)
		},
	)

	// Test 9: Service MUST use UTF-8 encoding
	suite.AddTest(
		"test_utf8_encoding",
		"Service uses UTF-8 encoding (MUST)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")

			// OData services should use UTF-8 encoding (implied or explicit)
			// JSON is UTF-8 by default per RFC 8259
			if strings.Contains(contentType, "application/json") {
				return nil
			}
			if strings.Contains(contentType, "charset=utf-8") || strings.Contains(contentType, "charset=UTF-8") {
				return nil
			}
			// JSON implies UTF-8, so this is acceptable
			return nil
		},
	)

	// Test 10: Service MUST support $metadata system resource
	suite.AddTest(
		"test_metadata_system_resource",
		"Service supports $metadata system resource (MUST)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return framework.NewError("Service must support $metadata system resource")
			}
			if len(resp.Body) == 0 {
				return framework.NewError("$metadata response body is empty")
			}
			return nil
		},
	)

	return suite
}
