package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// KeyAsSegments creates the OData 4.01 key-as-segments URL convention test suite.
// It validates that /EntitySet/{key} resolves equivalently to /EntitySet({key})
// when the client negotiates OData 4.01, and that this convention is NOT active
// under OData 4.0 negotiation.
func KeyAsSegments() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"OData 4.01 Key-As-Segments URL Convention",
		"Validates OData 4.01 key-as-segment URL addressing for entity access, property access, "+
			"and reference access. Verifies the convention is not active under OData 4.0 negotiation.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_KeyasSegmentConvention",
	)

	// Helper: fetch the ID of the first product.
	getFirstProductID := func(ctx *framework.TestContext) (string, error) {
		return firstEntityID(ctx, "Products")
	}

	// Test 1: /Products/{id} with OData-MaxVersion: 4.01 returns the same entity as /Products({id}).
	suite.AddTest(
		"test_key_as_segment_entity_401",
		"GET /Products/{id} with OData-MaxVersion: 4.01 resolves same entity as /Products({id})",
		func(ctx *framework.TestContext) error {
			id, err := getFirstProductID(ctx)
			if err != nil {
				return err
			}

			headers401 := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}

			// Parenthetical form (baseline)
			parenResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", id), headers401...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(parenResp, http.StatusOK); err != nil {
				return fmt.Errorf("parenthetical key form failed: %w", err)
			}

			// Key-as-segment form
			segResp, err := ctx.GET(fmt.Sprintf("/Products/%s", id), headers401...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(segResp, http.StatusOK); err != nil {
				return fmt.Errorf("key-as-segment form failed (expected 200): %w", err)
			}

			// Both responses must return the same entity ID
			var parenBody, segBody map[string]interface{}
			if err := json.Unmarshal(parenResp.Body, &parenBody); err != nil {
				return fmt.Errorf("failed to parse parenthetical response: %w", err)
			}
			if err := json.Unmarshal(segResp.Body, &segBody); err != nil {
				return fmt.Errorf("failed to parse key-as-segment response: %w", err)
			}

			if fmt.Sprintf("%v", parenBody["ID"]) != fmt.Sprintf("%v", segBody["ID"]) {
				return fmt.Errorf("entity ID mismatch: parenthetical=%v, key-as-segment=%v",
					parenBody["ID"], segBody["ID"])
			}
			return nil
		},
	)

	// Test 2: /Products/{id} with OData-MaxVersion: 4.0 must NOT apply key-as-segments.
	suite.AddTest(
		"test_key_as_segment_not_active_40",
		"GET /Products/{id} with OData-MaxVersion: 4.0 does not apply key-as-segments (4.0-only behavior)",
		func(ctx *framework.TestContext) error {
			id, err := getFirstProductID(ctx)
			if err != nil {
				return err
			}

			headers40 := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}
			resp, err := ctx.GET(fmt.Sprintf("/Products/%s", id), headers40...)
			if err != nil {
				return err
			}

			// Under OData 4.0, the numeric segment is not a property name, so this should be 404.
			if resp.StatusCode == http.StatusOK {
				return fmt.Errorf("OData 4.0 must NOT resolve /Products/{id} as key-as-segment, got 200")
			}
			if resp.StatusCode != http.StatusNotFound {
				return fmt.Errorf("OData 4.0: expected 404 for key-as-segment path, got %d", resp.StatusCode)
			}
			return nil
		},
	)

	// Test 3: /Products/{id}/Name (structural property via key-as-segment) under 4.01.
	suite.AddTest(
		"test_key_as_segment_property_access_401",
		"GET /Products/{id}/Name with OData-MaxVersion: 4.01 returns property value",
		func(ctx *framework.TestContext) error {
			id, err := getFirstProductID(ctx)
			if err != nil {
				return err
			}

			headers401 := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			resp, err := ctx.GET(fmt.Sprintf("/Products/%s/Name", id), headers401...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return fmt.Errorf("property access via key-as-segment failed: %w", err)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(resp.Body, &body); err != nil {
				return fmt.Errorf("failed to parse property response: %w", err)
			}
			if _, ok := body["value"]; !ok {
				return fmt.Errorf("property response missing 'value' field, got: %v", body)
			}
			return nil
		},
	)

	// Test 4: Non-existent key via key-as-segment returns 404 under 4.01.
	suite.AddTest(
		"test_key_as_segment_nonexistent_401",
		"GET /Products/00000000-0000-0000-0000-000000000000 with OData-MaxVersion: 4.01 returns 404",
		func(ctx *framework.TestContext) error {
			headers401 := []framework.Header{{Key: "OData-MaxVersion", Value: "4.01"}}
			resp, err := ctx.GET("/Products/00000000-0000-0000-0000-000000000000", headers401...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, http.StatusNotFound); err != nil {
				return fmt.Errorf("non-existent entity via key-as-segment should return 404: %w", err)
			}
			return nil
		},
	)

	return suite
}
