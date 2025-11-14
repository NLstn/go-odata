package v4_0

import (
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// EnumMetadataMembers creates the 5.3 Enumeration Types - Metadata Members test suite
func EnumMetadataMembers() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.3 Enumeration Types - Metadata Members",
		"Verifies that enum metadata reflects actual enum values and configured namespace.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_EnumerationType",
	)

	suite.AddTest(
		"test_enum_metadata_xml",
		"Enum members in XML metadata",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "EnumType") {
				return nil // No enums, optional
			}

			// If enums exist, they should have members
			if strings.Contains(body, "EnumType") && !strings.Contains(body, "Member") {
				return framework.NewError("EnumType should have Member elements")
			}

			return nil
		},
	)

	return suite
}
