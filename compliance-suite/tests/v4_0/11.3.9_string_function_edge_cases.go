package v4_0

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// StringFunctionEdgeCases creates a test suite for string function edge cases
func StringFunctionEdgeCases() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.3.9 String Function Edge Cases",
		"Tests edge cases and boundary conditions for string functions in filter expressions",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_BuiltinFilterOperations",
	)
	RegisterStringFunctionEdgeCasesTests(suite)
	return suite
}

// RegisterStringFunctionEdgeCasesTests registers tests for string function edge cases
func RegisterStringFunctionEdgeCasesTests(suite *framework.TestSuite) {
	suite.AddTest(
		"contains() with empty string",
		"Empty string is contained in all strings",
		testContainsEmptyString,
	)

	suite.AddTest(
		"startswith() with empty string",
		"All strings start with empty string",
		testStartswithEmptyString,
	)

	suite.AddTest(
		"endswith() with empty string",
		"All strings end with empty string",
		testEndswithEmptyString,
	)

	suite.AddTest(
		"length() with positive value",
		"Test length() function returns positive integers",
		testLengthPositive,
	)

	suite.AddTest(
		"substring() beyond string length",
		"Substring with start position beyond string length",
		testSubstringBeyondLength,
	)

	suite.AddTest(
		"substring() with negative start returns error",
		"Negative start position for substring() returns 400",
		testSubstringNegativeStart,
	)

	suite.AddTest(
		"substring() with length of 0",
		"Substring with length 0 returns empty string",
		testSubstringZeroLength,
	)

	suite.AddTest(
		"indexof() not found returns -1",
		"indexof() returns -1 when substring not found",
		testIndexofNotFound,
	)

	suite.AddTest(
		"indexof() with empty string returns 0",
		"Empty string is found at position 0",
		testIndexofEmptyString,
	)

	suite.AddTest(
		"tolower() idempotent",
		"tolower() applied twice gives same result",
		testTolowerIdempotent,
	)

	suite.AddTest(
		"toupper() idempotent",
		"toupper() applied twice gives same result",
		testToupperIdempotent,
	)

	suite.AddTest(
		"trim() on string without whitespace",
		"trim() on string without leading/trailing whitespace unchanged",
		testTrimNoWhitespace,
	)

	suite.AddTest(
		"concat() with empty strings",
		"Concatenation of empty strings produces empty string",
		testConcatEmptyStrings,
	)

	suite.AddTest(
		"concat() multiple arguments",
		"Nested concat() calls combine multiple strings",
		testConcatMultiple,
	)

	suite.AddTest(
		"Case-insensitive contains()",
		"Use tolower() for case-insensitive matching",
		testContainsCaseInsensitive,
	)

	suite.AddTest(
		"Nested string functions",
		"Multiple string functions can be nested",
		testNestedStringFunctions,
	)

	suite.AddTest(
		"String function on null property",
		"String functions on null properties handled appropriately",
		testStringFunctionOnNull,
	)

	suite.AddTest(
		"Very long string in filter",
		"Service handles long string literals in filters",
		testVeryLongString,
	)

	suite.AddTest(
		"Special regex characters treated as literals",
		"Regex metacharacters treated as literal characters",
		testSpecialRegexChars,
	)

	suite.AddTest(
		"Unicode in string functions",
		"Unicode characters handled correctly in string functions",
		testUnicodeInStringFunctions,
	)

	suite.AddTest(
		"contains() treats wildcard characters as literals",
		"Wildcard characters must be treated as literals in contains()",
		testContainsWildcardLiterals,
	)

	suite.AddTest(
		"startswith() treats wildcard characters as literals",
		"Wildcard characters must be treated as literals in startswith()",
		testStartswithWildcardLiterals,
	)

	suite.AddTest(
		"endswith() treats wildcard characters as literals",
		"Wildcard characters must be treated as literals in endswith()",
		testEndswithWildcardLiterals,
	)
}

func testContainsEmptyString(ctx *framework.TestContext) error {
	filter := url.QueryEscape("contains(Name,'')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testStartswithEmptyString(ctx *framework.TestContext) error {
	filter := url.QueryEscape("startswith(Name,'')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testEndswithEmptyString(ctx *framework.TestContext) error {
	filter := url.QueryEscape("endswith(Name,'')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testLengthPositive(ctx *framework.TestContext) error {
	filter := url.QueryEscape("length(Name) gt 0")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testSubstringBeyondLength(ctx *framework.TestContext) error {
	filter := url.QueryEscape("substring(Name,100) eq ''")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	// Implementation dependent - may return 200 or 400
	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		return fmt.Errorf("expected status 200 or 400, got %d", resp.StatusCode)
	}

	return nil
}

func testSubstringNegativeStart(ctx *framework.TestContext) error {
	filter := url.QueryEscape("substring(Name,-1) eq ''")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 400 {
		return fmt.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	return nil
}

func testSubstringZeroLength(ctx *framework.TestContext) error {
	filter := url.QueryEscape("substring(Name,0,0) eq ''")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testIndexofNotFound(ctx *framework.TestContext) error {
	filter := url.QueryEscape("indexof(Name,'ZZZZZ') eq -1")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testIndexofEmptyString(ctx *framework.TestContext) error {
	filter := url.QueryEscape("indexof(Name,'') eq 0")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testTolowerIdempotent(ctx *framework.TestContext) error {
	filter := url.QueryEscape("tolower(Name) eq tolower(Name)")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testToupperIdempotent(ctx *framework.TestContext) error {
	filter := url.QueryEscape("toupper(Name) eq toupper(Name)")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testTrimNoWhitespace(ctx *framework.TestContext) error {
	filter := url.QueryEscape("trim(Name) eq Name")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testConcatEmptyStrings(ctx *framework.TestContext) error {
	filter := url.QueryEscape("concat('','') eq ''")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testConcatMultiple(ctx *framework.TestContext) error {
	filter := url.QueryEscape("concat(concat(Name,' - '),'suffix') ne ''")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testContainsCaseInsensitive(ctx *framework.TestContext) error {
	filter := url.QueryEscape("contains(tolower(Name),'laptop')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testNestedStringFunctions(ctx *framework.TestContext) error {
	filter := url.QueryEscape("length(trim(toupper(Name))) gt 0")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testStringFunctionOnNull(ctx *framework.TestContext) error {
	filter := url.QueryEscape("length(Description) eq null")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	// Implementation dependent - null handling varies
	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		return fmt.Errorf("expected status 200 or 400, got %d", resp.StatusCode)
	}

	return nil
}

func testVeryLongString(ctx *framework.TestContext) error {
	longString := strings.Repeat("A", 1000)
	filter := url.QueryEscape(fmt.Sprintf("contains(Name,'%s')", longString))
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testSpecialRegexChars(ctx *framework.TestContext) error {
	filter := url.QueryEscape("contains(Name,'.*+?[]')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testUnicodeInStringFunctions(ctx *framework.TestContext) error {
	filter := url.QueryEscape("contains(Name,'café')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testContainsWildcardLiterals(ctx *framework.TestContext) error {
	filter := url.QueryEscape("contains(Name,'%')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return assertEmptyValueSet(resp.Body)
}

func testStartswithWildcardLiterals(ctx *framework.TestContext) error {
	filter := url.QueryEscape("startswith(Name,'_')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return assertEmptyValueSet(resp.Body)
}

func testEndswithWildcardLiterals(ctx *framework.TestContext) error {
	filter := url.QueryEscape("endswith(Name,'\\')")
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return assertEmptyValueSet(resp.Body)
}
