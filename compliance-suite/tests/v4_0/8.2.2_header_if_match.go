package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderIfMatch creates the 8.2.2 If-Match Header test suite
func HeaderIfMatch() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.2 If-Match Header",
		"Tests If-Match and If-None-Match headers for optimistic concurrency control.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderIfMatch",
	)

	suite.AddTest(
		"test_etag_in_get_response",
		"ETag header present in GET response",
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

			etag := resp.Headers.Get("ETag")
			if etag == "" {
				// ETag is optional but recommended
				return ctx.Skip("ETag header not supported by service")
			}

			// Validate ETag format - should be a quoted string
			if !strings.HasPrefix(etag, "\"") || !strings.HasSuffix(etag, "\"") {
				if !strings.HasPrefix(etag, "W/\"") {
					return fmt.Errorf("ETag must be a quoted string or weak ETag (W/\"...\"), got: %s", etag)
				}
			}

			ctx.Log(fmt.Sprintf("ETag received: %s", etag))
			return nil
		},
	)

	suite.AddTest(
		"test_if_match_with_valid_etag",
		"If-Match with valid ETag allows update",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			// Get current entity and ETag
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			etag := resp.Headers.Get("ETag")
			if etag == "" {
				return ctx.Skip("ETag not supported - cannot test If-Match")
			}

			// Update with matching ETag
			update := map[string]interface{}{
				"Name": "Updated with If-Match",
			}

			patchResp, err := ctx.PATCH(productPath, update,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "If-Match", Value: etag},
			)
			if err != nil {
				return err
			}

			// Should succeed with 200 or 204
			if patchResp.StatusCode != 200 && patchResp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204 with matching ETag, got %d", patchResp.StatusCode)
			}

			ctx.Log(fmt.Sprintf("Update succeeded with If-Match: %s", etag))
			return nil
		},
	)

	suite.AddTest(
		"test_if_match_with_mismatched_etag",
		"If-Match with mismatched ETag returns 412",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			// Get current entity to verify ETag support
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			etag := resp.Headers.Get("ETag")
			if etag == "" {
				return ctx.Skip("ETag not supported - cannot test If-Match")
			}

			// Try to update with intentionally wrong ETag
			wrongETag := "\"wrong-etag-value\""
			update := map[string]interface{}{
				"Name": "Should fail",
			}

			patchResp, err := ctx.PATCH(productPath, update,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "If-Match", Value: wrongETag},
			)
			if err != nil {
				return err
			}

			// Must return 412 Precondition Failed
			if patchResp.StatusCode != 412 {
				return fmt.Errorf("expected status 412 Precondition Failed with mismatched ETag, got %d", patchResp.StatusCode)
			}

			ctx.Log("Correctly rejected update with mismatched ETag (412)")
			return nil
		},
	)

	suite.AddTest(
		"test_if_match_star",
		"If-Match: * matches any version",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			// Verify entity exists
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			etag := resp.Headers.Get("ETag")
			if etag == "" {
				return ctx.Skip("ETag not supported - cannot test If-Match: *")
			}

			// Update with If-Match: *
			update := map[string]interface{}{
				"Name": "Updated with If-Match: *",
			}

			patchResp, err := ctx.PATCH(productPath, update,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "If-Match", Value: "*"},
			)
			if err != nil {
				return err
			}

			// Should succeed - * matches any existing entity
			if patchResp.StatusCode != 200 && patchResp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204 with If-Match: *, got %d", patchResp.StatusCode)
			}

			ctx.Log("If-Match: * correctly matched existing entity")
			return nil
		},
	)

	suite.AddTest(
		"test_if_match_star_nonexistent",
		"If-Match: * on non-existent entity returns 404",
		func(ctx *framework.TestContext) error {
			// Check if any entity supports ETag first
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if resp.Headers.Get("ETag") == "" {
				return ctx.Skip("ETag not supported")
			}

			// Try to update non-existent entity with If-Match: *
			nonExistentPath := nonExistingEntityPath("Products")
			update := map[string]interface{}{
				"Name": "Should fail",
			}

			patchResp, err := ctx.PATCH(nonExistentPath, update,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "If-Match", Value: "*"},
			)
			if err != nil {
				return err
			}

			// Should return 404 - entity doesn't exist
			if patchResp.StatusCode != 404 {
				return fmt.Errorf("expected status 404 for non-existent entity with If-Match: *, got %d", patchResp.StatusCode)
			}

			ctx.Log("Correctly returned 404 for If-Match: * on non-existent entity")
			return nil
		},
	)

	suite.AddTest(
		"test_if_none_match_with_matching_etag",
		"If-None-Match with matching ETag returns 304",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			// Get current entity and ETag
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			etag := resp.Headers.Get("ETag")
			if etag == "" {
				return ctx.Skip("ETag not supported - cannot test If-None-Match")
			}

			// GET with If-None-Match using same ETag
			getResp, err := ctx.GET(productPath,
				framework.Header{Key: "If-None-Match", Value: etag},
			)
			if err != nil {
				return err
			}

			// Should return 304 Not Modified
			if getResp.StatusCode != 304 {
				return fmt.Errorf("expected status 304 Not Modified with matching If-None-Match, got %d", getResp.StatusCode)
			}

			// Body should be empty for 304 responses
			if len(getResp.Body) > 0 {
				return fmt.Errorf("304 response should have empty body, got %d bytes", len(getResp.Body))
			}

			ctx.Log("Correctly returned 304 Not Modified for matching ETag")
			return nil
		},
	)

	suite.AddTest(
		"test_if_none_match_with_different_etag",
		"If-None-Match with different ETag returns 200",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			// Verify ETag support
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if resp.Headers.Get("ETag") == "" {
				return ctx.Skip("ETag not supported - cannot test If-None-Match")
			}

			// GET with If-None-Match using different ETag
			differentETag := "\"different-etag\""
			getResp, err := ctx.GET(productPath,
				framework.Header{Key: "If-None-Match", Value: differentETag},
			)
			if err != nil {
				return err
			}

			// Should return 200 and full entity
			if getResp.StatusCode != 200 {
				return fmt.Errorf("expected status 200 with non-matching If-None-Match, got %d", getResp.StatusCode)
			}

			// Should have body with entity
			var entity map[string]interface{}
			if err := json.Unmarshal(getResp.Body, &entity); err != nil {
				return fmt.Errorf("expected valid entity in response body: %w", err)
			}

			ctx.Log("Correctly returned 200 with entity for non-matching ETag")
			return nil
		},
	)

	suite.AddTest(
		"test_etag_changes_after_update",
		"ETag changes after successful update",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			// Get initial ETag
			resp1, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp1, 200); err != nil {
				return err
			}

			etag1 := resp1.Headers.Get("ETag")
			if etag1 == "" {
				return ctx.Skip("ETag not supported")
			}

			// Update entity
			update := map[string]interface{}{
				"Name": "Updated to change ETag",
			}
			patchResp, err := ctx.PATCH(productPath, update,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			if patchResp.StatusCode != 200 && patchResp.StatusCode != 204 {
				return fmt.Errorf("update failed with status %d", patchResp.StatusCode)
			}

			// Get updated entity and new ETag
			resp2, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp2, 200); err != nil {
				return err
			}

			etag2 := resp2.Headers.Get("ETag")
			if etag2 == "" {
				return fmt.Errorf("ETag missing after update")
			}

			// ETags should be different
			if etag1 == etag2 {
				return fmt.Errorf("ETag should change after update, but remained: %s", etag1)
			}

			ctx.Log(fmt.Sprintf("ETag correctly changed from %s to %s", etag1, etag2))
			return nil
		},
	)

	return suite
}
