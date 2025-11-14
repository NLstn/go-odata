package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ComplexFilter creates the 5.2.1 Complex Type Filtering test suite
func ComplexFilter() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.2.1 Complex Type Filtering",
		"Validates that nested complex properties can participate in $filter expressions.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType",
	)

	suite.AddTest(
		"test_filter_nested_complex_property",
		"Filter by nested complex property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingAddress/City eq 'Seattle'")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			// Check response has results with Seattle
			body := string(resp.Body)
			if !strings.Contains(body, `"City":"Seattle"`) {
				return framework.NewError("Filtered response missing expected City value")
			}

			return nil
		},
	)

	return suite
}
