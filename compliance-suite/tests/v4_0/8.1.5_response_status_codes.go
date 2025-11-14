package v4_0

import (
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
		"test_status_404_not_found",
		"404 Not Found for non-existent entity",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 404)
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
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
