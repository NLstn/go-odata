package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// SingletonOperations creates the 11.2.16 Singleton Operations test suite
func SingletonOperations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.16 Singleton Operations",
		"Tests singleton entity operations including GET, PATCH, PUT and proper error responses for invalid operations (POST, DELETE).",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Singletons",
	)

	// Test 1: GET singleton returns 200
	suite.AddTest(
		"test_get_singleton",
		"GET singleton returns 200",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Company")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 2: GET singleton returns valid JSON structure
	suite.AddTest(
		"test_singleton_json_structure",
		"Singleton returns proper JSON structure",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Company")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			// Check for @odata.context
			if _, ok := result["@odata.context"]; !ok {
				return fmt.Errorf("response missing '@odata.context'")
			}

			// Check that it's not wrapped in a value array (singletons return direct entity)
			if _, ok := result["value"]; ok {
				return fmt.Errorf("singleton should not be wrapped in 'value' array")
			}

			return nil
		},
	)

	// Test 3: GET singleton with $select
	suite.AddTest(
		"test_singleton_select",
		"Singleton supports $select query option",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Company?$select=Name,CEO")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 4: PATCH singleton updates entity
	suite.AddTest(
		"test_patch_singleton",
		"PATCH updates singleton entity",
		func(ctx *framework.TestContext) error {
			// Get original
			origResp, err := ctx.GET("/Company")
			if err != nil {
				return err
			}

			var original map[string]interface{}
			if err := json.Unmarshal(origResp.Body, &original); err != nil {
				return fmt.Errorf("failed to parse original: %w", err)
			}

			// Update CEO field
			resp, err := ctx.PATCH("/Company", map[string]interface{}{
				"CEO": "Test CEO",
			})
			if err != nil {
				return err
			}

			// Restore original (cleanup)
			_, _ = ctx.PATCH("/Company", original)

			// PATCH should return 204 No Content or 200 OK
			if resp.StatusCode != 204 && resp.StatusCode != 200 {
				return fmt.Errorf("expected status 204 or 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 5: PUT singleton replaces entity
	suite.AddTest(
		"test_put_singleton",
		"PUT replaces singleton entity",
		func(ctx *framework.TestContext) error {
			// Get original
			origResp, err := ctx.GET("/Company")
			if err != nil {
				return err
			}

			var original map[string]interface{}
			if err := json.Unmarshal(origResp.Body, &original); err != nil {
				return fmt.Errorf("failed to parse original: %w", err)
			}

			// Replace with new data
			newData := map[string]interface{}{
				"ID":          original["ID"],
				"Name":        "Test Company",
				"CEO":         "Test CEO",
				"Founded":     2000,
				"HeadQuarter": "Test HQ",
				"Version":     1,
			}

			resp, err := ctx.PUT("/Company", newData)
			if err != nil {
				return err
			}

			// Restore original (cleanup)
			_, _ = ctx.PUT("/Company", original)

			// PUT should return 204 No Content or 200 OK
			if resp.StatusCode != 204 && resp.StatusCode != 200 {
				return fmt.Errorf("expected status 204 or 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 6: POST to singleton should fail (405 Method Not Allowed)
	suite.AddTest(
		"test_post_singleton_fails",
		"POST to singleton returns error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Company", map[string]interface{}{
				"Name": "New Company",
			})
			if err != nil {
				return err
			}

			// Only 405 Method Not Allowed is compliant
			if resp.StatusCode == 405 {
				return nil
			}

			if resp.StatusCode == 501 {
				return fmt.Errorf("status code 501 indicates singleton POST handling is not implemented and is non-compliant")
			}

			return fmt.Errorf("expected status 405 Method Not Allowed, got %d", resp.StatusCode)
		},
	)

	// Test 7: DELETE singleton should fail (405 Method Not Allowed)
	suite.AddTest(
		"test_delete_singleton_fails",
		"DELETE singleton returns error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.DELETE("/Company")
			if err != nil {
				return err
			}

			// Only 405 Method Not Allowed is compliant
			if resp.StatusCode == 405 {
				return nil
			}

			if resp.StatusCode == 501 {
				return fmt.Errorf("status code 501 indicates singleton DELETE handling is not implemented and is non-compliant")
			}

			return fmt.Errorf("expected status 405 Method Not Allowed, got %d", resp.StatusCode)
		},
	)

	// Test 8: Singleton appears in service document
	suite.AddTest(
		"test_singleton_in_service_document",
		"Singleton appears in service document",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, `"kind":"Singleton"`) && !strings.Contains(bodyStr, `"kind": "Singleton"`) {
				return fmt.Errorf("service document does not list singleton with kind=Singleton")
			}

			return nil
		},
	)

	// Test 9: Singleton has proper metadata in $metadata
	suite.AddTest(
		"test_singleton_in_metadata",
		"Singleton defined in metadata document",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			bodyStr := string(resp.Body)
			if !strings.Contains(bodyStr, "Singleton") {
				return fmt.Errorf("metadata document does not define singleton")
			}

			return nil
		},
	)

	// Test 10: Singleton property access
	suite.AddTest(
		"test_singleton_property_access",
		"Singleton property access works",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Company/Name")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("singleton properties not implemented (optional feature)")
			}

			return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
		},
	)

	return suite
}
