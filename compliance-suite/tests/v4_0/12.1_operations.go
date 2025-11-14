package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// Operations creates the 12.1 Operations (Actions and Functions) test suite
func Operations() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"12.1 Operations",
		"Tests OData operations (actions and functions) including bound and unbound operations, parameter passing, and proper invocation syntax.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Operations",
	)

	var productPath string
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		if productPath != "" {
			return productPath, nil
		}
		path, err := firstEntityPath(ctx, "Products")
		if err != nil {
			return "", err
		}
		productPath = path
		return productPath, nil
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

			// May return 200, 404 (not implemented), or 501
			if resp.StatusCode == 200 {
				return nil
			} else if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("Unbound functions not implemented")
			}

			return framework.NewError("Unexpected status code for unbound function")
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

			// May return 200, 404 (not implemented), or 501
			if resp.StatusCode == 200 {
				return nil
			} else if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("Function parameters not implemented")
			}

			return framework.NewError("Unexpected status code for function with parameters")
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

			// May return 200, 404 (not implemented), or 501
			if resp.StatusCode == 200 {
				return nil
			} else if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("Bound functions not implemented")
			}

			return framework.NewError("Unexpected status code for bound function")
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

			// May return 200, 404 (not implemented), or 501
			if resp.StatusCode == 200 {
				return nil
			} else if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("Collection-bound functions not implemented")
			}

			return framework.NewError("Unexpected status code for collection-bound function")
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

			// May return 200/204, 404 (not implemented), or 501
			if resp.StatusCode == 200 || resp.StatusCode == 204 {
				return nil
			} else if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("Unbound actions not implemented")
			}

			return framework.NewError("Unexpected status code for unbound action")
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

			// May return 200/204, 404 (not implemented), or 501
			if resp.StatusCode == 200 || resp.StatusCode == 204 {
				return nil
			} else if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("Bound actions not implemented")
			}

			return framework.NewError("Unexpected status code for bound action")
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
				return ctx.AssertJSONField(resp, "value")
			} else if resp.StatusCode == 404 || resp.StatusCode == 501 {
				return ctx.Skip("Operations not implemented")
			}

			return framework.NewError("Unexpected status code")
		},
	)

	return suite
}
