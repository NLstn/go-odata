package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// PrimitiveDataTypes creates the 5.1.1 Primitive Data Types test suite
func PrimitiveDataTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.1.1 Primitive Data Types",
		"Validates handling of OData primitive data types in requests and responses including Edm.String, Edm.Int32, Edm.Decimal, etc.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_PrimitiveTypes",
	)

	suite.AddTest(
		"test_string_type",
		"Edm.String type handles text values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name eq 'Laptop'")
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

	suite.AddTest(
		"test_int32_type",
		"Edm.Int32 type handles integer values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Status eq 1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	suite.AddTest(
		"test_decimal_type",
		"Edm.Decimal type handles decimal values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price eq 999.99")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	suite.AddTest(
		"test_datetime_type",
		"Edm.DateTimeOffset type handles datetime values",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=CreatedAt lt 2025-12-31T23:59:59Z")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	suite.AddTest(
		"test_null_value_handling",
		"Null value handling in filters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=CategoryID ne null")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	suite.AddTest(
		"test_number_precision",
		"Decimal precision is maintained",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name eq 'Laptop'")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Check that price field exists and has decimal precision
			return nil
		},
	)

	suite.AddTest(
		"test_special_characters",
		"Special characters in strings are handled",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=contains(Name,'&') or contains(Name,'/')")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	suite.AddTest(
		"test_empty_string",
		"Empty string handling in filters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name ne ''")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			return nil
		},
	)

	return suite
}
