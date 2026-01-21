package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// DataValidation creates the 11.4.15 Data Validation and Constraints test suite
func DataValidation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.15 Data Validation",
		"Tests that the service enforces data validation rules, required fields, and constraints on entity creation and updates.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html",
	)

	// Test 1: Missing required field returns error
	suite.AddTest(
		"test_missing_required_field",
		"Missing required field returns 400",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Price":      99.99,
				"CategoryID": 1,
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 2: Invalid data type returns error
	suite.AddTest(
		"test_invalid_data_type",
		"Invalid data type returns 400",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Name":       "Test Product",
				"Price":      "not-a-number",
				"CategoryID": 1,
				"Status":     1,
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 3: Malformed JSON returns error
	suite.AddTest(
		"test_malformed_json",
		"Malformed JSON returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POSTRaw("/Products", []byte(`{"Name":"Test","Price":99.99,}`), "application/json")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 4: Content-Type header missing or incorrect
	suite.AddTest(
		"test_missing_content_type",
		"Missing Content-Type returns 415",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POSTRaw("/Products", []byte(`{"Name":"Test","Price":99.99,"CategoryID":1,"Status":1}`), "")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 415); err != nil {
				return err
			}

			if err := ctx.AssertJSONField(resp, "error"); err != nil {
				return err
			}

			var payload map[string]interface{}
			if err := ctx.GetJSON(resp, &payload); err != nil {
				return err
			}

			errorValue, ok := payload["error"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("error field is not an object")
			}

			message, ok := errorValue["message"].(string)
			if !ok || strings.TrimSpace(message) == "" {
				return fmt.Errorf("error.message is missing or empty")
			}

			return nil
		},
	)

	return suite
}
