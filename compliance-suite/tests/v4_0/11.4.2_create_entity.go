package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// CreateEntity creates the 11.4.2 Create an Entity (POST) test suite
func CreateEntity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.2 Create an Entity (POST)",
		"Tests entity creation according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_CreateanEntity",
	)

	// Track created IDs for cleanup
	var createdIDs []string

	// Test 1: POST should return 201 Created
	suite.AddTest(
		"test_post_returns_201",
		"POST entity returns 201 Created",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestProduct1", 99.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if resp.StatusCode != 201 {
				return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if id, ok := result["ID"]; ok {
				parsedID, err := parseEntityID(id)
				if err != nil {
					return err
				}
				createdIDs = append(createdIDs, parsedID)
			}

			return nil
		},
	)

	// Test 2: Location header should be present in 201 response
	suite.AddTest(
		"test_location_header",
		"Location header present in 201 response",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestProduct2", 199.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if resp.StatusCode != 201 {
				return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
			}

			location := resp.Headers.Get("Location")
			if location == "" {
				return fmt.Errorf("Location header missing")
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err == nil {
				if id, ok := result["ID"]; ok {
					parsedID, err := parseEntityID(id)
					if err != nil {
						return err
					}
					createdIDs = append(createdIDs, parsedID)
				}
			}

			return nil
		},
	)

	// Test 3: Created entity should be returned in response body
	suite.AddTest(
		"test_entity_in_response",
		"Created entity returned in response body",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestProduct3", 299.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if resp.StatusCode != 201 {
				return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if _, ok := result["ID"]; !ok {
				return fmt.Errorf("response missing 'ID' field")
			}

			name, ok := result["Name"].(string)
			if !ok || name != "ComplianceTestProduct3" {
				return fmt.Errorf("response missing or incorrect 'Name' field")
			}

			if id, ok := result["ID"]; ok {
				parsedID, err := parseEntityID(id)
				if err != nil {
					return err
				}
				createdIDs = append(createdIDs, parsedID)
			}

			return nil
		},
	)

	// Test 4: POST with Prefer: return=minimal should return 201 Created with empty body
	suite.AddTest(
		"test_prefer_minimal",
		"POST with Prefer: return=minimal returns 201",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestProduct4", 399.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Prefer", Value: "return=minimal"})
			if err != nil {
				return err
			}

			// Per OData v4.01 spec, POST with return=minimal should return 201 Created with empty body
			if resp.StatusCode != 201 {
				return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
			}

			location := resp.Headers.Get("Location")
			if location == "" {
				return fmt.Errorf("Location header missing")
			}

			// Extract ID from location for cleanup
			if strings.Contains(location, "Products(") {
				start := strings.Index(location, "Products(")
				if start != -1 {
					idPart := location[start+len("Products("):]
					end := strings.Index(idPart, ")")
					if end != -1 {
						createdIDs = append(createdIDs, idPart[:end])
					}
				}
			}

			return nil
		},
	)

	// Test 5: OData-EntityId header should be present in 201 response
	suite.AddTest(
		"test_odata_entityid_header",
		"OData-EntityId header in 201 response",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ComplianceTestProduct5", 499.99)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Prefer", Value: "return=minimal"})
			if err != nil {
				return err
			}

			// Per OData v4.01 spec, POST with return=minimal should return 201 Created with empty body
			if resp.StatusCode != 201 {
				return fmt.Errorf("expected status 201, got %d", resp.StatusCode)
			}

			entityID := resp.Headers.Get("OData-EntityId")
			if entityID == "" {
				return fmt.Errorf("OData-EntityId header missing")
			}

			return nil
		},
	)

	return suite
}
