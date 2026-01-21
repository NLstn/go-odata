package core

import (
	"fmt"

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

			namespace, err := metadataNamespace(resp.Body)
			if err != nil {
				return err
			}

			target := fmt.Sprintf("%s.Product/CreatedAt", namespace)
			found, err := hasAnnotation(resp.Body, target, "Org.OData.Core.V1.Computed")
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("expected Core.Computed annotation on %s", target)
			}

			return nil
		},
	)

	suite.AddTest(
		"computed_property_not_settable_on_create",
		"POST request ignores computed properties in request body",
		func(ctx *framework.TestContext) error {
			// Attempt to create an entity with a computed property set
			clientValue := "2026-01-21T00:00:00Z"
			payload := fmt.Sprintf(`{
				"Name": "Test Product",
				"Price": 99.99,
				"CreatedAt": "%s"
			}`, clientValue)

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

			if resp.StatusCode != 201 {
				return nil
			}

			var created map[string]interface{}
			if err := ctx.GetJSON(resp, &created); err != nil {
				return err
			}
			id, ok := created["ID"]
			if !ok {
				return fmt.Errorf("created entity missing ID field")
			}

			fetchResp, err := ctx.GET(fmt.Sprintf("/Products(%v)", id))
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
			createdAt, ok := fetched["CreatedAt"].(string)
			if !ok {
				return fmt.Errorf("fetched entity missing CreatedAt field")
			}
			if createdAt == clientValue {
				return fmt.Errorf("computed CreatedAt should be server-controlled, got client value %s", createdAt)
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

			clientValue := "2030-01-01T00:00:00Z"
			updatePayload := fmt.Sprintf(`{"CreatedAt": "%s"}`, clientValue)
			resp, err := ctx.PATCH(fmt.Sprintf("/Products(%v)", id), updatePayload,
				framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// Service should either ignore the update (200/204) or reject it (400)
			if resp.StatusCode != 200 && resp.StatusCode != 204 && resp.StatusCode != 400 {
				return fmt.Errorf("expected status 200, 204, or 400, got %d: %s", resp.StatusCode, string(resp.Body))
			}

			if resp.StatusCode != 200 && resp.StatusCode != 204 {
				return nil
			}

			fetchResp, err := ctx.GET(fmt.Sprintf("/Products(%v)", id))
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
			createdAt, ok := fetched["CreatedAt"].(string)
			if !ok {
				return fmt.Errorf("fetched entity missing CreatedAt field")
			}
			if createdAt == clientValue {
				return fmt.Errorf("computed CreatedAt should be server-controlled, got client value %s", createdAt)
			}

			return nil
		},
	)

	return suite
}
