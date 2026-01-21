package capabilities

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// UpdateRestrictions creates tests for the Capabilities.UpdateRestrictions annotation
// Tests that entity sets annotated with UpdateRestrictions properly enforce update capabilities
func UpdateRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.UpdateRestrictions Annotation",
		"Validates that entity sets annotated with Org.OData.Capabilities.V1.UpdateRestrictions properly advertise and enforce update capabilities.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Capabilities.V1.md#UpdateRestrictions",
	)

	suite.AddTest(
		"metadata_includes_update_restrictions",
		"Metadata document includes Capabilities.UpdateRestrictions annotations where defined",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Capabilities.UpdateRestrictions") &&
				!strings.Contains(body, "Org.OData.Capabilities.V1.UpdateRestrictions") {
				ctx.Log("Info: No Capabilities.UpdateRestrictions annotations found in metadata")
			}

			return nil
		},
	)

	suite.AddTest(
		"updatable_entity_set_accepts_patch",
		"PATCH to entity in updatable entity set succeeds",
		func(ctx *framework.TestContext) error {
			// First create an entity to update
			createPayload := `{"Name": "Update Test Product", "Price": 29.99}`
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

			// Update the entity
			updatePayload := `{"Name": "Updated Product Name"}`
			resp, err := ctx.PATCH(fmt.Sprintf("/Products(%v)", id), updatePayload,
				framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// Should succeed for updatable entity sets
			if resp.StatusCode != 200 && resp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204 for updatable entity, got %d: %s", resp.StatusCode, string(resp.Body))
			}

			return nil
		},
	)

	return suite
}
