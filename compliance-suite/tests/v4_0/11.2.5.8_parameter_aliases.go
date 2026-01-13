package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ParameterAliases creates the 11.2.5.8 Parameter Aliases test suite
func ParameterAliases() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.8 Parameter Aliases",
		"Tests parameter alias support in system query options.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ParameterAliases",
	)

	suite.AddTest(
		"test_filter_parameter_alias",
		"$filter supports parameter aliases",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price gt @p&@p=10")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %v", err)
			}

			value, ok := result["value"].([]interface{})
			if !ok {
				return framework.NewError("response missing value array")
			}

			// Success: parameter alias was accepted by server (status 200)
			// The actual filter results depend on data, so we don't assert on count
			_ = value // Acknowledge we have the value
			return nil
		},
	)

	suite.AddTest(
		"test_top_parameter_alias",
		"$top supports parameter aliases",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=@t&@t=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %v", err)
			}

			value, ok := result["value"].([]interface{})
			if !ok {
				return framework.NewError("response missing value array")
			}

			// Verify that $top=@t&@t=1 respected the limit
			if len(value) > 1 {
				return fmt.Errorf("expected at most 1 product with $top=1, got %d", len(value))
			}

			return nil
		},
	)

	return suite
}
