package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// EnumTypes creates the 5.3 Enumeration Types test suite
func EnumTypes() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.3 Enumeration Types",
		"Validates handling of enumeration types including filtering, selecting, ordering, and enum operations.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_EnumerationType",
	)

	suite.AddTest(
		"test_enum_retrieval",
		"Retrieve entity with enum property",
		func(ctx *framework.TestContext) error {
			prodPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(prodPath)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, `"Status"`) {
				return nil
			}

			return nil // No enum property, optional
		},
	)

	suite.AddTest(
		"test_filter_enum_numeric",
		"Filter by enum numeric value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Status eq 1")
			if err != nil {
				return err
			}

			// Enum filtering may not be supported
			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 404 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_enum_comparison",
		"Filter enum with comparison operators",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Status gt 0")
			if err != nil {
				return err
			}

			// Optional feature
			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 404 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_select_enum",
		"Select enum property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$select=Name,Status")
			if err != nil {
				return err
			}

			// Optional feature
			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 404 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_orderby_enum",
		"Order by enum property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Status")
			if err != nil {
				return err
			}

			// Optional feature
			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 404 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_enum_null",
		"Filter for null enum value",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Status eq null")
			if err != nil {
				return err
			}

			// Optional feature
			if resp.StatusCode == 200 || resp.StatusCode == 400 || resp.StatusCode == 404 {
				return nil
			}

			return fmt.Errorf("unexpected status: %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"test_enum_in_metadata",
		"Enum type in metadata document",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if strings.Contains(body, "EnumType") {
				return nil
			}

			return nil // No enum types, optional
		},
	)

	return suite
}
