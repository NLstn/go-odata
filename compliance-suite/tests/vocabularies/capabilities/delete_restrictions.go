package capabilities

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// DeleteRestrictions creates tests for the Capabilities.DeleteRestrictions annotation
// Tests that entity sets annotated with DeleteRestrictions properly enforce delete capabilities
func DeleteRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.DeleteRestrictions Annotation",
		"Validates that entity sets annotated with Org.OData.Capabilities.V1.DeleteRestrictions properly advertise and enforce delete capabilities.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Capabilities.V1.md#DeleteRestrictions",
	)

	suite.AddTest(
		"metadata_includes_delete_restrictions",
		"Metadata document includes Capabilities.DeleteRestrictions annotations where defined",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Capabilities.DeleteRestrictions") &&
				!strings.Contains(body, "Org.OData.Capabilities.V1.DeleteRestrictions") {
				ctx.Log("Warning: No Capabilities.DeleteRestrictions annotations found in metadata")
			}

			return nil
		},
	)

	suite.AddTest(
		"deletable_entity_accepts_delete",
		"DELETE request to entity in deletable entity set succeeds",
		func(ctx *framework.TestContext) error {
			// First create an entity to delete
			createPayload := `{"Name": "Delete Test Product", "Price": 9.99}`
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

			// Delete the entity
			resp, err := ctx.DELETE(fmt.Sprintf("/Products(%v)", id))
			if err != nil {
				return err
			}

			// Should succeed for deletable entity sets
			if err := ctx.AssertStatusCode(resp, 204); err != nil {
				return fmt.Errorf("expected status 204 for deletable entity: %w", err)
			}

			return nil
		},
	)

	return suite
}
