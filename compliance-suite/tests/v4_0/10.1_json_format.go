package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// JSONFormat creates the 10.1 JSON Format test suite
func JSONFormat() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"10.1 JSON Format",
		"Validates JSON format requirements for OData responses including entity representation, collections, metadata annotations, and proper JSON structure.",
		"https://docs.oasis-open.org/odata/odata-json-format/v4.0/odata-json-format-v4.0.html",
	)

	// Test 1: Collection response has 'value' property
	suite.AddTest(
		"test_collection_value",
		"Collection response has 'value' property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("collection response missing 'value' property")
			}

			return nil
		},
	)

	// Test 2: Entity response has @odata.context
	suite.AddTest(
		"test_odata_context",
		"Response includes @odata.context annotation",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["@odata.context"]; !ok {
				return fmt.Errorf("response missing @odata.context annotation")
			}

			return nil
		},
	)

	// Test 3: Valid JSON structure
	suite.AddTest(
		"test_valid_json",
		"Response is valid JSON structure",
		func(ctx *framework.TestContext) error {
			// Get a single entity
			allResp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if allResp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", allResp.StatusCode)
			}

			var allResult map[string]interface{}
			if err := json.Unmarshal(allResp.Body, &allResult); err != nil {
				return fmt.Errorf("failed to parse products JSON: %w", err)
			}

			value, ok := allResult["value"].([]interface{})
			if !ok || len(value) == 0 {
				return fmt.Errorf("no products available")
			}

			firstItem, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first item is not an object")
			}

			productID := firstItem["ID"]

			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)", productID))
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("response is not valid JSON: %w", err)
			}

			return nil
		},
	)

	// Test 4: Collection 'value' is JSON array
	suite.AddTest(
		"test_array_format",
		"Collection 'value' is JSON array",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			value, ok := result["value"]
			if !ok {
				return fmt.Errorf("response missing 'value' property")
			}

			if _, ok := value.([]interface{}); !ok {
				return fmt.Errorf("'value' property is not an array")
			}

			return nil
		},
	)

	// Test 5: Metadata annotations use @ prefix
	suite.AddTest(
		"test_metadata_annotations",
		"Metadata annotations use @ prefix",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Check for @ prefix in metadata annotations
			foundODataAnnotation := false
			for key := range result {
				if strings.HasPrefix(key, "@odata.") {
					foundODataAnnotation = true
					break
				}
			}

			if !foundODataAnnotation {
				return fmt.Errorf("metadata annotations not in correct format (missing @odata. prefix)")
			}

			return nil
		},
	)

	// Test 6: Content-Type includes proper JSON media type
	suite.AddTest(
		"test_content_type",
		"Content-Type includes proper JSON media type",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("Content-Type not properly set, got: %s", contentType)
			}

			return nil
		},
	)

	return suite
}
