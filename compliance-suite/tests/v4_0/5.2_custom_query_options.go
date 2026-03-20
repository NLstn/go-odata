package v4_0

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// CustomQueryOptions creates the 5.2 Custom Query Options test suite.
func CustomQueryOptions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.2 Custom Query Options",
		"Validates that custom query options (not starting with $ or @) are accepted and do not interfere with system query option processing.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#_Custom_Query_Options",
	)

	suite.AddTest(
		"test_custom_query_option_is_ignored_not_rejected",
		"custom query option names not starting with $ or @ are accepted",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?debug-mode=true&$top=2&$orderby=ID")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return framework.NewError(fmt.Sprintf("custom query option should not be rejected: %v", err))
			}

			return ctx.AssertJSONField(resp, "value")
		},
	)

	suite.AddTest(
		"test_custom_query_option_does_not_change_semantics",
		"custom query options are ignored by protocol semantics and do not change selected result set",
		func(ctx *framework.TestContext) error {
			canonical, err := ctx.GET("/Products?$top=3&$orderby=ID&$select=ID")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(canonical, http.StatusOK); err != nil {
				return err
			}

			variant, err := ctx.GET("/Products?$top=3&$orderby=ID&$select=ID&trace-id=abc123")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(variant, http.StatusOK); err != nil {
				return err
			}

			extractIDs := func(body []byte) ([]string, error) {
				var payload struct {
					Value []map[string]interface{} `json:"value"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					return nil, err
				}
				ids := make([]string, 0, len(payload.Value))
				for i, entity := range payload.Value {
					id, ok := entity["ID"]
					if !ok {
						return nil, fmt.Errorf("entity %d missing ID field", i)
					}
					ids = append(ids, fmt.Sprintf("%v", id))
				}
				sort.Strings(ids)
				return ids, nil
			}

			canonicalIDs, err := extractIDs(canonical.Body)
			if err != nil {
				return framework.NewError(fmt.Sprintf("canonical response is not valid JSON: %v", err))
			}

			variantIDs, err := extractIDs(variant.Body)
			if err != nil {
				return framework.NewError(fmt.Sprintf("variant response is not valid JSON: %v", err))
			}

			if len(canonicalIDs) != len(variantIDs) {
				return framework.NewError(fmt.Sprintf("custom query option changed result count: canonical=%d variant=%d", len(canonicalIDs), len(variantIDs)))
			}

			for i := range canonicalIDs {
				if canonicalIDs[i] != variantIDs[i] {
					return framework.NewError("custom query option changed selected entities")
				}
			}

			return nil
		},
	)

	return suite
}
