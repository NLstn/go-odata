package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// PropertyAccess creates the 11.2.3 Addressing Individual Properties test suite
func PropertyAccess() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.3 Addressing Individual Properties",
		"Tests addressing and accessing individual properties according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_AddressingIndividualProperties",
	)

	// Test 1: Access a primitive property
	suite.AddTest(
		"test_primitive_property",
		"Access a primitive property (Name)",
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
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("property response missing 'value' field")
			}

			return nil
		},
	)

	// Test 2: Access a primitive property with $value
	suite.AddTest(
		"test_property_value",
		"Access primitive property raw value with $value",
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
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// Should return plain text without JSON wrapping
			bodyStr := string(resp.Body)
			if len(bodyStr) == 0 {
				return fmt.Errorf("$value response is empty")
			}

			// Verify it's not JSON by checking it doesn't have "value" field
			var testJSON map[string]interface{}
			if json.Unmarshal(resp.Body, &testJSON) == nil {
				if _, hasValue := testJSON["value"]; hasValue {
					return fmt.Errorf("$value should return raw text, not JSON with 'value' field")
				}
			}

			return nil
		},
	)

	// Test 3: Access non-existent property should return 404
	suite.AddTest(
		"test_nonexistent_property",
		"Access non-existent property returns 404",
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

			// Test non-existent property
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)/NonExistentProperty", productID))
			if err != nil {
				return err
			}
			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404 for non-existent property, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 4: Access property of non-existent entity should return 404
	suite.AddTest(
		"test_property_of_nonexistent_entity",
		"Access property of non-existent entity returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products(00000000-0000-0000-0000-000000000000)/Name")
			if err != nil {
				return err
			}
			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404 for property of non-existent entity, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 5: Property access should have proper Content-Type
	suite.AddTest(
		"test_property_content_type",
		"Property access returns proper Content-Type",
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
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type to contain 'application/json', got: %s", contentType)
			}

			return nil
		},
	)

	// Test 6: $value should have text/plain Content-Type for strings
	suite.AddTest(
		"test_value_content_type",
		"$value has text/plain Content-Type for string property",
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
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "text/plain") {
				return fmt.Errorf("expected Content-Type to contain 'text/plain', got: %s", contentType)
			}

			return nil
		},
	)

	return suite
}
