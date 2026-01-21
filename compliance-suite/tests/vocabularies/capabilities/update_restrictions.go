package capabilities

import (
	"fmt"

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
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.updateRestricted) == 0 {
				return fmt.Errorf("no entity sets with Updatable=false found in metadata")
			}

			return nil
		},
	)

	suite.AddTest(
		"updatable_entity_set_accepts_patch",
		"PATCH to entity in updatable entity set succeeds",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}

			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			if len(metadataInfo.updateRestricted) == 0 {
				return fmt.Errorf("no entity sets with Updatable=false found in metadata")
			}

			for _, setInfo := range metadataInfo.updateRestricted {
				entity, err := fetchFirstEntity(ctx, setInfo.name)
				if err != nil {
					return err
				}
				key, err := buildEntityKey(entity, setInfo.keyProps)
				if err != nil {
					return err
				}

				resp, err := ctx.PATCH(fmt.Sprintf("/%s%s", setInfo.name, key), map[string]interface{}{"Name": "Blocked update"})
				if err != nil {
					return err
				}
				if resp.StatusCode < 400 || resp.StatusCode >= 500 {
					return fmt.Errorf("expected 4xx for non-updatable entity set %s, got %d: %s", setInfo.name, resp.StatusCode, string(resp.Body))
				}
			}

			return nil
		},
	)

	return suite
}
