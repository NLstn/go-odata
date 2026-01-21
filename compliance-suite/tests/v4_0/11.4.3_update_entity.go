package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// UpdateEntity creates the 11.4.3 Update an Entity test suite
func UpdateEntity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.3 Update an Entity",
		"Tests PATCH and PUT operations for updating entities according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_UpdateanEntity",
	)

	// Test 1: PATCH updates specified properties only
	suite.AddTest(
		"test_patch_update",
		"PATCH updates specified properties (partial update)",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "UpdateEntityPatch", 199.99)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			resp, err := ctx.PATCH(productPath, map[string]interface{}{
				"Price": 149.99,
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 204 && resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200 or 204, got %d", resp.StatusCode)
			}

			// Verify the update
			verifyResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(verifyResp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse verification JSON: %w", err)
			}

			price, ok := result["Price"].(float64)
			if !ok || price != 149.99 {
				return fmt.Errorf("price not updated correctly")
			}

			return nil
		},
	)

	// Test 2: PATCH with invalid property returns error
	suite.AddTest(
		"test_patch_invalid_property",
		"PATCH with invalid property returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "UpdateEntityInvalidProp", 199.99)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			resp, err := ctx.PATCH(productPath, map[string]interface{}{
				"NonExistentProperty": "value",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 3: PATCH to non-existent entity returns 404
	suite.AddTest(
		"test_patch_not_found",
		"PATCH to non-existent entity returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.PATCH("/Products(00000000-0000-0000-0000-000000000000)", map[string]interface{}{
				"Price": 100,
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 4: Content-Type header validation
	suite.AddTest(
		"test_patch_no_content_type",
		"PATCH without Content-Type returns 400 or 415",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "UpdateEntityNoContentType", 250.00)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)
			payload, err := json.Marshal(map[string]interface{}{
				"Price": 99.99,
			})
			if err != nil {
				return fmt.Errorf("failed to marshal payload: %w", err)
			}
			resp, err := ctx.PATCHRawNoContentType(productPath, payload)
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 && resp.StatusCode != 415 {
				return fmt.Errorf("expected status 400 or 415, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// PUT tests - full entity replacement
	suite.AddTest(
		"test_put_full_replacement",
		"PUT replaces entire entity (full update)",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "UpdateEntityPut", 299.99)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			// Get the current entity to get required fields
			getResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(getResp, 200); err != nil {
				return err
			}

			var current map[string]interface{}
			if err := json.Unmarshal(getResp.Body, &current); err != nil {
				return fmt.Errorf("failed to parse current entity: %w", err)
			}

			// Create complete replacement entity
			replacement := map[string]interface{}{
				"Name":       "Completely Replaced Product",
				"Price":      399.99,
				"CategoryID": current["CategoryID"], // Keep required foreign key
				"Status":     current["Status"],     // Keep required status
			}

			resp, err := ctx.PUT(productPath, replacement,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			// Should return 200 or 204
			if resp.StatusCode != 200 && resp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204, got %d", resp.StatusCode)
			}

			// Verify the replacement
			verifyResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(verifyResp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse verification JSON: %w", err)
			}

			// Check that all fields were replaced
			if name, ok := result["Name"].(string); !ok || name != "Completely Replaced Product" {
				return fmt.Errorf("name not replaced correctly")
			}

			if price, ok := result["Price"].(float64); !ok || price != 399.99 {
				return fmt.Errorf("price not replaced correctly")
			}

			ctx.Log("Entity fully replaced with PUT")
			return nil
		},
	)

	suite.AddTest(
		"test_put_response_body_validation",
		"PUT with 200 response includes updated entity body",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "PutResponseBody", 199.99)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)

			// Get current entity for required fields
			getResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			var current map[string]interface{}
			if err := json.Unmarshal(getResp.Body, &current); err != nil {
				return err
			}

			replacement := map[string]interface{}{
				"Name":       "Updated Name",
				"Price":      249.99,
				"CategoryID": current["CategoryID"],
				"Status":     current["Status"],
			}

			resp, err := ctx.PUT(productPath, replacement,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			if resp.StatusCode == 204 {
				// 204 No Content is fine, body validation not needed
				ctx.Log("PUT returned 204 No Content (valid)")
				return nil
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200 or 204, got %d", resp.StatusCode)
			}

			// With 200, body should contain updated entity
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("200 response should have valid JSON body: %w", err)
			}

			// Should have ID field
			if _, ok := result["ID"]; !ok {
				return fmt.Errorf("200 response body should include entity with ID")
			}

			// Should have updated values
			if name, ok := result["Name"].(string); !ok || name != "Updated Name" {
				return fmt.Errorf("200 response body should reflect updated values")
			}

			ctx.Log("PUT with 200 response correctly includes updated entity")
			return nil
		},
	)

	suite.AddTest(
		"test_put_nonexistent_entity",
		"PUT on non-existent entity returns 404",
		func(ctx *framework.TestContext) error {
			nonExistentPath := nonExistingEntityPath("Products")

			// Get a valid CategoryID first
			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return err
			}

			replacement := map[string]interface{}{
				"Name":       "Should Not Be Created",
				"Price":      99.99,
				"CategoryID": categoryID,
				"Status":     1,
			}

			resp, err := ctx.PUT(nonExistentPath, replacement,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			// PUT should not create new entities - use POST for that
			// Should return 404
			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404 for PUT on non-existent entity, got %d", resp.StatusCode)
			}

			ctx.Log("Correctly returned 404 for PUT on non-existent entity")
			return nil
		},
	)

	return suite
}
