package v4_0

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// ComplexOrderBy creates the 5.2.2 Complex Type Ordering test suite
func ComplexOrderBy() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.2.2 Complex Type Ordering",
		"Validates that nested complex properties can be used in $orderby expressions.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ComplexType",
	)

	suite.AddTest(
		"test_orderby_nested_complex_property",
		"Order by nested complex property",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$orderby=Dimensions/Length desc")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				// Skip if server doesn't support ordering by complex properties
				if resp.StatusCode == 400 || resp.StatusCode == 501 {
					return framework.NewError("Server does not support $orderby on complex properties")
				}
				return err
			}

			var result map[string]interface{}
			if err := json.Unmarshal(resp.Body, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			// Verify ordering
			value, ok := result["value"].([]interface{})
			if !ok || len(value) == 0 {
				return nil // Empty result is acceptable
			}

			// Verify the entities are ordered by Dimensions/Length descending
			var prev *float64
			for i, raw := range value {
				entity, ok := raw.(map[string]interface{})
				if !ok {
					return framework.NewError("Invalid entity format")
				}

				var lengthPtr *float64
				if dimsRaw, ok := entity["Dimensions"]; ok && dimsRaw != nil {
					if dims, ok := dimsRaw.(map[string]interface{}); ok {
						if l, ok := dims["Length"]; ok && l != nil {
							if lf, ok := l.(float64); ok {
								lengthPtr = &lf
							}
						}
					}
				}

				// If previous length is set and current length is set, ensure non-increasing order
				if prev != nil && lengthPtr != nil {
					if *lengthPtr > *prev+1e-9 { // allow tiny float tolerance
						return fmt.Errorf("entity at index %d has Length %v which is greater than previous %v", i, *lengthPtr, *prev)
					}
				}

				// If previous length is set and current length is nil, that's fine (nulls last)
				// If previous is nil and current is set, that's a violation (nulls should come last)
				if prev == nil && lengthPtr != nil && i > 0 {
					return fmt.Errorf("non-null Length encountered after null at index %d", i)
				}

				// Update prev only when we have a non-nil length; once we hit nil, the rest must be nil as well
				if lengthPtr != nil {
					prev = lengthPtr
				} else {
					// Set prev to nil to indicate we've entered the null tail
					prev = nil
				}
			}

			return nil
		},
	)

	return suite
}
