package v4_01

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderMaxVersion creates the 8.2.9 Header OData-MaxVersion test suite for OData 4.01 behavior.
func HeaderMaxVersion() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.9 Header OData-MaxVersion",
		"Tests OData 4.01 max-version negotiation for default and higher-than-supported requests.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderODataMaxVersion",
	)

	suite.AddTest(
		"test_request_without_maxversion_returns_highest_supported",
		"Requests without OData-MaxVersion return the highest supported version 4.01",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			resp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if version != "4.01" {
				return framework.NewError(fmt.Sprintf("expected highest supported OData-Version 4.01, got %q", version))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_higher_maxversion_returns_highest_supported",
		"Requests with OData-MaxVersion above 4.01 return the highest supported version 4.01",
		func(ctx *framework.TestContext) error {
			productPath, err := firstEntityPath(ctx, "Products")
			if err != nil {
				return err
			}

			resp, err := ctx.GET(productPath, framework.Header{Key: "OData-MaxVersion", Value: "5.0"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if version != "4.01" {
				return framework.NewError(fmt.Sprintf("expected highest supported OData-Version 4.01, got %q", version))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_service_document_higher_maxversion_returns_highest_supported",
		"Service document requests with OData-MaxVersion above 4.01 return OData-Version 4.01",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/", framework.Header{Key: "OData-MaxVersion", Value: "5.0"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if version != "4.01" {
				return framework.NewError(fmt.Sprintf("expected OData-Version 4.01 on service document, got %q", version))
			}

			return nil
		},
	)

	return suite
}
