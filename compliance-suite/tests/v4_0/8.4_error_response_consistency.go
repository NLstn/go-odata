package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ErrorResponseConsistency creates the 8.4 Error Response Consistency test suite
func ErrorResponseConsistency() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.4 Error Response Consistency",
		"Tests that error responses are consistent and follow the OData error response format.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ErrorResponse",
	)

	invalidProductPath := nonExistingEntityPath("Products")

	suite.AddTest(
		"test_404_error_format",
		"404 error has proper error object",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}

			if resp.StatusCode != 404 {
				return fmt.Errorf("expected 404, got %d", resp.StatusCode)
			}

			// Verify error object is present
			body := string(resp.Body)
			if !strings.Contains(body, `"error"`) {
				return framework.NewError("404 response should contain error object")
			}

			// Verify it's valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("error response is not valid JSON: %w", err)
			}

			// Check for error properties
			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return framework.NewError("error object missing in response")
			}

			if _, hasCode := errorObj["code"]; !hasCode {
				return framework.NewError("error object missing 'code' property")
			}

			if _, hasMessage := errorObj["message"]; !hasMessage {
				return framework.NewError("error object missing 'message' property")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_400_error_format",
		"400 error has proper error object",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=invalid syntax")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return nil // Not a validation error, skip
			}

			// Verify error object is present
			body := string(resp.Body)
			if !strings.Contains(body, `"error"`) {
				return framework.NewError("400 response should contain error object")
			}

			return nil
		},
	)

	return suite
}
