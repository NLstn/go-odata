package v4_01

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// MatchesPatternFilter creates the OData v4.01 Section 11.5.3.3 matchesPattern compliance test suite.
// matchesPattern(field, pattern) is a v4.01-only string function that filters entities
// using a POSIX ERE (Extended Regular Expression) pattern.
func MatchesPatternFilter() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.5.3.3 matchesPattern filter function (OData v4.01)",
		"Validates the OData v4.01 matchesPattern(field,pattern) filter function for POSIX ERE regex matching",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_matchesPattern",
	)

	// Test 1: Basic matchesPattern with anchored pattern (4.01 negotiated)
	suite.AddTest(
		"test_matchespattern_basic_4_01",
		"matchesPattern returns entities whose Name matches the pattern when OData-MaxVersion: 4.01 is negotiated",
		func(ctx *framework.TestContext) error {
			pattern := `^[A-Z]`
			filterExpr := fmt.Sprintf("matchesPattern(Name,'%s')", pattern)
			encodedFilter := url.QueryEscape(filterExpr)
			resp, err := ctx.GETWithHeaders(
				"/Products?$filter="+encodedFilter,
				map[string]string{"OData-MaxVersion": "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(entities) == 0 {
				return framework.NewError("matchesPattern filter returned no items; expected at least one product with a Name starting with an uppercase letter")
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid test pattern %q: %w", pattern, err)
			}
			for i, entity := range entities {
				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing string Name field", i))
				}
				if !re.MatchString(name) {
					return framework.NewError(fmt.Sprintf("entity %d has Name=%q which does not match pattern %q", i, name, pattern))
				}
			}
			return nil
		},
	)

	// Test 2: matchesPattern with suffix pattern (anchored at end)
	suite.AddTest(
		"test_matchespattern_suffix_pattern",
		"matchesPattern filters entities by a suffix POSIX ERE pattern",
		func(ctx *framework.TestContext) error {
			pattern := `e$`
			filterExpr := fmt.Sprintf("matchesPattern(Name,'%s')", pattern)
			encodedFilter := url.QueryEscape(filterExpr)
			resp, err := ctx.GETWithHeaders(
				"/Products?$filter="+encodedFilter,
				map[string]string{"OData-MaxVersion": "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid test pattern %q: %w", pattern, err)
			}
			for i, entity := range entities {
				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing string Name field", i))
				}
				if !re.MatchString(name) {
					return framework.NewError(fmt.Sprintf("entity %d has Name=%q which does not match pattern %q", i, name, pattern))
				}
			}
			return nil
		},
	)

	// Test 3: matchesPattern combined with another filter
	suite.AddTest(
		"test_matchespattern_combined_filter",
		"matchesPattern can be combined with other filter expressions using 'and'",
		func(ctx *framework.TestContext) error {
			pattern := `^[A-Z]`
			filterExpr := fmt.Sprintf("matchesPattern(Name,'%s') and Price gt 10", pattern)
			encodedFilter := url.QueryEscape(filterExpr)
			resp, err := ctx.GETWithHeaders(
				"/Products?$filter="+encodedFilter,
				map[string]string{"OData-MaxVersion": "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid test pattern %q: %w", pattern, err)
			}
			for i, entity := range entities {
				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing string Name field", i))
				}
				if !re.MatchString(name) {
					return framework.NewError(fmt.Sprintf("entity %d has Name=%q which does not match pattern %q", i, name, pattern))
				}
			}
			return nil
		},
	)

	// Test 4: matchesPattern with case-insensitive pattern (using ERE alternation)
	suite.AddTest(
		"test_matchespattern_case_pattern",
		"matchesPattern handles case-specific patterns correctly",
		func(ctx *framework.TestContext) error {
			// Match names containing 'laptop' or 'Laptop'
			pattern := `[Ll]aptop`
			filterExpr := fmt.Sprintf("matchesPattern(Name,'%s')", pattern)
			encodedFilter := url.QueryEscape(filterExpr)
			resp, err := ctx.GETWithHeaders(
				"/Products?$filter="+encodedFilter,
				map[string]string{"OData-MaxVersion": "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid test pattern %q: %w", pattern, err)
			}
			for i, entity := range entities {
				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing string Name field", i))
				}
				if !re.MatchString(name) {
					return framework.NewError(fmt.Sprintf("entity %d has Name=%q which does not match pattern %q", i, name, pattern))
				}
			}
			return nil
		},
	)

	// Test 5: matchesPattern with invalid syntax returns 400
	suite.AddTest(
		"test_matchespattern_missing_pattern_arg",
		"matchesPattern with wrong number of arguments returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GETWithHeaders(
				"/Products?$filter=matchesPattern(Name)",
				map[string]string{"OData-MaxVersion": "4.01"},
			)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	// Test 6: Verify matchesPattern is OData v4.01-specific by checking it works when 4.01 is negotiated
	suite.AddTest(
		"test_matchespattern_version_negotiation_4_01",
		"matchesPattern works correctly when OData-MaxVersion: 4.01 is negotiated",
		func(ctx *framework.TestContext) error {
			pattern := `Laptop`
			filterExpr := fmt.Sprintf("matchesPattern(Name,'%s')", pattern)
			encodedFilter := url.QueryEscape(filterExpr)
			resp, err := ctx.GETWithHeaders(
				"/Products?$filter="+encodedFilter,
				map[string]string{"OData-MaxVersion": "4.01"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return err
			}
			if len(entities) == 0 {
				return framework.NewError("expected at least one product with 'Laptop' in the name")
			}

			for i, entity := range entities {
				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing string Name field", i))
				}
				if !strings.Contains(name, "Laptop") {
					return framework.NewError(fmt.Sprintf("entity %d has Name=%q which does not contain 'Laptop'", i, name))
				}
			}
			return nil
		},
	)

	// Test 7: matchesPattern is rejected when OData-MaxVersion: 4.0 is negotiated (4.01-only feature)
	suite.AddTest(
		"test_matchespattern_version_negotiation_4_0_rejected",
		"matchesPattern returns 400 Bad Request when OData-MaxVersion: 4.0 is negotiated (4.01-only feature)",
		func(ctx *framework.TestContext) error {
			pattern := `^[A-Z]`
			filterExpr := fmt.Sprintf("matchesPattern(Name,'%s')", pattern)
			encodedFilter := url.QueryEscape(filterExpr)
			resp, err := ctx.GETWithHeaders(
				"/Products?$filter="+encodedFilter,
				map[string]string{"OData-MaxVersion": "4.0"},
			)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	return suite
}
