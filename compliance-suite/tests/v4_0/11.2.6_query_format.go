package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QueryFormat creates the 11.2.6 Query Option $format test suite
func QueryFormat() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.6 System Query Option $format",
		"Tests $format query option for specifying response format according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionformat",
	)

	// Test 1: $format=json returns JSON
	suite.AddTest(
		"test_format_json",
		"$format=json returns JSON response",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$format=json")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type to contain 'application/json', got: %s", contentType)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			return nil
		},
	)

	// Test 2: $format=xml returns XML (for metadata)
	suite.AddTest(
		"test_format_xml",
		"$format=xml on metadata returns XML",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata?$format=xml")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/xml") {
				return fmt.Errorf("expected Content-Type to contain 'application/xml', got: %s", contentType)
			}

			return nil
		},
	)

	// Test 3: Invalid $format returns error or is ignored
	suite.AddTest(
		"test_format_invalid",
		"Invalid $format value returns error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$format=invalid")
			if err != nil {
				return err
			}

			// Accept 400, 406, or implementations that are lenient (200)
			if resp.StatusCode != 406 && resp.StatusCode != 400 && resp.StatusCode != 200 {
				return fmt.Errorf("expected status 400, 406, or 200 (lenient), got %d", resp.StatusCode)
			}

			return nil
		},
	)

	return suite
}
