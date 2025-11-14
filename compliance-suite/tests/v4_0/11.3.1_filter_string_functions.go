package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FilterStringFunctions creates the 11.3.1 String Functions test suite
func FilterStringFunctions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.1 String Functions in $filter",
		"Tests string functions (contains, startswith, endswith, length, etc.) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations",
	)

	// Test 1: contains function
	suite.AddTest(
		"test_contains_function",
		"contains() function filters string values",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("contains(Name,'Laptop')")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' property")
			}

			return nil
		},
	)

	// Test 2: startswith function
	suite.AddTest(
		"test_startswith_function",
		"startswith() function filters by prefix",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("startswith(Name,'Gaming')")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' property")
			}

			return nil
		},
	)

	// Test 3: endswith function
	suite.AddTest(
		"test_endswith_function",
		"endswith() function filters by suffix",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("endswith(Name,'Mouse')")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' property")
			}

			return nil
		},
	)

	// Test 4: length function
	suite.AddTest(
		"test_length_function",
		"length() function returns string length",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("length(Name) gt 10")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' property")
			}

			return nil
		},
	)

	// Test 5: indexof function
	suite.AddTest(
		"test_indexof_function",
		"indexof() function finds substring position",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("indexof(Name,'Pro') eq 0")
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

	// Test 6: substring function
	suite.AddTest(
		"test_substring_function",
		"substring() function extracts substring",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("substring(Name,0,3) eq 'Gam'")
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

	// Test 7: tolower function
	suite.AddTest(
		"test_tolower_function",
		"tolower() function converts to lowercase",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("tolower(Name) eq 'laptop'")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' property")
			}

			return nil
		},
	)

	// Test 8: toupper function
	suite.AddTest(
		"test_toupper_function",
		"toupper() function converts to uppercase",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("toupper(Name) eq 'LAPTOP'")
			resp, err := ctx.GET("/Products?$filter=" + filter)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["value"]; !ok {
				return fmt.Errorf("response missing 'value' property")
			}

			return nil
		},
	)

	// Test 9: trim function
	suite.AddTest(
		"test_trim_function",
		"trim() function removes whitespace",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("trim(Name) eq 'Laptop'")
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

	// Test 10: concat function
	suite.AddTest(
		"test_concat_function",
		"concat() function concatenates strings",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("concat(Name,' Test') eq 'Laptop Test'")
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
