package v4_01

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderODataVersion creates the 8.2.6 Header OData-Version test suite for OData 4.01 negotiation.
func HeaderODataVersion() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.6 Header OData-Version",
		"Tests OData 4.01 version negotiation behavior for default responses and explicit 4.01 negotiation.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_HeaderODataVersion",
	)

	suite.AddTest(
		"test_default_response_version_401",
		"Service document defaults to OData-Version 4.01 when no OData-MaxVersion is supplied",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if version != "4.01" {
				return framework.NewError(fmt.Sprintf("expected OData-Version 4.01 by default, got %q", version))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_explicit_maxversion_401_response",
		"Service document responds with OData-Version 4.01 when OData-MaxVersion is 4.01",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/", framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if version != "4.01" {
				return framework.NewError(fmt.Sprintf("expected OData-Version 4.01, got %q", version))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_entity_collection_default_version_401",
		"Entity collection responses default to OData-Version 4.01 when unconstrained",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$top=1")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			version := strings.TrimSpace(resp.Headers.Get("OData-Version"))
			if version != "4.01" {
				return framework.NewError(fmt.Sprintf("expected OData-Version 4.01 on default entity response, got %q", version))
			}

			return nil
		},
	)

	return suite
}
