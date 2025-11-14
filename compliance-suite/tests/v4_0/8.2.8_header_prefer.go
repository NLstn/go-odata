package v4_0

import (
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// HeaderPrefer creates the 8.2.8 Prefer Header test suite
func HeaderPrefer() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"8.2.8 Prefer Header",
		"Tests Prefer header handling for client preferences like return=minimal and return=representation.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_HeaderPrefer",
	)

	suite.AddTest(
		"test_prefer_return_minimal",
		"Prefer: return=minimal",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Products", map[string]interface{}{
				"Name":       "Prefer Test",
				"Price":      99.99,
				"CategoryID": 1,
			}, framework.Header{
				Key:   "Prefer",
				Value: "return=minimal",
			})
			if err != nil {
				return err
			}

			// Must accept successful creation
			if resp.StatusCode != 201 && resp.StatusCode != 204 && resp.StatusCode != 200 {
				return fmt.Errorf("Expected successful creation (200/201/204), got %d", resp.StatusCode)
			}

			// When return=minimal is honored, should return 204 No Content
			// When not honored, may return 200/201 with content
			// Both are valid per OData spec section 8.2.8
			if resp.StatusCode == 204 {
				// Honored: minimal response, no body
				if len(resp.Body) > 0 {
					return framework.NewError("return=minimal honored but body is not empty")
				}
			}
			// If 200/201, server chose not to honor preference, which is acceptable

			return nil
		},
	)

	suite.AddTest(
		"test_prefer_return_representation",
		"Prefer: return=representation",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.POST("/Products", map[string]interface{}{
				"Name":       "Prefer Test 2",
				"Price":      99.99,
				"CategoryID": 1,
			}, framework.Header{
				Key:   "Prefer",
				Value: "return=representation",
			})
			if err != nil {
				return err
			}

			// Must accept successful creation  
			if resp.StatusCode != 201 && resp.StatusCode != 200 {
				return fmt.Errorf("Expected successful creation (200/201), got %d", resp.StatusCode)
			}

			// When return=representation is honored, should return entity in response body
			// Even if not explicitly honored, 201 Created should include representation
			if resp.StatusCode == 200 || resp.StatusCode == 201 {
				if len(resp.Body) == 0 {
					return framework.NewError("Expected entity representation in response body")
				}
				// Verify it's valid JSON with the entity
				if err := ctx.AssertJSONField(resp, "Name"); err != nil {
					return fmt.Errorf("Response body should contain created entity: %v", err)
				}
			}

			return nil
		},
	)

	return suite
}
