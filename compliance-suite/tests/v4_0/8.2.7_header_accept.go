package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderAccept creates the 8.2.7 Accept Header test suite
func HeaderAccept() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.7 Header Accept",
		"Tests Accept header content negotiation and handling of different media types",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderAccept",
	)

	registerHeaderAcceptTests(suite)
	return suite
}

func registerHeaderAcceptTests(suite *framework.TestSuite) {
	suite.AddTest(
		"Accept application/json",
		"Accept: application/json should return JSON",
		func(ctx *framework.TestContext) error {
			// Use a valid product key for the compliance server (GUID keys)
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/json",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Accept */* returns JSON",
		"Accept: */* should return JSON (default)",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "*/*",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Unsupported Accept returns 406",
		"Unsupported media type should return 406 or support XML",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/xml",
			})
			if err != nil {
				return err
			}

			switch resp.StatusCode {
			case 406:
				return nil
			case 200:
				// Server might support XML
				contentType := resp.Headers.Get("Content-Type")
				if strings.Contains(strings.ToLower(contentType), "application/xml") {
					return nil
				}
				return fmt.Errorf("expected 406 or XML response, got JSON with HTTP 200")
			default:
				return fmt.Errorf("expected HTTP 406, got %d", resp.StatusCode)
			}
		},
	)

	suite.AddTest(
		"Accept with odata.metadata parameter",
		"Accept with parameters should be supported",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "application/json;odata.metadata=minimal",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Accept quality values respected",
		"Accept with quality values should prefer higher quality",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}
			resp, err := ctx.GET(productPath, framework.Header{
				Key:   "Accept",
				Value: "text/plain;q=0.5, application/json;q=1.0",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("expected JSON, got %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Metadata accepts application/xml",
		"Metadata document should support application/xml",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{
				Key:   "Accept",
				Value: "application/xml",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/xml") {
				return fmt.Errorf("expected Content-Type application/xml, got %s", contentType)
			}

			return nil
		},
	)
}
