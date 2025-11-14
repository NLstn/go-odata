package v4_0

import (
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// AdvancedApply creates the 11.2.5.4.1 Advanced $apply Transformations test suite
func AdvancedApply() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.4.1 Advanced $apply Transformations",
		"Tests advanced $apply query option transformations including nested groupby, multiple aggregations, filter before/after aggregation, and complex transformation pipelines.",
		"https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/odata-data-aggregation-ext-v4.0.html",
	)

	// Test 1: Multiple aggregations in single aggregate
	suite.AddTest(
		"test_multiple_aggregations",
		"Multiple aggregations in single statement",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("aggregate(Price with sum as TotalPrice,Price with average as AvgPrice,Price with max as MaxPrice)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: groupby with multiple properties
	suite.AddTest(
		"test_groupby_multiple_properties",
		"groupby with multiple grouping properties",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("groupby((CategoryID,Status))")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 3: groupby with multiple aggregate methods
	suite.AddTest(
		"test_groupby_with_multiple_aggregates",
		"groupby with multiple aggregation methods",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("groupby((CategoryID),aggregate(Price with sum as Total,Price with average as Average,$count as Count))")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 4: Filter before aggregation
	suite.AddTest(
		"test_filter_before_aggregate",
		"Filter before aggregation",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("filter(Price gt 50)/aggregate(Price with sum as Total)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 5: Filter before groupby
	suite.AddTest(
		"test_filter_before_groupby",
		"Filter before groupby transformation",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("filter(Price gt 50)/groupby((CategoryID),aggregate($count as Count))")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 6: Multiple transformations in sequence
	suite.AddTest(
		"test_transformation_pipeline",
		"Transformation pipeline",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("filter(Price gt 10)/groupby((CategoryID))/filter($count gt 1)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 7: Aggregate with countdistinct
	suite.AddTest(
		"test_countdistinct",
		"Aggregate with countdistinct",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("aggregate(CategoryID with countdistinct as UniqueCategories)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 8: groupby followed by filter
	suite.AddTest(
		"test_groupby_then_filter",
		"Filter after groupby/aggregate",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("groupby((CategoryID),aggregate(Price with sum as Total))/filter(Total gt 100)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 9: Min and max aggregation together
	suite.AddTest(
		"test_min_max_aggregate",
		"Min and max aggregation together",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("aggregate(Price with min as MinPrice,Price with max as MaxPrice)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 10: $apply with $top
	suite.AddTest(
		"test_apply_with_top",
		"$apply works with $top",
		func(ctx *framework.TestContext) error {
			applyFilter := url.QueryEscape("groupby((CategoryID),aggregate($count as Count))")
			resp, err := ctx.GET("/Products?$apply=" + applyFilter + "&$top=2")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 11: $apply with $orderby
	suite.AddTest(
		"test_apply_with_orderby",
		"$apply works with $orderby",
		func(ctx *framework.TestContext) error {
			applyFilter := url.QueryEscape("groupby((CategoryID),aggregate(Price with sum as Total))")
			orderby := url.QueryEscape("Total desc")
			resp, err := ctx.GET("/Products?$apply=" + applyFilter + "&$orderby=" + orderby)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 12: Average aggregation
	suite.AddTest(
		"test_average_aggregation",
		"Average aggregation method",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("aggregate(Price with average as AvgPrice)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 13: Invalid aggregation method
	suite.AddTest(
		"test_invalid_aggregation_method",
		"Invalid aggregation method returns error",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("aggregate(Price with invalid as Result)")
			resp, err := ctx.GET("/Products?$apply=" + filter)
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 400)
		},
	)

	return suite
}
