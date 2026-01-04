package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderContentId creates the 8.2.4 Content-ID Header test suite
func HeaderContentId() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.4 Content-ID Header",
		"Tests Content-ID header handling in batch requests.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderContentID",
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

	// Test 1: Content-ID echoed in single request batch response
	suite.AddTest(
		"test_content_id_echo_single",
		"Content-ID MUST be echoed back in batch response",
		func(ctx *framework.TestContext) error {
			segment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: myRequest1

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

			// Per OData v4 spec section 11.7.4, Content-ID MUST be echoed back
			if !strings.Contains(string(resp.Body), "Content-ID: myRequest1") {
				return framework.NewError("Content-ID MUST be echoed back in batch response per OData v4 spec")
			}

			return nil
		},
	)

	// Test 2: Multiple Content-IDs echoed in batch response
	suite.AddTest(
		"test_content_id_echo_multiple",
		"Multiple Content-IDs MUST be echoed back in batch response",
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
Content-ID: getFirst

GET %s HTTP/1.1
Accept: application/json


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: getSecond

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

			body := string(resp.Body)
			if !strings.Contains(body, "Content-ID: getFirst") {
				return framework.NewError("First Content-ID MUST be echoed back")
			}
			if !strings.Contains(body, "Content-ID: getSecond") {
				return framework.NewError("Second Content-ID MUST be echoed back")
			}

			return nil
		},
	)

	// Test 3: Content-ID echoed in changeset response using PATCH on existing entities
	suite.AddTest(
		"test_content_id_echo_changeset",
		"Content-ID MUST be echoed back in changeset responses",
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
Content-Type: multipart/mixed; boundary=changeset_boundary

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

PATCH %s HTTP/1.1
Content-Type: application/json

{"Description":"Updated via batch Content-ID test 1"}

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH %s HTTP/1.1
Content-Type: application/json

{"Description":"Updated via batch Content-ID test 2"}

--changeset_boundary--

--batch_boundary--`, firstSegment, secondSegment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "Content-ID: 1") {
				return framework.NewError("Content-ID: 1 MUST be echoed back in changeset response")
			}
			if !strings.Contains(body, "Content-ID: 2") {
				return framework.NewError("Content-ID: 2 MUST be echoed back in changeset response")
			}

			return nil
		},
	)

	// Test 4: No Content-ID when not provided
	suite.AddTest(
		"test_no_content_id_when_not_provided",
		"No Content-ID should be present when not provided in request",
		func(ctx *framework.TestContext) error {
			segment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			// Request without Content-ID header
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

			// When no Content-ID is provided, none should be in the response
			if strings.Contains(string(resp.Body), "Content-ID:") {
				return framework.NewError("Content-ID should not be present when not provided in request")
			}

			return nil
		},
	)

	return suite
}
