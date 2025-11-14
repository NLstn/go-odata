package v4_0

import (
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// FilterDateFunctions creates the 11.3.2 Date/Time Functions test suite
func FilterDateFunctions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.2 Date and Time Functions in $filter",
		"Tests date/time functions (year, month, day, hour, minute, second, date, time, now) in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations",
	)

	// Test 1: year function
	suite.AddTest(
		"test_year_function",
		"year() function extracts year from date",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("year(CreatedAt) eq 2024")
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

	// Test 2: month function
	suite.AddTest(
		"test_month_function",
		"month() function extracts month from date",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("month(CreatedAt) eq 1")
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

	// Test 3: day function
	suite.AddTest(
		"test_day_function",
		"day() function extracts day from date",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("day(CreatedAt) eq 15")
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

	// Test 4: hour function
	suite.AddTest(
		"test_hour_function",
		"hour() function extracts hour from datetime",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("hour(CreatedAt) lt 12")
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

	// Test 5: minute function
	suite.AddTest(
		"test_minute_function",
		"minute() function extracts minute from datetime",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("minute(CreatedAt) eq 30")
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

	// Test 6: second function
	suite.AddTest(
		"test_second_function",
		"second() function extracts second from datetime",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("second(CreatedAt) lt 60")
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

	// Test 7: date function
	suite.AddTest(
		"test_date_function",
		"date() function extracts date portion",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("date(CreatedAt) eq 2024-01-15")
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

	// Test 8: time function
	suite.AddTest(
		"test_time_function",
		"time() function extracts time portion",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("time(CreatedAt) lt 12:00:00")
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

	// Test 9: now function
	suite.AddTest(
		"test_now_function",
		"now() function returns current datetime",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("CreatedAt lt now()")
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

	// Test 10: Combined date functions
	suite.AddTest(
		"test_combined_date_functions",
		"Combined date functions work together",
		func(ctx *framework.TestContext) error {
			filter := url.QueryEscape("year(CreatedAt) eq 2024 and month(CreatedAt) ge 6")
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
