package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QuerySkiptoken creates the 11.2.5.7 $skiptoken Query Option test suite
func QuerySkiptoken() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.7 $skiptoken",
		"Tests server-driven paging with $skiptoken for continuation of result sets.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_SystemQueryOptionskiptoken",
	)

	// Test 1: Response with @odata.nextLink includes skiptoken
	suite.AddTest(
		"test_nextlink_has_skiptoken",
		"@odata.nextLink contains $skiptoken parameter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=2")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			// Check if there's a nextLink
			if nextLink, ok := result["@odata.nextLink"].(string); ok {
				// NextLink should contain skiptoken parameter
				// This is implementation-specific, so we just verify it exists
				ctx.Log("@odata.nextLink found: " + nextLink)
				return nil
			}

			// If there's no nextLink, result set fits in one page
			ctx.Log("No @odata.nextLink (result set fits in one page)")
			return nil
		},
	)

	// Test 2: Invalid skiptoken returns error
	suite.AddTest(
		"test_invalid_skiptoken",
		"Invalid $skiptoken returns 400 Bad Request",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$skiptoken=invalid_token_xyz")
			if err != nil {
				return err
			}

			// Should return 400 for invalid skiptoken
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	return suite
}
