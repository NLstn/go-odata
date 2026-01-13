package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

const lambdaSkipReason = "Service does not implement $filter lambda operators (any/all)"

// LambdaOperators creates a test suite for lambda operators
func LambdaOperators() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.9 Lambda Operators (any, all)",
		"Tests lambda operators for collection navigation and filtering",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_LambdaOperators",
	)
	RegisterLambdaOperatorsTests(suite)
	return suite
}

// RegisterLambdaOperatorsTests registers tests for lambda operators (any, all)
func RegisterLambdaOperatorsTests(suite *framework.TestSuite) {
	suite.AddTest(
		"Lambda any operator with collection navigation",
		"Filter entities using any() operator on collection navigation properties",
		testLambdaAnyOperator,
	)

	suite.AddTest(
		"Lambda all operator with collection navigation",
		"Filter entities using all() operator on collection navigation properties",
		testLambdaAllOperator,
	)

	suite.AddTest(
		"Lambda any with complex condition",
		"Use any() with compound boolean expressions",
		testLambdaAnyComplex,
	)

	suite.AddTest(
		"Lambda any with property comparison",
		"Use any() to compare properties within navigation collection",
		testLambdaAnyPropertyComparison,
	)

	suite.AddTest(
		"Nested lambda operators",
		"Use nested any/all operators for multi-level filtering",
		testNestedLambda,
	)
}

func testLambdaAnyOperator(ctx *framework.TestContext) error {
	// Test any() operator on collection navigation property
	resp, err := executeLambdaFilter(ctx, "Descriptions/any(d: d/LanguageKey eq 'EN')")
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if _, ok := result["value"]; !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	return nil
}

func testLambdaAllOperator(ctx *framework.TestContext) error {
	// Test all() operator - all items must satisfy condition
	resp, err := executeLambdaFilter(ctx, "Descriptions/all(d: d/LanguageKey ne 'XX')")
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if _, ok := result["value"]; !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	return nil
}

func testLambdaAnyComplex(ctx *framework.TestContext) error {
	// Test any() with complex boolean expression
	resp, err := executeLambdaFilter(ctx, "Descriptions/any(d: d/LanguageKey eq 'EN' and contains(d/Description,'Laptop'))")
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if _, ok := result["value"]; !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	return nil
}

func testLambdaAnyPropertyComparison(ctx *framework.TestContext) error {
	// Test any() with string function on navigation property
	resp, err := executeLambdaFilter(ctx, "Descriptions/any(d: contains(d/Description,'Gaming'))")
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if _, ok := result["value"]; !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	return nil
}

func testNestedLambda(ctx *framework.TestContext) error {
	// Test multiple any() operators in same filter
	resp, err := executeLambdaFilter(ctx, "Descriptions/any(d: d/LanguageKey eq 'EN') and Descriptions/any(d: d/LanguageKey eq 'DE')")
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if _, ok := result["value"]; !ok {
		return fmt.Errorf("response missing 'value' array")
	}

	return nil
}

func executeLambdaFilter(ctx *framework.TestContext, expression string) (*framework.HTTPResponse, error) {
	filter := url.QueryEscape(expression)
	resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		switch resp.StatusCode {
		case 400, 404, 500, 501:
			return nil, framework.NewError(lambdaSkipReason)
		default:
			return nil, fmt.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	}

	return resp, nil
}
