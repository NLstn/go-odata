package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// Operations creates the 12.1 Operations (Actions and Functions) test suite
func Operations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"12.1 Operations",
		"Tests OData operations (actions and functions) including bound and unbound operations, parameter passing, and proper invocation syntax.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Operations",
	)

	// Helper function to get product path for each test
	// Note: Must refetch on each call because database is reseeded between tests
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		return firstEntityPath(ctx, "Products")
	}

	// Test 1: Unbound function invocation
	suite.AddTest(
		"test_unbound_function",
		"Unbound function invocation",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/GetTopProducts()")
			if err != nil {
				return err
			}

			// If operation exists in metadata, must work properly
			if resp.StatusCode == 200 {
				// Validate response is valid OData collection or entity
				return ctx.AssertJSONField(resp, "value")
			}

			// 404 indicates function not defined (acceptable if not in metadata)
			if resp.StatusCode == 404 {
				return framework.NewError("Unbound function not defined in service")
			}

			return fmt.Errorf("Unexpected status code %d for unbound function", resp.StatusCode)
		},
	)

	// Test 2: Unbound function with parameters
	suite.AddTest(
		"test_unbound_function_parameters",
		"Unbound function with parameters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/GetTopProducts(count=3)")
			if err != nil {
				return err
			}

			// If function exists, parameters must be handled correctly
			if resp.StatusCode == 200 {
				return ctx.AssertJSONField(resp, "value")
			}

			// 404 indicates function not defined
			if resp.StatusCode == 404 {
				return framework.NewError("Unbound function with parameters not defined in service")
			}

			return fmt.Errorf("Unexpected status code %d for function with parameters", resp.StatusCode)
		},
	)

	// Test 3: Bound function on entity
	suite.AddTest(
		"test_bound_function",
		"Bound function on entity",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path + "/GetTotalPrice(taxRate=0.08)")
			if err != nil {
				return err
			}

			// If bound function exists, must return valid result
			if resp.StatusCode == 200 {
				// Should return a value
				if len(resp.Body) == 0 {
					return framework.NewError("Bound function returned empty body")
				}
				return nil
			}

			// 404 indicates function not bound to this entity type
			if resp.StatusCode == 404 {
				return framework.NewError("Bound function not defined for this entity type")
			}

			return fmt.Errorf("Unexpected status code %d for bound function", resp.StatusCode)
		},
	)

	// Test 4: Bound function on collection
	suite.AddTest(
		"test_bound_function_collection",
		"Bound function on collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/GetAveragePrice()")
			if err != nil {
				return err
			}

			// If function exists, must work correctly
			if resp.StatusCode == 200 {
				if len(resp.Body) == 0 {
					return framework.NewError("Collection-bound function returned empty body")
				}
				return nil
			}

			// 404 indicates function not bound to this collection
			if resp.StatusCode == 404 {
				return framework.NewError("Collection-bound function not defined for Products")
			}

			return fmt.Errorf("Unexpected status code %d for collection-bound function", resp.StatusCode)
		},
	)

	// Test 5: Unbound action invocation
	suite.AddTest(
		"test_unbound_action",
		"Unbound action invocation",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{}

			resp, err := ctx.POST("/ResetProducts", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// Actions must return success when invoked
			if resp.StatusCode == 200 || resp.StatusCode == 204 {
				return nil
			}

			// 404 indicates action not defined
			if resp.StatusCode == 404 {
				return framework.NewError("Unbound action not defined in service")
			}

			return fmt.Errorf("Unexpected status code %d for unbound action", resp.StatusCode)
		},
	)

	// Test 6: Bound action on entity
	suite.AddTest(
		"test_bound_action",
		"Bound action on entity",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"percentage": 10,
			}

			resp, err := ctx.POST(path+"/ApplyDiscount", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// Bound actions must work when defined
			if resp.StatusCode == 200 || resp.StatusCode == 204 {
				return nil
			}

			// 404 indicates action not bound to this entity type
			if resp.StatusCode == 404 {
				return framework.NewError("Bound action not defined for this entity type")
			}

			return fmt.Errorf("Unexpected status code %d for bound action", resp.StatusCode)
		},
	)

	// Test 7: Operation returns collection
	suite.AddTest(
		"test_operation_returns_collection",
		"Operation returns collection",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/GetTopProducts()")
			if err != nil {
				return err
			}

			if resp.StatusCode == 200 {
				// Must return valid OData collection format
				if err := ctx.AssertJSONField(resp, "value"); err != nil {
					return fmt.Errorf("Operation returning collection must include 'value' array: %v", err)
				}
				return nil
			}

			// 404 indicates operation not defined
			if resp.StatusCode == 404 {
				return framework.NewError("Operation not defined in service")
			}

			return fmt.Errorf("Unexpected status code %d", resp.StatusCode)
		},
	)

	return suite
}
