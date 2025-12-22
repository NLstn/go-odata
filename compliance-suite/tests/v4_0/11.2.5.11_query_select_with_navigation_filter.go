package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QuerySelectWithNavigationFilter creates test suite for combining $select with navigation property filters
// This tests that SELECT clauses properly qualify column names when JOINs are present
func QuerySelectWithNavigationFilter() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.11 System Query Option $select with Navigation Property Filters",
		"Tests $select combined with $filter on navigation properties to ensure proper SQL generation with qualified column names",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptionselect",
	)
	RegisterQuerySelectWithNavigationFilterTests(suite)
	return suite
}

// RegisterQuerySelectWithNavigationFilterTests registers tests for $select with navigation filters
func RegisterQuerySelectWithNavigationFilterTests(suite *framework.TestSuite) {
	suite.AddTest(
		"Select single property with navigation filter",
		"Combine $select with $filter on navigation property to ensure ambiguous column errors don't occur",
		testSelectWithNavigationFilter,
	)

	suite.AddTest(
		"Select multiple properties with navigation filter",
		"Test $select with multiple properties and navigation filter",
		testSelectMultipleWithNavigationFilter,
	)

	suite.AddTest(
		"Select with navigation filter on different properties",
		"Test various navigation property comparisons",
		testSelectWithVariousNavigationFilters,
	)

	suite.AddTest(
		"Select with navigation filter and expand",
		"Combine $select, $filter, and $expand on navigation properties",
		testSelectWithNavigationFilterAndExpand,
	)
}

func testSelectWithNavigationFilter(ctx *framework.TestContext) error {
	// Test: /Products?$select=Name&$filter=Category/Name eq 'Electronics'
	// This ensures that when a JOIN is added for the filter, the SELECT clause
	// uses qualified column names to avoid ambiguous references
	
	filter := url.QueryEscape("Category/Name eq 'Electronics'")
	select_ := url.QueryEscape("Name")
	
	resp, err := ctx.GET("/Products?$select=" + select_ + "&$filter=" + filter)
	if err != nil {
		return err
	}

	// The key test is that this doesn't return an error about ambiguous columns
	// which would happen if column names aren't properly qualified
	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(resp.Body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Verify response structure
	value, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	// If there are results, verify the selected properties are present
	if len(value) > 0 {
		item, ok := value[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("first item is not an object")
		}

		// Verify Name field is present (selected)
		if _, ok := item["Name"]; !ok {
			return fmt.Errorf("selected field 'Name' is missing")
		}

		// Verify ID is present (key properties are always included)
		if _, ok := item["ID"]; !ok {
			return fmt.Errorf("key property 'ID' is missing")
		}

		// Verify that non-selected fields are not present
		if _, ok := item["Description"]; ok {
			return fmt.Errorf("non-selected field 'Description' should not be present")
		}
	}

	return nil
}

func testSelectMultipleWithNavigationFilter(ctx *framework.TestContext) error {
	// Test: /Products?$select=Name,Price&$filter=Category/Name eq 'Electronics'
	
	filter := url.QueryEscape("Category/Name eq 'Electronics'")
	select_ := url.QueryEscape("Name,Price")
	
	resp, err := ctx.GET("/Products?$select=" + select_ + "&$filter=" + filter)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(resp.Body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	value, ok := result["value"].([]interface{})
	if !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	if len(value) > 0 {
		item, ok := value[0].(map[string]interface{})
		if !ok {
			return fmt.Errorf("first item is not an object")
		}

		// Verify selected fields are present
		if _, ok := item["Name"]; !ok {
			return fmt.Errorf("selected field 'Name' is missing")
		}
		if _, ok := item["Price"]; !ok {
			return fmt.Errorf("selected field 'Price' is missing")
		}

		// Verify key property is present
		if _, ok := item["ID"]; !ok {
			return fmt.Errorf("key property 'ID' is missing")
		}
	}

	return nil
}

func testSelectWithVariousNavigationFilters(ctx *framework.TestContext) error {
	// Test different filter operators with navigation properties
	tests := []struct {
		filter string
		desc   string
	}{
		{"Category/Name ne 'Books'", "not equals"},
		{"Category/Name eq 'Electronics' or Category/Name eq 'Computers'", "logical or"},
		{"Category/Name eq 'Electronics' and Price gt 100", "combined with regular property"},
	}

	select_ := url.QueryEscape("Name")

	for _, tt := range tests {
		filter := url.QueryEscape(tt.filter)
		resp, err := ctx.GET("/Products?$select=" + select_ + "&$filter=" + filter)
		if err != nil {
			return fmt.Errorf("test '%s' failed: %v", tt.desc, err)
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("test '%s': expected status 200, got %d. Body: %s", tt.desc, resp.StatusCode, string(resp.Body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return fmt.Errorf("test '%s': failed to parse JSON: %w", tt.desc, err)
		}

		if _, ok := result["value"]; !ok {
			return fmt.Errorf("test '%s': response missing 'value' array", tt.desc)
		}
	}

	return nil
}

func testSelectWithNavigationFilterAndExpand(ctx *framework.TestContext) error {
	// Test: /Products?$select=Name,Category&$filter=Category/Name eq 'Electronics'&$expand=Category
	// This tests that $select, $filter, and $expand can all work together with navigation properties
	
	filter := url.QueryEscape("Category/Name eq 'Electronics'")
	select_ := url.QueryEscape("Name,Category")
	expand := url.QueryEscape("Category")
	
	resp, err := ctx.GET("/Products?$select=" + select_ + "&$filter=" + filter + "&$expand=" + expand)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(resp.Body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	if _, ok := result["value"].([]interface{}); !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	// Note: The current implementation requires navigation properties to be
	// explicitly included in $select to be returned when $expand is used.
	// This test validates that the query doesn't fail with ambiguous column errors,
	// which was the main issue being fixed.

	return nil
}
