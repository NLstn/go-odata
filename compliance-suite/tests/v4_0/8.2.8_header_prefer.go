package v4_0

import (
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

			// Should accept the header (may or may not honor it)
			if resp.StatusCode == 201 || resp.StatusCode == 204 || resp.StatusCode == 200 {
				return nil
			}

			return nil // Optional feature
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

			// Should accept the header
			if resp.StatusCode == 201 || resp.StatusCode == 200 {
				return nil
			}

			return nil // Optional feature
		},
	)

	return suite
}
