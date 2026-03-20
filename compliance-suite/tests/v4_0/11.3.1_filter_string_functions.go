package v4_0

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

func fetchStringFilterItems(ctx *framework.TestContext, filterExpr string) ([]map[string]interface{}, error) {
	filter := url.QueryEscape(filterExpr)
	resp, err := ctx.GET("/Products?$filter=" + filter)
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}

	return ctx.ParseEntityCollection(resp)
}

func productName(item map[string]interface{}) (string, error) {
	name, ok := item["Name"].(string)
	if !ok {
		return "", fmt.Errorf("item missing Name field or Name is not a string")
	}
	return name, nil
}

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
			items, err := fetchStringFilterItems(ctx, "contains(Name,'Laptop')")
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("contains filter returned no items")
			}
			for i, item := range items {
				name, err := productName(item)
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				if !strings.Contains(name, "Laptop") {
					return fmt.Errorf("item %d has Name=%q which does not satisfy contains(Name,'Laptop')", i, name)
				}
			}

			return nil
		},
	)

	// Test 2: startswith function
	suite.AddTest(
		"test_startswith_function",
		"startswith() function filters by prefix",
		func(ctx *framework.TestContext) error {
			items, err := fetchStringFilterItems(ctx, "startswith(Name,'Wireless')")
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("startswith filter returned no items")
			}
			for i, item := range items {
				name, err := productName(item)
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				if !strings.HasPrefix(name, "Wireless") {
					return fmt.Errorf("item %d has Name=%q which does not satisfy startswith(Name,'Wireless')", i, name)
				}
			}

			return nil
		},
	)

	// Test 3: endswith function
	suite.AddTest(
		"test_endswith_function",
		"endswith() function filters by suffix",
		func(ctx *framework.TestContext) error {
			items, err := fetchStringFilterItems(ctx, "endswith(Name,'Mouse')")
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("endswith filter returned no items")
			}
			for i, item := range items {
				name, err := productName(item)
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				if !strings.HasSuffix(name, "Mouse") {
					return fmt.Errorf("item %d has Name=%q which does not satisfy endswith(Name,'Mouse')", i, name)
				}
			}

			return nil
		},
	)

	// Test 4: length function
	suite.AddTest(
		"test_length_function",
		"length() function returns string length",
		func(ctx *framework.TestContext) error {
			items, err := fetchStringFilterItems(ctx, "length(Name) gt 10")
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("length filter returned no items")
			}
			for i, item := range items {
				name, err := productName(item)
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				if len(name) <= 10 {
					return fmt.Errorf("item %d has Name=%q with length=%d, expected > 10", i, name, len(name))
				}
			}

			return nil
		},
	)

	// Test 5: indexof function
	suite.AddTest(
		"test_indexof_function",
		"indexof() function finds substring position",
		func(ctx *framework.TestContext) error {
			items, err := fetchStringFilterItems(ctx, "indexof(Name,'Lap') eq 0")
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("indexof filter returned no items")
			}
			for i, item := range items {
				name, err := productName(item)
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				if !strings.HasPrefix(name, "Lap") {
					return fmt.Errorf("item %d has Name=%q which does not satisfy indexof(Name,'Lap') eq 0", i, name)
				}
			}
			return nil
		},
	)

	// Test 6: substring function
	suite.AddTest(
		"test_substring_function",
		"substring() function extracts substring",
		func(ctx *framework.TestContext) error {
			items, err := fetchStringFilterItems(ctx, "substring(Name,0,4) eq 'Lapt'")
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("substring filter returned no items")
			}
			for i, item := range items {
				name, err := productName(item)
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				if len(name) < 4 || name[:4] != "Lapt" {
					return fmt.Errorf("item %d has Name=%q which does not satisfy substring(Name,0,4) eq 'Lapt'", i, name)
				}
			}
			return nil
		},
	)

	// Test 7: tolower function
	suite.AddTest(
		"test_tolower_function",
		"tolower() function converts to lowercase",
		func(ctx *framework.TestContext) error {
			items, err := fetchStringFilterItems(ctx, "tolower(Name) eq 'laptop'")
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("tolower filter returned no items")
			}
			for i, item := range items {
				name, err := productName(item)
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				if strings.ToLower(name) != "laptop" {
					return fmt.Errorf("item %d has Name=%q which does not satisfy tolower(Name) eq 'laptop'", i, name)
				}
			}

			return nil
		},
	)

	// Test 8: toupper function
	suite.AddTest(
		"test_toupper_function",
		"toupper() function converts to uppercase",
		func(ctx *framework.TestContext) error {
			items, err := fetchStringFilterItems(ctx, "toupper(Name) eq 'LAPTOP'")
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("toupper filter returned no items")
			}
			for i, item := range items {
				name, err := productName(item)
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				if strings.ToUpper(name) != "LAPTOP" {
					return fmt.Errorf("item %d has Name=%q which does not satisfy toupper(Name) eq 'LAPTOP'", i, name)
				}
			}

			return nil
		},
	)

	// Test 9: trim function
	suite.AddTest(
		"test_trim_function",
		"trim() function removes whitespace",
		func(ctx *framework.TestContext) error {
			items, err := fetchStringFilterItems(ctx, "trim(Name) eq 'Laptop'")
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("trim filter returned no items")
			}
			for i, item := range items {
				name, err := productName(item)
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				if strings.TrimSpace(name) != "Laptop" {
					return fmt.Errorf("item %d has Name=%q which does not satisfy trim(Name) eq 'Laptop'", i, name)
				}
			}
			return nil
		},
	)

	// Test 10: concat function
	suite.AddTest(
		"test_concat_function",
		"concat() function concatenates strings",
		func(ctx *framework.TestContext) error {
			items, err := fetchStringFilterItems(ctx, "concat(Name,' Test') eq 'Laptop Test'")
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return fmt.Errorf("concat filter returned no items")
			}
			for i, item := range items {
				name, err := productName(item)
				if err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
				if name+" Test" != "Laptop Test" {
					return fmt.Errorf("item %d has Name=%q which does not satisfy concat(Name,' Test') eq 'Laptop Test'", i, name)
				}
			}
			return nil
		},
	)

	return suite
}
