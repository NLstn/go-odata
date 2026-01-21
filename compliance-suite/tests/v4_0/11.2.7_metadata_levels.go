package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// MetadataLevels creates the 11.2.7 Metadata Levels test suite
func MetadataLevels() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.7 Metadata Levels",
		"Tests odata.metadata parameter values according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_metadataURLs",
	)

	// Test 1: odata.metadata=minimal (default)
	suite.AddTest(
		"test_metadata_minimal",
		"odata.metadata=minimal includes @odata.context",
		func(ctx *framework.TestContext) error {
			format := url.QueryEscape("application/json;odata.metadata=minimal")
			resp, err := ctx.GET("/Products?$format=" + format)
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
				return fmt.Errorf("metadata=minimal missing @odata.context")
			}

			return nil
		},
	)

	// Test 2: odata.metadata=full includes type annotations
	suite.AddTest(
		"test_metadata_full",
		"odata.metadata=full includes type annotations",
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

			// Test metadata=full
			format := url.QueryEscape("application/json;odata.metadata=full")
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)?$format=%s", productID, format))
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

			// Check for @odata.context
			if _, ok := result["@odata.context"]; !ok {
				return fmt.Errorf("metadata=full missing @odata.context")
			}

			// Check for type annotations (@odata.type or @odata.id)
			hasTypeAnnotations := false
			for key := range result {
				if key == "@odata.type" || key == "@odata.id" {
					hasTypeAnnotations = true
					break
				}
			}

			if !hasTypeAnnotations {
				return fmt.Errorf("metadata=full missing type annotations (@odata.type or @odata.id)")
			}

			return nil
		},
	)

	// Test 3: odata.metadata=none excludes metadata
	suite.AddTest(
		"test_metadata_none_excludes",
		"odata.metadata=none excludes @odata.context",
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

			// Test metadata=none
			format := url.QueryEscape("application/json;odata.metadata=none")
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)?$format=%s", productID, format))
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

			// Should NOT have @odata.context
			if _, ok := result["@odata.context"]; ok {
				return fmt.Errorf("metadata=none should not include @odata.context")
			}

			return nil
		},
	)

	// Test 4: metadata=none still returns data
	suite.AddTest(
		"test_metadata_none_returns_data",
		"odata.metadata=none still returns entity data",
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

			// Test metadata=none
			format := url.QueryEscape("application/json;odata.metadata=none")
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)?$format=%s", productID, format))
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

			// Should have entity data (ID or Name)
			if _, hasID := result["ID"]; !hasID {
				if _, hasName := result["Name"]; !hasName {
					return fmt.Errorf("metadata=none should return entity data (missing ID and Name)")
				}
			}

			return nil
		},
	)

	// Test 5: Invalid metadata value should work or return error
	suite.AddTest(
		"test_metadata_invalid",
		"Invalid odata.metadata value handling",
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

			// Test invalid metadata value
			format := url.QueryEscape("application/json;odata.metadata=invalid")
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)?$format=%s", productID, format))
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 && resp.StatusCode != 406 {
				return fmt.Errorf("expected status 400 or 406, got %d", resp.StatusCode)
			}

			if err := ctx.AssertHeaderContains(resp, "Content-Type", "application/json"); err != nil {
				return err
			}

			if resp.Headers.Get("OData-Version") == "" {
				return framework.NewError("missing OData-Version header")
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("response is not valid JSON: %w", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("error response must have 'error' object")
			}

			code, ok := errorObj["code"].(string)
			if !ok || code == "" {
				return fmt.Errorf("error object must include non-empty 'code' property")
			}

			message := errorObj["message"]
			switch msg := message.(type) {
			case string:
				if msg == "" {
					return fmt.Errorf("error.message must be non-empty string")
				}
			case map[string]interface{}:
				value, ok := msg["value"].(string)
				if !ok || value == "" {
					return fmt.Errorf("error.message.value must be non-empty string")
				}
			default:
				return fmt.Errorf("error.message must be string or object, got %T", message)
			}

			return nil
		},
	)

	// Test 6: Collection with metadata=full
	suite.AddTest(
		"test_collection_metadata_full",
		"Collection with metadata=full includes @odata.context",
		func(ctx *framework.TestContext) error {
			format := url.QueryEscape("application/json;odata.metadata=full")
			resp, err := ctx.GET("/Products?$top=2&$format=" + format)
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
				return fmt.Errorf("collection metadata=full missing @odata.context")
			}

			return nil
		},
	)

	return suite
}
