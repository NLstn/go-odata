package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ODataBind creates the 11.4.2.1 @odata.bind Annotation test suite
func ODataBind() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.2.1 @odata.bind Annotation",
		"Tests binding navigation properties using @odata.bind annotation according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_BindingaNavigationProperty",
	)

	// Test 1: POST with @odata.bind using relative URL
	suite.AddTest(
		"test_bind_post_relative_url",
		"POST with @odata.bind (relative URL)",
		func(ctx *framework.TestContext) error {
			categoryIDs, err := fetchEntityIDs(ctx, "Categories", 1)
			if err != nil {
				return err
			}
			payload := map[string]interface{}{
				"Name":                "BindTestProduct1",
				"Price":               99.99,
				"Category@odata.bind": fmt.Sprintf("Categories(%s)", categoryIDs[0]),
				"Status":              1,
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			// Verify CategoryID was set to 1
			var data map[string]interface{}
			if err := ctx.GetJSON(resp, &data); err != nil {
				return err
			}

			categoryID, err := parseEntityID(data["CategoryID"])
			if err != nil {
				return err
			}
			if categoryID != categoryIDs[0] {
				return framework.NewError(fmt.Sprintf("Expected CategoryID=%s, got %s", categoryIDs[0], categoryID))
			}

			return nil
		},
	)

	// Test 2: POST with @odata.bind referencing non-existent entity should fail
	suite.AddTest(
		"test_bind_post_nonexistent",
		"POST with @odata.bind to non-existent entity returns 400",
		func(ctx *framework.TestContext) error {
			const nonexistentCategoryID = "00000000-0000-0000-0000-000000000000"
			payload := map[string]interface{}{
				"Name":                "BindTestProduct3",
				"Price":               299.99,
				"Category@odata.bind": fmt.Sprintf("Categories(%s)", nonexistentCategoryID),
				"Status":              1,
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 3: POST with invalid @odata.bind format should fail
	suite.AddTest(
		"test_bind_post_invalid_format",
		"POST with invalid @odata.bind format returns 400",
		func(ctx *framework.TestContext) error {
			payload := map[string]interface{}{
				"Name":                "BindTestProduct5",
				"Price":               499.99,
				"Category@odata.bind": "InvalidFormat",
				"Status":              1,
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 4: PATCH with @odata.bind to update navigation property
	suite.AddTest(
		"test_bind_patch_update",
		"PATCH with @odata.bind updates navigation property",
		func(ctx *framework.TestContext) error {
			categoryIDs, err := fetchEntityIDs(ctx, "Categories", 2)
			if err != nil {
				return err
			}
			// First create a product with Category 1
			createPayload := map[string]interface{}{
				"Name":                "BindTestProduct6",
				"Price":               599.99,
				"Category@odata.bind": fmt.Sprintf("Categories(%s)", categoryIDs[0]),
				"Status":              1,
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

			productID, err := parseEntityID(createData["ID"])
			if err != nil {
				return err
			}

			// Update to Category 2 using @odata.bind
			updatePayload := map[string]interface{}{
				"Category@odata.bind": fmt.Sprintf("Categories(%s)", categoryIDs[1]),
			}

			updateResp, err := ctx.PATCH(fmt.Sprintf("/Products(%s)", productID), updatePayload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(updateResp, 204); err != nil {
				if err := ctx.AssertStatusCode(updateResp, 200); err != nil {
					return err
				}
			}

			// Fetch the product and verify CategoryID is 2
			getResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", productID))
			if err != nil {
				return err
			}

			var getData map[string]interface{}
			if err := ctx.GetJSON(getResp, &getData); err != nil {
				return err
			}

			categoryID, err := parseEntityID(getData["CategoryID"])
			if err != nil {
				return err
			}
			if categoryID != categoryIDs[1] {
				return framework.NewError(fmt.Sprintf("Expected CategoryID=%s, got %s", categoryIDs[1], categoryID))
			}

			return nil
		},
	)

	// Test 5: PATCH with @odata.bind and other properties together
	suite.AddTest(
		"test_bind_patch_mixed",
		"PATCH with @odata.bind and other properties",
		func(ctx *framework.TestContext) error {
			categoryIDs, err := fetchEntityIDs(ctx, "Categories", 2)
			if err != nil {
				return err
			}
			// Create a product
			createPayload := map[string]interface{}{
				"Name":                "BindTestProduct7",
				"Price":               699.99,
				"Category@odata.bind": fmt.Sprintf("Categories(%s)", categoryIDs[0]),
				"Status":              1,
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

			productID, err := parseEntityID(createData["ID"])
			if err != nil {
				return err
			}

			// Update both price and category
			updatePayload := map[string]interface{}{
				"Price":               799.99,
				"Category@odata.bind": fmt.Sprintf("Categories(%s)", categoryIDs[1]),
			}

			updateResp, err := ctx.PATCH(fmt.Sprintf("/Products(%s)", productID), updatePayload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(updateResp, 204); err != nil {
				if err := ctx.AssertStatusCode(updateResp, 200); err != nil {
					return err
				}
			}

			// Fetch and verify both updates
			getResp, err := ctx.GET(fmt.Sprintf("/Products(%s)", productID))
			if err != nil {
				return err
			}

			var getData map[string]interface{}
			if err := ctx.GetJSON(getResp, &getData); err != nil {
				return err
			}

			price, priceOk := getData["Price"].(float64)
			categoryID, err := parseEntityID(getData["CategoryID"])
			if err != nil {
				return err
			}

			if !priceOk || price != 799.99 || categoryID != categoryIDs[1] {
				return framework.NewError(fmt.Sprintf("Expected Price=799.99 and CategoryID=%s, got Price=%v CategoryID=%s", categoryIDs[1], price, categoryID))
			}

			return nil
		},
	)

	return suite
}
