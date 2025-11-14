package v4_0

import (
	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// UnicodeStrings creates the 7.1.1 Unicode and Internationalization test suite
func UnicodeStrings() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"7.1.1 Unicode and Internationalization",
		"Tests handling of Unicode characters including multi-byte characters, emoji, international text, and proper URL encoding.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_LiteralDataValues",
	)

	suite.AddTest(
		"test_latin_extended",
		"Basic multi-byte Unicode characters (Latin Extended)",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=contains(Name,'café')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_cyrillic",
		"Cyrillic characters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=contains(Name,'Привет')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_chinese",
		"Chinese characters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=contains(Name,'中文')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_arabic",
		"Arabic characters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=contains(Name,'مرحبا')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_emoji",
		"Emoji characters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=contains(Name,'🚀')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	suite.AddTest(
		"test_mixed_scripts",
		"Mixed script characters",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=contains(Name,'Hello世界')")
			if err != nil {
				return err
			}
			return ctx.AssertStatusCode(resp, 200)
		},
	)

	return suite
}
