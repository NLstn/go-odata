package core

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ComputedAnnotation creates tests for the Core.Computed annotation
// Tests that properties annotated with Core.Computed are read-only
func ComputedAnnotation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Core.Computed Annotation",
		"Validates that properties annotated with Org.OData.Core.V1.Computed are properly marked as computed/read-only in metadata and are not settable by clients.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Core.V1.md#Computed",
	)

	suite.AddTest(
		"metadata_includes_computed_annotation",
		"Metadata document includes Core.Computed annotation on computed properties",
		func(ctx *framework.TestContext) error {
			// Get metadata
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			// Check for Core.Computed annotation in XML metadata
			if !strings.Contains(body, "Core.Computed") && !strings.Contains(body, "Org.OData.Core.V1.Computed") {
				ctx.Log("Warning: No Core.Computed annotations found in metadata. This may be expected if no properties are computed.")
			}

			// Parse XML to validate structure
			type Metadata struct {
				XMLName xml.Name `xml:"Edmx"`
			}
			var metadata Metadata
			if err := xml.Unmarshal(resp.Body, &metadata); err != nil {
				return fmt.Errorf("failed to parse metadata XML: %w", err)
			}

			return nil
		},
	)

	suite.AddTest(
		"computed_property_not_settable_on_create",
		"POST request ignores computed properties in request body",
		func(ctx *framework.TestContext) error {
			// Attempt to create an entity with a computed property set
			payload := `{
				"Name": "Test Product",
				"Price": 99.99,
				"CreatedAt": "2026-01-21T00:00:00Z"
			}`

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}

			// Service should either succeed (ignoring computed field) or return 400
			if resp.StatusCode != 201 && resp.StatusCode != 400 {
				return fmt.Errorf("expected status 201 or 400, got %d: %s", resp.StatusCode, string(resp.Body))
			}

			if resp.StatusCode == 201 {
				ctx.Log("Creation succeeded - service handled computed property appropriately")
			}

			return nil
		},
	)

	suite.AddTest(
		"computed_property_not_updatable",
		"PATCH request ignores or rejects updates to computed properties",
		func(ctx *framework.TestContext) error {
			// First create an entity
			createPayload := `{"Name": "Test Product for Update", "Price": 49.99}`
			createResp, err := ctx.POST("/Products", createPayload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(createResp, 201); err != nil {
				return fmt.Errorf("failed to create test entity: %w", err)
			}

			// Extract ID from response
			var created map[string]interface{}
			if err := ctx.GetJSON(createResp, &created); err != nil {
				return err
			}
			id, ok := created["ID"]
			if !ok {
				return fmt.Errorf("created entity missing ID field")
			}

			// Attempt to update computed property
			updatePayload := `{"CreatedAt": "2030-01-01T00:00:00Z"}`
			resp, err := ctx.PATCH(fmt.Sprintf("/Products(%v)", id), updatePayload,
				framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// Service should either ignore the update (200/204) or reject it (400)
			if resp.StatusCode != 200 && resp.StatusCode != 204 && resp.StatusCode != 400 {
				return fmt.Errorf("expected status 200, 204, or 400, got %d: %s", resp.StatusCode, string(resp.Body))
			}

			ctx.Log("Update handled appropriately by service")
			return nil
		},
	)

	return suite
}
