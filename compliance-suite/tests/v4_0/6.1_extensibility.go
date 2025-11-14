package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// Extensibility creates the 6.1 Extensibility test suite
func Extensibility() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"6.1 Extensibility",
		"Tests OData v4 extensibility features including support for instance annotations and proper handling of unknown elements.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Extensibility",
	)

	suite.AddTest(
		"test_ignores_unknown_headers",
		"Service accepts requests with custom headers",
		func(ctx *framework.TestContext) error {
			// Services MUST NOT require clients to understand custom headers
			resp, err := ctx.GET("/Products", framework.Header{
				Key:   "X-Custom-Header",
				Value: "test",
			})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_no_custom_headers_required",
		"Standard operations work without custom headers",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_instance_annotation_format",
		"Instance annotations use @ prefix",
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

			body := string(resp.Body)
			if !strings.Contains(body, `"@odata`) {
				return framework.NewError("Response should contain OData instance annotations with @ prefix")
			}

			return nil
		},
	)

	return suite
}
