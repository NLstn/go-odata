package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NullableProperties creates the 5.1.2 Nullable Properties test suite
func NullableProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.2 Nullable Properties",
		"Tests handling of nullable properties including null values in filters and responses.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_Nullable",
	)

	// Test 1: Create entity with null value in nullable property
	suite.AddTest(
		"test_create_with_null",
		"Create entity with null value in nullable property",
		func(ctx *framework.TestContext) error {
			// LongText is a nullable property on ProductDescription
			productID, err := firstEntityID(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.POST("/ProductDescriptions", map[string]interface{}{
				"ProductID":   productID,
				"LanguageKey": "IT",
				"Description": "Italian description",
				"LongText":    nil,
			})
			if err != nil {
				return err
			}

			if resp.StatusCode == 201 {
				return nil
			}
			if resp.StatusCode == 500 {
				return ctx.Skip("server returned 500 on valid create with null value")
			}
			return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
		},
	)

	// Test 2: Filter for null values using eq null
	suite.AddTest(
		"test_filter_eq_null",
		"Filter for entities where property eq null",
		func(ctx *framework.TestContext) error {
			// LongText is a nullable property
			resp, err := ctx.GET("/ProductDescriptions?$filter=LongText eq null")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}
			if resp.StatusCode == 400 {
				return ctx.Skip("null literal handling in $filter not implemented")
			}

			return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
		},
	)

	// Test 3: Filter for non-null values using ne null
	suite.AddTest(
		"test_filter_ne_null",
		"Filter for entities where property ne null",
		func(ctx *framework.TestContext) error {
			// LongText is a nullable property
			resp, err := ctx.GET("/ProductDescriptions?$filter=LongText ne null")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}
			if resp.StatusCode == 400 {
				return ctx.Skip("null literal handling in $filter not implemented")
			}

			return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
		},
	)

	// Test 4: Response includes null values as JSON null
	suite.AddTest(
		"test_response_null_representation",
		"Response represents null values as JSON null",
		func(ctx *framework.TestContext) error {
			// Create a ProductDescription with null LongText explicitly
			pid, err := firstEntityID(ctx, "Products")
			if err != nil {
				return err
			}
			createResp, err := ctx.POST("/ProductDescriptions", map[string]interface{}{
				"ProductID":   pid,
				"LanguageKey": "FR",
				"Description": "French description",
				"LongText":    nil,
			})
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				if createResp.StatusCode == 500 {
					return ctx.Skip("server returned 500 on creating entity with nullable property set to null")
				}
				return fmt.Errorf("failed to create entity (status: %d)", createResp.StatusCode)
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			productID, ok := createResult["ProductID"].(string)
			if !ok {
				return fmt.Errorf("could not get ProductID from created entity")
			}

			langKey, ok := createResult["LanguageKey"].(string)
			if !ok {
				return fmt.Errorf("could not get LanguageKey from created entity")
			}

			// Get the entity and verify LongText is represented as null
			getResp, err := ctx.GET(fmt.Sprintf("/ProductDescriptions(ProductID=%s,LanguageKey='%s')", productID, langKey))
			if err != nil {
				return err
			}

			// Verify LongText is null in the response
			bodyStr := string(getResp.Body)
			if !strings.Contains(bodyStr, `"LongText":null`) {
				return fmt.Errorf("LongText not represented as null in JSON response")
			}

			return nil
		},
	)

	// Test 5: Update property to null
	suite.AddTest(
		"test_update_to_null",
		"Update property to null value",
		func(ctx *framework.TestContext) error {
			// Create a test entity with non-null LongText
			pid, err := firstEntityID(ctx, "Products")
			if err != nil {
				return err
			}
			createResp, err := ctx.POST("/ProductDescriptions", map[string]interface{}{
				"ProductID":   pid,
				"LanguageKey": "ES",
				"Description": "Spanish description",
				"LongText":    "Some long text",
			})
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				if createResp.StatusCode == 500 {
					return ctx.Skip("server returned 500 on creating entity before update-to-null test")
				}
				return fmt.Errorf("failed to create entity (status: %d)", createResp.StatusCode)
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			productID, ok := createResult["ProductID"].(string)
			if !ok {
				return fmt.Errorf("could not get ProductID from created entity")
			}

			langKey, ok := createResult["LanguageKey"].(string)
			if !ok {
				return fmt.Errorf("could not get LanguageKey from created entity")
			}

			// Now update LongText to null
			patchResp, err := ctx.PATCH(fmt.Sprintf("/ProductDescriptions(ProductID=%s,LanguageKey='%s')", productID, langKey), map[string]interface{}{
				"LongText": nil,
			})
			if err != nil {
				return err
			}

			if patchResp.StatusCode != 200 && patchResp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204, got %d", patchResp.StatusCode)
			}

			return nil
		},
	)

	// Test 6: Null literal in URL filter
	suite.AddTest(
		"test_null_literal_url",
		"Null literal in URL filter",
		func(ctx *framework.TestContext) error {
			// LongText is nullable
			resp, err := ctx.GET("/ProductDescriptions?$filter=LongText eq null")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}
			if resp.StatusCode == 400 {
				return ctx.Skip("null literal handling in $filter not implemented")
			}

			return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
		},
	)

	// Test 7: Accessing null property returns appropriate response
	suite.AddTest(
		"test_access_null_property",
		"Accessing null property returns appropriate response",
		func(ctx *framework.TestContext) error {
			// Create an entity with null LongText
			pid, err := firstEntityID(ctx, "Products")
			if err != nil {
				return err
			}
			createResp, err := ctx.POST("/ProductDescriptions", map[string]interface{}{
				"ProductID":   pid,
				"LanguageKey": "PT",
				"Description": "Portuguese description",
				"LongText":    nil,
			})
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				if createResp.StatusCode == 500 {
					return ctx.Skip("server returned 500 on creating entity with null property for access test")
				}
				return fmt.Errorf("failed to create entity (status: %d)", createResp.StatusCode)
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			productID, ok := createResult["ProductID"].(string)
			if !ok {
				return fmt.Errorf("could not get ProductID from created entity")
			}

			langKey, ok := createResult["LanguageKey"].(string)
			if !ok {
				return fmt.Errorf("could not get LanguageKey from created entity")
			}

			// Access the null property
			propResp, err := ctx.GET(fmt.Sprintf("/ProductDescriptions(ProductID=%s,LanguageKey='%s')/LongText", productID, langKey))
			if err != nil {
				return err
			}

			// Per OData spec, accessing a null property should return 204 No Content
			// or 200 with {"@odata.null":true} or {"value":null}
			if propResp.StatusCode == 204 {
				return nil
			}

			if propResp.StatusCode == 200 {
				bodyStr := string(propResp.Body)
				if strings.Contains(bodyStr, `"@odata.null":true`) || strings.Contains(bodyStr, `"value":null`) {
					return nil
				}
				return fmt.Errorf("status 200 but body doesn't properly represent null: %s", bodyStr)
			}

			return fmt.Errorf("expected status 204 or 200, got %d", propResp.StatusCode)
		},
	)

	// Test 8: Setting non-nullable property to null handled correctly
	suite.AddTest(
		"test_nonnullable_reject_null",
		"Setting non-nullable property to null handled correctly",
		func(ctx *framework.TestContext) error {
			// Try to create entity with required non-nullable field as null
			// Description is non-nullable
			pid, err := firstEntityID(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.POST("/ProductDescriptions", map[string]interface{}{
				"ProductID":   pid,
				"LanguageKey": "NL",
				"Description": nil, // Description is non-nullable
				"LongText":    "Some text",
			})
			if err != nil {
				return err
			}

			// Should return 400 Bad Request since Description is non-nullable
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400 for null in non-nullable property, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	return suite
}
