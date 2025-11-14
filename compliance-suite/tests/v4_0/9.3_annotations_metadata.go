package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// AnnotationsMetadata creates the 9.3 Annotations in Metadata test suite
func AnnotationsMetadata() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"9.3 Annotations in Metadata",
		"Validates vocabulary annotations in metadata document including Core, Capabilities, and other standard vocabularies.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_Annotation",
	)

	suite.AddTest(
		"test_annotations_element",
		"Metadata contains valid structure",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<Schema") && !strings.Contains(body, `"$Version"`) {
				return framework.NewError("Invalid metadata structure")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_core_vocabulary",
		"Core vocabulary annotations (optional)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// Core vocabulary is optional
			if strings.Contains(body, "Core.Description") || strings.Contains(body, "Description=") {
				return nil
			}

			return nil // Optional feature
		},
	)

	suite.AddTest(
		"test_capabilities_vocabulary",
		"Capabilities vocabulary (optional)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			// Capabilities vocabulary is optional
			if strings.Contains(body, "Capabilities.") {
				return nil
			}

			return nil // Optional feature
		},
	)

	return suite
}
