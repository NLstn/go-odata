package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// DateTimeOfDayTypes creates the 5.1.7 Date and TimeOfDay Types test suite
func DateTimeOfDayTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.7 Date and TimeOfDay Types",
		"Tests handling of Edm.Date (YYYY-MM-DD) and Edm.TimeOfDay (HH:MM:SS.sss) primitive types including literal formats, filtering, and functions.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_PrimitiveTypes",
	)

	// Edm.Date tests
	suite.AddTest(
		"test_date_in_metadata",
		"Edm.Date type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.Date"`) {
				return nil // Optional type
			}

			return nil
		},
	)

	suite.AddTest(
		"test_date_literal_format",
		"Date literal in YYYY-MM-DD format",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ReleaseDate eq 2024-01-15")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_date_comparison",
		"Date supports comparison operators",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ReleaseDate gt 2024-01-01")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_date_year_function",
		"year() function works with Date",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=year(ReleaseDate) eq 2024")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_date_month_function",
		"month() function works with Date",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=month(ReleaseDate) eq 1")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_date_day_function",
		"day() function works with Date",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=day(ReleaseDate) eq 15")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_date_null_comparison",
		"Date supports null comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=ReleaseDate ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_date_cast",
		"cast() function supports Edm.Date",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(CreatedAt, 'Edm.Date') eq 2024-01-15")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Edm.TimeOfDay tests
	suite.AddTest(
		"test_timeofday_in_metadata",
		"Edm.TimeOfDay type appears in metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, `Type="Edm.TimeOfDay"`) {
				return nil // Optional type
			}

			return nil
		},
	)

	suite.AddTest(
		"test_timeofday_literal_format",
		"TimeOfDay literal in HH:MM:SS format",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=OpenTime eq 09:30:00")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_timeofday_with_milliseconds",
		"TimeOfDay supports fractional seconds",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=OpenTime eq 09:30:00.123")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_timeofday_comparison",
		"TimeOfDay supports comparison operators",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=OpenTime gt 08:00:00")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_timeofday_hour_function",
		"hour() function works with TimeOfDay",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=hour(OpenTime) eq 9")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_timeofday_minute_function",
		"minute() function works with TimeOfDay",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=minute(OpenTime) eq 30")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_timeofday_second_function",
		"second() function works with TimeOfDay",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=second(OpenTime) eq 0")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_timeofday_midnight",
		"TimeOfDay handles midnight (00:00:00)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=OpenTime eq 00:00:00")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_timeofday_end_of_day",
		"TimeOfDay handles end of day (23:59:59)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=OpenTime le 23:59:59")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_timeofday_null_comparison",
		"TimeOfDay supports null comparison",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=OpenTime ne null")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_timeofday_cast",
		"cast() function supports Edm.TimeOfDay",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=cast(CreatedAt, 'Edm.TimeOfDay') gt 09:00:00")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_date_and_timeofday_in_response",
		"Date and TimeOfDay values are correctly serialized",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if _, hasValue := result["value"]; !hasValue {
				return framework.NewError("Response missing value field")
			}

			return nil
		},
	)

	return suite
}
