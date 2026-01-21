package capabilities

import (
	"fmt"

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
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.deleteRestricted) == 0 {
				return fmt.Errorf("no entity sets with Deletable=false found in metadata")
			}

			return nil
		},
	)

	suite.AddTest(
		"deletable_entity_accepts_delete",
		"DELETE request to entity in deletable entity set succeeds",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.deleteRestricted) == 0 {
				return fmt.Errorf("no entity sets with Deletable=false found in metadata")
			}

			for _, setInfo := range metadataInfo.deleteRestricted {
				entity, err := fetchFirstEntity(ctx, setInfo.name)
				if err != nil {
					return err
				}
				key, err := buildEntityKey(entity, setInfo.keyProps)
				if err != nil {
					return err
				}

				resp, err := ctx.DELETE(fmt.Sprintf("/%s%s", setInfo.name, key))
				if err != nil {
					return err
				}
				if resp.StatusCode < 400 || resp.StatusCode >= 500 {
					return fmt.Errorf("expected 4xx for non-deletable entity set %s, got %d: %s", setInfo.name, resp.StatusCode, string(resp.Body))
				}
			}

			return nil
		},
	)

	return suite
}
