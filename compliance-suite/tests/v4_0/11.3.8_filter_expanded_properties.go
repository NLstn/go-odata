package v4_0

import (
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FilterExpandedProperties creates a test suite for filtering on expanded properties
func FilterExpandedProperties() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.8 Filter on Expanded Properties",
		"Tests filtering entities based on properties of expanded navigation entities",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_ExpandSystemQueryOption",
	)
	RegisterFilterExpandedPropertiesTests(suite)
	return suite
}

// RegisterFilterExpandedPropertiesTests registers tests for filtering on expanded navigation properties
func RegisterFilterExpandedPropertiesTests(suite *framework.TestSuite) {
	suite.AddTest(
		"Filter on collection navigation with any()",
		"Filter entities based on properties of collection navigation using any()",
		testFilterAnyOnNavigation,
	)

	suite.AddTest(
		"Filter on collection navigation with all()",
		"Filter entities using all() operator on collection navigation",
		testFilterAllOnNavigation,
	)

	suite.AddTest(
		"Filter with any() and complex condition",
		"Use any() with compound boolean expressions on navigation properties",
		testFilterAnyComplex,
	)

	suite.AddTest(
		"Expand with filter on expanded entities",
		"Apply $filter to expanded navigation collection",
		testExpandWithNestedFilter,
	)

	suite.AddTest(
		"Filter main and expanded entities",
		"Combine filter on main entity with filter on expanded entities",
		testFilterBothLevels,
	)

	suite.AddTest(
		"Any with string function on navigation",
		"Use string functions within any() lambda expression",
		testAnyWithStringFunction,
	)

	suite.AddTest(
		"Multiple any() filters on same navigation",
		"Apply multiple any() conditions on same collection navigation",
		testMultipleAnyFilters,
	)

	suite.AddTest(
		"Navigation filter with or condition",
		"Use or operator within any() lambda expression",
		testNavigationFilterOr,
	)

	suite.AddTest(
		"Nested condition in any() with function",
		"Combine functions and comparisons within any() expression",
		testNestedAnyCondition,
	)

	suite.AddTest(
		"Expand and filter same navigation property",
		"Apply both filter and expand to same navigation collection",
		testExpandAndFilterSameNav,
	)

	suite.AddTest(
		"Filter with not and any on navigation",
		"Use not operator with any() on navigation property",
		testNotAnyOnNavigation,
	)

	suite.AddTest(
		"Complex filter combining entity and navigation",
		"Combine entity property filters with navigation property filters",
		testComplexCombinedFilter,
	)
}

func testFilterAnyOnNavigation(ctx *framework.TestContext) error {
	// Filter products that have descriptions in English
	filter := url.QueryEscape("Descriptions/any(d: d/LanguageKey eq 'EN')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testFilterAllOnNavigation(ctx *framework.TestContext) error {
	// Filter products where all descriptions are NOT in XX language (non-existent)
	filter := url.QueryEscape("Descriptions/all(d: d/LanguageKey ne 'XX')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testFilterAnyComplex(ctx *framework.TestContext) error {
	// Filter products with English descriptions containing "Laptop"
	filter := url.QueryEscape("Descriptions/any(d: d/LanguageKey eq 'EN' and contains(d/Description, 'Laptop'))")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testExpandWithNestedFilter(ctx *framework.TestContext) error {
	expand := url.QueryEscape("Descriptions($filter=LanguageKey eq 'EN')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$expand=%s", expand))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testFilterBothLevels(ctx *framework.TestContext) error {
	filter := url.QueryEscape("Price gt 100")
	expand := url.QueryEscape("Descriptions($filter=LanguageKey eq 'EN')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s&$expand=%s", filter, expand))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testAnyWithStringFunction(ctx *framework.TestContext) error {
	// Filter products with descriptions containing "laptop" (case insensitive via contains)
	filter := url.QueryEscape("Descriptions/any(d: contains(d/Description, 'laptop'))")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testMultipleAnyFilters(ctx *framework.TestContext) error {
	// Filter products that have both EN and DE descriptions
	filter := url.QueryEscape("Descriptions/any(d: d/LanguageKey eq 'EN') and Descriptions/any(d: d/LanguageKey eq 'DE')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testNavigationFilterOr(ctx *framework.TestContext) error {
	// Filter products with descriptions in EN or DE language
	filter := url.QueryEscape("Descriptions/any(d: d/LanguageKey eq 'EN' or d/LanguageKey eq 'DE')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testNestedAnyCondition(ctx *framework.TestContext) error {
	// Complex condition with string function and comparison
	filter := url.QueryEscape("Descriptions/any(d: contains(d/Description, 'laptop') and d/LanguageKey eq 'EN')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testExpandAndFilterSameNav(ctx *framework.TestContext) error {
	// Apply both $filter and $expand on same navigation property
	filter := url.QueryEscape("Descriptions/any(d: d/LanguageKey eq 'EN')")
	expand := "Descriptions"
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s&$expand=%s", filter, expand))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testNotAnyOnNavigation(ctx *framework.TestContext) error {
	// Filter products that do NOT have any descriptions in French
	filter := url.QueryEscape("not Descriptions/any(d: d/LanguageKey eq 'FR')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testComplexCombinedFilter(ctx *framework.TestContext) error {
	// Combine entity property filter with navigation property filter
	filter := url.QueryEscape("Price gt 100 and Descriptions/any(d: d/LanguageKey eq 'EN')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}
