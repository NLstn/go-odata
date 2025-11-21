package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// InstanceAnnotations creates the 11.6 Annotations test suite
func InstanceAnnotations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.6 Annotations",
		"Tests handling of instance annotations, control information, and custom annotations in OData responses.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_InstanceAnnotations",
	)

	// Test 1: Standard @odata.context annotation
	suite.AddTest(
		"test_odata_context",
		"@odata.context annotation present",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.context")
		},
	)

	// Test 2: @odata.count annotation
	suite.AddTest(
		"test_odata_count",
		"@odata.count annotation with $count",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.count")
		},
	)

	// Test 3: @odata.id annotation for entities
	suite.AddTest(
		"test_odata_id",
		"@odata.id annotation for entity",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.id")
		},
	)

	// Test 4: Annotations in metadata=full
	suite.AddTest(
		"test_annotations_full_metadata",
		"Annotations in metadata=full",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{Key: "Accept", Value: "application/json;odata.metadata=full"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertBodyContains(resp, "@odata")
		},
	)

	// Test 5: No annotations in metadata=none
	suite.AddTest(
		"test_no_annotations_none_metadata",
		"No annotations in metadata=none",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{Key: "Accept", Value: "application/json;odata.metadata=none"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// @odata.context should not be present
			body := string(resp.Body)
			if framework.ContainsAny(body, `"@odata.context"`) {
				return framework.NewError("@odata.context should not be present in metadata=none")
			}

			return nil
		},
	)

	// Test 6: Annotations in collections
	suite.AddTest(
		"test_annotations_in_collections",
		"Annotations in collection responses",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "@odata.context")
		},
	)

	return suite
}
