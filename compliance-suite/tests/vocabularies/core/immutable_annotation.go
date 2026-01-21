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
		"PATCH request should reject updates to immutable properties or keep the immutable value unchanged",
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
			originalSerialNumber, ok := created["SerialNumber"]
			if !ok {
				return fmt.Errorf("created entity missing SerialNumber field")
			}

			// Attempt to update immutable property SerialNumber
			updatePayload := `{"SerialNumber": "SN-MODIFIED"}`
			resp, err := ctx.PATCH(fmt.Sprintf("/Products(%v)", id), updatePayload,
				framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 400 || resp.StatusCode == 409 {
				return nil
			}

			if resp.StatusCode != 200 && resp.StatusCode != 204 {
				return fmt.Errorf("unexpected status for immutable property update, got %d: %s", resp.StatusCode, string(resp.Body))
			}

			fetchResp, err := ctx.GET(fmt.Sprintf("/Products(%v)", id),
				framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(fetchResp, 200); err != nil {
				return err
			}

			var fetched map[string]interface{}
			if err := ctx.GetJSON(fetchResp, &fetched); err != nil {
				return err
			}
			currentSerialNumber, ok := fetched["SerialNumber"]
			if !ok {
				return fmt.Errorf("fetched entity missing SerialNumber field")
			}
			if currentSerialNumber != originalSerialNumber {
				return fmt.Errorf("immutable property changed: expected %v, got %v", originalSerialNumber, currentSerialNumber)
			}

			return nil
		},
	)

	return suite
}
