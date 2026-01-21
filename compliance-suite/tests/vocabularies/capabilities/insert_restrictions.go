package capabilities

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// InsertRestrictions creates tests for the Capabilities.InsertRestrictions annotation
// Tests that entity sets annotated with InsertRestrictions properly enforce insert capabilities
func InsertRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities.InsertRestrictions Annotation",
		"Validates that entity sets annotated with Org.OData.Capabilities.V1.InsertRestrictions properly advertise and enforce insert capabilities.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Capabilities.V1.md#InsertRestrictions",
	)

	suite.AddTest(
		"metadata_includes_insert_restrictions",
		"Metadata document includes Capabilities.InsertRestrictions annotations where defined",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Capabilities.InsertRestrictions") &&
				!strings.Contains(body, "Org.OData.Capabilities.V1.InsertRestrictions") {
				ctx.Log("Warning: No Capabilities.InsertRestrictions annotations found in metadata")
			}

			return nil
		},
	)

	suite.AddTest(
		"non_insertable_entity_set_rejects_post",
		"POST to entity set with Insertable=false returns appropriate error",
		func(ctx *framework.TestContext) error {
			// This test would need to know which entity sets are non-insertable
			// For now, we'll skip it unless we can discover this from metadata
			return ctx.Skip("Test requires entity set known to be non-insertable")
		},
	)

	suite.AddTest(
		"insertable_entity_set_accepts_post",
		"POST to entity set with Insertable=true or no restriction succeeds",
		func(ctx *framework.TestContext) error {
			payload := `{"Name": "Capabilities Test Product", "Price": 79.99}`

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}

			// Should succeed for insertable entity sets
			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return fmt.Errorf("expected status 201 for insertable entity set: %w", err)
			}

			return nil
		},
	)

	return suite
}
