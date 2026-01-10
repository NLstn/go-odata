package v4_0

import (
	"fmt"
	"strings"

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
		"OData-MaxVersion 4.0 returns OData-Version 4.0",
		"Request with OData-MaxVersion: 4.0 should return OData-Version: 4.0",
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

			// Per OData v4 spec section 8.2.6: Services respond with the maximum supported version
			// that is less than or equal to the requested OData-MaxVersion
			if strings.TrimSpace(odataVersion) != "4.0" {
				return fmt.Errorf("expected OData-Version: 4.0, got %q (spec requires response version <= OData-MaxVersion)", odataVersion)
			}

			return nil
		},
	)

	suite.AddTest(
		"OData-MaxVersion 4.01 returns OData-Version 4.01",
		"Request with OData-MaxVersion: 4.01 should return OData-Version: 4.01",
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

			// Per OData v4 spec: Services respond with the maximum supported version
			// that is less than or equal to the requested OData-MaxVersion
			if strings.TrimSpace(odataVersion) != "4.01" {
				return fmt.Errorf("expected OData-Version: 4.01, got %q", odataVersion)
			}

			return nil
		},
	)

	suite.AddTest(
		"Unsupported OData-MaxVersion returns 406",
		"Request with OData-MaxVersion below 4.0 should return 406 Not Acceptable",
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

			// OData spec says service should reject if it cannot satisfy the version constraint
			if resp.StatusCode != 406 {
				return fmt.Errorf("expected HTTP 406 Not Acceptable, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	suite.AddTest(
		"Request without OData-MaxVersion returns highest supported version",
		"Request without OData-MaxVersion should default to highest supported (4.01)",
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

			// Without OData-MaxVersion, service should return its highest supported version
			if strings.TrimSpace(odataVersion) != "4.01" {
				return fmt.Errorf("expected OData-Version: 4.01 (highest supported), got %q", odataVersion)
			}

			return nil
		},
	)

	suite.AddTest(
		"Invalid OData-MaxVersion format returns 406 or ignored",
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

			// Implementation may either reject invalid format (406) or ignore it (200)
			if resp.StatusCode == 406 || resp.StatusCode == 200 {
				return nil
			}

			return fmt.Errorf("expected HTTP 406 or 200, got %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"OData-MaxVersion higher than supported returns highest supported",
		"Request with OData-MaxVersion: 5.0 should return OData-Version: 4.01 (our max)",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "OData-MaxVersion",
				Value: "5.0",
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

			// When client accepts higher version than we support, respond with our max
			if strings.TrimSpace(odataVersion) != "4.01" {
				return fmt.Errorf("expected OData-Version: 4.01, got %q", odataVersion)
			}

			return nil
		},
	)

	suite.AddTest(
		"Service document respects OData-MaxVersion 4.0",
		"Service document request with OData-MaxVersion: 4.0 should return OData-Version: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/", framework.Header{
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
			if strings.TrimSpace(odataVersion) != "4.0" {
				return fmt.Errorf("expected OData-Version: 4.0, got %q", odataVersion)
			}

			return nil
		},
	)

	suite.AddTest(
		"Metadata document respects OData-MaxVersion 4.0",
		"Metadata document request with OData-MaxVersion: 4.0 should return OData-Version: 4.0",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{
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
			if strings.TrimSpace(odataVersion) != "4.0" {
				return fmt.Errorf("expected OData-Version: 4.0, got %q", odataVersion)
			}

			return nil
		},
	)
}
