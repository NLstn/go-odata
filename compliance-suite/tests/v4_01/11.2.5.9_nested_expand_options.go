package v4_01

import (
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NestedExpandOptions creates the 11.2.5.9 Nested Expand with Query Options test suite for OData v4.01.
func NestedExpandOptions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.9 Nested Expand with Query Options",
		"Validates nested $expand options for $count and $levels in OData v4.01.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptionexpand",
	)

	suite.AddTest(
		"test_expand_with_count_true",
		"Expand with $count=true includes @odata.count annotation",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($count=true)")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var body map[string]interface{}
			if err := ctx.GetJSON(resp, &body); err != nil {
				return err
			}

			entities, ok := body["value"].([]interface{})
			if !ok || len(entities) == 0 {
				return fmt.Errorf("expected non-empty value array in response")
			}

			entity, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected expanded entity to be a JSON object")
			}

			if _, ok := entity["Descriptions@odata.count"]; !ok {
				return fmt.Errorf("missing Descriptions@odata.count annotation")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_expand_with_levels_integer",
		"Expand with $levels=2 returns expanded results",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($levels=2)")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertBodyContains(resp, "Descriptions")
		},
	)

	suite.AddTest(
		"test_expand_with_count_and_levels",
		"Expand with both $count=true and $levels=2 returns annotations",
		func(ctx *framework.TestContext) error {
			expand := url.QueryEscape("Descriptions($count=true;$levels=2)")
			resp, err := ctx.GET("/Products?$top=1&$expand=" + expand)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return ctx.AssertBodyContains(resp, "Descriptions@odata.count")
		},
	)

	return suite
}
