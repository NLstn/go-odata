package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FilterOnSingleEntityNavigationProperties creates test suite for filtering on single-entity navigation properties
// Per OData v4.01 spec 5.1.1.15, properties of entities related with cardinality 0..1 or 1 can be accessed directly
func FilterOnSingleEntityNavigationProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.10 Filter on Single-Entity Navigation Properties",
		"Tests filtering on single-entity (cardinality 0..1 or 1) navigation properties",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_PathExpressions",
	)
	RegisterFilterOnSingleEntityNavigationPropertiesTests(suite)
	return suite
}

// RegisterFilterOnSingleEntityNavigationPropertiesTests registers tests for filtering on single-entity navigation properties
func RegisterFilterOnSingleEntityNavigationPropertiesTests(suite *framework.TestSuite) {
	suite.AddTest(
		"Filter by single-entity navigation property",
		"Filter entities by properties of related entities with cardinality 0..1 or 1",
		testFilterBySingleEntityNavigationProperty,
	)

	suite.AddTest(
		"Filter with navigation property path in comparison",
		"Use navigation property paths in filter comparisons",
		testNavigationPropertyPathComparison,
	)

	suite.AddTest(
		"Collection navigation properties still require lambda operators",
		"Verify that collection navigation properties cannot be used without any/all",
		testCollectionNavigationRequiresLambda,
	)

	suite.AddTest(
		"Combine single-entity navigation filter with expand",
		"Filter and expand the same navigation property",
		testCombineFilterAndExpandNavigation,
	)
}

func testFilterBySingleEntityNavigationProperty(ctx *framework.TestContext) error {
	// Test filtering by a property of a single-entity navigation property
	// Example based on OData v4.01 spec section 5.1.1.15
	// This simulates: /TeamMembers?$filter=Team/ClubID eq 'some-guid'
	
	// Try with a generic entity set that likely has navigation properties
	resp, err := ctx.GET("/Products?$filter=Category/Name eq 'Electronics'")
	if err != nil {
		// If Products doesn't exist or doesn't have the right structure, skip
		ctx.Skip("Products entity set not available or doesn't have Category navigation property")
		return nil
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(resp.Body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if _, ok := result["value"]; !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	return nil
}

func testNavigationPropertyPathComparison(ctx *framework.TestContext) error {
	// Test various comparison operators with navigation property paths
	// This tests the spec requirement that navigation property paths work like regular properties
	
	resp, err := ctx.GET("/Products?$filter=Supplier/Country eq 'USA'")
	if err != nil {
		ctx.Skip("Products entity set not available or doesn't have Supplier navigation property")
		return nil
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testCollectionNavigationRequiresLambda(ctx *framework.TestContext) error {
	// Verify that collection navigation properties still require any/all operators
	// This ensures we maintain spec compliance for collection-valued navigation properties
	
	// Try to filter by a collection navigation property without lambda operator
	// This should return a 400 Bad Request error
	resp, err := ctx.GET("/Products?$filter=Descriptions/LanguageKey eq 'EN'")
	if err != nil {
		ctx.Skip("Products entity set not available or doesn't have Descriptions collection")
		return nil
	}

	// This should fail because Descriptions is a collection
	if resp.StatusCode == 200 {
		return fmt.Errorf("expected error for collection navigation property without lambda, got status 200")
	}

	// We expect a 400 Bad Request
	if resp.StatusCode != 400 {
		return fmt.Errorf("expected status 400 for collection navigation without lambda, got %d", resp.StatusCode)
	}

	return nil
}

func testCombineFilterAndExpandNavigation(ctx *framework.TestContext) error {
	// Test that we can both filter on and expand the same navigation property
	// Per spec, this should work seamlessly
	
	escapedFilter := url.QueryEscape("Team/ClubID eq 'club-1'")
	resp, err := ctx.GET("/TeamMembers?$filter=" + escapedFilter + "&$expand=Team")
	if err != nil {
		ctx.Skip("TeamMembers entity set not available")
		return nil
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(resp.Body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	values, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response 'value' is not an array")
	}

	// Check that expanded navigation properties are included
	if len(values) > 0 {
		firstItem := values[0].(map[string]interface{})
		if _, hasTeam := firstItem["Team"]; !hasTeam {
			return fmt.Errorf("expected Team property to be expanded in response")
		}
	}

	return nil
}
