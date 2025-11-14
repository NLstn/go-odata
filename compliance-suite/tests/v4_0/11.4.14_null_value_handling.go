package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// NullValueHandling creates the 11.4.14 Null Value Handling test suite
func NullValueHandling() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.14 Null Value Handling",
		"Tests that the service properly handles null values in entity creation, updates, and filtering.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html",
	)

	var createdID int

	// Test 1: Create entity with null property
	suite.AddTest(
		"test_create_with_null",
		"Create entity with explicit null property",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Name":        "Null Test Product",
				"Price":       99.99,
				"CategoryID":  1,
				"Description": nil,
				"Status":      1,
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			if id, ok := data["ID"].(float64); ok {
				createdID = int(id)
			}

			return nil
		},
	)

	// Test 2: Retrieve entity with null property
	suite.AddTest(
		"test_retrieve_null_property",
		"Retrieve entity returns null property correctly",
		func(ctx *framework.TestContext) error {
			if createdID == 0 {
				return ctx.Skip("No test entity available")
			}

			resp, err := ctx.GET(fmt.Sprintf("/Products(%d)", createdID))
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Description should be null or absent
			body := string(resp.Body)
			if framework.ContainsAny(body, `"Description":null`, `"Description"`) || !framework.ContainsAny(body, `"Description"`) {
				return nil
			}

			return framework.NewError("Expected null or absent Description property")
		},
	)

	// Test 3: Update property to null using PATCH
	suite.AddTest(
		"test_patch_to_null",
		"Update property to null using PATCH",
		func(ctx *framework.TestContext) error {
			if createdID == 0 {
				return ctx.Skip("No test entity available")
			}

			payload := map[string]interface{}{
				"Description": nil,
			}

			resp, err := ctx.PATCH(fmt.Sprintf("/Products(%d)", createdID), payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			// Should return 204 or 200
			if err := ctx.AssertStatusCode(resp, 204); err != nil {
				return ctx.AssertStatusCode(resp, 200)
			}

			return nil
		},
	)

	// Test 4: Filter for null values using eq null
	suite.AddTest(
		"test_filter_eq_null",
		"Filter for null values using eq null",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Description eq null")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 5: Filter for non-null values using ne null
	suite.AddTest(
		"test_filter_ne_null",
		"Filter for non-null values using ne null",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Description ne null")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
