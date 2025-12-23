package v4_0

import (
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// URLEncoding creates the 11.2.14 URL Encoding test suite
func URLEncoding() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.14 URL Encoding",
		"Tests proper handling of URL encoding in resource paths, query parameters, and special characters according to RFC 3986.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html",
	)

	// Test 1: String literal with spaces in filter
	suite.AddTest(
		"test_filter_with_spaces",
		"Filter with URL-encoded spaces",
		func(ctx *framework.TestContext) error {
			// URL-encode the query parameter properly
			path := "/Products?$filter=" + url.QueryEscape("Name eq 'Gaming Laptop'")
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 2: String literal with special characters in filter
	suite.AddTest(
		"test_filter_special_chars",
		"Filter with encoded special characters (&)",
		func(ctx *framework.TestContext) error {
			path := "/Products?$filter=" + url.QueryEscape("contains(Name,'&')")
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 3: Query option with encoded $ symbol
	suite.AddTest(
		"test_encoded_dollar_sign",
		"Query option with encoded dollar sign",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?%24top=5")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 4: Filter with URL-encoded operators
	suite.AddTest(
		"test_encoded_operators",
		"Filter with URL-encoded operators",
		func(ctx *framework.TestContext) error {
			path := "/Products?$filter=" + url.QueryEscape("Price gt 50 and Price lt 200")
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 5: String literal with single quote (escaped)
	suite.AddTest(
		"test_filter_single_quote",
		"Filter with single quote in string literal",
		func(ctx *framework.TestContext) error {
			// Single quotes in OData string literals are escaped by doubling them
			path := "/Products?$filter=" + url.QueryEscape("contains(Name,'''')")
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 6: Parentheses in filter expressions
	suite.AddTest(
		"test_parentheses_encoding",
		"Filter with URL-encoded parentheses",
		func(ctx *framework.TestContext) error {
			// Use Status (integer) instead of ID (UUID) for type-safe comparison across databases
			path := "/Products?$filter=" + url.QueryEscape("(Price gt 100) and (Status lt 10)")
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 7: Mixed encoded and unencoded characters
	suite.AddTest(
		"test_mixed_encoding",
		"Mixed encoded and unencoded parameters",
		func(ctx *framework.TestContext) error {
			path := "/Products?$filter=" + url.QueryEscape("Price gt 50") + "&$top=10"
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 8: Plus sign encoding
	suite.AddTest(
		"test_plus_sign",
		"Plus sign handling in URL",
		func(ctx *framework.TestContext) error {
			path := "/Products?$filter=" + url.QueryEscape("Price gt 0")
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 9: Percent encoding in string literals
	suite.AddTest(
		"test_percent_in_string",
		"Percent sign in filter string",
		func(ctx *framework.TestContext) error {
			path := "/Products?$filter=" + url.QueryEscape("contains(Name,'%')")
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 10: Reserved characters in query string
	suite.AddTest(
		"test_reserved_chars",
		"Reserved characters handled gracefully",
		func(ctx *framework.TestContext) error {
			// Semicolon is a reserved character
			path := "/Products?$filter=" + url.QueryEscape("Price gt 10;id lt 100")
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			// This may fail as semicolon is a query parameter separator in some contexts
			// We're testing that the server handles it gracefully
			if resp.StatusCode != 200 && resp.StatusCode != 400 {
				return fmt.Errorf("expected status 200 or 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 11: Unicode characters in filter
	suite.AddTest(
		"test_unicode_characters",
		"Unicode characters in filter",
		func(ctx *framework.TestContext) error {
			// Test with UTF-8 characters (é)
			path := "/Products?$filter=" + url.QueryEscape("contains(Name,'é')")
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 12: Case sensitivity of query options
	suite.AddTest(
		"test_query_option_case",
		"Query option case handling",
		func(ctx *framework.TestContext) error {
			// Query options should be case-insensitive (though lowercase is recommended)
			resp, err := ctx.GET("/Products?$FILTER=Status%20eq%201")
			if err != nil {
				return err
			}

			// Some implementations may be case-sensitive
			if resp.StatusCode != 200 && resp.StatusCode != 400 {
				return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			return nil
		},
	)

	return suite
}
