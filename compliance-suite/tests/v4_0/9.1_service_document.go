package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ServiceDocument creates the 9.1 Service Document test suite
func ServiceDocument() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"9.1 Service Document",
		"Tests that the service document is properly formatted according to OData v4 specification, including required metadata, entity sets, and singletons.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ServiceDocument",
	)

	// Test 1: Service document should be accessible at root
	suite.AddTest(
		"test_service_document_accessible",
		"Service document accessible at /",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: Service document should have @odata.context
	suite.AddTest(
		"test_odata_context",
		"Service document contains @odata.context",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			// Check for @odata.context field
			if err := ctx.AssertJSONField(resp, "@odata.context"); err != nil {
				return err
			}

			// Verify it contains $metadata
			body := string(resp.Body)
			if !strings.Contains(body, "$metadata") {
				return framework.NewError("@odata.context must reference $metadata")
			}

			return nil
		},
	)

	// Test 3: Service document should have value array
	suite.AddTest(
		"test_value_array",
		"Service document contains value array",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/")
			if err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "value")
		},
	)

	// Test 4: Service document entity sets should have required properties
	suite.AddTest(
		"test_entity_set_properties",
		"Entity sets have required properties (name, kind, url)",
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
				return framework.NewError("Service document must contain at least one item in value array")
			}

			// Check first item has required properties
			firstItem, ok := valueArray[0].(map[string]interface{})
			if !ok {
				return framework.NewError("Items in value array must be objects")
			}

			if _, ok := firstItem["name"]; !ok {
				return framework.NewError("Entity set items must have 'name' property")
			}

			if _, ok := firstItem["kind"]; !ok {
				return framework.NewError("Entity set items must have 'kind' property")
			}

			if _, ok := firstItem["url"]; !ok {
				return framework.NewError("Entity set items must have 'url' property")
			}

			return nil
		},
	)

	// Test 5: Entity set kind should be "EntitySet"
	suite.AddTest(
		"test_entity_set_kind",
		"Entity sets have kind=\"EntitySet\"",
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

			// Check for at least one EntitySet
			foundEntitySet := false
			for _, item := range valueArray {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				kind, ok := itemMap["kind"]
				if ok && kind == "EntitySet" {
					foundEntitySet = true
					break
				}
			}

			if !foundEntitySet {
				return framework.NewError("Service document must contain at least one item with kind=\"EntitySet\"")
			}

			return nil
		},
	)

	// Test 6: Singleton should have kind="Singleton" (if any)
	suite.AddTest(
		"test_singleton_kind",
		"Singletons have kind=\"Singleton\" (if present)",
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

			// If there are singletons, verify they have the correct kind
			for _, item := range valueArray {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				kind, ok := itemMap["kind"]
				if ok && kind == "Singleton" {
					// Found a singleton, verify it has a name
					if _, hasName := itemMap["name"]; !hasName {
						return framework.NewError("Singletons must have a 'name' property")
					}
				}
			}

			// If no singletons found, test passes
			return nil
		},
	)

	return suite
}
