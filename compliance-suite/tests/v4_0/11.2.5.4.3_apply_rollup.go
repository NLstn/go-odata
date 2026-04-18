package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// QueryApplyRollup creates the test suite for the rollup() transformation.
// Spec reference: OData Data Aggregation Extension v4.0, Section 3.2.2
func QueryApplyRollup() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.4 $apply rollup() Transformation",
		"Tests the rollup() transformation within groupby() for hierarchical aggregation according to OData v4.0 Data Aggregation Extension Section 3.2.2.",
		"https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html#sec_Transformationrollup",
	)

	// Test 1: rollup(null, prop) — single dimension with grand total
	suite.AddTest(
		"test_rollup_single_with_grand_total",
		"rollup(null, prop) produces subtotals and grand total row",
		func(ctx *framework.TestContext) error {
			query := url.QueryEscape("groupby((rollup(null,CategoryID)),aggregate(Price with sum as Total))")
			resp, err := ctx.GET("/Products?$apply=" + query)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var body struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if len(body.Value) == 0 {
				return framework.NewError("response should contain at least one row")
			}

			// There must be a grand total row where CategoryID is null
			grandTotalFound := false
			for _, row := range body.Value {
				cat, hasCat := row["CategoryID"]
				if hasCat && cat == nil {
					grandTotalFound = true
					// The grand total row must have a Total aggregate value
					totalRaw, hasTotal := firstPresent(row, "Total", "total")
					if !hasTotal {
						return framework.NewError("grand total row must contain Total aggregate field")
					}
					if totalRaw == nil {
						return framework.NewError("grand total row Total must not be null")
					}
				}
			}
			if !grandTotalFound {
				return framework.NewError("rollup(null, CategoryID) must produce a grand total row with CategoryID=null")
			}

			return nil
		},
	)

	// Test 2: rollup(prop) without null — no grand total row
	suite.AddTest(
		"test_rollup_single_without_grand_total",
		"rollup(prop) without null does not produce a grand total row",
		func(ctx *framework.TestContext) error {
			query := url.QueryEscape("groupby((rollup(CategoryID)),aggregate(Price with sum as Total))")
			resp, err := ctx.GET("/Products?$apply=" + query)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var body struct {
				Value []map[string]interface{} `json:"value"`
			}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			// There must NOT be a grand total row (all GroupBy dimensions are null)
			for _, row := range body.Value {
				if cat, ok := row["CategoryID"]; ok && cat == nil {
					return framework.NewError("rollup(CategoryID) without null must not produce a grand total row (CategoryID=null)")
				}
			}

			return nil
		},
	)

	// Test 3: rollup produces multiple aggregation levels
	suite.AddTest(
		"test_rollup_produces_hierarchical_levels",
		"rollup() produces one row per aggregation level",
		func(ctx *framework.TestContext) error {
			// First count categories
			catResp, err := ctx.GET("/Products?$apply=groupby((CategoryID))")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(catResp, 200); err != nil {
				return err
			}
			catCount, err := applyValueCount(catResp.Body)
			if err != nil {
				return err
			}

			// rollup(null, CategoryID) should produce catCount + 1 rows (categories + grand total)
			query := url.QueryEscape("groupby((rollup(null,CategoryID)),aggregate(Price with sum as Total))")
			resp, err := ctx.GET("/Products?$apply=" + query)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			rollupCount, err := applyValueCount(resp.Body)
			if err != nil {
				return err
			}

			expectedCount := catCount + 1
			if rollupCount != expectedCount {
				return framework.NewError(fmt.Sprintf(
					"rollup(null, CategoryID) should produce %d rows (%d categories + 1 grand total), got %d",
					expectedCount, catCount, rollupCount,
				))
			}

			return nil
		},
	)

	// Test 4: filter before rollup
	suite.AddTest(
		"test_filter_before_rollup",
		"filter before rollup applies to input rows before aggregation",
		func(ctx *framework.TestContext) error {
			query := url.QueryEscape("filter(Price gt 0)/groupby((rollup(null,CategoryID)),aggregate(Price with sum as Total))")
			resp, err := ctx.GET("/Products?$apply=" + query)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 5: rollup with invalid property returns 400
	suite.AddTest(
		"test_rollup_invalid_property_returns_400",
		"rollup() with non-existent property returns HTTP 400",
		func(ctx *framework.TestContext) error {
			query := url.QueryEscape("groupby((rollup(null,NonExistentProperty)),aggregate(Price with sum as Total))")
			resp, err := ctx.GET("/Products?$apply=" + query)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
