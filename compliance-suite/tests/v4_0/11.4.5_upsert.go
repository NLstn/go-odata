package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// Upsert creates the 11.4.5 Upsert Operations test suite
func Upsert() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.5 Upsert Operations",
		"Tests upsert (PUT) operations according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_UpsertanEntity",
	)

	// Test 1: PUT to existing entity updates it
	suite.AddTest(
		"test_put_update_existing",
		"PUT updates existing entity",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Original Product", 99.99)
			if err != nil {
				return err
			}
			// First create an entity
			createResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				return fmt.Errorf("failed to create entity (status: %d)", createResp.StatusCode)
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			id, err := parseEntityID(createResult["ID"])
			if err != nil {
				return err
			}

			// Now PUT to update it
			putResp, err := ctx.PUT(fmt.Sprintf("/Products(%s)", id), map[string]interface{}{
				"ID":          id,
				"Name":        "Updated Product",
				"Price":       199.99,
				"Description": "Updated via PUT",
			})
			if err != nil {
				return err
			}

			if putResp.StatusCode != 200 && putResp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204, got %d", putResp.StatusCode)
			}

			return nil
		},
	)

	// Test 2: PUT to non-existent entity creates it (if supported)
	suite.AddTest(
		"test_put_create_nonexistent",
		"PUT to non-existent entity (insert)",
		func(ctx *framework.TestContext) error {
			const nonexistentID = "00000000-0000-0000-0000-000000000000"
			resp, err := ctx.PUT(fmt.Sprintf("/Products(%s)", nonexistentID), map[string]interface{}{
				"ID":          nonexistentID,
				"Name":        "Upserted Product",
				"Price":       299.99,
				"Description": "Created via PUT",
			})
			if err != nil {
				return err
			}

			// Should return 201 (created) or 404 (not supported)
			if resp.StatusCode != 201 && resp.StatusCode != 404 {
				return fmt.Errorf("expected status 201 or 404, got %d", resp.StatusCode)
			}

			return nil
		},
	)

	// Test 3: PUT with missing required fields should fail
	suite.AddTest(
		"test_put_incomplete_entity",
		"PUT with incomplete entity returns 400",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Test Product", 50.00)
			if err != nil {
				return err
			}
			// First create an entity
			createResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				return fmt.Errorf("failed to create entity")
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			id, err := parseEntityID(createResult["ID"])
			if err != nil {
				return err
			}

			// Try PUT with incomplete data
			putResp, err := ctx.PUT(fmt.Sprintf("/Products(%s)", id), map[string]interface{}{
				"Name": "Incomplete",
			})
			if err != nil {
				return err
			}

			// Should return 400 or accept it (200/204) if server fills in defaults
			if putResp.StatusCode != 400 && putResp.StatusCode != 200 && putResp.StatusCode != 204 {
				return fmt.Errorf("expected status 400, 200, or 204, got %d", putResp.StatusCode)
			}

			return nil
		},
	)

	// Test 4: PUT should return proper headers
	suite.AddTest(
		"test_put_response_headers",
		"PUT response includes proper headers",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Header Test", 75.00)
			if err != nil {
				return err
			}
			// First create an entity
			createResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				return fmt.Errorf("failed to create entity")
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			id, err := parseEntityID(createResult["ID"])
			if err != nil {
				return err
			}

			// PUT to update
			putResp, err := ctx.PUT(fmt.Sprintf("/Products(%s)", id), map[string]interface{}{
				"ID":          id,
				"Name":        "Header Test Product",
				"Price":       99.99,
				"Description": "Testing headers",
			})
			if err != nil {
				return err
			}

			if putResp.StatusCode != 200 && putResp.StatusCode != 204 {
				return fmt.Errorf("expected status 200 or 204, got %d", putResp.StatusCode)
			}

			// Check for OData-Version header
			version := putResp.Headers.Get("OData-Version")
			if version == "" {
				return fmt.Errorf("OData-Version header missing")
			}

			return nil
		},
	)

	// Test 5: PUT with If-Match header
	suite.AddTest(
		"test_put_if_match",
		"PUT with If-Match for optimistic concurrency",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "ETag Test", 85.00)
			if err != nil {
				return err
			}
			// First create an entity
			createResp, err := ctx.POST("/Products", payload)
			if err != nil {
				return err
			}

			if createResp.StatusCode != 201 {
				return fmt.Errorf("failed to create entity")
			}

			var createResult map[string]interface{}
			if err := json.Unmarshal(createResp.Body, &createResult); err != nil {
				return fmt.Errorf("failed to parse created entity: %w", err)
			}

			id, err := parseEntityID(createResult["ID"])
			if err != nil {
				return err
			}

			// Get the entity to retrieve ETag
			getResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", id))
			if err != nil {
				return err
			}

			etag := getResp.Headers.Get("ETag")
			if etag != "" {
				// If ETag is supported, try PUT with If-Match
				putResp, err := ctx.PUT(fmt.Sprintf("/Products(%s)", id), map[string]interface{}{
					"ID":          id,
					"Name":        "Conditional Update",
					"Price":       149.99,
					"Description": "With ETag",
				}, framework.Header{Key: "If-Match", Value: etag})
				if err != nil {
					return err
				}

				if putResp.StatusCode != 200 && putResp.StatusCode != 204 {
					return fmt.Errorf("expected status 200 or 204, got %d", putResp.StatusCode)
				}
			}
			// ETags are optional, so pass if not supported

			return nil
		},
	)

	return suite
}
