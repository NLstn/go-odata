package v4_01

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// JSONBatch creates the OData JSON Format v4.01 §19 JSON Batch Requests compliance test suite.
//
// JSON batch is a 4.01-only feature.  The suite verifies:
//   - Basic JSON batch acceptance and response format (§19.2 / §19.5)
//   - Per-request id echo
//   - dependsOn failure propagation (424 Failed Dependency)
//   - atomicityGroup rollback-on-failure semantics
//   - Prefer: continue-on-error support
//   - Rejection of JSON batch when the client negotiates OData 4.0 (version gating)
func JSONBatch() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"19 JSON Batch Requests (OData 4.01)",
		"Tests OData 4.01 JSON batch request and response semantics.",
		"https://docs.oasis-open.org/odata/odata-json-format/v4.01/odata-json-format-v4.01.html#sec_BatchRequestsandResponses",
	)

	// ── helpers ──────────────────────────────────────────────────────────────

	// postJSONBatch sends a JSON batch request and returns the decoded response envelope.
	postJSONBatch := func(ctx *framework.TestContext, body map[string]interface{}, headers ...framework.Header) (*framework.HTTPResponse, error) {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal batch body: %w", err)
		}
		hdrs := append([]framework.Header{{Key: "Content-Type", Value: "application/json"}}, headers...)
		return ctx.POSTRaw("/$batch", b, "application/json", hdrs...)
	}

	makeRequests := func(reqs ...map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{"requests": reqs}
	}

	// ── §19.2  JSON batch envelope accepted ──────────────────────────────────

	suite.AddTest(
		"test_json_batch_accepted",
		"JSON batch (application/json) is accepted and returns 200",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, makeRequests(map[string]interface{}{
				"id":     "r1",
				"method": "GET",
				"url":    "Products",
			}))
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// ── §19.5  JSON batch response format ────────────────────────────────────

	suite.AddTest(
		"test_json_batch_response_format",
		"JSON batch response has application/json Content-Type and 'responses' array",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, makeRequests(map[string]interface{}{
				"id":     "r1",
				"method": "GET",
				"url":    "Products",
			}))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			ct := resp.Headers.Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				return framework.NewError(fmt.Sprintf("expected application/json Content-Type, got %q", ct))
			}

			var envelope map[string]interface{}
			if err := json.Unmarshal(resp.Body, &envelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse JSON batch response: %v", err))
			}
			if _, ok := envelope["responses"]; !ok {
				return framework.NewError("JSON batch response must contain a 'responses' member")
			}
			return nil
		},
	)

	// ── §19.5  Per-request id is echoed ──────────────────────────────────────

	suite.AddTest(
		"test_json_batch_id_echoed",
		"Each response object echoes the id of the corresponding request",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, makeRequests(map[string]interface{}{
				"id":     "my-unique-request-id",
				"method": "GET",
				"url":    "Products",
			}))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var envelope map[string]interface{}
			if err := json.Unmarshal(resp.Body, &envelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse response: %v", err))
			}

			responses, ok := envelope["responses"].([]interface{})
			if !ok || len(responses) == 0 {
				return framework.NewError("expected at least one response")
			}

			r0, ok := responses[0].(map[string]interface{})
			if !ok {
				return framework.NewError("response item is not an object")
			}
			if r0["id"] != "my-unique-request-id" {
				return framework.NewError(fmt.Sprintf("expected id %q, got %v", "my-unique-request-id", r0["id"]))
			}
			return nil
		},
	)

	// ── §19.2  Multiple independent requests ─────────────────────────────────

	suite.AddTest(
		"test_json_batch_multiple_requests",
		"Multiple independent requests in a JSON batch all receive responses",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, makeRequests(
				map[string]interface{}{"id": "r1", "method": "GET", "url": "Products"},
				map[string]interface{}{"id": "r2", "method": "GET", "url": "Products"},
			))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var envelope map[string]interface{}
			if err := json.Unmarshal(resp.Body, &envelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse response: %v", err))
			}

			responses, ok := envelope["responses"].([]interface{})
			if !ok {
				return framework.NewError("no 'responses' array")
			}
			if len(responses) != 2 {
				return framework.NewError(fmt.Sprintf("expected 2 responses, got %d", len(responses)))
			}
			return nil
		},
	)

	// ── §19.2  dependsOn failure propagation ─────────────────────────────────

	suite.AddTest(
		"test_json_batch_dependson_failure_propagation",
		"A request whose dependsOn dependency failed receives 424 Failed Dependency",
		func(ctx *framework.TestContext) error {
			// r1 will fail (non-existent entity); r2 depends on r1.
			resp, err := postJSONBatch(ctx, makeRequests(
				map[string]interface{}{
					"id":     "r1",
					"method": "GET",
					"url":    "Products(2147483647)", // highly unlikely to exist
				},
				map[string]interface{}{
					"id":        "r2",
					"method":    "GET",
					"url":       "Products",
					"dependsOn": []string{"r1"},
				},
			))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var envelope map[string]interface{}
			if err := json.Unmarshal(resp.Body, &envelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse response: %v", err))
			}

			responses, ok := envelope["responses"].([]interface{})
			if !ok || len(responses) != 2 {
				return framework.NewError(fmt.Sprintf("expected 2 responses, got %d", len(responses)))
			}

			r1 := responses[0].(map[string]interface{})
			r2 := responses[1].(map[string]interface{})

			if r1["status"] == float64(200) {
				return framework.NewError("r1 should have failed (entity not found)")
			}
			if r2["id"] != "r2" {
				return framework.NewError(fmt.Sprintf("expected r2 id, got %v", r2["id"]))
			}
			if r2["status"] != float64(424) {
				return framework.NewError(fmt.Sprintf("expected r2 status 424, got %v", r2["status"]))
			}
			return nil
		},
	)

	// ── §19.2  atomicityGroup rollback semantics ──────────────────────────────

	suite.AddTest(
		"test_json_batch_atomicitygroup_rollback",
		"When a request in an atomicityGroup fails, all group responses become 424",
		func(ctx *framework.TestContext) error {
			// r1 succeeds (creates entity); r2 fails (non-existent delete).
			// Both are in the same atomicityGroup, so both should end up as 4xx.
			resp, err := postJSONBatch(ctx, makeRequests(
				map[string]interface{}{
					"id":             "r1",
					"method":         "POST",
					"url":            "Products",
					"headers":        map[string]interface{}{"Content-Type": "application/json"},
					"body":           map[string]interface{}{"Name": "AtomicRollbackTest", "Price": 1.0},
					"atomicityGroup": "g1",
				},
				map[string]interface{}{
					"id":             "r2",
					"method":         "DELETE",
					"url":            "Products(2147483647)", // non-existent
					"atomicityGroup": "g1",
				},
			))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var envelope map[string]interface{}
			if err := json.Unmarshal(resp.Body, &envelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse response: %v", err))
			}

			responses, ok := envelope["responses"].([]interface{})
			if !ok || len(responses) != 2 {
				return framework.NewError(fmt.Sprintf("expected 2 responses, got %d", len(responses)))
			}

			r1 := responses[0].(map[string]interface{})
			if r1["status"] != float64(424) {
				return framework.NewError(fmt.Sprintf("expected r1 status 424 (rolled back), got %v", r1["status"]))
			}

			r2 := responses[1].(map[string]interface{})
			if r2["status"] == float64(200) || r2["status"] == float64(201) {
				return framework.NewError(fmt.Sprintf("expected r2 to fail, got status %v", r2["status"]))
			}

			// Verify the entity was NOT persisted.
			listResp, err := ctx.GET("/Products?$filter=Name eq 'AtomicRollbackTest'")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(listResp, 200); err != nil {
				return err
			}
			var listEnvelope map[string]interface{}
			if err := json.Unmarshal(listResp.Body, &listEnvelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse product list: %v", err))
			}
			items, _ := listEnvelope["value"].([]interface{})
			if len(items) > 0 {
				return framework.NewError("entity should have been rolled back but still exists")
			}
			return nil
		},
	)

	// ── §19.2  Prefer: continue-on-error ─────────────────────────────────────

	suite.AddTest(
		"test_json_batch_continue_on_error",
		"With Prefer: continue-on-error, independent requests run even after a failure",
		func(ctx *framework.TestContext) error {
			// r1 fails; r2 is independent (no dependsOn) and should execute.
			resp, err := postJSONBatch(ctx,
				makeRequests(
					map[string]interface{}{
						"id":     "r1",
						"method": "GET",
						"url":    "Products(2147483647)", // will fail
					},
					map[string]interface{}{
						"id":     "r2",
						"method": "GET",
						"url":    "Products",
					},
				),
				framework.Header{Key: "Prefer", Value: "continue-on-error"},
			)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var envelope map[string]interface{}
			if err := json.Unmarshal(resp.Body, &envelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse response: %v", err))
			}

			responses, ok := envelope["responses"].([]interface{})
			if !ok || len(responses) != 2 {
				return framework.NewError(fmt.Sprintf("expected 2 responses with continue-on-error, got %d", len(responses)))
			}

			r2 := responses[1].(map[string]interface{})
			if r2["status"] != float64(200) {
				return framework.NewError(fmt.Sprintf("expected r2 status 200 (continued after error), got %v", r2["status"]))
			}
			return nil
		},
	)

	// ── Version gating: JSON batch MUST NOT apply under OData-MaxVersion: 4.0 ─

	suite.AddTest(
		"test_json_batch_rejected_for_odata_40",
		"JSON batch is rejected when client negotiates OData-MaxVersion: 4.0",
		func(ctx *framework.TestContext) error {
			body, _ := json.Marshal(makeRequests(map[string]interface{}{
				"id":     "r1",
				"method": "GET",
				"url":    "Products",
			}))

			resp, err := ctx.POSTRaw("/$batch", body, "application/json",
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"},
			)
			if err != nil {
				return err
			}
			// JSON batch is a 4.01-only feature; a 4.0 client should get 4xx.
			if resp.StatusCode < 400 {
				return framework.NewError(fmt.Sprintf(
					"expected a 4xx error when using JSON batch with OData-MaxVersion: 4.0, got %d",
					resp.StatusCode,
				))
			}
			return nil
		},
	)

	// ── §19.5  Empty batch returns empty responses array ──────────────────────

	suite.AddTest(
		"test_json_batch_empty_requests",
		"An empty JSON batch returns an empty responses array",
		func(ctx *framework.TestContext) error {
			resp, err := postJSONBatch(ctx, map[string]interface{}{"requests": []interface{}{}})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			var envelope map[string]interface{}
			if err := json.Unmarshal(resp.Body, &envelope); err != nil {
				return framework.NewError(fmt.Sprintf("failed to parse response: %v", err))
			}

			responses, ok := envelope["responses"].([]interface{})
			if !ok {
				// An empty array may be marshalled as nil by some decoders; treat nil as empty.
				if envelope["responses"] == nil {
					return nil
				}
				return framework.NewError(fmt.Sprintf("expected 'responses' array, got %T", envelope["responses"]))
			}
			if len(responses) != 0 {
				return framework.NewError(fmt.Sprintf("expected 0 responses, got %d", len(responses)))
			}
			return nil
		},
	)

	return suite
}
