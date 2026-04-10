package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

// CaseInsensitiveSystemQueryOptions creates the 11.2.17 case-insensitive system query options test suite.
func CaseInsensitiveSystemQueryOptions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.17 Case-Insensitive System Query Options",
		"Validates OData 4.01 requirement that system query option names are case-insensitive and may be specified without the $ prefix.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_SystemQueryOptions",
	)

	suite.AddTest(
		"test_filter_case_and_dollar_prefix",
		"$filter accepts mixed case and no-$ forms equivalently",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$filter=Price%20gt%2010"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$FILTER=Price%20gt%2010",
				"/Products?$Filter=Price%20gt%2010",
				"/Products?filter=Price%20gt%2010",
			})
		},
	)

	suite.AddTest(
		"test_select_case_and_dollar_prefix",
		"$select accepts mixed case and no-$ forms equivalently",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$select=ID,Name"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$SELECT=ID,Name",
				"/Products?$Select=ID,Name",
				"/Products?select=ID,Name",
			})
		},
	)

	suite.AddTest(
		"test_orderby_case_and_dollar_prefix",
		"$orderby accepts mixed case and no-$ forms equivalently",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$orderby=Price%20desc"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$ORDERBY=Price%20desc",
				"/Products?$OrderBy=Price%20desc",
				"/Products?orderby=Price%20desc",
			})
		},
	)

	suite.AddTest(
		"test_top_case_and_dollar_prefix",
		"$top accepts mixed case and no-$ forms equivalently",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$top=3"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$TOP=3",
				"/Products?$Top=3",
				"/Products?top=3",
			})
		},
	)

	suite.AddTest(
		"test_skip_case_and_dollar_prefix",
		"$skip accepts mixed case and no-$ forms equivalently",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$skip=1"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$SKIP=1",
				"/Products?$Skip=1",
				"/Products?skip=1",
			})
		},
	)

	suite.AddTest(
		"test_count_case_and_dollar_prefix",
		"$count accepts mixed case and no-$ forms equivalently",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$count=true&$top=3"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$COUNT=true&$top=3",
				"/Products?$Count=true&$top=3",
				"/Products?count=true&$top=3",
			})
		},
	)

	suite.AddTest(
		"test_expand_case_and_dollar_prefix",
		"$expand accepts mixed case and no-$ forms equivalently",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$expand=Category&$top=2"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$EXPAND=Category&$top=2",
				"/Products?$Expand=Category&$top=2",
				"/Products?expand=Category&$top=2",
			})
		},
	)

	suite.AddTest(
		"test_search_case_and_dollar_prefix",
		"$search accepts mixed case and no-$ forms equivalently",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$search=Laptop"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$SEARCH=Laptop",
				"/Products?$Search=Laptop",
				"/Products?search=Laptop",
			})
		},
	)

	suite.AddTest(
		"test_compute_case_and_dollar_prefix",
		"$compute accepts mixed case and no-$ forms equivalently",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$compute=Price%20mul%202%20as%20DoublePrice&$select=ID,DoublePrice&$top=2"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$COMPUTE=Price%20mul%202%20as%20DoublePrice&$select=ID,DoublePrice&$top=2",
				"/Products?$Compute=Price%20mul%202%20as%20DoublePrice&$select=ID,DoublePrice&$top=2",
				"/Products?compute=Price%20mul%202%20as%20DoublePrice&$select=ID,DoublePrice&$top=2",
			})
		},
	)

	suite.AddTest(
		"test_apply_case_and_dollar_prefix",
		"$apply accepts mixed case and no-$ forms equivalently",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$apply=filter(Price%20gt%2010)"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$APPLY=filter(Price%20gt%2010)",
				"/Products?$Apply=filter(Price%20gt%2010)",
				"/Products?apply=filter(Price%20gt%2010)",
			})
		},
	)

	suite.AddTest(
		"test_apply_transformation_keywords_case_insensitive",
		"4.01 parses mixed-case $apply transformation names",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$apply=filter(Price%20gt%2010)/orderby(Price%20desc)/top(1)"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$apply=FILTER(Price%20gt%2010)/ORDERBY(Price%20desc)/TOP(1)",
				"/Products?$apply=FiLtEr(Price%20gt%2010)/OrDeRbY(Price%20desc)/ToP(1)",
			})
		},
	)

	suite.AddTest(
		"test_filter_operator_names_case_insensitive",
		"4.01 logical/arithmetic operator names are case-insensitive",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$filter=Price%20gt%2010%20and%20Price%20lt%201000"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$filter=Price%20GT%2010%20AnD%20Price%20Lt%201000",
				"/Products?$filter=Price%20gT%2010%20aNd%20Price%20lT%201000",
			})
		},
	)

	suite.AddTest(
		"test_filter_function_names_case_insensitive",
		"4.01 canonical function names are case-insensitive",
		func(ctx *framework.TestContext) error {
			canonical := "/Products?$filter=contains(Name,'Laptop')"
			return assertEquivalentResponses(ctx, canonical, []string{
				"/Products?$filter=CONTAINS(Name,'Laptop')",
				"/Products?$filter=ConTaIns(Name,'Laptop')",
			})
		},
	)

	suite.AddTest(
		"test_duplicate_system_query_option_rejected_across_forms",
		"same system query option must not appear multiple times across case/$ variants",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filter=Price%20gt%2010&FILTER=Price%20gt%2010")
			if err != nil {
				return err
			}

			return ctx.AssertODataError(resp, http.StatusBadRequest, "must not appear more than once")
		},
	)

	suite.AddTest(
		"test_unknown_option_rejected_with_dollar",
		"unknown query options with $ are rejected",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/Products?$filtre=Price%20gt%2010")
			if err != nil {
				return err
			}

			if err := ctx.AssertODataError(resp, http.StatusBadRequest, "unknown query option"); err != nil {
				return framework.NewError(fmt.Sprintf("expected unknown option to be rejected with strict 400 payload: %v", err))
			}

			return nil
		},
	)

	suite.AddTest(
		"test_unknown_option_rejected_without_dollar",
		"unknown parameters without $ are treated as custom query params and ignored",
		func(ctx *framework.TestContext) error {
			// Per OData 4.01 spec: without a $ prefix the server only treats a parameter as a
			// system query option if its name matches a known system query option name (e.g. "filter",
			// "top"). Unknown names without $ (e.g. "filtre") are custom query parameters — they must
			// not start with $ or @@ — and MUST be silently ignored by the service (200 response).
			resp, err := ctx.GET("/Products?filtre=Price%20gt%2010")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, http.StatusOK)
		},
	)

	suite.AddTest(
		"test_negotiated_4_0_keeps_strict_query_option_and_expression_casing",
		"when negotiated to 4.0, mixed-case/$-less system options and upper-case expression keywords are not treated as 4.01 features",
		func(ctx *framework.TestContext) error {
			headers := []framework.Header{{Key: "OData-MaxVersion", Value: "4.0"}}

			canonicalResp, err := ctx.GET("/Products?$filter=Price%20gt%2010&$top=1", headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(canonicalResp, http.StatusOK); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 canonical query should still work: %v", err))
			}

			mixedCaseOptionResp, err := ctx.GET("/Products?$FILTER=Price%20gt%2010&$top=1", headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertODataError(mixedCaseOptionResp, http.StatusBadRequest, "unknown query option"); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 should reject mixed-case system option names with strict error payload: %v", err))
			}

			noDollarOptionResp, err := ctx.GET("/Products?filter=Price%20gt%2010&$top=1", headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(noDollarOptionResp, http.StatusOK); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 should treat non-$ query options as custom options: %v", err))
			}

			upperCaseOperatorResp, err := ctx.GET("/Products?$filter=Price%20GT%2010&$top=1", headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertODataError(upperCaseOperatorResp, http.StatusBadRequest, "unexpected token"); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 should reject upper-case logical/comparison operators with strict error payload: %v", err))
			}

			upperCaseFunctionResp, err := ctx.GET("/Products?$filter=CONTAINS(Name,'Laptop')&$top=1", headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertODataError(upperCaseFunctionResp, http.StatusBadRequest, "unsupported function"); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 should reject upper-case canonical function names with strict error payload: %v", err))
			}

			mixedCaseApplyResp, err := ctx.GET("/Products?$APPLY=filter(Price%20gt%2010)&$top=1", headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertODataError(mixedCaseApplyResp, http.StatusBadRequest, "unknown query option"); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 should reject mixed-case $apply option name with strict error payload: %v", err))
			}

			noDollarApplyResp, err := ctx.GET("/Products?apply=filter(Price%20gt%2010)&$top=1", headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(noDollarApplyResp, http.StatusOK); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 should treat no-$ apply as custom option and ignore it: %v", err))
			}

			mixedCaseApplyTransformationResp, err := ctx.GET("/Products?$apply=FILTER(Price%20gt%2010)&$top=1", headers...)
			if err != nil {
				return err
			}
			if err := ctx.AssertODataError(mixedCaseApplyTransformationResp, http.StatusBadRequest, "unknown transformation"); err != nil {
				return framework.NewError(fmt.Sprintf("4.0 should reject mixed-case apply transformation names with strict error payload: %v", err))
			}

			return nil
		},
	)

	return suite
}

func assertEquivalentResponses(ctx *framework.TestContext, canonical string, variants []string) error {
	canonicalResp, err := ctx.GET(canonical)
	if err != nil {
		return err
	}
	if err := ctx.AssertStatusCode(canonicalResp, http.StatusOK); err != nil {
		return framework.NewError(fmt.Sprintf("canonical request failed: %v", err))
	}

	var canonicalPayload interface{}
	if err := json.Unmarshal(canonicalResp.Body, &canonicalPayload); err != nil {
		return framework.NewError(fmt.Sprintf("canonical response is not valid JSON: %v", err))
	}

	for _, variant := range variants {
		resp, err := ctx.GET(variant)
		if err != nil {
			return err
		}
		if err := ctx.AssertStatusCode(resp, http.StatusOK); err != nil {
			return framework.NewError(fmt.Sprintf("variant request failed (%s): %v", variant, err))
		}

		var payload interface{}
		if err := json.Unmarshal(resp.Body, &payload); err != nil {
			return framework.NewError(fmt.Sprintf("variant response is not valid JSON (%s): %v", variant, err))
		}

		if !reflect.DeepEqual(canonicalPayload, payload) {
			return framework.NewError(fmt.Sprintf("response for %s differs from canonical request %s", variant, canonical))
		}
	}

	return nil
}
