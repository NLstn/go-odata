package v4_0

import (
	"encoding/json"
	"encoding/xml"
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

			if resp.StatusCode != 406 && resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400 or 406 for invalid $format, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type to contain 'application/json', got: %s", contentType)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("response is not valid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("error response must have 'error' object property")
			}

			code, ok := errorObj["code"].(string)
			if !ok || code == "" {
				return fmt.Errorf("error object must have 'code' property as non-empty string")
			}

			message, ok := errorObj["message"].(string)
			if !ok || message == "" {
				return fmt.Errorf("error object must have 'message' property as non-empty string")
			}

			return nil
		},
	)

	// Test 4: $format=atom returns Atom/XML for collection
	suite.AddTest(
		"test_format_atom_collection",
		"$format=atom returns Atom/XML for entity collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$format=atom")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/atom+xml") {
				return fmt.Errorf("expected Content-Type to contain 'application/atom+xml', got: %s", contentType)
			}

			// Validate well-formed XML
			if err := xml.Unmarshal(resp.Body, new(interface{})); err != nil {
				return fmt.Errorf("response is not valid XML: %v", err)
			}

			// Validate Atom feed structure: must contain <feed element
			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, "<feed") {
				return fmt.Errorf("Atom feed response must contain <feed> element")
			}

			return nil
		},
	)

	// Test 5: $format=atom returns Atom/XML for single entity
	suite.AddTest(
		"test_format_atom_entity",
		"$format=atom returns Atom/XML for single entity",
		func(ctx *framework.TestContext) error {
			// First get a product ID from the collection
			listResp, err := ctx.GET("/Products?$format=json&$top=1")
			if err != nil {
				return err
			}
			if listResp.StatusCode != 200 {
				return fmt.Errorf("failed to list products: status %d", listResp.StatusCode)
			}
			var listResult map[string]interface{}
			if err := json.Unmarshal(listResp.Body, &listResult); err != nil {
				return fmt.Errorf("failed to parse product list: %v", err)
			}
			values, ok := listResult["value"].([]interface{})
			if !ok || len(values) == 0 {
				// No products to test with; skip this test
				return nil
			}

			firstProduct, ok := values[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("unexpected product format")
			}

			// Build the entity URL using the first available numeric ID field
			var productID interface{}
			for _, key := range []string{"ID", "Id", "ProductID", "ProductId"} {
				if v, exists := firstProduct[key]; exists {
					productID = v
					break
				}
			}
			if productID == nil {
				// Can't determine key; skip
				return nil
			}

			entityURL := fmt.Sprintf("/Products(%v)?$format=atom", productID)
			resp, err := ctx.GET(entityURL)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/atom+xml") {
				return fmt.Errorf("expected Content-Type to contain 'application/atom+xml', got: %s", contentType)
			}

			// Validate well-formed XML
			if err := xml.Unmarshal(resp.Body, new(interface{})); err != nil {
				return fmt.Errorf("response is not valid XML: %v", err)
			}

			// Validate Atom entry structure
			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, "<entry") {
				return fmt.Errorf("Atom entry response must contain <entry> element")
			}

			return nil
		},
	)

	// Test 6: Accept: application/atom+xml returns Atom/XML
	suite.AddTest(
		"test_accept_atom_xml",
		"Accept: application/atom+xml header returns Atom/XML for entity collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GETWithHeaders("/Products", map[string]string{
				"Accept": "application/atom+xml",
			})
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/atom+xml") {
				return fmt.Errorf("expected Content-Type to contain 'application/atom+xml', got: %s", contentType)
			}

			// Validate well-formed XML
			if err := xml.Unmarshal(resp.Body, new(interface{})); err != nil {
				return fmt.Errorf("response is not valid XML: %v", err)
			}

			return nil
		},
	)

	return suite
}
