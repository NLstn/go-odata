package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ResourcePath creates the 11.1 Resource Path test suite
func ResourcePath() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.1 Resource Path",
		"Tests OData v4 resource path conventions for addressing resources",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_URLComponents",
	)

	// Test 1: Service root path
	suite.AddTest(
		"test_service_root_path",
		"Service root path returns 200",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 2: Entity set path
	suite.AddTest(
		"test_entity_set_path",
		"Entity set path returns 200",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 3: Single entity by key
	suite.AddTest(
		"test_entity_by_key",
		"Entity by key path returns 200",
		func(ctx *framework.TestContext) error {
			// Get a product ID first
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", resp.StatusCode)
			}

			var result struct {
				Value []struct {
					ID string `json:"ID"`
				} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(result.Value) == 0 {
				return ctx.Skip("no products available for testing")
			}

			productID := result.Value[0].ID
			resp, err = ctx.GET(fmt.Sprintf("/Products(%s)", url.PathEscape(productID)))
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 4: Single entity by key with property name
	suite.AddTest(
		"test_entity_by_named_key",
		"Entity by named key path (ID=value) works or returns 404",
		func(ctx *framework.TestContext) error {
			// Get a product ID first
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", resp.StatusCode)
			}

			var result struct {
				Value []struct {
					ID string `json:"ID"`
				} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(result.Value) == 0 {
				return ctx.Skip("no products available for testing")
			}

			productID := result.Value[0].ID
			resp, err = ctx.GET(fmt.Sprintf("/Products(ID=%s)", url.PathEscape(productID)))
			if err != nil {
				return err
			}

			// Named key syntax may not be implemented
			switch resp.StatusCode {
			case 200:
				return nil
			case 404:
				return ctx.Skip(fmt.Sprintf("named key syntax (ID=%s) not implemented", productID))
			default:
				return fmt.Errorf("named key syntax should return 200 or 404 (got %d)", resp.StatusCode)
			}
		},
	)

	// Test 5: Property path
	suite.AddTest(
		"test_property_path",
		"Property path is valid (200 or 404)",
		func(ctx *framework.TestContext) error {
			// Get a product ID first
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", resp.StatusCode)
			}

			var result struct {
				Value []struct {
					ID string `json:"ID"`
				} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(result.Value) == 0 {
				return ctx.Skip("no products available for testing")
			}

			productID := result.Value[0].ID
			resp, err = ctx.GET(fmt.Sprintf("/Products(%s)/Name", url.PathEscape(productID)))
			if err != nil {
				return err
			}

			// Should return 200 with property value or 404 if not supported
			if resp.StatusCode != 200 && resp.StatusCode != 404 {
				return fmt.Errorf("property path should return 200 or 404 (got %d)", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 6: Property value with $value
	suite.AddTest(
		"test_property_value",
		"Property $value path works or returns 404",
		func(ctx *framework.TestContext) error {
			// Get a product ID first
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", resp.StatusCode)
			}

			var result struct {
				Value []struct {
					ID string `json:"ID"`
				} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(result.Value) == 0 {
				return ctx.Skip("no products available for testing")
			}

			productID := result.Value[0].ID
			resp, err = ctx.GET(fmt.Sprintf("/Products(%s)/Name/$value", url.PathEscape(productID)))
			if err != nil {
				return err
			}

			switch resp.StatusCode {
			case 200:
				return nil
			case 404:
				return ctx.Skip("property $value not implemented")
			default:
				return fmt.Errorf("property $value should return 200 or 404 (got %d)", resp.StatusCode)
			}
		},
	)

	// Test 7: Navigation property path
	suite.AddTest(
		"test_navigation_property",
		"Navigation property path is accessible (200 or 404)",
		func(ctx *framework.TestContext) error {
			// Get a product ID first
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", resp.StatusCode)
			}

			var result struct {
				Value []struct {
					ID string `json:"ID"`
				} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(result.Value) == 0 {
				return ctx.Skip("no products available for testing")
			}

			productID := result.Value[0].ID
			resp, err = ctx.GET(fmt.Sprintf("/Products(%s)/Category", url.PathEscape(productID)))
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 && resp.StatusCode != 404 {
				return fmt.Errorf("navigation property should return 200 or 404 (got %d)", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 8: Chained navigation paths
	suite.AddTest(
		"test_chained_navigation",
		"Chained navigation paths work or are not implemented",
		func(ctx *framework.TestContext) error {
			// Get a product ID first
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", resp.StatusCode)
			}

			var result struct {
				Value []struct {
					ID string `json:"ID"`
				} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(result.Value) == 0 {
				return ctx.Skip("no products available for testing")
			}

			productID := result.Value[0].ID
			resp, err = ctx.GET(fmt.Sprintf("/Products(%s)/Category/Products", url.PathEscape(productID)))
			if err != nil {
				return err
			}

			switch resp.StatusCode {
			case 200:
				return nil
			case 404, 501:
				return ctx.Skip("chained navigation not implemented")
			default:
				return fmt.Errorf("chained navigation should return 200, 404, or 501 (got %d)", resp.StatusCode)
			}
		},
	)

	// Test 9: System resource $metadata
	suite.AddTest(
		"test_metadata_resource",
		"$metadata system resource path returns 200",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 10: Invalid resource path returns 404
	suite.AddTest(
		"test_invalid_resource",
		"Invalid resource path returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/InvalidResource")
			if err != nil {
				return err
			}
			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 11: Entity with non-existent key returns 404
	suite.AddTest(
		"test_nonexistent_key",
		"Non-existent entity key returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products(99999999-9999-9999-9999-999999999999)")
			if err != nil {
				return err
			}
			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 12: Case sensitivity in resource paths
	suite.AddTest(
		"test_case_sensitivity",
		"Resource path case sensitivity (lowercase 'products' returns 404 or 200)",
		func(ctx *framework.TestContext) error {
			// OData resource paths are case-sensitive by default
			resp, err := ctx.GET("/products")
			if err != nil {
				return err
			}

			// Should return 404 for lowercase (case mismatch)
			// However, some servers may be case-insensitive (allowed but not required)
			if resp.StatusCode != 404 && resp.StatusCode != 200 {
				return fmt.Errorf("case mismatch should return 404 or 200 (got %d)", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 13: Path with query options
	suite.AddTest(
		"test_path_with_query",
		"Path with query options returns 200",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=5")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 14: Empty path segments handling
	suite.AddTest(
		"test_empty_path_segments",
		"Empty path segments should return error or redirect",
		func(ctx *framework.TestContext) error {
			// Products// should be invalid (empty segment)
			// Per OData URL conventions, empty path segments are not valid
			resp, err := ctx.GET("/Products//")
			if err != nil {
				return err
			}

			// OData spec: empty path segments are invalid and should return 404, 400, or 301 (redirect)
			if resp.StatusCode == 200 {
				return fmt.Errorf("server accepted invalid URL with empty path segments (should return 400, 404, or 301)")
			}
			if resp.StatusCode != 404 && resp.StatusCode != 400 && resp.StatusCode != 301 {
				return fmt.Errorf("empty path segments must return 404, 400, or 301 per OData spec (got %d)", resp.StatusCode)
			}
			return nil
		},
	)

	return suite
}
