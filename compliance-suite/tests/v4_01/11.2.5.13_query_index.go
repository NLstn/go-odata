package v4_01

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QueryIndex creates the 11.2.5.13 $index Query Option test suite
func QueryIndex() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.13 $index Query Option",
		"Validates the $index system query option which returns the zero-based ordinal position of each item in a collection. This is an OData v4.01 feature.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_index",
	)

	var (
		productPath string
		categoryID  string
	)

	parseEntityIDValue := func(value interface{}) (string, error) {
		if value == nil {
			return "", fmt.Errorf("entity ID is nil")
		}
		return fmt.Sprintf("%v", value), nil
	}

	getFirstEntityID := func(ctx *framework.TestContext, entitySet string) (string, error) {
		resp, err := ctx.GET(fmt.Sprintf("/%s?$top=1&$select=ID", entitySet))
		if err != nil {
			return "", err
		}
		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return "", fmt.Errorf("list %s: %w", entitySet, err)
		}
		var body struct {
			Value []map[string]interface{} `json:"value"`
		}
		if err := json.Unmarshal(resp.Body, &body); err != nil {
			return "", fmt.Errorf("parse %s list: %w", entitySet, err)
		}
		if len(body.Value) == 0 {
			return "", fmt.Errorf("no %s available", entitySet)
		}
		id, err := parseEntityIDValue(body.Value[0]["ID"])
		if err != nil {
			return "", err
		}
		return id, nil
	}

	getProductPath := func(ctx *framework.TestContext) (string, error) {
		if productPath != "" {
			return productPath, nil
		}
		id, err := getFirstEntityID(ctx, "Products")
		if err != nil {
			return "", err
		}
		productPath = fmt.Sprintf("/Products(%s)", id)
		return productPath, nil
	}

	getCategoryID := func(ctx *framework.TestContext) (string, error) {
		if categoryID != "" {
			return categoryID, nil
		}
		id, err := getFirstEntityID(ctx, "Categories")
		if err != nil {
			return "", err
		}
		categoryID = id
		return categoryID, nil
	}

	// Test 1: $index without other query options
	suite.AddTest(
		"test_index_basic",
		"$index query option basic support",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("Missing $index support is a compliance defect: %v", err))
			}

			body := string(resp.Body)
			if !strings.Contains(body, `"@odata.index"`) {
				ctx.Log("Response missing @odata.index annotations (optional check)")
			}

			return nil
		},
	)

	// Test 2: $index with $top
	suite.AddTest(
		"test_index_with_top",
		"$index works with $top",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$top=5")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("Missing $index with $top support: %v", err))
			}

			return nil
		},
	)

	// Test 3: $index with $skip
	suite.AddTest(
		"test_index_with_skip",
		"$index works with $skip",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$skip=2")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("Missing $index with $skip support: %v", err))
			}

			return nil
		},
	)

	// Test 4: $index with $orderby
	suite.AddTest(
		"test_index_with_orderby",
		"$index works with $orderby",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$orderby=Price")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("Missing $index with $orderby support: %v", err))
			}

			return nil
		},
	)

	// Test 5: $index with $filter
	suite.AddTest(
		"test_index_with_filter",
		"$index works with $filter",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$filter=Price gt 50")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("Missing $index with $filter support: %v", err))
			}

			return nil
		},
	)

	// Test 6: $index response format
	suite.AddTest(
		"test_index_response_format",
		"$index response has valid JSON structure",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$top=3")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `"value"`) {
				return framework.NewError("Response missing value array")
			}

			return nil
		},
	)

	// Test 7: $index with $expand
	suite.AddTest(
		"test_index_with_expand",
		"$index works with $expand",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$expand=Category")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("Missing $index with $expand support: %v", err))
			}

			return nil
		},
	)

	// Test 8: $index on entity should fail
	suite.AddTest(
		"test_index_on_entity",
		"$index rejected on single entity",
		func(ctx *framework.TestContext) error {
			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(path + "?$index")
			if err != nil {
				return err
			}

			// $index should not work on single entities
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return nil // Good, it was rejected
			}

			return framework.NewError("$index should be rejected on single entity")
		},
	)

	// Test 9: $index with complex query combination
	suite.AddTest(
		"test_index_complex_query",
		"$index works with complex query combinations",
		func(ctx *framework.TestContext) error {
			catID, err := getCategoryID(ctx)
			if err != nil {
				return err
			}
			resp, err := ctx.GET(fmt.Sprintf("/Products?$index&$filter=CategoryID eq %s&$orderby=Name&$top=5", catID))
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("Missing $index complex query support: %v", err))
			}

			return nil
		},
	)

	// Test 10: $index with $count
	suite.AddTest(
		"test_index_with_count",
		"$index works with $count",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$count=true")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("Missing $index with $count support: %v", err))
			}

			return nil
		},
	)

	// Test 11: Check if @odata.index annotation is included
	suite.AddTest(
		"test_index_annotation_presence",
		"@odata.index annotation presence (optional)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$top=2")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `"@odata.index"`) {
				ctx.Log("Response missing @odata.index annotations (optional check)")
			}

			return nil
		},
	)

	// Test 12: $index value starts at 0
	suite.AddTest(
		"test_index_starts_at_zero",
		"$index starts at zero (optional verification)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$top=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if matched, _ := regexp.MatchString(`"@odata\.index"\s*:\s*0`, body); matched {
				return nil
			}

			ctx.Log("Unable to verify that @odata.index starts at 0 (optional check)")
			return nil
		},
	)

	// Test 13: $index with $select
	suite.AddTest(
		"test_index_with_select",
		"$index works with $select",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$select=Name,Price")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return framework.NewError(fmt.Sprintf("Missing $index with $select support: %v", err))
			}

			return nil
		},
	)

	// Test 14: $index case sensitivity
	suite.AddTest(
		"test_index_case_sensitivity",
		"$index is case-sensitive",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$INDEX")
			if err != nil {
				return err
			}

			// Should reject uppercase $INDEX
			if resp.StatusCode == 200 {
				return framework.NewError("Service treated $INDEX as valid; expected rejection")
			}

			if err := ctx.AssertStatusCode(resp, 400); err != nil {
				return framework.NewError(fmt.Sprintf("Expected HTTP 400 for uppercase $INDEX but got %d", resp.StatusCode))
			}

			return nil
		},
	)

	// Test 15: Multiple $index parameters (invalid)
	suite.AddTest(
		"test_multiple_index_params",
		"Duplicate $index parameters handled",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$index")
			if err != nil {
				return err
			}

			// Should reject duplicate parameters
			if resp.StatusCode == 200 {
				return framework.NewError("Service accepted duplicate $index parameters")
			}

			if err := ctx.AssertStatusCode(resp, 400); err != nil {
				return framework.NewError(fmt.Sprintf("Expected HTTP 400 for duplicate $index parameters but got %d", resp.StatusCode))
			}

			return nil
		},
	)

	return suite
}
