package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ErrorResponses creates the 8.3 Error Responses test suite
func ErrorResponses() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.3 Error Responses",
		"Tests error response format and structure according to OData v4 specification",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ErrorResponse",
	)

	registerErrorResponseTests(suite)
	return suite
}

// validateErrorCodeAndMessage validates that an error object has required 'code' and 'message' fields
// per OData v4 specification. The 'code' must be a non-empty string, and 'message' must be either
// a non-empty string or an object with a non-empty 'value' property.
func validateErrorCodeAndMessage(errorObj map[string]interface{}) error {
	code, ok := errorObj["code"].(string)
	if !ok || code == "" {
		return fmt.Errorf("missing or empty 'code' in error object")
	}

	message, ok := errorObj["message"]
	if !ok {
		return fmt.Errorf("missing 'message' in error object")
	}
	switch msg := message.(type) {
	case string:
		if msg == "" {
			return fmt.Errorf("'message' must not be empty")
		}
	case map[string]interface{}:
		value, ok := msg["value"].(string)
		if !ok || value == "" {
			return fmt.Errorf("message object must have non-empty 'value'")
		}
	default:
		return fmt.Errorf("message must be string or object, got %T", message)
	}
	return nil
}

func registerErrorResponseTests(suite *framework.TestSuite) {
	invalidProductPath := nonExistingEntityPath("Products")

	suite.AddTest(
		"404 error response contains 'error' object",
		"404 error should contain properly structured error object",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}

			if resp.StatusCode != 404 {
				return fmt.Errorf("expected status 404, got %d", resp.StatusCode)
			}

			// Strictly validate error response structure per OData spec
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("response is not valid JSON: %v", err)
			}

			// Error response MUST have an "error" property at the root level
			errorObj, ok := result["error"]
			if !ok {
				return fmt.Errorf("error response must have 'error' property at root level")
			}

			// The "error" property must be an object, not a string or other type
			_, ok = errorObj.(map[string]interface{})
			if !ok {
				return fmt.Errorf("'error' property must be an object")
			}

			return nil
		},
	)

	suite.AddTest(
		"Error object contains 'code' property",
		"Error object must contain 'code' property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("no 'error' object in response")
			}

			code, ok := errorObj["code"].(string)
			if !ok || code == "" {
				return fmt.Errorf("'code' property is not a non-empty string")
			}

			return nil
		},
	)

	suite.AddTest(
		"Error object contains 'message' property",
		"Error object must contain 'message' property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("no 'error' object in response")
			}

			message, ok := errorObj["message"].(string)
			if !ok || message == "" {
				return fmt.Errorf("'message' property is not a non-empty string")
			}

			return nil
		},
	)

	suite.AddTest(
		"Error response has application/json Content-Type",
		"Error response should have application/json Content-Type",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(strings.ToLower(contentType), "application/json") {
				return fmt.Errorf("Content-Type: %s", contentType)
			}

			return nil
		},
	)

	suite.AddTest(
		"Invalid query returns 400 with error object",
		"Invalid filter syntax should return 400 with properly structured error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?" + url.QueryEscape("$filter") + "=invalid%20syntax")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected status 400 for invalid syntax, got %d", resp.StatusCode)
			}

			// Strictly validate error response structure
			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("response is not valid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("error response must have 'error' object property")
			}

			// Verify error has required 'code' property as non-empty string
			code, ok := errorObj["code"].(string)
			if !ok || code == "" {
				return fmt.Errorf("error object must have 'code' property as non-empty string")
			}

			// Verify error has required 'message' property as non-empty string
			message, ok := errorObj["message"].(string)
			if !ok || message == "" {
				return fmt.Errorf("error object must have 'message' property as non-empty string")
			}

			return nil
		},
	)

	suite.AddTest(
		"Unsupported version returns 406 with error",
		"Unsupported OData version should return 406 with error",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products", framework.Header{
				Key:   "OData-MaxVersion",
				Value: "3.0",
			})
			if err != nil {
				return err
			}

			if resp.StatusCode == 406 {
				if !strings.Contains(string(resp.Body), `"error"`) {
					return fmt.Errorf("no error object")
				}
				return nil
			}

			return fmt.Errorf("expected status 406, got %d", resp.StatusCode)
		},
	)

	suite.AddTest(
		"Error response includes OData-Version header",
		"Error response should include OData-Version header",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}

			odataVersion := resp.Headers.Get("OData-Version")
			if odataVersion == "" {
				return fmt.Errorf("no OData-Version header")
			}

			return nil
		},
	)

	suite.AddTest(
		"Error code has no whitespace",
		"Error code should not contain whitespace",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("no 'error' object in response")
			}

			code, ok := errorObj["code"].(string)
			if !ok {
				return fmt.Errorf("no 'code' in error object")
			}

			if strings.ContainsAny(code, " \t\n\r") {
				return fmt.Errorf("error code should not contain whitespace, got: %q", code)
			}

			ctx.Log(fmt.Sprintf("Error code: %s", code))
			return nil
		},
	)

	suite.AddTest(
		"Error target field validation",
		"Error object may include 'target' field for property-specific errors",
		func(ctx *framework.TestContext) error {
			// Try to create a product with invalid data
			invalidPayload := map[string]interface{}{
				"Name":  "Test Product",
				"Price": "not-a-number", // Invalid price type
			}

			resp, err := ctx.POST("/Products", invalidPayload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			// Should return 400 for validation error
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected 400 for invalid data, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("no 'error' object in response")
			}

			if err := validateErrorCodeAndMessage(errorObj); err != nil {
				return err
			}

			// Target is optional, but if present should be a string
			if target, ok := errorObj["target"]; ok {
				targetStr, ok := target.(string)
				if !ok {
					return fmt.Errorf("'target' must be a string if present, got %T", target)
				}
				if targetStr == "" {
					return fmt.Errorf("'target' must not be empty if present")
				}
				ctx.Log(fmt.Sprintf("Error target: %s", targetStr))
			} else {
				ctx.Log("No 'target' field (optional)")
			}

			return nil
		},
	)

	suite.AddTest(
		"Error details array validation",
		"Error object may include 'details' array for multiple errors",
		func(ctx *framework.TestContext) error {
			// Try various operations that might return detailed errors
			resp, err := ctx.GET("/Products?$filter=InvalidFunction()")
			if err != nil {
				return err
			}

			if resp.StatusCode != 400 {
				return fmt.Errorf("expected 400 for invalid filter, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("no 'error' object in response")
			}

			if err := validateErrorCodeAndMessage(errorObj); err != nil {
				return err
			}

			// Details is optional, but if present must be an array
			if details, ok := errorObj["details"]; ok {
				detailsArray, ok := details.([]interface{})
				if !ok {
					return fmt.Errorf("'details' must be an array if present, got %T", details)
				}

				// Each detail should be an object with code and message
				for i, detail := range detailsArray {
					detailObj, ok := detail.(map[string]interface{})
					if !ok {
						return fmt.Errorf("details[%d] must be an object", i)
					}

					// Each detail must have code
					if _, ok := detailObj["code"].(string); !ok {
						return fmt.Errorf("details[%d] must have 'code' as string", i)
					}

					// Each detail must have message
					if _, ok := detailObj["message"].(string); !ok {
						return fmt.Errorf("details[%d] must have 'message' as string", i)
					}
				}

				ctx.Log(fmt.Sprintf("Error has %d details", len(detailsArray)))
			} else {
				ctx.Log("No 'details' array (optional)")
			}

			return nil
		},
	)

	suite.AddTest(
		"Error innererror validation",
		"Error object may include 'innererror' for debugging",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("no 'error' object in response")
			}

			// Innererror is optional, but if present can be object or nested
			if innererror, ok := errorObj["innererror"]; ok {
				// Can be an object or string (implementation-specific)
				switch v := innererror.(type) {
				case map[string]interface{}:
					ctx.Log(fmt.Sprintf("innererror object with %d fields", len(v)))
				case string:
					ctx.Log(fmt.Sprintf("innererror string: %s", v))
				default:
					return fmt.Errorf("innererror has unexpected type: %T", innererror)
				}
			} else {
				ctx.Log("No 'innererror' (optional)")
			}

			return nil
		},
	)

	suite.AddTest(
		"Multiple validation errors in create",
		"Creating entity with multiple invalid fields",
		func(ctx *framework.TestContext) error {
			// Try to create with multiple problems
			invalidPayload := map[string]interface{}{
				"Name": "", // Empty name might be invalid
				// Missing required fields
			}

			resp, err := ctx.POST("/Products", invalidPayload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
			)
			if err != nil {
				return err
			}

			// Should return 400 for validation errors
			if resp.StatusCode != 400 {
				return fmt.Errorf("expected 400 for invalid data, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("no 'error' object in response")
			}

			// Must have code and message
			if err := validateErrorCodeAndMessage(errorObj); err != nil {
				return err
			}

			// Server may include details for multiple validation errors
			ctx.Log("Validation error response properly structured")
			return nil
		},
	)

	suite.AddTest(
		"Error message structure validation",
		"Error message can be string or object with value and lang",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET(invalidProductPath)
			if err != nil {
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("invalid JSON: %v", err)
			}

			errorObj, ok := result["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("no 'error' object in response")
			}

			message := errorObj["message"]
			switch msg := message.(type) {
			case string:
				// Simple string message - most common
				if msg == "" {
					return fmt.Errorf("message string must not be empty")
				}
				ctx.Log(fmt.Sprintf("Simple message: %s", msg))
			case map[string]interface{}:
				// Object with value and optional lang
				value, ok := msg["value"].(string)
				if !ok || value == "" {
					return fmt.Errorf("message object must have 'value' as non-empty string")
				}
				// lang is optional
				if lang, ok := msg["lang"]; ok {
					if langStr, ok := lang.(string); ok {
						ctx.Log(fmt.Sprintf("Message: %s (lang: %s)", value, langStr))
					} else {
						return fmt.Errorf("message.lang must be a string if present")
					}
				} else {
					ctx.Log(fmt.Sprintf("Message object: %s", value))
				}
			default:
				return fmt.Errorf("message must be string or object, got %T", message)
			}

			return nil
		},
	)
}
