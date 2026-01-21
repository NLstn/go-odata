package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ReturningResults creates the 11.4.12 Returning Results from Data Modification test suite
func ReturningResults() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.12 Returning Results from Modifications",
		"Tests returning entities from POST, PATCH, PUT operations with Prefer header.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html",
	)

	// Test 1: POST with return=minimal returns 201 with no body
	suite.AddTest(
		"test_post_return_minimal",
		"POST with return=minimal returns 201 with no body",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Name":   "Minimal Return Test",
				"Price":  99.99,
				"Status": 1,
			}

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Prefer", Value: "return=minimal"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			if len(resp.Body) != 0 {
				return framework.NewError("Expected empty response body")
			}

			return ctx.AssertHeader(resp, "Preference-Applied", "return=minimal")
		},
	)

	// Test 2: POST with return=representation returns 201 with entity
	suite.AddTest(
		"test_post_return_representation",
		"POST with return=representation returns 201 with entity",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Name":   "Representation Return Test",
				"Price":  149.99,
				"Status": 1,
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

			return ctx.AssertBodyContains(resp, "Representation Return Test")
		},
	)

	// Test 3: PATCH with return=representation returns entity
	suite.AddTest(
		"test_patch_return_representation",
		"PATCH with return=representation returns 200 with entity",
		func(ctx *framework.TestContext) error {
			// First create an entity
			createPayload := map[string]interface{}{
				"Name":   "Patch Test",
				"Price":  50.00,
				"Status": 1,
			}

			createResp, err := ctx.POST("/Products", createPayload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(createResp, 201); err != nil {
				return err
			}

			var createData map[string]interface{}
			if err := ctx.GetJSON(createResp, &createData); err != nil {
				return err
			}

			id, ok := createData["ID"].(string)
			if !ok {
				return framework.NewError("Could not extract entity ID")
			}

			// Now PATCH with return=representation
			updatePayload := map[string]interface{}{
				"Price": 75.00,
			}

			patchResp, err := ctx.PATCH(fmt.Sprintf("/Products(%s)", id), updatePayload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Prefer", Value: "return=representation"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(patchResp, 200); err != nil {
				return err
			}

			return ctx.AssertBodyContains(patchResp, `"Price":75`)
		},
	)

	// Test 4: PATCH with return=minimal returns 204 No Content
	suite.AddTest(
		"test_patch_return_minimal",
		"PATCH with return=minimal returns 204 with no body",
		func(ctx *framework.TestContext) error {
			// First create an entity
			createPayload := map[string]interface{}{
				"Name":   "Patch Minimal Test",
				"Price":  60.00,
				"Status": 1,
			}

			createResp, err := ctx.POST("/Products", createPayload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(createResp, 201); err != nil {
				return err
			}

			var createData map[string]interface{}
			if err := ctx.GetJSON(createResp, &createData); err != nil {
				return err
			}

			id, ok := createData["ID"].(string)
			if !ok {
				return framework.NewError("Could not extract entity ID")
			}

			// Now PATCH with return=minimal
			updatePayload := map[string]interface{}{
				"Price": 85.00,
			}

			patchResp, err := ctx.PATCH(fmt.Sprintf("/Products(%s)", id), updatePayload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Prefer", Value: "return=minimal"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(patchResp, 204); err != nil {
				return err
			}

			if len(patchResp.Body) != 0 {
				return framework.NewError("Expected empty response body")
			}

			return ctx.AssertHeader(patchResp, "Preference-Applied", "return=minimal")
		},
	)

	return suite
}
