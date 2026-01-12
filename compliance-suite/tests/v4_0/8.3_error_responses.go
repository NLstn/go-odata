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
}
