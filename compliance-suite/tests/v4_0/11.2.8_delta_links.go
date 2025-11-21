package v4_0

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// DeltaLinks creates the 11.2.8 Delta Links test suite
func DeltaLinks() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.8 Delta Links",
		"Tests delta link support for tracking changes according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_RequestingChanges",
	)

	var currentToken string
	var createdProductID string

	// Test 1: Initial delta request
	suite.AddTest(
		"test_initial_delta_request",
		"Initial delta request applies track-changes preference",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{Key: "Prefer", Value: "odata.track-changes"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Check for Preference-Applied header
			prefApplied := resp.Headers.Get("Preference-Applied")
			if !strings.Contains(strings.ToLower(prefApplied), "odata.track-changes") {
				return framework.NewError("Preference-Applied header missing odata.track-changes")
			}

			// Parse delta link
			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			deltaLink, ok := data["@odata.deltaLink"].(string)
			if !ok {
				return framework.NewError("Delta link not found in initial response")
			}

			token, err := extractDeltaToken(deltaLink)
			if err != nil {
				return err
			}
			currentToken = token

			return nil
		},
	)

	// Test 2: Delta feed includes newly created entity
	suite.AddTest(
		"test_delta_includes_creation",
		"Delta feed includes newly created entity",
		func(ctx *framework.TestContext) error {
			if currentToken == "" {
				return ctx.Skip("No delta token available from previous test")
			}

			// Create a new product
			payload, err := buildProductPayload(ctx, "Track Changes Widget", 42.42)
			if err != nil {
				return err
			}

			createResp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(createResp, 201); err != nil {
				return err
			}

			// Parse created product ID
			var createdData map[string]interface{}
			if err := ctx.GetJSON(createResp, &createdData); err != nil {
				return err
			}

			id, err := parseEntityID(createdData["ID"])
			if err != nil {
				return err
			}
			createdProductID = id

			// Get delta feed
			deltaQuery := url.Values{}
			deltaQuery.Set("$deltatoken", currentToken)
			deltaResp, err := ctx.GET("/Products?" + deltaQuery.Encode())
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(deltaResp, 200); err != nil {
				return err
			}

			// Check if created entity is in delta
			if err := ctx.AssertBodyContains(deltaResp, "Track Changes Widget"); err != nil {
				return err
			}

			// Update current token
			var deltaData map[string]interface{}
			if err := ctx.GetJSON(deltaResp, &deltaData); err != nil {
				return err
			}

			if deltaLink, ok := deltaData["@odata.deltaLink"].(string); ok {
				token, err := extractDeltaToken(deltaLink)
				if err == nil {
					currentToken = token
				}
			}

			return nil
		},
	)

	// Test 3: Delta feed reports deleted entity
	suite.AddTest(
		"test_delta_includes_deletion",
		"Delta feed reports deleted entity",
		func(ctx *framework.TestContext) error {
			if createdProductID == "" {
				return ctx.Skip("No product created in previous test")
			}

			if currentToken == "" {
				return ctx.Skip("No delta token available")
			}

			// Verify entity exists before attempting deletion
			checkResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", createdProductID))
			if err != nil {
				return err
			}

			if checkResp.StatusCode == 404 {
				return ctx.Skip("Product no longer exists (may have been deleted in previous test run)")
			}

			// Delete the product
			deleteResp, err := ctx.DELETE(fmt.Sprintf("/Products(%s)", createdProductID))
			if err != nil {
				return err
			}

			// Accept both 204 (deleted) or 404 (already deleted)
			if deleteResp.StatusCode != 204 && deleteResp.StatusCode != 404 {
				return fmt.Errorf("expected 204 or 404, got %d", deleteResp.StatusCode)
			}

			// If entity was already deleted, skip the test
			if deleteResp.StatusCode == 404 {
				return ctx.Skip("Product was already deleted")
			}

			// Get delta feed
			deltaQuery := url.Values{}
			deltaQuery.Set("$deltatoken", currentToken)
			deltaResp, err := ctx.GET("/Products?" + deltaQuery.Encode())
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(deltaResp, 200); err != nil {
				return err
			}

			// Check for @odata.removed marker
			if err := ctx.AssertBodyContains(deltaResp, "@odata.removed"); err != nil {
				return err
			}

			// Check for the deleted ID
			if err := ctx.AssertBodyContains(deltaResp, fmt.Sprintf(`"ID":"%s"`, createdProductID)); err != nil {
				return err
			}

			return nil
		},
	)

	return suite
}

func extractDeltaToken(deltaLink string) (string, error) {
	parsed, err := url.Parse(deltaLink)
	if err != nil {
		return "", fmt.Errorf("invalid delta link: %w", err)
	}
	token := parsed.Query().Get("$deltatoken")
	if token == "" {
		token = parsed.Query().Get("%24deltatoken")
	}
	if token == "" {
		return "", framework.NewError("Delta link missing $deltatoken parameter")
	}
	return token, nil
}
