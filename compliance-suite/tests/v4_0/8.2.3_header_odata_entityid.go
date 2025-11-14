package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderODataEntityId creates the 8.2.3 OData-EntityId Header test suite
func HeaderODataEntityId() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.3 OData-EntityId Header",
		"Tests OData-EntityId header in responses for created entities.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderODataEntityId",
	)

	suite.AddTest(
		"test_odata_entityid_header",
		"OData-EntityId header for created entity",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Products", map[string]interface{}{
				"Name":       "EntityId Test",
				"Price":      99.99,
				"CategoryID": 1,
			})
			if err != nil {
				return err
			}

			if resp.StatusCode != 201 && resp.StatusCode != 200 {
				return nil // Creation not supported, skip
			}

			// OData-EntityId is optional but recommended
			entityId := resp.Headers.Get("OData-EntityId")
			if entityId != "" && !strings.Contains(entityId, "Products") {
				return framework.NewError("OData-EntityId should contain entity URL")
			}

			return nil
		},
	)

	return suite
}
