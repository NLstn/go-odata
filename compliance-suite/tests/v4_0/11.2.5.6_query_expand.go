package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QueryExpand creates the 11.2.5.6 System Query Option $expand test suite
func QueryExpand() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.6 System Query Option $expand",
		"Tests $expand query option for expanding related entities according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionexpand",
	)

	// Test 1: Basic $expand returns related entities inline
	suite.AddTest(
		"test_expand_basic",
		"$expand includes related entities inline",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions")
			resp, err := ctx.GET("/Products?$expand=" + expand + "&$top=1")
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

			// Get the value array
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) == 0 {
				return fmt.Errorf("response contains no items")
			}

			// Check first item
			item, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first item is not an object")
			}

			// Verify Descriptions field is present
			descriptions, ok := item["Descriptions"]
			if !ok {
				return fmt.Errorf("descriptions field is missing")
			}

			// Verify Descriptions is an array (expanded data)
			if _, ok := descriptions.([]interface{}); !ok {
				return fmt.Errorf("descriptions field is not an array (not properly expanded)")
			}

			return nil
		},
	)

	// Test 2: $expand with $select on expanded entity
	suite.AddTest(
		"test_expand_with_select",
		"$expand with nested $select",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($select=Description)")
			resp, err := ctx.GET("/Products?$expand=" + expand + "&$top=1")
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

			// Get the value array
			value, ok := result["value"].([]interface{})
			if !ok {
				return fmt.Errorf("response missing 'value' array")
			}

			if len(value) == 0 {
				return fmt.Errorf("response contains no items")
			}

			// Check first item
			item, ok := value[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first item is not an object")
			}

			// Verify Descriptions field is present and expanded
			descriptions, ok := item["Descriptions"]
			if !ok {
				return fmt.Errorf("descriptions field is missing")
			}

			// Verify it's an array
			descArray, ok := descriptions.([]interface{})
			if !ok {
				return fmt.Errorf("descriptions field is not an array")
			}

			// If there are descriptions, verify they contain Description field
			if len(descArray) > 0 {
				desc, ok := descArray[0].(map[string]interface{})
				if !ok {
					return fmt.Errorf("first description is not an object")
				}
				if _, ok := desc["Description"]; !ok {
					return fmt.Errorf("expanded Descriptions missing Description field")
				}
			}

			return nil
		},
	)

	// Test 3: $expand on single entity
	suite.AddTest(
		"test_expand_single_entity",
		"$expand on single entity request",
		func(ctx *framework.TestContext) error {
			// First get a product ID
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

			// Now test $expand on single entity
			expand := url.QueryEscape("Descriptions")
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)?$expand=%s", productID, expand))
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

			// Verify Descriptions field is present and expanded
			descriptions, ok := result["Descriptions"]
			if !ok {
				return fmt.Errorf("descriptions field is missing")
			}

			// Verify it's expanded (should be an array)
			if _, ok := descriptions.([]interface{}); !ok {
				return fmt.Errorf("descriptions not expanded as array")
			}

			return nil
		},
	)

	return suite
}
