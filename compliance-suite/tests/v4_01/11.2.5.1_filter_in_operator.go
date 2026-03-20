package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// InOperator creates the 11.2.5.1 $filter in-operator test suite.
func InOperator() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.1 $filter In Operator",
		"Validates the OData 4.01 in operator in $filter expressions, including successful membership filtering and required 400 responses for invalid expressions.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_in",
	)

	suite.AddTest(
		"test_filter_in_string_membership",
		"String membership: Name in ('Laptop','Wireless Mouse') returns only matching entities",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in ('Laptop','Wireless Mouse')")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := decodeCollectionAllowEmpty(resp)
			if err != nil {
				return err
			}

			allowed := map[string]struct{}{"Laptop": {}, "Wireless Mouse": {}}
			for i, entity := range entities {
				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing string Name field", i))
				}
				if _, exists := allowed[name]; !exists {
					return framework.NewError(fmt.Sprintf("entity %d has Name=%q not in expected set", i, name))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_in_numeric_membership",
		"Numeric membership: Price in (29.99, 15.50) returns only matching entities",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price in (29.99,15.50)")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := decodeCollectionAllowEmpty(resp)
			if err != nil {
				return err
			}

			allowedPrices := map[float64]struct{}{29.99: {}, 15.50: {}}
			for i, entity := range entities {
				price, err := floatField(entity, "Price")
				if err != nil {
					return framework.NewError(fmt.Sprintf("entity %d: %v", i, err))
				}
				if _, exists := allowedPrices[price]; !exists {
					return framework.NewError(fmt.Sprintf("entity %d has Price=%f not in expected set", i, price))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_in_with_combined_expression",
		"Combined expression: Name in ('Laptop','Wireless Mouse') and Price gt 20 enforces both predicates",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in ('Laptop','Wireless Mouse') and Price gt 20")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
				return err
			}

			entities, err := decodeCollectionAllowEmpty(resp)
			if err != nil {
				return err
			}

			allowedNames := map[string]struct{}{"Laptop": {}, "Wireless Mouse": {}}
			for i, entity := range entities {
				name, ok := entity["Name"].(string)
				if !ok {
					return framework.NewError(fmt.Sprintf("entity %d missing string Name field", i))
				}
				if _, exists := allowedNames[name]; !exists {
					return framework.NewError(fmt.Sprintf("entity %d has Name=%q not in expected set", i, name))
				}

				price, err := floatField(entity, "Price")
				if err != nil {
					return framework.NewError(fmt.Sprintf("entity %d: %v", i, err))
				}
				if price <= 20 {
					return framework.NewError(fmt.Sprintf("entity %d has Price=%v, expected > 20", i, price))
				}
			}

			return nil
		},
	)

	suite.AddTest(
		"test_filter_in_type_mismatch_string_in_numeric",
		"Type mismatch: string property compared to numeric list returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in (1,2)")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_filter_in_type_mismatch_numeric_in_string",
		"Type mismatch: numeric property compared to string list returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price in ('expensive','cheap')")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_filter_in_malformed_missing_parentheses",
		"Malformed list syntax (missing parentheses) returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in 'Laptop','Wireless Mouse'")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_filter_in_malformed_missing_comma",
		"Malformed list syntax (missing comma) returns 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Name in ('Laptop' 'Wireless Mouse')")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	suite.AddTest(
		"test_filter_in_empty_expression",
		"Empty expression errors return 400",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusBadRequest)
		},
	)

	return suite
}

func decodeCollectionAllowEmpty(resp *framework.HTTPResponse) ([]map[string]interface{}, error) {
	var payload struct {
		Value []map[string]interface{} `json:"value"`
	}

	if err := decodeJSON(resp, &payload); err != nil {
		return nil, err
	}

	if payload.Value == nil {
		return nil, framework.NewError("response body does not contain 'value' collection")
	}

	return payload.Value, nil
}

func decodeJSON(resp *framework.HTTPResponse, target interface{}) error {
	if err := json.Unmarshal(resp.Body, target); err != nil {
		return framework.NewError(fmt.Sprintf("failed to parse JSON response: %v", err))
	}
	return nil
}

func intField(entity map[string]interface{}, key string) (int, error) {
	v, ok := entity[key]
	if !ok {
		return 0, fmt.Errorf("missing %q field", key)
	}

	n, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("field %q is %T, expected number", key, v)
	}

	return int(n), nil
}

func floatField(entity map[string]interface{}, key string) (float64, error) {
	v, ok := entity[key]
	if !ok {
		return 0, fmt.Errorf("missing %q field", key)
	}

	n, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("field %q is %T, expected number", key, v)
	}

	return n, nil
}
