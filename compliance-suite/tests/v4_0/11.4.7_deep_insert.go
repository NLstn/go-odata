package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// DeepInsert creates the 11.4.7 Deep Insert test suite
func DeepInsert() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.7 Deep Insert",
		"Tests creating entities with related entities in a single request (deep insert).",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_CreateRelatedEntitiesWhenCreatinganE",
	)

	// Test 1: Deep insert returns 201 Created
	suite.AddTest(
		"test_deep_insert_status",
		"Deep insert returns 201 Created",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Deep Insert Test", 50.00)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 201)
		},
	)

	// Test 2: Deep insert with invalid data returns error
	suite.AddTest(
		"test_deep_insert_invalid_data",
		"Deep insert with invalid data returns 400",
		func(ctx *framework.TestContext) error {
			categoryID, err := firstEntityID(ctx, "Categories")
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"Name":       "Test",
				"Price":      "invalid_price",
				"CategoryID": categoryID,
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 3: Deep insert response includes created entity
	suite.AddTest(
		"test_deep_insert_response_body",
		"Deep insert response includes created entity",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Response Test Product", 75.00)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Prefer", Value: "return=representation"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			return ctx.AssertJSONField(resp, "Name")
		},
	)

	// Test 4: Deep insert returns Location header
	suite.AddTest(
		"test_deep_insert_location_header",
		"Deep insert returns Location header",
		func(ctx *framework.TestContext) error {
			payload, err := buildProductPayload(ctx, "Location Test Product", 80.00)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			location := resp.Headers.Get("Location")
			if location == "" {
				return framework.NewError("Location header not found")
			}

			return nil
		},
	)

	return suite
}
