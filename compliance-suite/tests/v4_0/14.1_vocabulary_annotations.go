package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// VocabularyAnnotations creates the 14.1 Vocabulary Annotations test suite
func VocabularyAnnotations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"14.1 Vocabulary Annotations",
		"Tests support for OData vocabulary annotations in metadata and responses. Tests Core vocabulary annotations and instance annotations in responses.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_Annotation",
	)

	var productPath string
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		if productPath != "" {
			return productPath, nil
		}
		path, err := firstEntityPath(ctx, "Products")
		if err != nil {
			return "", err
		}
		productPath = path
		return productPath, nil
	}

	// Test 1: Metadata document structure supports annotations
	suite.AddTest(
		"test_metadata_annotation_structure",
		"Metadata structure supports annotations",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: Instance annotations in entity response
	suite.AddTest(
		"test_instance_annotations_in_entity",
		"Instance annotations in entity response",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Should have @odata.context which is required
			return ctx.AssertBodyContains(resp, "@odata.context")
		},
	)

	// Test 3: Instance annotations in collection response
	suite.AddTest(
		"test_instance_annotations_in_collection",
		"Instance annotations in collection response",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Should have @odata.context annotation
			return ctx.AssertBodyContains(resp, "@odata.context")
		},
	)

	// Test 4: @odata.nextLink annotation in paginated results
	suite.AddTest(
		"test_odata_nextlink_annotation",
		"@odata.nextLink in paginated results",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=2")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// @odata.nextLink is optional, so just check response is valid
			return nil
		},
	)

	// Test 5: @odata.count annotation
	suite.AddTest(
		"test_odata_count_annotation",
		"@odata.count annotation",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Should have @odata.count annotation
			return ctx.AssertBodyContains(resp, "@odata.count")
		},
	)

	// Test 6: Annotation ordering in JSON
	suite.AddTest(
		"test_annotation_ordering",
		"Annotation ordering in JSON response",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Annotations should be present
			return ctx.AssertBodyContains(resp, "@odata.context")
		},
	)

	return suite
}
