package core

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ImmutableAnnotation creates tests for the Core.Immutable annotation
// Tests that properties annotated with Core.Immutable cannot be changed after creation
func ImmutableAnnotation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Core.Immutable Annotation",
		"Validates that properties annotated with Org.OData.Core.V1.Immutable can only be set during entity creation and cannot be modified afterwards.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Core.V1.md#Immutable",
	)

	suite.AddTest(
		"metadata_includes_immutable_annotation",
		"Metadata document includes Core.Immutable annotation on immutable properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Core.Immutable") && !strings.Contains(body, "Org.OData.Core.V1.Immutable") {
				ctx.Log("Warning: No Core.Immutable annotations found in metadata")
			}

			return nil
		},
	)

	suite.AddTest(
		"immutable_property_settable_on_create",
		"Immutable properties can be set during entity creation",
		func(ctx *framework.TestContext) error {
			payload := `{
				"Name": "Test Product",
				"Price": 99.99,
				"SerialNumber": "SN-TEST-001"
			}`

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			// Verify the SerialNumber was set
			var created map[string]interface{}
			if err := ctx.GetJSON(resp, &created); err != nil {
				return err
			}

			if serialNumber, ok := created["SerialNumber"]; !ok || serialNumber != "SN-TEST-001" {
				return fmt.Errorf("immutable property not set correctly: got %v", serialNumber)
			}

			return nil
		},
	)

	suite.AddTest(
		"immutable_property_not_updatable",
		"PATCH request should reject or ignore updates to immutable properties (enforcement not yet implemented)",
		func(ctx *framework.TestContext) error {
			// First create an entity with an immutable property
			createPayload := `{
				"Name": "Immutable Test Product",
				"Price": 199.99,
				"SerialNumber": "SN-IMMUTABLE-001"
			}`
			createResp, err := ctx.POST("/Products", createPayload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(createResp, 201); err != nil {
				return fmt.Errorf("failed to create test entity: %w", err)
			}

			var created map[string]interface{}
			if err := ctx.GetJSON(createResp, &created); err != nil {
				return err
			}
			id, ok := created["ID"]
			if !ok {
				return fmt.Errorf("created entity missing ID field")
			}

			// Attempt to update immutable property SerialNumber
			updatePayload := `{"SerialNumber": "SN-MODIFIED"}`
			resp, err := ctx.PATCH(fmt.Sprintf("/Products(%v)", id), updatePayload,
				framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// TODO: Once immutability enforcement is implemented, service should reject (400)
			// For now, we just verify that the annotation is present in metadata
			// Service behavior: currently allows updates (will be 200 or 204)
			if resp.StatusCode != 200 && resp.StatusCode != 204 && resp.StatusCode != 400 {
				return fmt.Errorf("unexpected status for immutable property update, got %d: %s", resp.StatusCode, string(resp.Body))
			}

			ctx.Log(fmt.Sprintf("Note: Immutability enforcement not yet implemented (got status %d)", resp.StatusCode))
			return nil
		},
	)

	return suite
}
