package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"

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

	parseIndexedEntities := func(ctx *framework.TestContext, resp *framework.HTTPResponse) ([]map[string]interface{}, error) {
		items, err := ctx.ParseEntityCollection(resp)
		if err != nil {
			return nil, err
		}
		if err := ctx.AssertMinCollectionSize(items, 1); err != nil {
			return nil, err
		}
		return items, nil
	}

	assertSequentialIndexes := func(items []map[string]interface{}, start int) error {
		for i, item := range items {
			rawIndex, ok := item["@odata.index"]
			if !ok {
				return fmt.Errorf("item %d is missing @odata.index", i)
			}

			indexValue, ok := rawIndex.(float64)
			if !ok {
				return fmt.Errorf("item %d has non-numeric @odata.index value %T", i, rawIndex)
			}

			expected := float64(start + i)
			if indexValue != expected {
				return fmt.Errorf("item %d has @odata.index %.0f, expected %.0f", i, indexValue, expected)
			}
		}

		return nil
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

			items, err := parseIndexedEntities(ctx, resp)
			if err != nil {
				return err
			}

			if err := assertSequentialIndexes(items, 0); err != nil {
				return framework.NewError(fmt.Sprintf("$index response must include sequential @odata.index annotations: %v", err))
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

			items, err := parseIndexedEntities(ctx, resp)
			if err != nil {
				return framework.NewError(fmt.Sprintf("$index response must include a non-empty value collection: %v", err))
			}
			if len(items) != 3 {
				return framework.NewError(fmt.Sprintf("expected 3 items from $top=3 query, got %d", len(items)))
			}
			if err := assertSequentialIndexes(items, 0); err != nil {
				return framework.NewError(fmt.Sprintf("$index response format must include sequential annotations: %v", err))
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

			if err := ctx.AssertStatusCode(resp, http.StatusBadRequest); err != nil {
				return framework.NewError(fmt.Sprintf("expected HTTP 400 when $index is used on a single entity: %v", err))
			}

			if !ctx.IsValidJSON(resp) {
				return framework.NewError("single-entity $index rejection must return a valid JSON error payload")
			}

			if err := ctx.AssertBodyContains(resp, "$index query option is not applicable to individual entities"); err != nil {
				return framework.NewError(fmt.Sprintf("single-entity $index rejection should explain why the request is invalid: %v", err))
			}

			return nil
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
		"@odata.index annotation presence is required",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$top=2")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := parseIndexedEntities(ctx, resp)
			if err != nil {
				return err
			}
			if len(items) != 2 {
				return framework.NewError(fmt.Sprintf("expected 2 items from $top=2 query, got %d", len(items)))
			}
			if err := assertSequentialIndexes(items, 0); err != nil {
				return framework.NewError(fmt.Sprintf("@odata.index annotations must be present and sequential: %v", err))
			}

			return nil
		},
	)

	// Test 12: $index value starts at 0
	suite.AddTest(
		"test_index_starts_at_zero",
		"$index starts at zero",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$index&$top=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := parseIndexedEntities(ctx, resp)
			if err != nil {
				return err
			}
			if len(items) != 1 {
				return framework.NewError(fmt.Sprintf("expected 1 item from $top=1 query, got %d", len(items)))
			}
			if err := assertSequentialIndexes(items, 0); err != nil {
				return framework.NewError(fmt.Sprintf("first $index value must be zero: %v", err))
			}

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

	// Test 14: $index case insensitivity (OData 4.01)
	suite.AddTest(
		"test_index_case_sensitivity",
		"$INDEX is accepted case-insensitively per OData 4.01",
		func(ctx *framework.TestContext) error {
			// OData 4.01 makes all system query option names case-insensitive.
			// $INDEX must be treated as equivalent to $index.
			resp, err := ctx.GET("/Products?$INDEX")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
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

	suite.AddTest(
		"test_index_version_negotiation_4_01_vs_4_0",
		"$index is accepted with OData-MaxVersion 4.01 and rejected when negotiated to 4.0",
		func(ctx *framework.TestContext) error {
			query := "/Products?$INDEX&$top=1"

			v401Headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			v401Resp, err := ctx.GET(query, v401Headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(v401Resp, http.StatusOK); err != nil {
				return framework.NewError(fmt.Sprintf("4.01 negotiated $index request should succeed: %v", err))
			}

			v40Headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}
			v40Resp, err := ctx.GET(query, v40Headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(v40Resp, http.StatusBadRequest); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 negotiated request must reject 4.01 $index behavior: %v", err))
			}
			if err := ctx.AssertODataError(v40Resp, http.StatusBadRequest, "unknown query option"); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 negotiated $index rejection must include strict OData error payload: %v", err))
			}

			return nil
		},
	)

	return suite
}
