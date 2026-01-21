package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ActionFunctionParameters creates the 11.4.13 Action and Function Parameter Validation test suite
func ActionFunctionParameters() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.13 Action/Function Parameters",
		"Tests parameter validation for actions and functions, including required parameters, type validation, and error handling.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Operations",
	)

	// Test 1: Unbound function with valid parameters
	suite.AddTest(
		"test_function_valid_params",
		"Unbound function with valid parameters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/FindProducts(name='Laptop',maxPrice=1000)")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Helper function to get a fresh product path for each test
	// Note: Must refetch on each call because database is reseeded between tests
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		return firstEntityPath(ctx, "Products")
	}
	invalidProductPath := nonExistingEntityPath("Products")

	// Test 2: Bound function with valid parameters
	suite.AddTest(
		"test_bound_function_valid_params",
		"Bound function with valid parameters",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path + "/GetTotalPrice(taxRate=0.08)")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 3: Bound function on non-existent entity should fail
	suite.AddTest(
		"test_bound_function_invalid_entity",
		"Bound function on invalid entity fails",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath + "/GetTotalPrice(taxRate=0.08)")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 404)
		},
	)

	// Test 4: Bound action with valid parameters
	suite.AddTest(
		"test_action_valid_params",
		"Bound action with valid parameters",
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

			// Should return 200 or 204
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return ctx.AssertStatusCode(resp, 204)
			}

			return nil
		},
	)

	// Test 5: Action with missing required parameter should fail
	suite.AddTest(
		"test_action_missing_param",
		"Action without required parameter fails",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			payload := map[string]interface{}{}

			resp, err := ctx.POST(path+"/ApplyDiscount", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 400); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "error")
		},
	)

	// Test 6: Action with invalid parameter type should fail
	suite.AddTest(
		"test_action_invalid_param_type",
		"Action with invalid parameter type fails",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"percentage": "invalid",
			}

			resp, err := ctx.POST(path+"/ApplyDiscount", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 400); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "error")
		},
	)

	return suite
}
