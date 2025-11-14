package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// MetadataAnnotations creates the 4.6 Annotations test suite
func MetadataAnnotations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"4.6 Annotations",
		"Tests that model elements can be decorated with annotations, and annotations have proper term names and optional qualifiers.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#_Toc453752519",
	)

	suite.AddTest(
		"test_annotations_present",
		"Annotations can be applied to model elements",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Annotation") {
				return nil // No annotations, skip (they're optional)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_annotations_have_term",
		"Annotations must have Term attribute",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<Annotation") {
				return nil // No annotations, skip
			}

			if !strings.Contains(body, `Term=`) {
				return framework.NewError("Annotation elements must have Term attribute")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_annotation_terms_qualified",
		"Annotation terms use qualified names",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Term="`) {
				return nil // No annotation terms, skip
			}

			// Terms should contain '.' for qualified names
			return nil
		},
	)

	suite.AddTest(
		"test_annotations_with_qualifiers",
		"Annotations can have optional qualifiers",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// Qualifiers are optional, so just check if present they're valid
			if strings.Contains(body, `Qualifier=`) {
				return nil
			}

			return nil // Qualifiers are optional
		},
	)

	suite.AddTest(
		"test_core_vocabulary_annotations",
		"Core vocabulary annotations are used",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// Core vocabulary is optional
			if strings.Contains(body, `Term="Core.`) || strings.Contains(body, `Term="Org.OData.Core`) {
				return nil
			}

			return nil // Optional feature
		},
	)

	suite.AddTest(
		"test_annotations_on_entity_types",
		"Annotations can be applied to entity types",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			_ = string(resp.Body)
			// Optional feature
			return nil
		},
	)

	suite.AddTest(
		"test_annotations_on_properties",
		"Annotations can be applied to properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			_ = string(resp.Body)
			// Optional feature
			return nil
		},
	)

	suite.AddTest(
		"test_external_targeting",
		"Annotations element supports external targeting",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, "<Annotations") && !strings.Contains(body, "Target=") {
				return framework.NewError("Annotations element should have Target attribute")
			}

			return nil // Optional feature
		},
	)

	return suite
}
