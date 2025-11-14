package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderMaxVersion creates the 8.2.9 OData-MaxVersion Header test suite
func HeaderMaxVersion() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.9 Header OData-MaxVersion",
		"Tests OData-MaxVersion header handling for version negotiation",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataMaxVersion",
	)

	registerHeaderMaxVersionTests(suite)
	return suite
}

func registerHeaderMaxVersionTests(suite *framework.TestSuite) {
	suite.AddTest(
		"OData-MaxVersion 4.0 respected",
		"Request with OData-MaxVersion: 4.0 should succeed",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "OData-MaxVersion",
				Value: "4.0",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return fmt.Errorf("no OData-Version header in response")
			}

			return nil
		},
	)

	suite.AddTest(
		"OData-MaxVersion 4.01 accepted",
		"Request with OData-MaxVersion: 4.01 should succeed",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "OData-MaxVersion",
				Value: "4.01",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return fmt.Errorf("no OData-Version header in response")
			}

			return nil
		},
	)

	suite.AddTest(
		"Unsupported OData-MaxVersion returns 400 or 406",
		"Request with unsupported OData-MaxVersion should return error or accept lower",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "OData-MaxVersion",
				Value: "3.0",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode == 400 || resp.StatusCode == 406 || resp.StatusCode == 200 {
				return nil
			}

			return fmt.Errorf("expected HTTP 400/406 or 200, got %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"Request without OData-MaxVersion succeeds",
		"Request without OData-MaxVersion should default to highest supported",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return fmt.Errorf("no OData-Version header")
			}

			return nil
		},
	)

	suite.AddTest(
		"Invalid OData-MaxVersion format returns 400 or ignored",
		"Invalid OData-MaxVersion format should return error or be ignored",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "OData-MaxVersion",
				Value: "invalid",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode == 400 || resp.StatusCode == 406 || resp.StatusCode == 200 {
				return nil
			}

			return fmt.Errorf("expected HTTP 400/406 or 200, got %d", resp.StatusCode)
		},
	)
}
