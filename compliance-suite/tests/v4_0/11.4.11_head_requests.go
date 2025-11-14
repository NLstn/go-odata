package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HEADRequests creates the 11.4.11 HEAD Requests test suite
func HEADRequests() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.11 HEAD Requests",
		"Validates HEAD requests for entities, collections, and documents as defined in OData v4.0 section 11.4.11.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HEADRequests",
	)
	invalidProductPath := nonExistingEntityPath("Products")

	// Test 1: HEAD request on entity collection
	suite.AddTest(
		"test_head_collection",
		"HEAD request on entity collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.HEAD("/Products")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 2: HEAD request on single entity
	suite.AddTest(
		"test_head_entity",
		"HEAD request on single entity",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				if err.Error() == "status code: 500 (expected 200)" {
					return ctx.Skip("GET request returns 500, skipping HEAD test")
				}
				return err
			}
			resp, err := ctx.HEAD(productPath)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 3: HEAD request returns no body
	suite.AddTest(
		"test_head_no_body",
		"HEAD request returns headers only",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.HEAD("/Products")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// HEAD should return headers only, no body
			if len(resp.Body) > 0 {
				return fmt.Errorf("HEAD should not return body content, got %d bytes", len(resp.Body))
			}

			return nil
		},
	)

	// Test 4: HEAD request includes Content-Length
	suite.AddTest(
		"test_head_content_length",
		"HEAD response includes Content-Length",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.HEAD("/Products")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentLength := resp.Headers.Get("Content-Length")
			if contentLength == "" {
				return fmt.Errorf("Content-Length header missing")
			}

			return nil
		},
	)

	// Test 5: HEAD request includes OData-Version
	suite.AddTest(
		"test_head_odata_version",
		"HEAD response includes OData-Version",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.HEAD("/Products")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			version := resp.Headers.Get("OData-Version")
			if version == "" {
				return fmt.Errorf("OData-Version header missing")
			}

			return nil
		},
	)

	// Test 6: HEAD request with query options
	suite.AddTest(
		"test_head_with_query",
		"HEAD request with query options",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.HEAD("/Products?$top=5")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 7: HEAD request on non-existent entity returns 404
	suite.AddTest(
		"test_head_not_found",
		"HEAD on non-existent entity returns 404",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.HEAD(invalidProductPath)
			if err != nil {
				return err
			}

			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 8: HEAD request includes Content-Type
	suite.AddTest(
		"test_head_content_type",
		"HEAD response includes Content-Type",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.HEAD("/Products")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if contentType == "" {
				return fmt.Errorf("Content-Type header missing")
			}

			return nil
		},
	)

	// Test 9: HEAD request on service document
	suite.AddTest(
		"test_head_service_document",
		"HEAD request on service document",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.HEAD("/")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 10: HEAD request on metadata document
	suite.AddTest(
		"test_head_metadata",
		"HEAD request on metadata document",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.HEAD("/$metadata")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 11: HEAD with Accept header
	suite.AddTest(
		"test_head_accept_header",
		"HEAD with Accept header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.HEAD("/Products", framework.Header{Key: "Accept", Value: "application/json"})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	return suite
}
