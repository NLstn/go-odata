package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// DeleteEntity creates the 11.4.4 Delete an Entity test suite
func DeleteEntity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.4 Delete an Entity",
		"Tests DELETE operations for removing entities according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_DeleteanEntity",
	)

	// Test 1: DELETE returns 204 No Content on success
	suite.AddTest(
		"test_delete_success",
		"DELETE returns 204 No Content",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Product To Delete", 10.00)
			if err != nil {
				return err
			}
			// Create entity to delete
			createResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				return fmt.Errorf("could not create test entity (status: %d)", createResp.StatusCode)
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			deleteID, err := parseEntityID(createResult["ID"])
			if err != nil {
				return err
			}

			// Delete the entity
			deleteResp, err := ctx.DELETE(fmt.Sprintf("/Products(%s)", deleteID))
			if err != nil {
				return err
			}

			if deleteResp.StatusCode != 204 {
				return fmt.Errorf("expected status 204, got %d", deleteResp.StatusCode)
			}

			return nil
		},
	)

	// Test 2: DELETE to non-existent entity returns 404
	suite.AddTest(
		"test_delete_nonexistent",
		"DELETE to non-existent entity returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.DELETE("/Products(00000000-0000-0000-0000-000000000000)")
			if err != nil {
				return err
			}

			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 3: Verify entity is actually deleted
	suite.AddTest(
		"test_verify_deleted",
		"Deleted entity returns 404 on GET",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Product To Verify Delete", 20.00)
			if err != nil {
				return err
			}
			// Create entity to delete
			createResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				return fmt.Errorf("could not create test entity")
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			verifyID, err := parseEntityID(createResult["ID"])
			if err != nil {
				return err
			}

			// Delete the entity
			_, err = ctx.DELETE(fmt.Sprintf("/Products(%s)", verifyID))
			if err != nil {
				return err
			}

			// Try to retrieve it
			verifyResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", verifyID))
			if err != nil {
				return err
			}

			if verifyResp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", verifyResp.StatusCode)
			}

			return nil
		},
	)

	return suite
}
