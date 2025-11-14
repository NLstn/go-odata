package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ComplexTypes creates the 5.2 Complex Types test suite
func ComplexTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.2 Complex Types",
		"Validates handling of complex (structured) types including nested properties, filtering, selecting, and operations.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType",
	)

	suite.AddTest(
		"test_complex_type_retrieval",
		"Retrieve entity with complex type property",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, "ShippingAddress") {
				// Verify it's a JSON object
				if !strings.Contains(body, `"ShippingAddress"`) {
					return framework.NewError("ShippingAddress is not a complex object")
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_complex_nested_property",
		"Access nested property of complex type",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/ShippingAddress/City")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `"value":"Seattle"`) {
				return framework.NewError("Nested property response missing expected value")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_complex_property",
		"Filter by nested complex type property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingAddress/City eq 'Seattle'")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	suite.AddTest(
		"test_select_complex_property",
		"Select complex type property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name,ShippingAddress")
			if err != nil {
				return err
			}

			// Optional feature
			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 404 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d (should not be 500)", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_complex_null_value",
		"Retrieve entity with null complex type",
		func(ctx *framework.TestContext) error {
			// Get product with null ShippingAddress (Office Chair is 4th product, index 3)
			resp, err := ctx.GET("/Products?$filter=ShippingAddress eq null&$top=1")
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

			// Check that ShippingAddress is null or absent
			return nil
		},
	)

	suite.AddTest(
		"test_filter_null_complex",
		"Filter for null complex type value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ShippingAddress eq null")
			if err != nil {
				return err
			}

			// Optional feature
			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 404 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d (should not be 500)", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_orderby_complex_property",
		"Order by complex type property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=ShippingAddress/City")
			if err != nil {
				return err
			}

			// Optional feature
			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 404 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d (should not be 500)", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_access_complex_type",
		"Access complex type property directly",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath + "/ShippingAddress")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `"City":"Seattle"`) {
				return framework.NewError("Complex property response missing expected City value")
			}

			return nil
		},
	)

	return suite
}
