package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// CountSegment creates the 11.2.4.2 $count segment test suite.
func CountSegment() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.4.2 $count Segment",
		"Tests the $count path segment for entity and navigation collections.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_Count",
	)

	// Test 1: $count on entity set returns text/plain integer
	suite.AddTest(
		"test_entityset_count_segment",
		"$count on entity set returns text/plain integer",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products/$count")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			if err := ctx.AssertHeaderContains(resp, "Content-Type", "text/plain"); err != nil {
				return err
			}
			if _, err := parseCountBody(resp.Body); err != nil {
				return err
			}
			return nil
		},
	)

	// Test 2: $count respects $filter and matches @odata.count
	suite.AddTest(
		"test_count_segment_with_filter",
		"$count segment respects $filter and matches @odata.count",
		func(ctx *framework.TestContext) error {
			filter := "Price gt 100"
			qp := url.Values{}
			qp.Set("$filter", filter)

			segmentResp, err := ctx.GET("/Products/$count?" + qp.Encode())
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(segmentResp, 200); err != nil {
				return err
			}
			segmentCount, err := parseCountBody(segmentResp.Body)
			if err != nil {
				return err
			}

			queryResp, err := ctx.GET("/Products?$count=true&" + qp.Encode())
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(queryResp, 200); err != nil {
				return err
			}

			var body map[string]interface{}
			if err := json.Unmarshal(queryResp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}
			countValue, ok := body["@odata.count"]
			if !ok {
				return fmt.Errorf("@odata.count missing from response")
			}
			countFloat, ok := countValue.(float64)
			if !ok {
				return fmt.Errorf("@odata.count is not a number")
			}
			if segmentCount != int(countFloat) {
				return fmt.Errorf("$count segment=%d does not match @odata.count=%d", segmentCount, int(countFloat))
			}

			return nil
		},
	)

	// Test 3: $count on navigation collection
	suite.AddTest(
		"test_navigation_count_segment",
		"$count on navigation collection returns integer",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath + "/Descriptions/$count")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			if err := ctx.AssertHeaderContains(resp, "Content-Type", "text/plain"); err != nil {
				return err
			}
			if _, err := parseCountBody(resp.Body); err != nil {
				return err
			}
			return nil
		},
	)

	return suite
}

func parseCountBody(body []byte) (int, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return 0, fmt.Errorf("empty $count response body")
	}
	count, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid $count response %q: %w", trimmed, err)
	}
	if count < 0 {
		return 0, fmt.Errorf("$count must be non-negative, got %d", count)
	}
	return count, nil
}
