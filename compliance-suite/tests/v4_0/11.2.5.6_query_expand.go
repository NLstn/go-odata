package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QueryExpand creates the 11.2.5.6 System Query Option $expand test suite
func QueryExpand() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.6 System Query Option $expand",
		"Tests $expand query option for expanding related entities according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_SystemQueryOptionexpand",
	)

	// Test 1: Basic $expand returns related entities inline
	suite.AddTest(
		"test_expand_basic",
		"$expand includes related entities inline",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions")
			resp, err := ctx.GET("/Products?$expand=" + expand + "&$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("response contains no items: %w", err)
			}

			item := items[0]

			// Verify Descriptions field is present
			descriptions, ok := item["Descriptions"]
			if !ok {
				return fmt.Errorf("descriptions field is missing")
			}

			// Verify Descriptions is an array (expanded data)
			if _, ok := descriptions.([]interface{}); !ok {
				return fmt.Errorf("descriptions field is not an array (not properly expanded)")
			}

			return nil
		},
	)

	// Test 2: $expand with $select on expanded entity
	suite.AddTest(
		"test_expand_with_select",
		"$expand with nested $select",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($select=Description)")
			filter := url.QueryEscape("Name eq 'Laptop'")
			resp, err := ctx.GET("/Products?$filter=" + filter + "&$expand=" + expand + "&$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("response contains no items: %w", err)
			}

			item := items[0]

			// Verify Descriptions field is present and expanded
			descriptions, ok := item["Descriptions"]
			if !ok {
				return fmt.Errorf("descriptions field is missing")
			}

			// Verify it's an array
			descArray, ok := descriptions.([]interface{})
			if !ok {
				return fmt.Errorf("descriptions field is not an array")
			}

			if len(descArray) == 0 {
				return fmt.Errorf("expected at least one expanded description for Laptop fixture")
			}

			desc, ok := descArray[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("first description is not an object")
			}
			if _, ok := desc["Description"]; !ok {
				return fmt.Errorf("expanded Descriptions missing Description field")
			}
			// Key properties may be present even when not listed in nested $select.
			// The assertion here focuses on ensuring the selected non-key field is included.

			return nil
		},
	)

	// Test 3: $expand on single entity
	suite.AddTest(
		"test_expand_single_entity",
		"$expand on single entity request",
		func(ctx *framework.TestContext) error {
			// First get a product ID
			allResp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(allResp, 200); err != nil {
				return fmt.Errorf("failed to get products: %w", err)
			}

			items, err := ctx.ParseEntityCollection(allResp)
			if err != nil {
				return fmt.Errorf("failed to parse products JSON: %w", err)
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("no products available")
			}

			firstItem := items[0]

			productID := firstItem["ID"]

			// Now test $expand on single entity
			expand := url.QueryEscape("Descriptions")
			resp, err := ctx.GET(fmt.Sprintf("/Products(%v)?$expand=%s", productID, expand))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Verify Descriptions field is present and expanded
			descriptions, ok := result["Descriptions"]
			if !ok {
				return fmt.Errorf("descriptions field is missing")
			}

			// Verify it's expanded (should be an array)
			if _, ok := descriptions.([]interface{}); !ok {
				return fmt.Errorf("descriptions not expanded as array")
			}

			return nil
		},
	)

	// Test 4: $expand with parent $select should still include single-valued navigation entity
	suite.AddTest(
		"test_expand_with_parent_select_belongs_to",
		"$expand single-valued navigation works when parent $select omits foreign key",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("CategoryID ne null")
			expand := url.QueryEscape("Category($select=Name)")
			resp, err := ctx.GET("/Products?$filter=" + filter + "&$top=1&$select=Name&$expand=" + expand)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return fmt.Errorf("response missing 'value' array or contains no items: %w", err)
			}

			item := items[0]

			categoryValue, exists := item["Category"]
			if !exists {
				return fmt.Errorf("category field missing from expanded response")
			}
			if categoryValue == nil {
				return fmt.Errorf("category field is null; expected expanded entity")
			}

			category, ok := categoryValue.(map[string]interface{})
			if !ok {
				return fmt.Errorf("category field is not an object")
			}

			if err := ctx.AssertEntityHasFields(category, "Name"); err != nil {
				return fmt.Errorf("expanded category missing selected Name property: %w", err)
			}

			return nil
		},
	)

	return suite
}
