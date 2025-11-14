package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// BatchRequests creates the 11.4.9 Batch Requests test suite
func BatchRequests() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.9 Batch Requests",
		"Tests batch request processing according to OData v4 specification.",
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

	// Test 1: $batch endpoint exists
	suite.AddTest(
		"test_batch_endpoint_responds",
		"$batch endpoint responds",
		func(ctx *framework.TestContext) error {
			segment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary--`, segment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: Batch response has multipart/mixed Content-Type
	suite.AddTest(
		"test_batch_response_content_type",
		"Batch response has multipart/mixed Content-Type",
		func(ctx *framework.TestContext) error {
			segment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary--`, segment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(contentType, "multipart/mixed") {
				return framework.NewError("Expected multipart/mixed Content-Type")
			}

			return nil
		},
	)

	// Test 3: Batch with multiple GET requests
	suite.AddTest(
		"test_batch_multiple_gets",
		"Batch with multiple GET requests",
		func(ctx *framework.TestContext) error {
			firstSegment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			secondSegment, err := getProductSegment(ctx, 1)
			if err != nil {
				return err
			}
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


--batch_boundary--`, firstSegment, secondSegment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Check for multiple HTTP responses in body
			responseCount := strings.Count(string(resp.Body), "HTTP/1.1")
			if responseCount < 2 {
				return framework.NewError("Expected at least 2 responses in batch")
			}

			return nil
		},
	)

	// Test 4: Invalid batch request returns 400
	suite.AddTest(
		"test_batch_invalid",
		"Invalid batch request returns 400",
		func(ctx *framework.TestContext) error {
			batchBody := `--batch_boundary
INVALID CONTENT
--batch_boundary--`

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
