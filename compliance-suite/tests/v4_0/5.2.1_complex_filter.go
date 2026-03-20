package v4_0

import (
	"fmt"

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

			items, err := ctx.ParseEntityCollection(resp)
			if err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
				return err
			}

			return ctx.AssertAllEntitiesSatisfy(items, "ShippingAddress/City eq 'Seattle'", func(entity map[string]interface{}) (bool, string) {
				addressRaw, ok := entity["ShippingAddress"]
				if !ok || addressRaw == nil {
					return false, "ShippingAddress is missing or null"
				}
				address, ok := addressRaw.(map[string]interface{})
				if !ok {
					return false, fmt.Sprintf("ShippingAddress has unexpected type %T", addressRaw)
				}
				city, ok := address["City"].(string)
				if !ok {
					return false, fmt.Sprintf("City has unexpected type %T", address["City"])
				}
				if city != "Seattle" {
					return false, fmt.Sprintf("City=%q", city)
				}
				return true, ""
			})
		},
	)

	suite.AddTest(
		"test_filter_invalid_nested_complex_property",
		"Invalid nested complex property paths return 400 with an OData error payload",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingAddress/Unknown eq 'Seattle'")
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, 400, "ShippingAddress/Unknown")
		},
	)

	return suite
}
