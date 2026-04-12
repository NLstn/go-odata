package v4_0

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"

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

func applyEntityIDs(respBody []byte) ([]string, error) {
	items, err := parseApplyItems(&framework.HTTPResponse{Body: respBody})
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(items))
	for i, item := range items {
		rawID, ok := firstPresent(item, "ID", "id")
		if !ok {
			return nil, fmt.Errorf("item %d missing ID field", i)
		}
		ids = append(ids, fmt.Sprintf("%v", rawID))
	}

	sort.Strings(ids)
	return ids, nil
}

func decodePayload(respBody []byte) (interface{}, error) {
	var payload interface{}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse response payload: %w", err)
	}
	return payload, nil
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
			baselineIDs, err := applyEntityIDs(baselineResp.Body)
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
			gotIDs, err := applyEntityIDs(resp.Body)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(gotIDs, baselineIDs) {
				return fmt.Errorf("identity result mismatch: expected IDs %v, got %v", baselineIDs, gotIDs)
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
			baselineIDs, err := applyEntityIDs(baselineResp.Body)
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
			gotIDs, err := applyEntityIDs(resp.Body)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(gotIDs, baselineIDs) {
				return fmt.Errorf("search transformation mismatch: expected IDs %v, got %v", baselineIDs, gotIDs)
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

	suite.AddTest(
		"test_apply_join_nested_sequence",
		"join nested identity sequence is behaviorally equivalent to plain join",
		func(ctx *framework.TestContext) error {
			plainExpr := url.QueryEscape("join(Descriptions as Description)")
			plainResp, err := ctx.GET("/Products?$apply=" + plainExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(plainResp, http.StatusOK); err != nil {
				return err
			}
			plainPayload, err := decodePayload(plainResp.Body)
			if err != nil {
				return err
			}

			applyExpr := url.QueryEscape("join(Descriptions as Description,identity)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}
			payload, err := decodePayload(resp.Body)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(payload, plainPayload) {
				return fmt.Errorf("join nested identity must match plain join payload")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_outerjoin_nested_sequence",
		"outerjoin nested identity sequence is behaviorally equivalent to plain outerjoin",
		func(ctx *framework.TestContext) error {
			plainExpr := url.QueryEscape("outerjoin(Descriptions as Description)")
			plainResp, err := ctx.GET("/Products?$apply=" + plainExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(plainResp, http.StatusOK); err != nil {
				return err
			}
			plainPayload, err := decodePayload(plainResp.Body)
			if err != nil {
				return err
			}

			applyExpr := url.QueryEscape("outerjoin(Descriptions as Description,identity)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}
			payload, err := decodePayload(resp.Body)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(payload, plainPayload) {
				return fmt.Errorf("outerjoin nested identity must match plain outerjoin payload")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_structural_concat_tail_filter",
		"concat supports filter as a structural tail transformation",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("concat(filter(Price gt 100),filter(Price gt 500))/filter(Price gt 1000)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			var body struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse join filter response: %w", err)
			}

			for i, item := range body.Value {
				rawPrice, ok := firstPresent(item, "Price", "price")
				if !ok {
					return fmt.Errorf("row %d missing Price", i)
				}
				price, ok := rawPrice.(float64)
				if !ok {
					return fmt.Errorf("row %d Price is not numeric", i)
				}
				if price <= 1000 {
					return fmt.Errorf("row %d has Price=%v, expected > 1000", i, price)
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_ancestors",
		"ancestors requires fully-specified hierarchy parameters and must reject empty invocation",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("ancestors()")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, http.StatusBadRequest, "")
		},
	)

	suite.AddTest(
		"test_apply_descendants",
		"descendants requires fully-specified hierarchy parameters and must reject empty invocation",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("descendants()")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, http.StatusBadRequest, "")
		},
	)

	suite.AddTest(
		"test_apply_traverse",
		"traverse requires fully-specified hierarchy parameters and must reject empty invocation",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("traverse()")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, http.StatusBadRequest, "")
		},
	)

	suite.AddTest(
		"test_apply_structural_join_tail_filter",
		"join supports filter as a structural tail transformation",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("join(Descriptions as Description)/filter(Price gt 1000)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			var body struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse outerjoin filter response: %w", err)
			}

			for i, item := range body.Value {
				rawPrice, ok := firstPresent(item, "Price", "price")
				if !ok {
					return fmt.Errorf("row %d missing Price", i)
				}
				price, ok := rawPrice.(float64)
				if !ok {
					return fmt.Errorf("row %d Price is not numeric", i)
				}
				if price <= 1000 {
					return fmt.Errorf("row %d has Price=%v, expected > 1000", i, price)
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_structural_outerjoin_tail_filter",
		"outerjoin supports filter as a structural tail transformation",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("outerjoin(Descriptions as Description)/filter(Price gt 1000)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}

			for i, item := range items {
				rawPrice, ok := firstPresent(item, "Price", "price")
				if !ok {
					return fmt.Errorf("row %d missing Price", i)
				}
				price, ok := rawPrice.(float64)
				if !ok {
					return fmt.Errorf("row %d Price is not numeric", i)
				}
				if price <= 1000 {
					return fmt.Errorf("row %d has Price=%v, expected > 1000", i, price)
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_service_defined_set_function",
		"unknown service-defined set function transformation is rejected with OData error payload",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("Default.CustomSetTransform()")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, http.StatusBadRequest, "")
		},
	)

	suite.AddTest(
		"test_apply_join_aggregate_tail",
		"join with aggregate tail returns aggregated count of all joined rows",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("join(Descriptions as Description)/aggregate($count as TotalCount)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}
			if len(items) != 1 {
				return fmt.Errorf("expected exactly 1 result row, got %d", len(items))
			}

			rawCount, ok := firstPresent(items[0], "TotalCount", "totalCount")
			if !ok {
				return fmt.Errorf("result row missing TotalCount field")
			}
			count, ok := rawCount.(float64)
			if !ok {
				return fmt.Errorf("TotalCount is not numeric, got %T: %v", rawCount, rawCount)
			}
			if count < 7 {
				return fmt.Errorf("expected TotalCount >= 7 (7 product descriptions linked to products), got %v", count)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_join_groupby_tail",
		"join with groupby tail groups joined rows by a parent property",
		func(ctx *framework.TestContext) error {
			applyExpr := url.QueryEscape("join(Descriptions as Description)/groupby((CategoryID))")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}
			if len(items) < 1 {
				return fmt.Errorf("expected at least 1 group, got 0")
			}
			// There are 3 categories total; only products with descriptions (Electronics and Kitchen)
			// appear in the join result, so we expect at most 3 groups.
			if len(items) > 3 {
				return fmt.Errorf("expected at most 3 category groups, got %d", len(items))
			}

			for i, item := range items {
				if _, ok := firstPresent(item, "CategoryID", "categoryID"); !ok {
					return fmt.Errorf("group %d missing CategoryID field", i)
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_concat_aggregate_tail",
		"concat with aggregate tail returns aggregated count of concatenated rows",
		func(ctx *framework.TestContext) error {
			// Get baseline counts for each filter
			gt0Resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("Price gt 0") + "&$count=true")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(gt0Resp, http.StatusOK); err != nil {
				return err
			}
			gt0Count, err := applyValueCount(gt0Resp.Body)
			if err != nil {
				return err
			}

			gt100Resp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("Price gt 100") + "&$count=true")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(gt100Resp, http.StatusOK); err != nil {
				return err
			}
			gt100Count, err := applyValueCount(gt100Resp.Body)
			if err != nil {
				return err
			}

			expectedTotal := gt0Count + gt100Count

			applyExpr := url.QueryEscape("concat(filter(Price gt 0),filter(Price gt 100))/aggregate($count as TotalCount)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}
			if len(items) != 1 {
				return fmt.Errorf("expected exactly 1 result row from aggregate, got %d", len(items))
			}

			rawCount, ok := firstPresent(items[0], "TotalCount", "totalCount")
			if !ok {
				return fmt.Errorf("result row missing TotalCount field")
			}
			count, ok := rawCount.(float64)
			if !ok {
				return fmt.Errorf("TotalCount is not numeric, got %T: %v", rawCount, rawCount)
			}
			if int(count) != expectedTotal {
				return fmt.Errorf("expected TotalCount=%d (Price>0: %d + Price>100: %d), got %v",
					expectedTotal, gt0Count, gt100Count, count)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_filter_then_concat",
		"filter followed by concat uses filtered input set for each concat sequence",
		func(ctx *framework.TestContext) error {
			baselineExpr := url.QueryEscape("filter(Price gt 0)")
			baselineResp, err := ctx.GET("/Products?$apply=" + baselineExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(baselineResp, http.StatusOK); err != nil {
				return err
			}
			baselineCount, err := applyValueCount(baselineResp.Body)
			if err != nil {
				return err
			}

			applyExpr := url.QueryEscape("filter(Price gt 0)/concat(identity,identity)/aggregate($count as Total)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			items, err := parseApplyItems(resp)
			if err != nil {
				return err
			}
			if len(items) != 1 {
				return fmt.Errorf("expected exactly 1 result row from aggregate, got %d", len(items))
			}

			rawTotal, ok := firstPresent(items[0], "Total", "total")
			if !ok {
				return fmt.Errorf("result row missing Total field")
			}
			total, ok := rawTotal.(float64)
			if !ok {
				return fmt.Errorf("Total is not numeric, got %T: %v", rawTotal, rawTotal)
			}

			expected := baselineCount * 2
			if int(total) != expected {
				return fmt.Errorf("expected Total=%d (2 * baseline %d), got %v", expected, baselineCount, total)
			}

			return nil
		},
	)

	suite.AddTest(
		"test_apply_filter_then_join",
		"filter followed by join applies filter to parent entities before joining",
		func(ctx *framework.TestContext) error {
			filteredExpandedResp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("Price gt 1000") + "&$expand=Descriptions")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(filteredExpandedResp, http.StatusOK); err != nil {
				return err
			}

			var expanded struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(filteredExpandedResp.Body, &expanded); err != nil {
				return fmt.Errorf("failed to parse filtered expanded products response: %w", err)
			}

			expectedRows := 0
			for i, product := range expanded.Value {
				rawDescriptions, ok := firstPresent(product, "Descriptions", "descriptions")
				if !ok || rawDescriptions == nil {
					continue
				}
				descriptions, ok := rawDescriptions.([]interface{})
				if !ok {
					return fmt.Errorf("expanded Descriptions value for product %d has unexpected type %T", i, rawDescriptions)
				}
				expectedRows += len(descriptions)
			}

			applyExpr := url.QueryEscape("filter(Price gt 1000)/join(Descriptions as D)")
			resp, err := ctx.GET("/Products?$apply=" + applyExpr)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			var body struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse filtered join response: %w", err)
			}
			if len(body.Value) != expectedRows {
				return fmt.Errorf("expected %d joined rows for filtered products, got %d", expectedRows, len(body.Value))
			}

			for i, item := range body.Value {
				rawPrice, ok := firstPresent(item, "Price", "price")
				if !ok {
					return fmt.Errorf("row %d missing Price", i)
				}
				price, ok := rawPrice.(float64)
				if !ok {
					return fmt.Errorf("row %d Price is not numeric", i)
				}
				if price <= 1000 {
					return fmt.Errorf("row %d has Price=%v, expected > 1000", i, price)
				}

				if _, ok := firstPresent(item, "D", "d"); !ok {
					return fmt.Errorf("row %d missing join alias D", i)
				}
			}

			return nil
		},
	)

	return suite
}
