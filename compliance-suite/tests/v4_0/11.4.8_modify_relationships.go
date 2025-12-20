package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ModifyRelationships creates the 11.4.8 Modify Relationship References test suite
func ModifyRelationships() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.8 Modify Relationships",
		"Tests modifying relationships between entities using $ref endpoints.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ManagingRelationships",
	)

	// Test 1: GET $ref returns reference URL
	suite.AddTest(
		"test_get_ref",
		"GET $ref returns reference URL",
		func(ctx *framework.TestContext) error {
			// Get a product with a category relationship
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			resp, err := ctx.GET(productPath + "/Category/$ref")
			if err != nil {
				return err
			}

			// $ref is mandatory in OData v4 - should not return 404 or 501
			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("$ref is mandatory in OData v4, but got status %d (not implemented)", resp.StatusCode)
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Verify response contains @odata.id with the reference
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON response: %w", err)
			}

			odataID, ok := result["@odata.id"]
			if !ok {
				return fmt.Errorf("response missing @odata.id field")
			}

			// Verify @odata.id is a string and not empty
			odataIDStr, ok := odataID.(string)
			if !ok || odataIDStr == "" {
				return fmt.Errorf("@odata.id must be a non-empty string, got: %v", odataID)
			}

			return nil
		},
	)

	// Test 2: PUT $ref creates/updates single-valued relationship
	suite.AddTest(
		"test_put_ref_single",
		"PUT $ref creates/updates single-valued relationship",
		func(ctx *framework.TestContext) error {
			// Get a product and a category
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return err
			}

			categorySegment := fmt.Sprintf("Categories(%s)", categoryID)

			// Update the product's category relationship using PUT $ref
			payload := map[string]interface{}{
				"@odata.id": fmt.Sprintf("%s/%s", ctx.ServerURL(), categorySegment),
			}

			resp, err := ctx.PUT(productPath+"/Category/$ref", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// $ref is mandatory in OData v4 - should not return 404 or 501
			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("$ref is mandatory in OData v4, but got status %d (not implemented)", resp.StatusCode)
			}

			// PUT $ref should return 204 No Content or 200 OK
			if resp.StatusCode != 204 && resp.StatusCode != 200 {
				return fmt.Errorf("expected status 204 or 200, got %d", resp.StatusCode)
			}

			// Verify the relationship was actually updated by reading it back
			verifyResp, err := ctx.GET(productPath + "/Category/$ref")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(verifyResp, 200); err != nil {
				return fmt.Errorf("failed to verify updated relationship: %w", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(verifyResp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse verification response: %w", err)
			}

			if _, ok := result["@odata.id"]; !ok {
				return fmt.Errorf("verification response missing @odata.id")
			}

			return nil
		},
	)

	// Test 3: POST $ref adds to collection-valued relationship
	suite.AddTest(
		"test_post_ref_collection",
		"POST $ref adds to collection-valued relationship",
		func(ctx *framework.TestContext) error {
			// Get two products to establish a relationship
			productIDs, err := fetchEntityIDs(ctx, "Products", 2)
			if err != nil {
				return err
			}

			if len(productIDs) < 2 {
				return fmt.Errorf("need at least 2 products for relationship test")
			}

			firstProductPath := fmt.Sprintf("/Products(%s)", productIDs[0])
			secondProductSegment := fmt.Sprintf("Products(%s)", productIDs[1])

			// Add second product to first product's RelatedProducts collection
			payload := map[string]interface{}{
				"@odata.id": fmt.Sprintf("%s/%s", ctx.ServerURL(), secondProductSegment),
			}

			resp, err := ctx.POST(firstProductPath+"/RelatedProducts/$ref", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// $ref is mandatory in OData v4 - should not return 404 or 501
			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("$ref is mandatory in OData v4, but got status %d (not implemented)", resp.StatusCode)
			}

			// POST $ref should return 204 No Content or 201 Created
			if resp.StatusCode != 204 && resp.StatusCode != 201 {
				return fmt.Errorf("expected status 204 or 201, got %d", resp.StatusCode)
			}

			// Verify the relationship was added by reading the collection
			verifyResp, err := ctx.GET(firstProductPath + "/RelatedProducts/$ref")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(verifyResp, 200); err != nil {
				return fmt.Errorf("failed to verify added relationship: %w", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(verifyResp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse verification response: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("verification response missing value array")
			}

			return nil
		},
	)

	// Test 4: DELETE $ref removes relationship
	suite.AddTest(
		"test_delete_ref",
		"DELETE $ref removes relationship",
		func(ctx *framework.TestContext) error {
			// Get a product with a category relationship
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			resp, err := ctx.DELETE(productPath + "/Category/$ref")
			if err != nil {
				return err
			}

			// $ref is mandatory in OData v4 - should not return 404 or 501
			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("$ref is mandatory in OData v4, but got status %d (not implemented)", resp.StatusCode)
			}

			// DELETE $ref should return 204 No Content
			if err := ctx.AssertStatusCode(resp, 204); err != nil {
				return err
			}

			// Verify the relationship was actually deleted
			verifyResp, err := ctx.GET(productPath + "/Category/$ref")
			if err != nil {
				return err
			}

			// After deletion, the reference should either be null (204) or not found (404)
			// or return an empty/null @odata.id (200 with null)
			if verifyResp.StatusCode != 204 && verifyResp.StatusCode != 404 && verifyResp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, 204, or 404 for deleted reference, got %d", verifyResp.StatusCode)
			}

			// If it returns 200, verify @odata.id is null or absent
			if verifyResp.StatusCode == 200 {
				var result map[string]interface{}
				if err := json.Unmarshal(verifyResp.Body, &result); err != nil {
					return fmt.Errorf("failed to parse verification response: %w", err)
				}

				odataID, hasID := result["@odata.id"]
				if hasID && odataID != nil {
					return fmt.Errorf("expected @odata.id to be null or absent after deletion, got: %v", odataID)
				}
			}

			return nil
		},
	)

	// Test 5: Invalid $ref URL returns error
	suite.AddTest(
		"test_invalid_ref_url",
		"Invalid $ref URL returns error",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			// Try to set an invalid reference URL
			payload := map[string]interface{}{
				"@odata.id": "this-is-not-a-valid-reference",
			}

			resp, err := ctx.PUT(productPath+"/Category/$ref", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// Should return 400 Bad Request for invalid reference
			if err := ctx.AssertStatusCode(resp, 400); err != nil {
				return fmt.Errorf("invalid @odata.id should return 400 Bad Request: %w", err)
			}

			return nil
		},
	)

	// Test 6: $ref to non-existent navigation property returns error
	suite.AddTest(
		"test_ref_nonexistent_property",
		"$ref to non-existent navigation property returns 404",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			// Try to access $ref on a non-existent navigation property
			resp, err := ctx.GET(productPath + "/NonExistentNavProperty/$ref")
			if err != nil {
				return err
			}

			// Should return 404 Not Found for non-existent navigation property
			if err := ctx.AssertStatusCode(resp, 404); err != nil {
				return fmt.Errorf("non-existent navigation property should return 404 Not Found: %w", err)
			}

			return nil
		},
	)

	return suite
}
