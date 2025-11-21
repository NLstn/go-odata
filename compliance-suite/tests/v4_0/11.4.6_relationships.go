package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// Relationships creates the 11.4.6 Managing Relationships test suite
func Relationships() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.6 Managing Relationships",
		"Tests relationship management (creating, updating, deleting links) according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ManagingRelationships",
	)

	// Helper functions to get fresh entity IDs for each test
	// Note: These must refetch IDs each time because the database is reseeded between tests
	ensureProductSegments := func(ctx *framework.TestContext) (string, string, string, error) {
		ids, err := fetchEntityIDs(ctx, "Products", 2)
		if err != nil {
			return "", "", "", err
		}
		if len(ids) == 0 {
			return "", "", "", fmt.Errorf("no products available for relationship tests")
		}
		productSegment := fmt.Sprintf("Products(%s)", ids[0])
		productPath := "/" + productSegment
		secondProductSegment := productSegment
		if len(ids) > 1 {
			secondProductSegment = fmt.Sprintf("Products(%s)", ids[1])
		}
		return productPath, productSegment, secondProductSegment, nil
	}

	ensureCategorySegment := func(ctx *framework.TestContext) (string, error) {
		id, err := firstEntityID(ctx, "Categories")
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Categories(%s)", id), nil
	}

	// Test 1: Read entity reference with $ref
	suite.AddTest(
		"test_read_entity_reference",
		"Read entity reference with $ref",
		func(ctx *framework.TestContext) error {
			productPath, _, _, err := ensureProductSegments(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Category/$ref")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("$ref operations not implemented")
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.id")
		},
	)

	// Test 2: Read collection of references
	suite.AddTest(
		"test_read_collection_references",
		"Read collection of entity references",
		func(ctx *framework.TestContext) error {
			productPath, _, _, err := ensureProductSegments(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/RelatedProducts/$ref")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("$ref operations not implemented")
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "value")
		},
	)

	// Test 3: Create entity reference (single-valued navigation)
	suite.AddTest(
		"test_create_entity_reference",
		"Create/update entity reference with PUT",
		func(ctx *framework.TestContext) error {
			productPath, _, _, err := ensureProductSegments(ctx)
			if err != nil {
				return err
			}
			categorySegment, err := ensureCategorySegment(ctx)
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"@odata.id": fmt.Sprintf("%s/%s", ctx.ServerURL(), categorySegment),
			}

			resp, err := ctx.PUT(productPath+"/Category/$ref", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("$ref operations not implemented")
			}

			// Should return 204 or 200
			if err := ctx.AssertStatusCode(resp, 204); err != nil {
				return ctx.AssertStatusCode(resp, 200)
			}

			return nil
		},
	)

	// Test 4: Add reference to collection with POST
	suite.AddTest(
		"test_add_reference_collection",
		"Add entity reference to collection with POST",
		func(ctx *framework.TestContext) error {
			productPath, _, secondProductSegment, err := ensureProductSegments(ctx)
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"@odata.id": fmt.Sprintf("%s/%s", ctx.ServerURL(), secondProductSegment),
			}

			resp, err := ctx.POST(productPath+"/RelatedProducts/$ref", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("$ref operations not fully implemented")
			}

			// Should return 204 or 201
			if err := ctx.AssertStatusCode(resp, 204); err != nil {
				return ctx.AssertStatusCode(resp, 201)
			}

			return nil
		},
	)

	// Test 5: Delete entity reference
	suite.AddTest(
		"test_delete_entity_reference",
		"Delete entity reference with DELETE",
		func(ctx *framework.TestContext) error {
			productPath, _, _, err := ensureProductSegments(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.DELETE(productPath + "/Category/$ref")
			if err != nil {
				return err
			}

			if resp.StatusCode == 404 {
				return ctx.Skip("$ref operations not fully implemented")
			}

			return ctx.AssertStatusCode(resp, 204)
		},
	)

	// Test 6: Invalid reference should return 400
	suite.AddTest(
		"test_invalid_reference",
		"Invalid @odata.id in reference returns 400",
		func(ctx *framework.TestContext) error {
			productPath, _, _, err := ensureProductSegments(ctx)
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"@odata.id": "invalid-reference",
			}

			resp, err := ctx.PUT(productPath+"/Category/$ref", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
