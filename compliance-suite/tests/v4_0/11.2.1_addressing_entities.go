package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// AddressingEntities creates the 11.2.1 Addressing Entities test suite
func AddressingEntities() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.1 Addressing Entities",
		"Tests various ways to address entities according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingEntities",
	)

	// Test 1: Address entity set returns collection
	suite.AddTest(
		"test_entity_set",
		"Addressing entity set returns collection",
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
				return fmt.Errorf("response missing 'value' array")
			}

			return nil
		},
	)

	// Test 2: Address single entity by key
	suite.AddTest(
		"test_single_entity",
		"Addressing single entity by key",
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

			// Now test single entity access
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)", productID))
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

			if _, ok := result["ID"]; !ok {
				return fmt.Errorf("response missing 'ID' field")
			}

			return nil
		},
	)

	// Test 3: Non-existent entity returns 404
	suite.AddTest(
		"test_nonexistent_entity",
		"Non-existent entity returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products(00000000-0000-0000-0000-000000000000)")
			if err != nil {
				return err
			}
			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404 for non-existent entity, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 4: Invalid entity set returns 404
	suite.AddTest(
		"test_invalid_entity_set",
		"Invalid entity set returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/NonExistentEntitySet")
			if err != nil {
				return err
			}
			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404 for invalid entity set, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 5: Accessing property of entity
	suite.AddTest(
		"test_entity_property",
		"Accessing property of entity",
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

			// Test property access
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)/Name", productID))
			if err != nil {
				return err
			}

			// Accept both 200 (property access supported) or 404 (not supported)
			if resp.StatusCode == 200 {
				var result map[string]interface{}
				if err := json.Unmarshal(resp.Body, &result); err != nil {
					return fmt.Errorf("failed to parse JSON: %w", err)
				}
				if _, ok := result["value"]; !ok {
					return fmt.Errorf("property response missing 'value' field")
				}
			} else if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 200 or 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 6: Accessing raw value of property
	suite.AddTest(
		"test_property_raw_value",
		"Accessing raw value of property with $value",
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

			// Test $value access
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)/Name/$value", productID))
			if err != nil {
				return err
			}

			// Accept both 200 ($value supported) or 404 (not supported)
			if resp.StatusCode == 200 {
				// Should return raw text, not JSON
				bodyStr := string(resp.Body)
				if len(bodyStr) == 0 {
					return fmt.Errorf("$value response is empty")
				}
				// Verify it's not JSON by checking for common JSON markers
				var testJSON map[string]interface{}
				if json.Unmarshal(resp.Body, &testJSON) == nil {
					// It's valid JSON, which means it's probably not raw value
					if _, hasValue := testJSON["value"]; hasValue {
						return fmt.Errorf("$value should return raw text, not JSON with 'value' field")
					}
				}
			} else if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 200 or 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	return suite
}
