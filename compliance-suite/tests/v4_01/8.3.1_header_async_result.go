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
		"4.01 asynchronous final responses include AsyncResult with the terminal status code",
		func(ctx *framework.TestContext) error {
			initialResp, err := startAsyncRequest(ctx, "4.01")
			if err != nil {
				return err
			}

			if initialResp.StatusCode == http.StatusOK {
				return ctx.Skip("service chose synchronous processing for this request; AsyncResult requirement applies only when async processing is used")
			}

			if err := ctx.AssertStatusCode(initialResp, http.StatusAccepted); err != nil {
				return framework.NewError(fmt.Sprintf("expected 202 for async processing or 200 for sync, got %d", initialResp.StatusCode))
			}

			finalResp, err := pollAsyncMonitor(ctx, initialResp.Headers.Get("Location"), "4.01")
			if err != nil {
				return err
			}

			if finalResp.StatusCode == http.StatusAccepted {
				return ctx.Skip("status monitor remained in 202 Accepted state during polling window")
			}

			asyncResult := finalResp.Headers.Get("AsyncResult")
			if asyncResult == "" {
				return framework.NewError("final async status monitor response missing AsyncResult header")
			}

			statusCode, err := strconv.Atoi(asyncResult)
			if err != nil {
				return framework.NewError(fmt.Sprintf("AsyncResult header is not an integer status code: %q", asyncResult))
			}
			if statusCode != finalResp.StatusCode {
				return framework.NewError(fmt.Sprintf("AsyncResult header %d does not match terminal HTTP status %d", statusCode, finalResp.StatusCode))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_negotiated_4_0_async_response_does_not_include_async_result",
		"4.0-negotiated asynchronous final responses do not include the 4.01-only AsyncResult header",
		func(ctx *framework.TestContext) error {
			initialResp, err := startAsyncRequest(ctx, "4.0")
			if err != nil {
				return err
			}

			if initialResp.StatusCode == http.StatusOK {
				return ctx.Skip("service chose synchronous processing for this request; fallback assertion applies only when async processing is used")
			}

			if err := ctx.AssertStatusCode(initialResp, http.StatusAccepted); err != nil {
				return framework.NewError(fmt.Sprintf("expected 202 for async processing or 200 for sync, got %d", initialResp.StatusCode))
			}

			finalResp, err := pollAsyncMonitor(ctx, initialResp.Headers.Get("Location"), "4.0")
			if err != nil {
				return err
			}

			if finalResp.StatusCode == http.StatusAccepted {
				return ctx.Skip("status monitor remained in 202 Accepted state during polling window")
			}

			if got := finalResp.Headers.Get("AsyncResult"); got != "" {
				return framework.NewError(fmt.Sprintf("expected no AsyncResult header for 4.0-negotiated async response, got %q", got))
			}

			if version := strings.TrimSpace(finalResp.Headers.Get("OData-Version")); version != "4.0" {
				return framework.NewError(fmt.Sprintf("expected terminal async response to remain negotiated to OData-Version 4.0, got %q", version))
			}

			return nil
		},
	)

	return suite
}

func startAsyncRequest(ctx *framework.TestContext, maxVersion string) (*framework.HTTPResponse, error) {
	var lastResp *framework.HTTPResponse
	for i := 0; i < 5; i++ {
		resp, err := ctx.GET(
			"/Products?$top=1",
			framework.Header{Key: "Prefer", Value: "respond-async"},
			framework.Header{Key: "OData-MaxVersion", Value: maxVersion},
			framework.Header{Key: "Accept", Value: "application/json"},
		)
		if err != nil {
			return nil, err
		}

		lastResp = resp
		if resp.StatusCode == http.StatusAccepted {
			return resp, nil
		}
	}

	if lastResp == nil {
		return nil, framework.NewError("failed to obtain response when requesting async processing")
	}

	return lastResp, nil
}

func pollAsyncMonitor(ctx *framework.TestContext, location, maxVersion string) (*framework.HTTPResponse, error) {
	if location == "" {
		return nil, framework.NewError("202 response missing Location header for status monitor resource")
	}

	statusPath := location
	if strings.HasPrefix(statusPath, ctx.ServerURL()) {
		statusPath = strings.TrimPrefix(statusPath, ctx.ServerURL())
	}
	if !strings.HasPrefix(statusPath, "/") {
		statusPath = "/" + strings.TrimPrefix(statusPath, "/")
	}

	var finalResp *framework.HTTPResponse
	var err error
	for i := 0; i < 10; i++ {
		finalResp, err = ctx.GET(
			statusPath,
			framework.Header{Key: "OData-MaxVersion", Value: maxVersion},
			framework.Header{Key: "Accept", Value: "application/json"},
		)
		if err != nil {
			return nil, err
		}
		if finalResp.StatusCode != http.StatusAccepted {
			return finalResp, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	if finalResp == nil {
		return nil, framework.NewError("failed to retrieve status monitor response")
	}

	return finalResp, nil
}
