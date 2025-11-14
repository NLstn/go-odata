package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// RequestingEntities creates the 11.4.1 Requesting Individual Entities test suite
func RequestingEntities() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.1 Requesting Individual Entities",
		"Tests requesting individual entities with various methods",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_RequestingIndividualEntities",
	)

	registerRequestingEntitiesTests(suite)
	return suite
}

func registerRequestingEntitiesTests(suite *framework.TestSuite) {
	invalidProductPath := nonExistingEntityPath("Products")

	suite.AddTest(
		"GET individual entity by key",
		"GET individual entity by key should return entity",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// Verify entity has ID property
			if !strings.Contains(string(resp.Body), `"ID"`) {
				return fmt.Errorf("entity missing ID property")
			}

			// Should not be wrapped in "value" array
			if strings.Contains(string(resp.Body), `"value"`) {
				return fmt.Errorf("single entity should not be wrapped in value array")
			}

			return nil
		},
	)

	suite.AddTest(
		"HEAD request for individual entity",
		"HEAD request should return 200 without body",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			getResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(getResp, 200); err != nil {
				return err
			}

			headResp, err := ctx.HEAD(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(headResp, 200); err != nil {
				return err
			}

			if len(headResp.Body) != 0 {
				return fmt.Errorf("expected no body in HEAD response, got %d bytes", len(headResp.Body))
			}

			// HEAD must provide the same metadata as GET responses.
			// Validate key headers to ensure parity with the entity retrieval.
			if ct := headResp.Headers.Get("Content-Type"); ct == "" {
				return fmt.Errorf("Content-Type header missing in HEAD response")
			} else if expected := getResp.Headers.Get("Content-Type"); expected != "" && ct != expected {
				return fmt.Errorf("Content-Type header mismatch: expected %q, got %q", expected, ct)
			}

			if ov := headResp.Headers.Get("OData-Version"); ov == "" {
				return fmt.Errorf("OData-Version header missing in HEAD response")
			} else if expected := getResp.Headers.Get("OData-Version"); expected != "" && ov != expected {
				return fmt.Errorf("OData-Version header mismatch: expected %q, got %q", expected, ov)
			}

			if etag := getResp.Headers.Get("ETag"); etag != "" {
				if headResp.Headers.Get("ETag") != etag {
					return fmt.Errorf("ETag header mismatch: expected %q", etag)
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"Conditional request with If-None-Match",
		"Request with If-None-Match should return 304 for matching ETag",
		func(ctx *framework.TestContext) error {
			// First request to get ETag
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			etag := resp.Headers.Get("ETag")
			if etag == "" {
				// ETag support is optional
				return ctx.Skip("ETag support not implemented")
			}

			// Second request with If-None-Match
			resp2, err := ctx.GET(productPath, framework.Header{
				Key:   "If-None-Match",
				Value: etag,
			})
			if err != nil {
				return err
			}

			if resp2.StatusCode != 304 {
				return fmt.Errorf("expected status 304, got %d", resp2.StatusCode)
			}

			return nil
		},
	)

	suite.AddTest(
		"Request non-existent entity returns 404",
		"Request for non-existent entity should return 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}

			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	suite.AddTest(
		"Request individual entity with $select",
		"Request entity with query options should work",
		func(ctx *framework.TestContext) error {
			selectParam := url.QueryEscape("$select") + "=Name,Price"
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "?" + selectParam)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// Verify response contains selected properties
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}

			if _, ok := result["Name"]; !ok {
				return fmt.Errorf("response missing Name property")
			}
			if _, ok := result["Price"]; !ok {
				return fmt.Errorf("response missing Price property")
			}

			return nil
		},
	)
}
