package v4_01

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderAsyncResult creates the 8.3.1 AsyncResult header test suite.
func HeaderAsyncResult() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.3.1 Header AsyncResult",
		"Validates that OData 4.01 asynchronous final responses include the AsyncResult header.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderAsyncResult",
	)

	suite.AddTest(
		"test_final_async_response_includes_async_result",
		"If status monitor returns 200 for async completion, it includes AsyncResult",
		func(ctx *framework.TestContext) error {
			initialResp, err := ctx.GET(
				"/Products?$top=1",
				framework.Header{Key: "Prefer", Value: "respond-async"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
				framework.Header{Key: "Accept", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			if initialResp.StatusCode == http.StatusOK {
				return ctx.Skip("service chose synchronous processing for this request; AsyncResult requirement applies to asynchronous final responses")
			}

			if err := ctx.AssertStatusCode(initialResp, http.StatusAccepted); err != nil {
				return framework.NewError(fmt.Sprintf("expected 202 for async processing or 200 for sync, got %d", initialResp.StatusCode))
			}

			location := initialResp.Headers.Get("Location")
			if location == "" {
				return framework.NewError("202 response missing Location header for status monitor resource")
			}

			statusPath := location
			if strings.HasPrefix(statusPath, ctx.ServerURL()) {
				statusPath = strings.TrimPrefix(statusPath, ctx.ServerURL())
			}
			if !strings.HasPrefix(statusPath, "/") {
				statusPath = "/" + strings.TrimPrefix(statusPath, "/")
			}

			var finalResp *framework.HTTPResponse
			for i := 0; i < 10; i++ {
				finalResp, err = ctx.GET(
					statusPath,
					framework.Header{Key: "OData-MaxVersion", Value: "4.01"},
					framework.Header{Key: "Accept", Value: "application/json"},
				)
				if err != nil {
					return err
				}

				if finalResp.StatusCode != http.StatusAccepted {
					break
				}

				time.Sleep(100 * time.Millisecond)
			}

			if finalResp == nil {
				return framework.NewError("failed to retrieve status monitor response")
			}

			if finalResp.StatusCode == http.StatusAccepted {
				return ctx.Skip("status monitor remained in 202 Accepted state during polling window")
			}

			if finalResp.StatusCode != http.StatusOK {
				return ctx.Skip(fmt.Sprintf("status monitor returned terminal status %d; AsyncResult assertion in this test applies when monitor response is 200", finalResp.StatusCode))
			}

			asyncResult := finalResp.Headers.Get("AsyncResult")
			if asyncResult == "" {
				return framework.NewError("final async status monitor response missing AsyncResult header")
			}

			statusCode, err := strconv.Atoi(asyncResult)
			if err != nil {
				return framework.NewError(fmt.Sprintf("AsyncResult header is not an integer status code: %q", asyncResult))
			}
			if statusCode < 100 || statusCode > 599 {
				return framework.NewError(fmt.Sprintf("AsyncResult header out of HTTP status code range: %d", statusCode))
			}

			return nil
		},
	)

	return suite
}
