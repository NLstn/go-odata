package v4_01

import (
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// WildcardSelectExpand creates the 11.2.5.14 Wildcard $select and $expand test suite for OData v4.01.
func WildcardSelectExpand() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.14 Wildcard $select and $expand",
		"Validates that $select=* and $expand=* work correctly per OData v4.01 sections 5.1.3 and 5.1.4, and that wildcard behavior is version-gated to OData v4.01.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptionselect",
	)

	// ---- $select=* tests ----

	suite.AddTest(
		"test_select_wildcard_with_maxversion_401",
		"$select=* with OData-MaxVersion: 4.01 returns HTTP 200 with all structural properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GETWithHeaders("/Products?$top=1&$select=*", map[string]string{
				"OData-MaxVersion": "4.01",
			})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return fmt.Errorf("with OData-MaxVersion:4.01, $select=* should succeed: %w", err)
			}
			var body map[string]interface{}
			if err := ctx.GetJSON(resp, &body); err != nil {
				return err
			}
			entities, ok := body["value"].([]interface{})
			if !ok || len(entities) == 0 {
				return fmt.Errorf("expected non-empty value array")
			}
			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}
			// Key property must always be present
			if _, ok := product["ID"]; !ok {
				return fmt.Errorf("expected 'ID' to be present with $select=*")
			}
			// Structural properties must be present
			if _, ok := product["Name"]; !ok {
				return fmt.Errorf("expected 'Name' to be present with $select=*")
			}
			return nil
		},
	)

	suite.AddTest(
		"test_select_wildcard_rejected_with_maxversion_40",
		"$select=* with OData-MaxVersion: 4.0 returns HTTP 400 (wildcard is OData v4.01-only)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GETWithHeaders("/Products?$top=1&$select=*", map[string]string{
				"OData-MaxVersion": "4.0",
			})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusBadRequest); err != nil {
				return fmt.Errorf("with OData-MaxVersion:4.0, $select=* should be rejected (400): %w", err)
			}
			return nil
		},
	)

	// ---- $expand=* tests ----

	suite.AddTest(
		"test_expand_wildcard_with_maxversion_401",
		"$expand=* with OData-MaxVersion: 4.01 returns HTTP 200 and expands all navigation properties",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GETWithHeaders("/Products?$top=1&$expand=*", map[string]string{
				"OData-MaxVersion": "4.01",
			})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return fmt.Errorf("with OData-MaxVersion:4.01, $expand=* should succeed: %w", err)
			}
			var body map[string]interface{}
			if err := ctx.GetJSON(resp, &body); err != nil {
				return err
			}
			entities, ok := body["value"].([]interface{})
			if !ok || len(entities) == 0 {
				return fmt.Errorf("expected non-empty value array")
			}
			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}
			// Product has a navigation property to Descriptions — it must be expanded.
			if _, ok := product["Descriptions"]; !ok {
				return fmt.Errorf("expected 'Descriptions' to be expanded with $expand=*")
			}
			return nil
		},
	)

	suite.AddTest(
		"test_expand_wildcard_rejected_with_maxversion_40",
		"$expand=* with OData-MaxVersion: 4.0 returns HTTP 400 (wildcard is OData v4.01-only)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GETWithHeaders("/Products?$top=1&$expand=*", map[string]string{
				"OData-MaxVersion": "4.0",
			})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusBadRequest); err != nil {
				return fmt.Errorf("with OData-MaxVersion:4.0, $expand=* should be rejected (400): %w", err)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_select_wildcard_with_expand_combination",
		"$select=* combined with $expand works correctly in OData v4.01",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GETWithHeaders("/Products?$top=1&$select=*&$expand=Descriptions", map[string]string{
				"OData-MaxVersion": "4.01",
			})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}
			var body map[string]interface{}
			if err := ctx.GetJSON(resp, &body); err != nil {
				return err
			}
			entities, ok := body["value"].([]interface{})
			if !ok || len(entities) == 0 {
				return fmt.Errorf("expected non-empty value array")
			}
			product, ok := entities[0].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected product to be an object")
			}
			if _, ok := product["ID"]; !ok {
				return fmt.Errorf("expected 'ID' to be present")
			}
			if _, ok := product["Name"]; !ok {
				return fmt.Errorf("expected 'Name' to be present with $select=*")
			}
			if _, ok := product["Descriptions"]; !ok {
				return fmt.Errorf("expected 'Descriptions' to be expanded")
			}
			return nil
		},
	)

	return suite
}
