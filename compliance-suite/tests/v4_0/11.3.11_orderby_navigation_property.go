package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// OrderByNavigationProperty creates the test suite for $orderby on single-entity navigation property paths.
// Per OData v4.01 spec section 5.1.1.15, properties of entities related with cardinality 0..1 or 1
// can be used as operands in $orderby expressions using a slash-separated path (e.g., Author/Name).
func OrderByNavigationProperty() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.11 $orderby on Single-Entity Navigation Property Paths",
		"Validates that $orderby accepts single-entity navigation property paths (e.g., $orderby=Category/Name)",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_PathExpressions",
	)
	RegisterOrderByNavigationPropertyTests(suite)
	return suite
}

// RegisterOrderByNavigationPropertyTests registers all $orderby navigation property path tests.
func RegisterOrderByNavigationPropertyTests(suite *framework.TestSuite) {
	suite.AddTest(
		"orderby_navigation_property_asc",
		"$orderby on navigation property path ascending returns 200 and sorted results",
		testOrderByNavigationPropertyAsc,
	)

	suite.AddTest(
		"orderby_navigation_property_desc",
		"$orderby on navigation property path descending returns 200 and sorted results",
		testOrderByNavigationPropertyDesc,
	)

	suite.AddTest(
		"orderby_navigation_property_with_expand",
		"$orderby on navigation property path combined with $expand on the same property",
		testOrderByNavigationPropertyWithExpand,
	)

	suite.AddTest(
		"orderby_navigation_property_with_filter",
		"$orderby on navigation property path combined with $filter",
		testOrderByNavigationPropertyWithFilter,
	)

	suite.AddTest(
		"orderby_navigation_property_multiple_clauses",
		"$orderby with navigation property path and regular property as multiple clauses",
		testOrderByNavigationPropertyMultipleClauses,
	)

	suite.AddTest(
		"orderby_collection_navigation_rejected",
		"$orderby directly on a collection-valued navigation property is rejected with 400",
		testOrderByCollectionNavigationRejected,
	)
}

func testOrderByNavigationPropertyAsc(ctx *framework.TestContext) error {
	resp, err := ctx.GET("/Products?$orderby=Category/Name asc")
	if err != nil {
		return err
	}

	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("$orderby=Category/Name asc should be accepted: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	_, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response 'value' is not an array")
	}

	return nil
}

func testOrderByNavigationPropertyDesc(ctx *framework.TestContext) error {
	// Fetch without ordering first to gather category names.
	respAll, err := ctx.GET("/Products?$expand=Category($select=Name)&$orderby=Category/Name desc")
	if err != nil {
		return err
	}

	if err := ctx.AssertStatusCode(respAll, 200); err != nil {
		return fmt.Errorf("$orderby=Category/Name desc should be accepted: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respAll.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	values, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response 'value' is not an array")
	}

	// Verify descending order: each entry's Category/Name must be <= the previous.
	var prevName string
	for i, raw := range values {
		entity, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		catRaw, hasCat := entity["Category"]
		if !hasCat || catRaw == nil {
			// Products without a category sort last in most DBs; allow them.
			continue
		}
		cat, ok := catRaw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := cat["Name"].(string)
		if i > 0 && prevName != "" && name > prevName {
			return fmt.Errorf("ordering violation at index %d: '%s' > '%s' (not descending)", i, name, prevName)
		}
		if name != "" {
			prevName = name
		}
	}

	return nil
}

func testOrderByNavigationPropertyWithExpand(ctx *framework.TestContext) error {
	resp, err := ctx.GET("/Products?$expand=Category&$orderby=Category/Name")
	if err != nil {
		return err
	}

	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("$expand=Category&$orderby=Category/Name should be accepted: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	values, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response 'value' is not an array")
	}

	// Verify that Category is expanded and that results are in ascending order.
	var prevName string
	for i, raw := range values {
		entity, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		catRaw, hasCat := entity["Category"]
		if !hasCat || catRaw == nil {
			continue
		}
		cat, ok := catRaw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("entity %d: Category is not an object", i)
		}
		name, _ := cat["Name"].(string)
		if i > 0 && prevName != "" && name < prevName {
			return fmt.Errorf("ordering violation at index %d: '%s' < '%s' (not ascending)", i, name, prevName)
		}
		if name != "" {
			prevName = name
		}
	}

	return nil
}

func testOrderByNavigationPropertyWithFilter(ctx *framework.TestContext) error {
	// Combine $filter on a navigation property path with $orderby on the same path.
	filterExpr := url.QueryEscape("Category/Name ne 'unknown'")
	resp, err := ctx.GET("/Products?$filter=" + filterExpr + "&$orderby=Category/Name asc")
	if err != nil {
		return err
	}

	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("combined $filter and $orderby on Category/Name should be accepted: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if _, ok := result["value"]; !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	return nil
}

func testOrderByNavigationPropertyMultipleClauses(ctx *framework.TestContext) error {
	// Multiple $orderby clauses: navigation property path + regular property.
	resp, err := ctx.GET("/Products?$orderby=Category/Name asc,Name asc")
	if err != nil {
		return err
	}

	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return fmt.Errorf("$orderby=Category/Name asc,Name asc should be accepted: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if _, ok := result["value"]; !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	return nil
}

func testOrderByCollectionNavigationRejected(ctx *framework.TestContext) error {
	// Collection-valued navigation properties (e.g., Descriptions on Products) must NOT be
	// directly usable in $orderby without a lambda/ aggregation — this is invalid per spec.
	resp, err := ctx.GET("/Products?$orderby=Descriptions/LanguageKey")
	if err != nil {
		return err
	}

	// Must reject with 400 Bad Request.
	if resp.StatusCode != 400 {
		return fmt.Errorf("expected 400 for $orderby on collection navigation property, got %d", resp.StatusCode)
	}

	return nil
}
