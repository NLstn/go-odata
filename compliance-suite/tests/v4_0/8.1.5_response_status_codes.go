package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ResponseStatusCodes creates the 8.1.5 Response Status Codes test suite
func ResponseStatusCodes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.1.5 Response Status Codes",
		"Validates correct HTTP status codes for successful operations, client errors, and server errors.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ResponseStatusCodes",
	)

	invalidProductPath := nonExistingEntityPath("Products")

	suite.AddTest(
		"test_status_200_ok",
		"200 OK for successful GET",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_status_201_created",
		"201 Created for successful POST",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Status Code Product", 18.25)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}
			if resp.Headers.Get("Location") == "" {
				return framework.NewError("201 Created response missing Location header")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_status_204_no_content",
		"204 No Content for successful DELETE",
		func(ctx *framework.TestContext) error {
			productID, err := createTestProduct(ctx, "Delete Status Product", 22.5)
			if err != nil {
				return err
			}

			resp, err := ctx.DELETE(fmt.Sprintf("/Products(%s)", productID))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 204); err != nil {
				return err
			}
			if len(resp.Body) != 0 {
				return framework.NewError("204 No Content response must not include a response body")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_status_404_not_found",
		"404 Not Found for non-existent entity",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 404, "does not exist")
		},
	)

	suite.AddTest(
		"test_status_400_bad_request",
		"400 Bad Request for invalid filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=invalid syntax")
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 400, "invalid $filter")
		},
	)

	suite.AddTest(
		"test_status_405_method_not_allowed",
		"405 Method Not Allowed for unsupported collection PUT",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.PUT("/Products", []byte(`{}`))
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 405)
		},
	)

	suite.AddTest(
		"test_status_406_not_acceptable",
		"406 Not Acceptable for unsupported entity response media type",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 406, "requested format")
		},
	)

	suite.AddTest(
		"test_status_415_unsupported_media_type",
		"415 Unsupported Media Type for unsupported request body content type",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Products", []byte("not-json"), framework.Header{Key: "Content-Type", Value: "text/plain"})
			if err != nil {
				return err
			}
			return ctx.AssertODataError(resp, 415, "Content-Type")
		},
	)

	return suite
}
