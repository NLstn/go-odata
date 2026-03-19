package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// BatchContentIDReferencing creates the 11.4.9.3 Referencing New Entities compliance
// test suite.
//
// OData v4 spec §11.4.9.3: within a changeset, a request bearing a Content-ID header
// creates an alias "$<contentID>" that subsequent requests in the same changeset may use
// as a URL prefix to refer to the newly created entity.
func BatchContentIDReferencing() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.9.3 Referencing New Entities in a Change Set",
		"Tests $<contentID> URL referencing within $batch changesets per OData v4 spec §11.4.9.3.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ReferencingNewEntities",
	)

	// Helper: return a POST body for a minimal Product.
	postProduct := func(name string) string {
		return fmt.Sprintf(`{"Name":%q,"Price":10.0,"Status":1}`, name)
	}

	// productHeaders are included in batch sub-requests that create Products.
	// The devserver's ODataBeforeCreate hook for Product requires X-User-Role: admin.
	const productHeaders = "Content-Type: application/json\nX-User-Role: admin\n"

	// -----------------------------------------------------------------------
	// Test 1: create entity, then read it back via $<contentID>
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_read_back",
		"Created entity can be read back via $<contentID> in same changeset",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_rb"
			changesetBoundary := "changeset_cid_rb"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

GET /$1 HTTP/1.1


--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID ReadBack Product"),
				changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			if !strings.Contains(respBody, "HTTP/1.1 201") {
				return framework.NewError("expected 201 Created for POST with Content-ID")
			}
			if !strings.Contains(respBody, "CID ReadBack Product") {
				return framework.NewError("GET via $1 did not return the created entity")
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 2: create entity, update via $<contentID>
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_patch",
		"Created entity can be updated via $<contentID> PATCH in same changeset",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_patch"
			changesetBoundary := "changeset_cid_patch"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /$1 HTTP/1.1
Content-Type: application/json
X-User-Role: admin

{"Name":"CID Patched Product","Price":20.0,"Status":1}

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Original Product"),
				changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			if !strings.Contains(respBody, "HTTP/1.1 201") {
				return framework.NewError("expected 201 Created for POST with Content-ID")
			}
			// PATCH returns 200 or 204.
			if !strings.Contains(respBody, "HTTP/1.1 200") &&
				!strings.Contains(respBody, "HTTP/1.1 204") {
				return framework.NewError("expected 200 or 204 for PATCH via $1")
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 3: create entity, delete via $<contentID>
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_delete",
		"Created entity can be deleted via $<contentID> DELETE in same changeset",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_del"
			changesetBoundary := "changeset_cid_del"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

DELETE /$1 HTTP/1.1


--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Delete Product"),
				changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			if !strings.Contains(respBody, "HTTP/1.1 201") {
				return framework.NewError("expected 201 Created for POST with Content-ID")
			}
			if !strings.Contains(respBody, "HTTP/1.1 204") {
				return framework.NewError("expected 204 No Content for DELETE via $1")
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 4: multiple independent Content-IDs in one changeset
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_multiple",
		"Multiple Content-IDs can coexist and be independently resolved",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_multi"
			changesetBoundary := "changeset_cid_multi"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 3

PATCH /$1 HTTP/1.1
Content-Type: application/json
X-User-Role: admin

{"Name":"Multi CID Alpha Updated","Price":10.0,"Status":1}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 4

PATCH /$2 HTTP/1.1
Content-Type: application/json
X-User-Role: admin

{"Name":"Multi CID Beta Updated","Price":10.0,"Status":1}

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("Multi CID Alpha"),
				changesetBoundary, postProduct("Multi CID Beta"),
				changesetBoundary, changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			createdCount := strings.Count(respBody, "HTTP/1.1 201")
			if createdCount != 2 {
				return framework.NewError(
					fmt.Sprintf("expected 2 × 201 Created, got %d", createdCount))
			}
			// Two PATCH responses (200 or 204 each).
			patchSuccess := strings.Count(respBody, "HTTP/1.1 200") +
				strings.Count(respBody, "HTTP/1.1 204")
			if patchSuccess < 2 {
				return framework.NewError(
					fmt.Sprintf("expected 2 successful PATCH responses, got %d", patchSuccess))
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 5: $<contentID> in navigation-property URL segment
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_navigation_property",
		"$<contentID> can be used in a navigation-property URL segment (GET /$1/Descriptions)",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_nav"
			changesetBoundary := "changeset_cid_nav"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

GET /$1/Descriptions HTTP/1.1


--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Nav Product"),
				changesetBoundary,
				changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			if !strings.Contains(respBody, "HTTP/1.1 201") {
				return framework.NewError("expected 201 Created for POST with Content-ID")
			}
			if !strings.Contains(respBody, "HTTP/1.1 200") {
				return framework.NewError(
					fmt.Sprintf("GET via $1/Descriptions did not return 200; body: %s", respBody))
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 6: Content-ID response echo
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_echo_in_response",
		"Content-ID is echoed back in every response MIME part (including $<contentID> requests)",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_echo"
			changesetBoundary := "changeset_cid_echo"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: create-req

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: patch-req

PATCH /$create-req HTTP/1.1
Content-Type: application/json

{"Name":"CID Echo Patched","Price":10.0,"Status":1}

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Echo Product"),
				changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			if !strings.Contains(respBody, "Content-ID: create-req") {
				return framework.NewError("response missing Content-ID: create-req")
			}
			if !strings.Contains(respBody, "Content-ID: patch-req") {
				return framework.NewError("response missing Content-ID: patch-req")
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 7: unresolvable $<contentID> causes failure (and changeset rollback)
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_unresolvable_causes_error",
		"Unresolvable $<contentID> reference causes request failure and changeset rollback",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_unres"
			changesetBoundary := "changeset_cid_unres"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /$99 HTTP/1.1
Content-Type: application/json

{"Name":"Should not apply","Price":10.0,"Status":1}

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Unres Product"),
				changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// The second sub-request must fail (4xx).
			respBody := string(resp.Body)
			parts := strings.Split(respBody, "HTTP/1.1 ")
			// Find the response for the $99 request (second sub-request).
			if len(parts) < 3 {
				return framework.NewError("expected at least 2 sub-responses in batch")
			}
			secondStatus := parts[2]
			statusCode := secondStatus[:3]
			if statusCode[0] != '4' {
				return framework.NewError(
					fmt.Sprintf("expected 4xx for unresolvable $99, got HTTP/1.1 %s", statusCode))
			}
			return nil
		},
	)

	return suite
}
