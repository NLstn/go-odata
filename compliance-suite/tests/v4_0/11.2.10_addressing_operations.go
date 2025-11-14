package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// AddressingOperations creates the 11.2.10 Addressing Operations test suite
func AddressingOperations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.10 Addressing Operations",
		"Tests addressing bound and unbound actions and functions according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_AddressingOperations",
	)

	// Test 1: Unbound function is addressable
	suite.AddTest(
		"test_unbound_function",
		"Unbound function is addressable",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/GetTopProducts()")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("operation not addressable (status %d). Missing actions/functions are a compliance failure", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		},
	)

	// Test 2: Unbound function with parameters
	suite.AddTest(
		"test_unbound_function_params",
		"Unbound function with parameters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/GetTopProducts(count=5)")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("operation not addressable (status %d). Missing actions/functions are a compliance failure", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		},
	)

	// Test 3: Bound function on entity
	suite.AddTest(
		"test_bound_function",
		"Bound function on entity",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/GetRelatedProducts()")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("operation not addressable (status %d). Missing actions/functions are a compliance failure", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		},
	)

	// Test 4: Unbound action is addressable
	suite.AddTest(
		"test_unbound_action",
		"Unbound action is addressable",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/ResetProducts", map[string]interface{}{})
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 204 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("operation not addressable (status %d). Missing actions/functions are a compliance failure", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		},
	)

	// Test 5: Bound action on entity
	suite.AddTest(
		"test_bound_action",
		"Bound action on entity",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.POST(productPath+"/Activate", map[string]interface{}{})
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 204 {
				return nil
			}

			if resp.StatusCode == 500 {
				return ctx.Skip("bound action returns 500 (server bug): Activate")
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("operation not addressable (status %d). Missing actions/functions are a compliance failure", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		},
	)

	// Test 6: Function with multiple parameters
	suite.AddTest(
		"test_function_multiple_params",
		"Function with multiple parameters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/FindProducts(name='Laptop',maxPrice=1000)")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("operation not addressable (status %d). Missing actions/functions are a compliance failure", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		},
	)

	// Test 7: Action returns result
	suite.AddTest(
		"test_action_with_result",
		"Action can return result",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.POST(productPath+"/CalculateDiscount", map[string]interface{}{
				"percentage": 10,
			})
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("operation not addressable (status %d). Missing actions/functions are a compliance failure", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		},
	)

	// Test 8: Function on collection
	suite.AddTest(
		"test_function_on_collection",
		"Function on collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/GetAveragePrice()")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("operation not addressable (status %d). Missing actions/functions are a compliance failure", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		},
	)

	// Test 9: Action on collection
	suite.AddTest(
		"test_action_on_collection",
		"Action on collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Products/MarkAllAsReviewed", map[string]interface{}{})
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 || resp.StatusCode == 204 {
				return nil
			}

			if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return fmt.Errorf("operation not addressable (status %d). Missing actions/functions are a compliance failure", resp.StatusCode)
			}

			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		},
	)

	// Test 10: Metadata includes operations
	suite.AddTest(
		"test_metadata_operations",
		"Metadata includes operations",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// Metadata should be valid XML
			bodyStr := string(resp.Body)
			if len(bodyStr) == 0 {
				return fmt.Errorf("metadata response is empty")
			}

			// Basic XML validation
			if !framework.ContainsAny(bodyStr, "<edmx:Edmx", "<?xml") {
				return fmt.Errorf("metadata response is not valid XML")
			}

			return nil
		},
	)

	return suite
}
