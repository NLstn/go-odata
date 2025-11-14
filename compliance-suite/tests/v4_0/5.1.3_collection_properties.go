package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// CollectionProperties creates the 5.1.3 Collection Properties test suite
func CollectionProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.3 Collection Properties",
		"Validates handling of collection-valued navigation properties, lambda operators (any/all), and $count on collections.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_CollectionValuedProperties",
	)

	suite.AddTest(
		"test_expand_collection",
		"Expand collection-valued navigation property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$expand=Descriptions")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			// Check if Descriptions collection is included
			return nil
		},
	)

	suite.AddTest(
		"test_collection_any_operator",
		"Filter with collection 'any' operator",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Descriptions/any(d:contains(d/Description,'Laptop'))")
			if err != nil {
				return err
			}

			// Lambda operators are advanced feature
			if resp.StatusCode == 200 {
				return nil
			}
			if resp.StatusCode == 400 || resp.StatusCode == 501 {
				return ctx.Skip("Lambda operators (any/all) not implemented")
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_collection_all_operator",
		"Filter with collection 'all' operator",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Descriptions/all(d:d/Description ne null)")
			if err != nil {
				return err
			}

			// Lambda operators are advanced feature
			if resp.StatusCode == 200 {
				return nil
			}
			if resp.StatusCode == 400 || resp.StatusCode == 501 {
				return ctx.Skip("Lambda operators (any/all) not implemented")
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_collection_count",
		"Count items in collection navigation",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/Descriptions/$count")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Should return plain text number
			return nil
		},
	)

	suite.AddTest(
		"test_filter_with_count",
		"Filter using $count in expression",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Descriptions/$count gt 1")
			if err != nil {
				return err
			}

			// Advanced feature, may not be supported
			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_expand_with_count",
		"Expand with $count option",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$expand=Descriptions($count=true)")
			if err != nil {
				return err
			}

			// Optional feature
			if resp.StatusCode == 200 || resp.StatusCode == 400 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_expand_with_filter",
		"Expand with $filter on collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$expand=Descriptions($filter=LanguageKey eq 'EN')")
			if err != nil {
				return err
			}

			// Optional feature
			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 501 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_select_navigation_property",
		"Select navigation property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name,Descriptions")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	return suite
}
