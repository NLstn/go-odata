package v4_0

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ODataAnnotations creates the 10.2 OData Annotations test suite
func ODataAnnotations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"10.2 OData Annotations",
		"Tests that required OData control information annotations are present in JSON responses.",
		"https://docs.oasis-open.org/odata/odata-json-format/v4.0/os/odata-json-format-v4.0-os.html#_Toc372793052",
	)

	suite.AddTest(
		"test_odata_context_required",
		"@odata.context required in responses",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1")
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

			context, ok := result["@odata.context"]
			if !ok {
				return fmt.Errorf("@odata.context annotation is required but missing")
			}

			contextStr, ok := context.(string)
			if !ok {
				return fmt.Errorf("@odata.context must be a string, got %T", context)
			}

			// Validate format: should contain $metadata
			if !strings.Contains(contextStr, "$metadata") {
				return fmt.Errorf("@odata.context must reference $metadata, got: %s", contextStr)
			}

			ctx.Log(fmt.Sprintf("@odata.context: %s", contextStr))
			return nil
		},
	)

	suite.AddTest(
		"test_odata_context_format_collection",
		"@odata.context format for collections",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
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

			context, ok := result["@odata.context"].(string)
			if !ok {
				return fmt.Errorf("@odata.context missing or not a string")
			}

			// Should end with #EntitySet format for collections
			if !strings.Contains(context, "#Products") && !strings.Contains(context, "#Collection") {
				return fmt.Errorf("@odata.context should reference entity set, got: %s", context)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_odata_context_format_entity",
		"@odata.context format for single entity",
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

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			context, ok := result["@odata.context"].(string)
			if !ok {
				return fmt.Errorf("@odata.context missing or not a string")
			}

			// Should contain $metadata and entity reference
			if !strings.Contains(context, "$metadata") {
				return fmt.Errorf("@odata.context must reference $metadata")
			}

			if !strings.Contains(context, "#Products") && !strings.Contains(context, "$entity") {
				return fmt.Errorf("@odata.context should reference entity or $entity, got: %s", context)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_odata_id_in_entity_response",
		"@odata.id present in entity responses",
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

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			// @odata.id is required for full metadata, recommended for minimal
			odataID, ok := result["@odata.id"]
			if !ok {
				// Check if we're in metadata=none mode
				return ctx.Skip("@odata.id not present - may be metadata=none")
			}

			idStr, ok := odataID.(string)
			if !ok {
				return fmt.Errorf("@odata.id must be a string, got %T", odataID)
			}

			// Should be a valid URL reference to the entity
			if idStr == "" {
				return fmt.Errorf("@odata.id must not be empty")
			}

			// Should reference the entity set and key
			if !strings.Contains(idStr, "Products") {
				return fmt.Errorf("@odata.id should reference entity set, got: %s", idStr)
			}

			ctx.Log(fmt.Sprintf("@odata.id: %s", idStr))
			return nil
		},
	)

	suite.AddTest(
		"test_odata_id_in_collection",
		"@odata.id present for entities in collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=3")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			if len(result.Value) == 0 {
				return fmt.Errorf("expected at least one entity in collection")
			}

			// Check first entity for @odata.id
			entity := result.Value[0]
			if _, ok := entity["@odata.id"]; !ok {
				return ctx.Skip("@odata.id not present - may be metadata=none")
			}

			// All entities should have @odata.id
			for i, ent := range result.Value {
				odataID, ok := ent["@odata.id"]
				if !ok {
					return fmt.Errorf("entity %d missing @odata.id", i)
				}

				idStr, ok := odataID.(string)
				if !ok {
					return fmt.Errorf("entity %d: @odata.id must be a string", i)
				}

				if idStr == "" {
					return fmt.Errorf("entity %d: @odata.id must not be empty", i)
				}
			}

			ctx.Log(fmt.Sprintf("All %d entities have @odata.id", len(result.Value)))
			return nil
		},
	)

	suite.AddTest(
		"test_odata_count_annotation",
		"@odata.count present when $count=true",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$count=true&$top=5")
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

			count, ok := result["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count required when $count=true is specified")
			}

			// Should be a number
			var countNum float64
			switch v := count.(type) {
			case float64:
				countNum = v
			case int:
				countNum = float64(v)
			default:
				return fmt.Errorf("@odata.count must be a number, got %T", count)
			}

			if countNum < 0 {
				return fmt.Errorf("@odata.count must be non-negative, got %f", countNum)
			}

			if math.Trunc(countNum) != countNum {
				return fmt.Errorf("@odata.count must be an integer value, got %f", countNum)
			}

			ctx.Log(fmt.Sprintf("@odata.count: %f", countNum))
			return nil
		},
	)

	suite.AddTest(
		"test_odata_nextlink_with_pagination",
		"@odata.nextLink present when results are paginated",
		func(ctx *framework.TestContext) error {
			// Request with $count=true to confirm pagination is required
			resp, err := ctx.GET("/Products?$count=true&$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result struct {
				Value    []map[string]interface{} `json:"value"`
				Count    float64                  `json:"@odata.count"`
				NextLink string                   `json:"@odata.nextLink"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			if result.Count <= 1 {
				return ctx.Skip("Not enough entities to require pagination for $top=1")
			}

			if result.NextLink == "" {
				return fmt.Errorf("@odata.nextLink required when @odata.count exceeds $top=1")
			}

			parsed, err := url.Parse(result.NextLink)
			if err != nil {
				return fmt.Errorf("invalid @odata.nextLink URL: %w", err)
			}

			query := parsed.Query()
			if !query.Has("$skip") && !query.Has("$skiptoken") {
				return fmt.Errorf("@odata.nextLink must include $skip or $skiptoken parameter, got: %s", result.NextLink)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_odata_type_annotation",
		"@odata.type annotation for derived types",
		func(ctx *framework.TestContext) error {
			// This test checks if @odata.type is present when type info is needed
			// Skip if no derived types in the model
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			if len(result.Value) == 0 {
				return fmt.Errorf("no entities to test")
			}

			// @odata.type is optional unless there are derived types
			// Just validate format if present
			entity := result.Value[0]
			if odataType, ok := entity["@odata.type"]; ok {
				typeStr, ok := odataType.(string)
				if !ok {
					return fmt.Errorf("@odata.type must be a string, got %T", odataType)
				}

				// Should start with # for type references
				if !strings.HasPrefix(typeStr, "#") {
					return fmt.Errorf("@odata.type should start with #, got: %s", typeStr)
				}

				ctx.Log(fmt.Sprintf("@odata.type: %s", typeStr))
			} else {
				ctx.Log("@odata.type not present (optional for base types)")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_odata_editlink_format",
		"@odata.editLink format validation if present",
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

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			// @odata.editLink is optional
			if editLink, ok := result["@odata.editLink"]; ok {
				editLinkStr, ok := editLink.(string)
				if !ok {
					return fmt.Errorf("@odata.editLink must be a string, got %T", editLink)
				}

				if editLinkStr == "" {
					return fmt.Errorf("@odata.editLink must not be empty if present")
				}

				ctx.Log(fmt.Sprintf("@odata.editLink: %s", editLinkStr))
			} else {
				ctx.Log("@odata.editLink not present (optional)")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_created_entity_has_required_annotations",
		"Created entity response contains required annotations",
		func(ctx *framework.TestContext) error {
			// Create a new product
			payload, err := buildProductPayload(ctx, "Annotation Test Product", 99.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			// @odata.context required
			if _, ok := result["@odata.context"]; !ok {
				return fmt.Errorf("@odata.context required in create response")
			}

			// Check for Location header (alternative to @odata.id in body)
			location := resp.Headers.Get("Location")
			odataID, hasOdataID := result["@odata.id"]

			if location == "" && !hasOdataID {
				return fmt.Errorf("either Location header or @odata.id must be present in create response")
			}

			if hasOdataID {
				if idStr, ok := odataID.(string); !ok || idStr == "" {
					return fmt.Errorf("@odata.id must be a non-empty string")
				}
			}

			ctx.Log("Created entity has required annotations")
			return nil
		},
	)

	suite.AddTest(
		"test_navigation_property_annotation",
		"Navigation properties have proper annotations",
		func(ctx *framework.TestContext) error {
			// Get a product with expanded Category
			resp, err := ctx.GET("/Products?$top=1&$expand=Category")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON response: %w", err)
			}

			if len(result.Value) == 0 {
				return fmt.Errorf("no entities to test")
			}

			entity := result.Value[0]

			// Check for Category navigation property
			category, hasCategory := entity["Category"]
			if !hasCategory {
				return ctx.Skip("Category navigation not present - may not be related")
			}

			// If expanded, should be an object with its own annotations
			if categoryObj, ok := category.(map[string]interface{}); ok {
				// Expanded navigation should have @odata.context or inherit from parent
				ctx.Log("Navigation property expanded successfully")

				// Check if expanded entity has ID
				if _, hasID := categoryObj["ID"]; !hasID {
					return fmt.Errorf("expanded navigation entity should have ID field")
				}
			}

			return nil
		},
	)

	return suite
}
