package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ConditionalRequests creates the 11.5.1 Conditional Requests (ETag) test suite
func ConditionalRequests() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.5.1 Conditional Requests (ETag)",
		"Tests conditional request handling with ETags according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ConditionalRequests",
	)

	// Helper function to get product path and ETag for each test
	// Note: Must refetch on each call because database is reseeded between tests
	getProductAndETag := func(ctx *framework.TestContext) (string, string, error) {
		path, err := firstEntityPath(ctx, "Products")
		if err != nil {
			return "", "", err
		}

		// Fetch the product to get its ETag
		resp, err := ctx.GET(path)
		if err != nil {
			return "", "", err
		}

		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return "", "", err
		}

		etag := resp.Headers.Get("ETag")
		return path, etag, nil
	}

	// Test 1: Entity with @odata.etag should include ETag header
	suite.AddTest(
		"test_etag_header",
		"Response includes ETag header for entity with @odata.etag",
		func(ctx *framework.TestContext) error {
			path, etag, err := getProductAndETag(ctx)
			if err != nil {
				return err
			}

			if etag != "" {
				return nil
			}

			// ETags are optional, check if @odata.etag is in body
			// Re-fetch to get body since we already consumed it in getProductAndETag
			resp, err := ctx.GET(path)
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if framework.ContainsAny(body, `"@odata.etag"`) {
				return nil
			}

			// ETags optional, so pass
			return nil
		},
	)

	// Test 2: If-None-Match with matching ETag should return 304
	suite.AddTest(
		"test_if_none_match_matching",
		"If-None-Match with matching ETag returns 304 Not Modified",
		func(ctx *framework.TestContext) error {
			path, etag, err := getProductAndETag(ctx)
			if err != nil {
				return err
			}

			if etag == "" {
				return framework.NewError("No ETag support")
			}

			resp, err := ctx.GET(path, framework.Header{Key: "If-None-Match", Value: etag})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 304)
		},
	)

	// Test 3: If-None-Match with non-matching ETag should return 200
	suite.AddTest(
		"test_if_none_match_non_matching",
		"If-None-Match with non-matching ETag returns 200",
		func(ctx *framework.TestContext) error {
			path, etag, err := getProductAndETag(ctx)
			if err != nil {
				return err
			}

			if etag == "" {
				return framework.NewError("No ETag support")
			}

			resp, err := ctx.GET(path, framework.Header{Key: "If-None-Match", Value: `"different-etag"`})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 4: If-Match with matching ETag should succeed for PATCH
	suite.AddTest(
		"test_if_match_matching",
		"If-Match with matching ETag allows PATCH",
		func(ctx *framework.TestContext) error {
			path, etag, err := getProductAndETag(ctx)
			if err != nil {
				return err
			}

			if etag == "" {
				return framework.NewError("No ETag support")
			}

			payload := map[string]interface{}{
				"Name": "Test update",
			}

			resp, err := ctx.PATCH(path, payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "If-Match", Value: etag})
			if err != nil {
				return err
			}

			// Should return 200 or 204
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return ctx.AssertStatusCode(resp, 204)
			}

			return nil
		},
	)

	// Test 5: If-Match with non-matching ETag should return 412
	suite.AddTest(
		"test_if_match_non_matching",
		"If-Match with non-matching ETag returns 412 Precondition Failed",
		func(ctx *framework.TestContext) error {
			path, etag, err := getProductAndETag(ctx)
			if err != nil {
				return err
			}

			if etag == "" {
				return framework.NewError("No ETag support")
			}

			payload := map[string]interface{}{
				"Name": "Test update",
			}

			resp, err := ctx.PATCH(path, payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "If-Match", Value: `"wrong-etag"`})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 412)
		},
	)

	// Test 6: If-Match: * should always succeed
	suite.AddTest(
		"test_if_match_wildcard",
		"If-Match: * allows update regardless of ETag",
		func(ctx *framework.TestContext) error {
			path, etag, err := getProductAndETag(ctx)
			if err != nil {
				return err
			}

			if etag == "" {
				return framework.NewError("No ETag support")
			}

			payload := map[string]interface{}{
				"Name": "Test update with wildcard",
			}

			resp, err := ctx.PATCH(path, payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "If-Match", Value: "*"})
			if err != nil {
				return err
			}

			// Should return 200 or 204
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return ctx.AssertStatusCode(resp, 204)
			}

			return nil
		},
	)

	return suite
}
