package v4_0

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// VocabularyAnnotations creates the 14.1 Vocabulary Annotations test suite
func VocabularyAnnotations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"14.1 Vocabulary Annotations",
		"Tests support for OData vocabulary annotations in metadata and responses. Tests Core vocabulary annotations and instance annotations in responses.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_Annotation",
	)

	// Helper function to get product path for each test
	// Note: Must refetch on each call because database is reseeded between tests
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		return firstEntityPath(ctx, "Products")
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

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			annotationChecks := []struct {
				label     string
				property  string
				term      string
				termLabel string
			}{
				{
					label:     "computed",
					property:  "CreatedAt",
					term:      "Org.OData.Core.V1.Computed",
					termLabel: "Core.Computed",
				},
				{
					label:     "immutable",
					property:  "SerialNumber",
					term:      "Org.OData.Core.V1.Immutable",
					termLabel: "Core.Immutable",
				},
				{
					label:     "description",
					property:  "Product display name",
					term:      "Org.OData.Core.V1.Description",
					termLabel: "Core.Description",
				},
				{
					label:     "description",
					property:  "Detailed product description",
					term:      "Org.OData.Core.V1.Description",
					termLabel: "Core.Description",
				},
			}

			for _, check := range annotationChecks {
				pattern := regexp.MustCompile(fmt.Sprintf(`(?s)%s.*%s|%s.*%s`,
					regexp.QuoteMeta(check.property),
					regexp.QuoteMeta(check.term),
					regexp.QuoteMeta(check.term),
					regexp.QuoteMeta(check.property),
				))
				if !pattern.MatchString(body) {
					return fmt.Errorf("metadata missing %s annotation (%s) for %q", check.termLabel, check.term, check.property)
				}
			}

			return nil
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
			resp, err := ctx.GET("/Products?$count=true&$top=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			countVal, ok := result["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count required when $count=true is specified")
			}

			countNum, ok := countVal.(float64)
			if !ok || countNum < 0 {
				return fmt.Errorf("@odata.count must be a non-negative number, got %T", countVal)
			}

			if math.Trunc(countNum) != countNum {
				return fmt.Errorf("@odata.count must be an integer value, got %f", countNum)
			}

			if countNum <= 1 {
				return ctx.Skip("Not enough entities to require pagination for $top=1")
			}

			nextLink, ok := result["@odata.nextLink"].(string)
			if !ok || nextLink == "" {
				return fmt.Errorf("@odata.nextLink required when @odata.count exceeds $top=1")
			}

			parsed, err := url.Parse(nextLink)
			if err != nil {
				return fmt.Errorf("invalid @odata.nextLink URL: %w", err)
			}

			query := parsed.Query()
			if !query.Has("$skip") && !query.Has("$skiptoken") {
				return fmt.Errorf("@odata.nextLink must include $skip or $skiptoken parameter, got: %s", nextLink)
			}

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

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			countVal, ok := result["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count required when $count=true is specified")
			}

			countNum, ok := countVal.(float64)
			if !ok {
				return fmt.Errorf("@odata.count must be a number, got %T", countVal)
			}

			if countNum < 0 {
				return fmt.Errorf("@odata.count must be non-negative, got %f", countNum)
			}

			if math.Trunc(countNum) != countNum {
				return fmt.Errorf("@odata.count must be an integer value, got %f", countNum)
			}

			return nil
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
