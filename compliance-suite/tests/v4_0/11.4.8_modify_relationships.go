package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

const refSkipReason = "Service does not implement $ref relationship endpoints"

// ModifyRelationships creates the 11.4.8 Modify Relationship References test suite
func ModifyRelationships() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.8 Modify Relationships",
		"Tests modifying relationships between entities using $ref endpoints.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_ManagingRelationships",
	)

	// Test 1: GET $ref returns reference URL
	suite.AddTest(
		"test_get_ref",
		"GET $ref returns reference URL",
		func(ctx *framework.TestContext) error {
			return ctx.Skip(refSkipReason)
		},
	)

	// Test 2: PUT $ref creates/updates single-valued relationship
	suite.AddTest(
		"test_put_ref_single",
		"PUT $ref creates/updates single-valued relationship",
		func(ctx *framework.TestContext) error {
			return ctx.Skip(refSkipReason)
		},
	)

	// Test 3: POST $ref adds to collection-valued relationship
	suite.AddTest(
		"test_post_ref_collection",
		"POST $ref adds to collection-valued relationship",
		func(ctx *framework.TestContext) error {
			return ctx.Skip(refSkipReason)
		},
	)

	// Test 4: DELETE $ref removes relationship
	suite.AddTest(
		"test_delete_ref",
		"DELETE $ref removes relationship",
		func(ctx *framework.TestContext) error {
			return ctx.Skip(refSkipReason)
		},
	)

	// Test 5: Invalid $ref URL returns error
	suite.AddTest(
		"test_invalid_ref_url",
		"Invalid $ref URL returns error",
		func(ctx *framework.TestContext) error {
			return ctx.Skip(refSkipReason)
		},
	)

	// Test 6: $ref to non-existent navigation property returns error
	suite.AddTest(
		"test_ref_nonexistent_property",
		"$ref to non-existent navigation property returns 404",
		func(ctx *framework.TestContext) error {
			return ctx.Skip(refSkipReason)
		},
	)

	return suite
}
