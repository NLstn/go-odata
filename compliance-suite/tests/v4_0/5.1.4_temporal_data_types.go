package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// TemporalDataTypes creates the 5.1.4 Temporal Data Types test suite
func TemporalDataTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.4 Temporal Data Types",
		"Validates handling of OData temporal types including Edm.Date, Edm.TimeOfDay, and Edm.Duration in filters and metadata.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_datetime_offset_support",
		"Edm.DateTimeOffset type is supported",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=year(CreatedAt) eq 2024")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_date_literal",
		"Date literal in YYYY-MM-DD format",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=date(CreatedAt) eq 2024-01-15")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_time_literal",
		"Time literal in HH:MM:SS format",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=time(CreatedAt) eq 14:30:00")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_date_comparison",
		"Date comparison with gt/lt operators",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=date(CreatedAt) gt 2024-01-01")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_time_comparison",
		"Time comparison with operators",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=time(CreatedAt) lt 12:00:00")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_date_time_combination",
		"Combine date() and time() functions",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=date(CreatedAt) eq 2024-01-15 and time(CreatedAt) gt 10:00:00")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
