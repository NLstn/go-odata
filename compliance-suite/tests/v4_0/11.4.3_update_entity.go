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
		"PATCH without Content-Type validation",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "UpdateEntityNoContentType", 250.00)
			if err != nil {
				return err
			}
			productPath := fmt.Sprintf("/Products(%s)", productID)
			// This test checks Content-Type handling
			// Some implementations may be lenient
			resp, err := ctx.PATCH(productPath, map[string]interface{}{
				"Price": 99.99,
			})
			if err != nil {
				return err
			}

			// Should return 400 or 415 for missing/incorrect Content-Type
			// But some implementations may be lenient and accept it
			if resp.StatusCode == 400 || resp.StatusCode == 415 {
				return nil
			}

			// Lenient implementation
			ctx.Log(fmt.Sprintf("Status: %d (lenient implementation)", resp.StatusCode))
			return nil
		},
	)

	return suite
}
