package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

func applyValueCount(respBody []byte) (int, error) {
	var body struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(respBody, &body); err != nil {
		return 0, fmt.Errorf("failed to parse response body: %w", err)
	}
	return len(body.Value), nil
}

func applyProductNames(respBody []byte) ([]string, error) {
	items, err := parseApplyItems(&framework.HTTPResponse{Body: respBody})
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(items))
	for i, item := range items {
		rawName, ok := firstPresent(item, "Name", "name")
		if !ok {
			return nil, fmt.Errorf("item %d missing Name field", i)
		}
		name, ok := rawName.(string)
		if !ok {
			return nil, fmt.Errorf("item %d Name field is not a string", i)
		}
		names = append(names, name)
	}

	return names, nil
}

// ApplyTransformationCatalog creates the 11.2.5.4.2 $apply transformation catalog suite.
func ApplyTransformationCatalog() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.4.2 $apply transformation catalog",
		"Tests the full $apply transformation catalog required by the OData aggregation extension.",
		"https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html",
	)

	suite.AddTest(
		"test_apply_identity",
		"identity transformation returns the original set",
		func(ctx *framework.TestContext) error {
			baselineResp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(baselineResp, 200); err != nil {
				return err
			}
			baselineCount, err := applyValueCount(baselineResp.Body)
			if err != nil {
				return err
			}

			resp, err := ctx.GET("/Products?$apply=identity")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			gotCount, err := applyValueCount(resp.Body)
			if err != nil {
				return err
			}
			if gotCount != baselineCount {
				return fmt.Errorf("identity count mismatch: expected %d, got %d", baselineCount, gotCount)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_orderby_skip_top_pipeline",
		"orderby/skip/top can be used as $apply set transformations",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("identity/orderby(Price desc)/skip(1)/top(2)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			names, err := applyProductNames(resp.Body)
			if err != nil {
				return err
			}
			if len(names) != 2 {
				return fmt.Errorf("expected 2 products, got %d", len(names))
			}
			if names[0] != "Laptop" || names[1] != "Smartphone" {
				return fmt.Errorf("expected [Laptop Smartphone], got %v", names)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_search_transformation",
		"search can be used as an $apply transformation",
		func(ctx *framework.TestContext) error {
			baselineResp, err := ctx.GET("/Products?$search=Laptop")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(baselineResp, 200); err != nil {
				return err
			}
			baselineCount, err := applyValueCount(baselineResp.Body)
			if err != nil {
				return err
			}

			applyExpr := url.QueryEscape("search(Laptop)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			count, err := applyValueCount(resp.Body)
			if err != nil {
				return err
			}
			if count != baselineCount {
				return fmt.Errorf("search transformation count mismatch: expected %d, got %d", baselineCount, count)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_topcount",
		"topcount returns top N by measure",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("topcount(2,Price)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			names, err := applyProductNames(resp.Body)
			if err != nil {
				return err
			}
			if len(names) != 2 {
				return fmt.Errorf("expected 2 products, got %d", len(names))
			}
			if names[0] != "Premium Laptop Pro" || names[1] != "Laptop" {
				return fmt.Errorf("expected [Premium Laptop Pro Laptop], got %v", names)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_bottomcount",
		"bottomcount returns bottom N by measure",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("bottomcount(2,Price)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			names, err := applyProductNames(resp.Body)
			if err != nil {
				return err
			}
			if len(names) != 2 {
				return fmt.Errorf("expected 2 products, got %d", len(names))
			}
			if names[0] != "Coffee Mug" || names[1] != "Wireless Mouse" {
				return fmt.Errorf("expected [Coffee Mug Wireless Mouse], got %v", names)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_toppercent_100",
		"toppercent(100,...) returns the full set",
		func(ctx *framework.TestContext) error {
			baselineResp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(baselineResp, 200); err != nil {
				return err
			}
			baselineCount, err := applyValueCount(baselineResp.Body)
			if err != nil {
				return err
			}

			applyExpr := url.QueryEscape("toppercent(100,Price)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			count, err := applyValueCount(resp.Body)
			if err != nil {
				return err
			}
			if count != baselineCount {
				return fmt.Errorf("expected %d items, got %d", baselineCount, count)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_bottompercent_100",
		"bottompercent(100,...) returns the full set",
		func(ctx *framework.TestContext) error {
			baselineResp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(baselineResp, 200); err != nil {
				return err
			}
			baselineCount, err := applyValueCount(baselineResp.Body)
			if err != nil {
				return err
			}

			applyExpr := url.QueryEscape("bottompercent(100,Price)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			count, err := applyValueCount(resp.Body)
			if err != nil {
				return err
			}
			if count != baselineCount {
				return fmt.Errorf("expected %d items, got %d", baselineCount, count)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_topsum_large_threshold",
		"topsum with very large threshold returns the full set",
		func(ctx *framework.TestContext) error {
			baselineResp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(baselineResp, 200); err != nil {
				return err
			}
			baselineCount, err := applyValueCount(baselineResp.Body)
			if err != nil {
				return err
			}

			applyExpr := url.QueryEscape("topsum(100000,Price)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			count, err := applyValueCount(resp.Body)
			if err != nil {
				return err
			}
			if count != baselineCount {
				return fmt.Errorf("expected %d items, got %d", baselineCount, count)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_topsum_cutoff_semantics",
		"topsum enforces cumulative-threshold cutoff in descending measure order",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("topsum(2500,Price)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			names, err := applyProductNames(resp.Body)
			if err != nil {
				return err
			}
			if len(names) != 2 {
				return fmt.Errorf("expected 2 products for topsum cutoff, got %d", len(names))
			}
			if names[0] != "Premium Laptop Pro" || names[1] != "Laptop" {
				return fmt.Errorf("expected [Premium Laptop Pro Laptop], got %v", names)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_bottomsum_large_threshold",
		"bottomsum with very large threshold returns the full set",
		func(ctx *framework.TestContext) error {
			baselineResp, err := ctx.GET("/Products")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(baselineResp, 200); err != nil {
				return err
			}
			baselineCount, err := applyValueCount(baselineResp.Body)
			if err != nil {
				return err
			}

			applyExpr := url.QueryEscape("bottomsum(100000,Price)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			count, err := applyValueCount(resp.Body)
			if err != nil {
				return err
			}
			if count != baselineCount {
				return fmt.Errorf("expected %d items, got %d", baselineCount, count)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_bottomsum_cutoff_semantics",
		"bottomsum enforces cumulative-threshold cutoff in ascending measure order",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("bottomsum(40,Price)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			names, err := applyProductNames(resp.Body)
			if err != nil {
				return err
			}
			if len(names) != 2 {
				return fmt.Errorf("expected 2 products for bottomsum cutoff, got %d", len(names))
			}
			if names[0] != "Coffee Mug" || names[1] != "Wireless Mouse" {
				return fmt.Errorf("expected [Coffee Mug Wireless Mouse], got %v", names)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_concat",
		"concat combines the results of multiple transformation sequences",
		func(ctx *framework.TestContext) error {
			upperExpr := url.QueryEscape("filter(Price gt 100)")
			upperResp, err := ctx.GET("/Products?$apply=" + upperExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(upperResp, 200); err != nil {
				return err
			}
			upperCount, err := applyValueCount(upperResp.Body)
			if err != nil {
				return err
			}

			lowerExpr := url.QueryEscape("filter(Price gt 500)")
			lowerResp, err := ctx.GET("/Products?$apply=" + lowerExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(lowerResp, 200); err != nil {
				return err
			}
			lowerCount, err := applyValueCount(lowerResp.Body)
			if err != nil {
				return err
			}

			concatExpr := url.QueryEscape("concat(filter(Price gt 100),filter(Price gt 500))")
			concatResp, err := ctx.GET("/Products?$apply=" + concatExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(concatResp, 200); err != nil {
				return err
			}
			concatCount, err := applyValueCount(concatResp.Body)
			if err != nil {
				return err
			}

			expected := upperCount + lowerCount
			if concatCount != expected {
				return fmt.Errorf("concat count mismatch: expected %d, got %d", expected, concatCount)
			}

			// These filters overlap, so concat must preserve duplicate rows from the
			// second sequence (UNION ALL semantics).
			if concatCount <= upperCount {
				return fmt.Errorf("expected concat with overlapping sequences to include duplicates: concat=%d upper=%d", concatCount, upperCount)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_join",
		"join duplicates parent rows for each related child and excludes parents with empty collections",
		func(ctx *framework.TestContext) error {
			descriptionResp, err := ctx.GET("/ProductDescriptions")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(descriptionResp, 200); err != nil {
				return err
			}
			descriptionCount, err := applyValueCount(descriptionResp.Body)
			if err != nil {
				return err
			}

			applyExpr := url.QueryEscape("join(Descriptions as Description)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}
			if len(items) != descriptionCount {
				return fmt.Errorf("expected join result size %d to equal description count, got %d", descriptionCount, len(items))
			}

			for i, item := range items {
				if _, ok := firstPresent(item, "Description", "description"); !ok {
					return fmt.Errorf("joined row %d missing Description alias", i)
				}
				if rawName, ok := firstPresent(item, "Name", "name"); ok {
					if name, ok := rawName.(string); ok && name == "Desk" {
						return fmt.Errorf("join should exclude products with empty Descriptions collection")
					}
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_outerjoin",
		"outerjoin preserves parents with empty collections by emitting a null alias row",
		func(ctx *framework.TestContext) error {
			productsResp, err := ctx.GET("/Products?$expand=Descriptions")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(productsResp, 200); err != nil {
				return err
			}

			var expanded struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(productsResp.Body, &expanded); err != nil {
				return fmt.Errorf("failed to parse expanded products response: %w", err)
			}

			emptyParents := 0
			descriptionCount := 0
			for _, product := range expanded.Value {
				rawDescriptions, ok := firstPresent(product, "Descriptions", "descriptions")
				if !ok || rawDescriptions == nil {
					emptyParents++
					continue
				}
				descriptions, ok := rawDescriptions.([]interface{})
				if !ok {
					return fmt.Errorf("expanded Descriptions value has unexpected type %T", rawDescriptions)
				}
				descriptionCount += len(descriptions)
				if len(descriptions) == 0 {
					emptyParents++
				}
			}

			applyExpr := url.QueryEscape("outerjoin(Descriptions as Description)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}
			expected := descriptionCount + emptyParents
			if len(items) != expected {
				return fmt.Errorf("expected outerjoin result size %d, got %d", expected, len(items))
			}

			foundNullAlias := false
			for _, item := range items {
				rawDesc, ok := firstPresent(item, "Description", "description")
				if ok && rawDesc == nil {
					foundNullAlias = true
					break
				}
			}
			if !foundNullAlias {
				return fmt.Errorf("expected outerjoin result to include at least one null Description alias")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_groupby_nested_sequence",
		"groupby second parameter accepts a transformation sequence",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("groupby((CategoryID),aggregate($count as GroupCount)/orderby(GroupCount desc)/top(2))")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}
			if len(items) != 2 {
				return fmt.Errorf("expected top(2) to limit grouped results to 2, got %d", len(items))
			}

			var prev float64
			for i, item := range items {
				rawCount, ok := firstPresent(item, "GroupCount", "groupcount")
				if !ok {
					return fmt.Errorf("group %d missing GroupCount", i)
				}
				count, ok := rawCount.(float64)
				if !ok {
					return fmt.Errorf("group %d GroupCount is not numeric", i)
				}
				if i > 0 && count > prev {
					return fmt.Errorf("groups are not ordered descending by GroupCount")
				}
				prev = count
			}

			return nil
		},
	)

	return suite
}
