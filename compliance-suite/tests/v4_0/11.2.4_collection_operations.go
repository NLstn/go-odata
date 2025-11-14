package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// CollectionOperations creates the 11.2.4 Collection Operations test suite
func CollectionOperations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.4 Collection Operations",
		"Validates addressing entity collections and understanding the difference between collections and single entities",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingEntities",
	)

	// Test 1: Collection returns array with value property
	suite.AddTest(
		"test_collection_format",
		"Collection returns array wrapped in 'value' property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("collection response missing 'value' property (OData v4 requires collections wrapped in value)")
			}

			return nil
		},
	)

	// Test 2: Single entity does not have value wrapper
	suite.AddTest(
		"test_single_entity_format",
		"Single entity returns object without 'value' wrapper",
		func(ctx *framework.TestContext) error {
			// Get a product ID first
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", resp.StatusCode)
			}

			var result struct {
				Value []struct {
					ID string `json:"ID"`
				} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(result.Value) == 0 {
				return ctx.Skip("no products available for testing")
			}

			productID := result.Value[0].ID
			resp, err = ctx.GET(fmt.Sprintf("/Products(%s)", url.PathEscape(productID)))
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var single map[string]interface{}
			if err := json.Unmarshal(resp.Body, &single); err != nil {
				return fmt.Errorf("failed to parse single entity: %w", err)
			}

			// Single entity should have ID property directly
			if _, ok := single["ID"]; !ok {
				return fmt.Errorf("single entity missing ID property")
			}

			// Single entity should NOT be wrapped in value array
			if _, ok := single["value"]; ok {
				return fmt.Errorf("single entity incorrectly wrapped in 'value' property")
			}

			return nil
		},
	)

	// Test 3: Collection has @odata.context
	suite.AddTest(
		"test_collection_context",
		"Collection includes @odata.context",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["@odata.context"]; !ok {
				return fmt.Errorf("collection response missing @odata.context")
			}

			return nil
		},
	)

	// Test 4: Collection returns 200 OK
	suite.AddTest(
		"test_collection_status",
		"Collection request returns 200 OK",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 5: Empty collection returns valid structure
	suite.AddTest(
		"test_empty_collection",
		"Empty collection returns valid structure with empty array",
		func(ctx *framework.TestContext) error {
			// Use a filter that should return no results
			filter := url.QueryEscape("ID eq -999999")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Should still have value array (just empty) and context
			if _, ok := result["value"]; !ok {
				return fmt.Errorf("empty collection missing 'value' property")
			}
			if _, ok := result["@odata.context"]; !ok {
				return fmt.Errorf("empty collection missing @odata.context")
			}

			return nil
		},
	)

	// Test 6: Collection supports query options
	suite.AddTest(
		"test_collection_query_options",
		"Collection supports query options like $top",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=5")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 7: Single entity does not support $top
	suite.AddTest(
		"test_single_entity_rejects_top",
		"Single entity rejects $top query option with 400",
		func(ctx *framework.TestContext) error {
			// Get a product ID first
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", resp.StatusCode)
			}

			var result struct {
				Value []struct {
					ID string `json:"ID"`
				} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(result.Value) == 0 {
				return ctx.Skip("no products available for testing")
			}

			productID := result.Value[0].ID
			resp, err = ctx.GET(fmt.Sprintf("/Products(%s)?$top=5", url.PathEscape(productID)))
			if err != nil {
				return err
			}

			// Should return 400 as $top is not applicable to single entities
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400 ($top not valid for single entity), got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 8: Single entity does not support $skip
	suite.AddTest(
		"test_single_entity_rejects_skip",
		"Single entity rejects $skip query option with 400",
		func(ctx *framework.TestContext) error {
			// Get a product ID first
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("failed to get products: status %d", resp.StatusCode)
			}

			var result struct {
				Value []struct {
					ID string `json:"ID"`
				} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if len(result.Value) == 0 {
				return ctx.Skip("no products available for testing")
			}

			productID := result.Value[0].ID
			resp, err = ctx.GET(fmt.Sprintf("/Products(%s)?$skip=5", url.PathEscape(productID)))
			if err != nil {
				return err
			}

			// Should return 400 as $skip is not applicable to single entities
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400 ($skip not valid for single entity), got %d", resp.StatusCode)
			}
			return nil
		},
	)

	return suite
}
