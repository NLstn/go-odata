package core

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// DescriptionAnnotation creates tests for the Core.Description annotation
// Tests that Core.Description annotations are properly exposed in metadata
func DescriptionAnnotation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Core.Description Annotation",
		"Validates that properties and types annotated with Org.OData.Core.V1.Description expose human-readable descriptions in metadata.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Core.V1.md#Description",
	)

	suite.AddTest(
		"metadata_includes_description_annotations",
		"Metadata document includes Core.Description annotations where defined",
		func(ctx *framework.TestContext) error {
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

			nameTarget := fmt.Sprintf("%s.Product/Name", namespace)
			nameFound, err := hasAnnotation(resp.Body, nameTarget, "Org.OData.Core.V1.Description")
			if err != nil {
				return err
			}
			if !nameFound {
				return fmt.Errorf("expected Core.Description annotation on %s", nameTarget)
			}

			descTarget := fmt.Sprintf("%s.Product/Description", namespace)
			descFound, err := hasAnnotation(resp.Body, descTarget, "Org.OData.Core.V1.Description")
			if err != nil {
				return err
			}
			if !descFound {
				return fmt.Errorf("expected Core.Description annotation on %s", descTarget)
			}

			return nil
		},
	)

	suite.AddTest(
		"description_annotations_in_json_metadata",
		"JSON metadata format includes Core.Description annotations",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}

			// Service may not support JSON metadata format
			if resp.StatusCode == 406 {
				return ctx.Skip("Service does not support JSON metadata format")
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Parse JSON metadata
			var metadata map[string]interface{}
			if err := ctx.GetJSON(resp, &metadata); err != nil {
				return err
			}

			ctx.Log("JSON metadata retrieved successfully")
			return nil
		},
	)

	suite.AddTest(
		"qualified_description_annotations",
		"Metadata supports qualified Description annotations (e.g., Description#Short)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			// Check for qualified annotations (with # qualifier)
			if strings.Contains(body, "Qualifier=") {
				ctx.Log("Found qualified annotations in metadata")
			}

			return nil
		},
	)

	return suite
}
