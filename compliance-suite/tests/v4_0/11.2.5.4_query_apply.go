package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QueryApply creates the 11.2.5.4 System Query Option $apply test suite
func QueryApply() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.4 System Query Option $apply",
		"Tests $apply query option for data aggregation according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html",
	)

	// Test 1: Basic aggregate transformation
	suite.AddTest(
		"test_apply_aggregate_count",
		"$apply with aggregate (count)",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("aggregate($count as Total)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var body struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse aggregate response: %w", err)
			}
			if len(body.Value) == 0 {
				return framework.NewError("Aggregate response should contain at least one result")
			}
			if _, ok := body.Value[0]["Total"]; !ok {
				return framework.NewError("Aggregate response must include Total field")
			}

			return nil
		},
	)

	// Test 2: groupby transformation
	suite.AddTest(
		"test_apply_groupby",
		"$apply with groupby",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("groupby((CategoryID))")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Check for CategoryID field in response
			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return err
			}

			return nil
		},
	)

	// Test 3: groupby with aggregate
	suite.AddTest(
		"test_apply_groupby_aggregate",
		"$apply with groupby and aggregate",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("groupby((CategoryID),aggregate($count as Count))")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			// Check for value array
			if err := ctx.AssertJSONField(resp, "value"); err != nil {
				return err
			}

			return nil
		},
	)

	// Test 4: filter transformation
	suite.AddTest(
		"test_apply_filter",
		"$apply with filter transformation",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("filter(Price gt 10)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	// Test 5: Invalid $apply expression should return 400
	suite.AddTest(
		"test_apply_invalid",
		"Invalid $apply expression returns 400",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("invalid(syntax)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 400); err != nil {
				return err
			}

			return nil
		},
	)

	return suite
}
