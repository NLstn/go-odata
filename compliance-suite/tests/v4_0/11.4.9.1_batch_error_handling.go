package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// BatchErrorHandling creates the 11.4.9.1 Batch Error Handling test suite
func BatchErrorHandling() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.9.1 Batch Error Handling",
		"Tests error handling in batch requests including changeset atomicity, error responses, and malformed requests.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_BatchRequests",
	)

	getProductSegment := func(ctx *framework.TestContext, index int) (string, error) {
		ids, err := fetchEntityIDs(ctx, "Products", index+1)
		if err != nil {
			return "", err
		}
		if len(ids) <= index {
			return "", fmt.Errorf("need at least %d product(s)", index+1)
		}
		return fmt.Sprintf("Products(%s)", ids[index]), nil
	}

	// Test 1: Batch endpoint responds to malformed boundary
	suite.AddTest(
		"test_malformed_boundary",
		"Malformed batch boundary handled correctly",
		func(ctx *framework.TestContext) error {
			segment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--wrong_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--wrong_boundary--`, segment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 2: Independent requests don't affect each other
	suite.AddTest(
		"test_independent_requests",
		"Independent requests don't affect each other",
		func(ctx *framework.TestContext) error {
			firstSegment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			secondSegment, err := getProductSegment(ctx, 1)
			if err != nil {
				return err
			}
			invalidSegment := nonExistingEntitySegment("Products")
			batchBody := fmt.Sprintf(`--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary--`, firstSegment, invalidSegment, secondSegment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Should have both 200 and 404 responses
			successCount := strings.Count(string(resp.Body), "HTTP/1.1 200")
			notFoundCount := strings.Count(string(resp.Body), "HTTP/1.1 404")

			if successCount < 2 || notFoundCount < 1 {
				return framework.NewError("Expected at least two 200 responses and one 404 response")
			}

			return nil
		},
	)

	// Test 3: Error responses in batch have proper format
	suite.AddTest(
		"test_error_format_in_batch",
		"Error responses in batch have proper format",
		func(ctx *framework.TestContext) error {
			invalidSegment := nonExistingEntitySegment("Products")
			batchBody := fmt.Sprintf(`--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary--`, invalidSegment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Should have error JSON structure or 404
			body := string(resp.Body)
			if strings.Contains(body, `"error"`) || strings.Contains(body, "404") {
				return nil
			}

			return framework.NewError("Expected error payload or 404 marker in batch response")
		},
	)

	return suite
}
