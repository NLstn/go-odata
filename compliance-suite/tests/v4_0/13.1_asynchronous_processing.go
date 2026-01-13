package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// AsynchronousProcessing creates the 13.1 Asynchronous Request Processing test suite
func AsynchronousProcessing() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"13.1 Asynchronous Processing",
		"Tests asynchronous request processing features including the Prefer: respond-async header, status monitor URLs, and proper async response patterns.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_AsynchronousRequests",
	)

	// Test 1: Service accepts Prefer: respond-async header
	suite.AddTest(
		"test_prefer_async_header_accepted",
		"Service accepts Prefer: respond-async header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			// Should return 200 (synchronous) or 202 (asynchronous)
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return ctx.AssertStatusCode(resp, 202)
			}

			return nil
		},
	)

	// Test 2: Service can respond synchronously even with Prefer: respond-async
	suite.AddTest(
		"test_synchronous_response_allowed",
		"Service can respond synchronously with async preference",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			// Both 200 and 202 are valid
			if resp.StatusCode == 200 || resp.StatusCode == 202 {
				return nil
			}

			return framework.NewError("Service should respond with 200 or 202")
		},
	)

	// Test 3: Async response returns 202 Accepted
	suite.AddTest(
		"test_async_returns_202",
		"Async 202 response",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 202 {
				return nil
			} else if resp.StatusCode == 200 {
				return framework.NewError("Service does not support asynchronous processing")
			}

			return framework.NewError("Unexpected status code")
		},
	)

	// Test 4: Async response includes Location header
	suite.AddTest(
		"test_async_location_header",
		"Async 202 response includes Location header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			if resp.StatusCode == 202 {
				location := resp.Headers.Get("Location")
				if location == "" {
					return framework.NewError("202 Accepted response must include Location header")
				}
				return nil
			} else if resp.StatusCode == 200 {
				return framework.NewError("Service does not support asynchronous processing")
			}

			return framework.NewError("Unexpected status code")
		},
	)

	// Test 5: POST with async preference
	suite.AddTest(
		"test_async_post_request",
		"POST request with async preference",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Name":       "Async Test",
				"Price":      99.99,
				"CategoryID": 1,
			}

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Prefer", Value: "respond-async"},
				framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// Should return 201, 202, or 400
			if resp.StatusCode == 201 || resp.StatusCode == 202 || resp.StatusCode == 400 {
				return nil
			}

			return framework.NewError("Unexpected status code for POST with async")
		},
	)

	// Test 6: Service handles multiple concurrent async requests
	suite.AddTest(
		"test_multiple_async_requests",
		"Service handles multiple async requests",
		func(ctx *framework.TestContext) error {
			resp1, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			resp2, err := ctx.GET("/Categories", framework.Header{Key: "Prefer", Value: "respond-async"})
			if err != nil {
				return err
			}

			// Both should succeed (200 or 202)
			if (resp1.StatusCode == 200 || resp1.StatusCode == 202) &&
				(resp2.StatusCode == 200 || resp2.StatusCode == 202) {
				return nil
			}

			return framework.NewError("Service should handle multiple async requests")
		},
	)

	return suite
}
