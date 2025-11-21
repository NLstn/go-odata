package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// Introduction creates the 1.1 Introduction test suite
func Introduction() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"1.1 Introduction",
		"Tests basic service requirements defined in the OData v4 introduction section, including service availability, protocol version support, and basic resource accessibility.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_Introduction",
	)

	// Test 1: Service root must be accessible
	suite.AddTest(
		"test_service_root_accessible",
		"Service root is accessible",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: Service document must be valid JSON
	suite.AddTest(
		"test_service_document_json",
		"Service document returns valid JSON",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			if !ctx.IsValidJSON(resp) {
				return framework.NewError("Service document is not valid JSON")
			}
			return nil
		},
	)

	// Test 3: Metadata document must be accessible
	suite.AddTest(
		"test_metadata_accessible",
		"Metadata document is accessible",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 4: Service document must contain @odata.context
	suite.AddTest(
		"test_service_context",
		"Service document contains @odata.context",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			return ctx.AssertJSONField(resp, "@odata.context")
		},
	)

	// Test 5: Service must support at least one entity set
	suite.AddTest(
		"test_has_entity_sets",
		"Service exposes at least one entity set",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			value, ok := data["value"]
			if !ok {
				return framework.NewError("Service document must contain value array")
			}

			valueArray, ok := value.([]interface{})
			if !ok {
				return framework.NewError("value must be an array")
			}

			if len(valueArray) == 0 {
				return framework.NewError("Service document must contain at least one entity set in value array")
			}

			return nil
		},
	)

	// Test 6: Service must be accessible via HTTP
	suite.AddTest(
		"test_http_access",
		"Service is accessible via HTTP",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
