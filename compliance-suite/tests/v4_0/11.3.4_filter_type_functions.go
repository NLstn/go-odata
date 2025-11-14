package v4_0

import (
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FilterTypeFunctions creates the 11.3.4 Type Functions test suite
func FilterTypeFunctions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.4 Type Functions in $filter",
		"Tests type checking and casting functions (isof, cast) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations",
	)

	// Test 1: isof function with property
	suite.AddTest(
		"test_isof_function_property",
		"isof() function checks property type",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("isof(Price,Edm.Decimal)")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 2: isof function with entity type
	suite.AddTest(
		"test_isof_function_entity",
		"isof() function checks entity type",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("isof('Product')")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 3: cast function
	suite.AddTest(
		"test_cast_function",
		"cast() function casts to specified type",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("cast(Status,Edm.String) eq '1'")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 4: isof with null check
	suite.AddTest(
		"test_isof_null_check",
		"isof() with null check returns true",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("isof(Name,Edm.String) eq true")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 5: Negative isof test
	suite.AddTest(
		"test_isof_negative",
		"isof() returns false for wrong type",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("isof(Price,Edm.String) eq false")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	return suite
}
