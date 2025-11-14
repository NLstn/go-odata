package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// AsynchronousRequests creates the 11.4.10 Asynchronous Requests test suite
func AsynchronousRequests() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.10 Asynchronous Requests",
		"Tests asynchronous request processing with Prefer: respond-async header.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_AsynchronousRequests",
	)

	// Test 1: Prefer respond-async header is accepted
	suite.AddTest(
		"test_async_header_accepted",
		"Prefer: respond-async header is accepted",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			// Should return either 200 (sync) or 202 (async)
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return ctx.AssertStatusCode(resp, 202)
			}

			return nil
		},
	)

	// Test 2: Async POST request
	suite.AddTest(
		"test_async_post",
		"Async POST request is handled",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Name":       "Async Test Product",
				"Price":      99.99,
				"CategoryID": 1,
				"Status":     1,
			}

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			// Should return 201 (sync) or 202 (async)
			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return ctx.AssertStatusCode(resp, 202)
			}

			return nil
		},
	)

	// Test 3: Check for Location header on 202 response
	suite.AddTest(
		"test_async_location_header",
		"202 response includes Location header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 202 {
				location := resp.Headers.Get("Location")
				if location == "" {
					return framework.NewError("202 response missing Location header")
				}
			}

			// If not 202, service processed synchronously which is also valid
			return nil
		},
	)

	return suite
}
