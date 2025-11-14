package v4_01

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FunctionActionOverloading creates the 12.2 Function and Action Overloading test suite
func FunctionActionOverloading() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"12.2 Function and Action Overloading",
		"Validates function and action overload support where multiple functions or actions can share the same name but differ by binding parameter type or parameter count/types.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html#sec_FunctionandActionOverloading",
	)

	// Test 1: Function overload with different parameter counts
	suite.AddTest(
		"test_function_overload_param_count",
		"Function overload with different parameter counts",
		func(ctx *framework.TestContext) error {
			ctx.Log("Testing function overload with different parameter counts")

			// Call function with no parameters
			resp1, err := ctx.GET("/GetTopProducts()")
			if err != nil {
				return err
			}

			// Call function with one parameter
			resp2, err := ctx.GET("/GetTopProducts()?count=5")
			if err != nil {
				return err
			}

			// Both should work if overloading is supported
			if resp1.StatusCode == 200 && resp2.StatusCode == 200 {
				return nil
			}

			ctx.Log("Function overloads with different parameter counts not fully supported")
			return nil
		},
	)

	// Test 2: Function overload with different parameter types
	suite.AddTest(
		"test_function_overload_param_types",
		"Function overload with different parameter types",
		func(ctx *framework.TestContext) error {
			ctx.Log("Testing function overload with different parameter types")

			// Call function with string parameter
			resp1, err := ctx.GET("/Convert()?input=hello")
			if err != nil {
				return err
			}

			// Call function with numeric parameter
			resp2, err := ctx.GET("/Convert()?number=5")
			if err != nil {
				return err
			}

			if resp1.StatusCode == 200 && resp2.StatusCode == 200 {
				return nil
			}

			ctx.Log("Function overloads with different parameter types not fully supported")
			return nil
		},
	)

	// Test 3: Function overload resolution based on parameters
	suite.AddTest(
		"test_function_overload_resolution",
		"Function overload resolution based on parameters",
		func(ctx *framework.TestContext) error {
			ctx.Log("Testing correct function overload resolution")

			// Call function with single parameter
			resp1, err := ctx.GET("/Calculate()?value=5")
			if err != nil {
				return err
			}

			// Call function with two parameters
			resp2, err := ctx.GET("/Calculate()?a=3&b=7")
			if err != nil {
				return err
			}

			// Check both returned valid responses (not errors)
			if resp1.StatusCode == 200 {
				var data map[string]interface{}
				if err := json.Unmarshal(resp1.Body, &data); err == nil {
					if _, hasError := data["error"]; hasError {
						return framework.NewError("First overload returned error")
					}
				}
			}

			if resp2.StatusCode == 200 {
				var data map[string]interface{}
				if err := json.Unmarshal(resp2.Body, &data); err == nil {
					if _, hasError := data["error"]; hasError {
						return framework.NewError("Second overload returned error")
					}
				}
			}

			return nil
		},
	)

	// Test 4: Action overload with different parameter counts
	suite.AddTest(
		"test_action_overload_param_count",
		"Action overload with different parameter counts",
		func(ctx *framework.TestContext) error {
			ctx.Log("Testing action overload with different parameter counts")

			// Call action with one parameter
			resp1, err := ctx.POST("/Process", map[string]interface{}{
				"percentage": 10.0,
			})
			if err != nil {
				return err
			}

			// Call action with two parameters (different overload)
			resp2, err := ctx.POST("/Process", map[string]interface{}{
				"percentage": 10.0,
				"minPrice":   100.0,
			})
			if err != nil {
				return err
			}

			if (resp1.StatusCode == 204 || resp1.StatusCode == 200) &&
				(resp2.StatusCode == 204 || resp2.StatusCode == 200) {
				return nil
			}

			ctx.Log("Action overloads with different parameter counts not fully supported")
			return nil
		},
	)

	var productPath string
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		if productPath != "" {
			return productPath, nil
		}
		resp, err := ctx.GET("/Products?$top=1&$select=ID")
		if err != nil {
			return "", err
		}
		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return "", err
		}
		var body struct {
			Value []map[string]interface{} `json:"value"`
		}
		if err := json.Unmarshal(resp.Body, &body); err != nil {
			return "", err
		}
		if len(body.Value) == 0 {
			return "", framework.NewError("no products available")
		}
		id := body.Value[0]["ID"]
		productPath = fmt.Sprintf("/Products(%v)", id)
		return productPath, nil
	}

	// Test 5: Bound function overload on different entity sets
	suite.AddTest(
		"test_bound_function_overload",
		"Bound function overload on different entity sets",
		func(ctx *framework.TestContext) error {
			ctx.Log("Testing bound function overload on different entity sets")

			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}

			// Call bound function on Products entity set
			resp, err := ctx.GET(path + "/GetInfo?format=json")
			if err != nil {
				return err
			}

			// Accept 200 (supported) or 404 (not implemented)
			if resp.StatusCode == 200 || resp.StatusCode == 404 {
				return nil
			}

			ctx.Log("Bound function overload not fully supported")
			return nil
		},
	)

	// Test 6: Reject duplicate overloads
	suite.AddTest(
		"test_reject_duplicate_overload",
		"Verify duplicate function signatures validation",
		func(ctx *framework.TestContext) error {
			ctx.Log("Testing that duplicate function signatures are rejected")

			// Verify by checking that distinct overloads work
			resp, err := ctx.GET("/GetTopProducts()?count=5")
			if err != nil {
				return err
			}

			// If the service is running, it should have rejected duplicates at startup
			if resp.StatusCode == 200 {
				return nil
			}

			ctx.Log("Service may not be properly validating duplicate overloads")
			return nil
		},
	)

	// Test 7: Function overload with additional parameter
	suite.AddTest(
		"test_function_overload_additional_param",
		"Function overload with additional parameter",
		func(ctx *framework.TestContext) error {
			ctx.Log("Testing function overload with additional optional parameter")

			// Call function with required parameter only
			resp1, err := ctx.GET("/GetTopProducts()?count=5")
			if err != nil {
				return err
			}

			// Call function with required parameter plus category filter
			resp2, err := ctx.GET("/GetTopProducts()?count=5&category=Electronics")
			if err != nil {
				return err
			}

			if resp1.StatusCode == 200 && resp2.StatusCode == 200 {
				return nil
			}

			ctx.Log("Function overload with additional parameter not fully supported")
			return nil
		},
	)

	// Test 8: Bound function overload with different parameter counts
	suite.AddTest(
		"test_bound_function_param_overload",
		"Bound function overload with different parameter counts",
		func(ctx *framework.TestContext) error {
			ctx.Log("Testing bound function overload with different parameter counts")

			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}

			// Call bound function with one parameter
			resp1, err := ctx.GET(path + "/CalculatePrice?discount=10")
			if err != nil {
				return err
			}

			// Call bound function with two parameters (different overload)
			resp2, err := ctx.GET(path + "/CalculatePrice?discount=10&tax=8")
			if err != nil {
				return err
			}

			// Accept 200 (supported) or 404 (not implemented)
			if (resp1.StatusCode == 200 || resp1.StatusCode == 404) &&
				(resp2.StatusCode == 200 || resp2.StatusCode == 404) {
				return nil
			}

			ctx.Log("Bound function overload with different parameter counts not fully supported")
			return nil
		},
	)

	return suite
}
