package v4_01

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

type operationMetadata struct {
	Name       string              `xml:"Name,attr"`
	IsBound    bool                `xml:"IsBound,attr"`
	Parameters []operationParamDef `xml:"Parameter"`
}

type operationParamDef struct {
	Name string `xml:"Name,attr"`
}

type schemaMetadata struct {
	Functions []operationMetadata `xml:"Function"`
	Actions   []operationMetadata `xml:"Action"`
}

type metadataDocument struct {
	DataServices struct {
		Schemas []schemaMetadata `xml:"Schema"`
	} `xml:"DataServices"`
}

// FunctionActionOverloading creates the 12.2 Function and Action Overloading test suite
func FunctionActionOverloading() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"12.2 Function and Action Overloading",
		"Validates function and action overload support where multiple functions or actions can share the same name but differ by binding parameter type or parameter count/types.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part3-csdl.html#sec_FunctionandActionOverloading",
	)

	var cachedMetadata *metadataDocument
	getMetadata := func(ctx *framework.TestContext) (*metadataDocument, error) {
		if cachedMetadata != nil {
			return cachedMetadata, nil
		}

		resp, err := ctx.GET("/$metadata")
		if err != nil {
			return nil, err
		}
		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return nil, err
		}

		var doc metadataDocument
		if err := xml.Unmarshal(resp.Body, &doc); err != nil {
			return nil, framework.NewError("unable to parse $metadata XML: %v", err)
		}

		cachedMetadata = &doc
		return cachedMetadata, nil
	}

	operationSignatures := func(doc *metadataDocument, name string, bound bool, kind string) map[string]struct{} {
		signatures := map[string]struct{}{}

		for _, schema := range doc.DataServices.Schemas {
			var ops []operationMetadata
			if kind == "Function" {
				ops = schema.Functions
			} else {
				ops = schema.Actions
			}

			for _, op := range ops {
				if op.Name != name || op.IsBound != bound {
					continue
				}

				params := op.Parameters
				if bound && len(params) > 0 {
					params = params[1:]
				}

				names := make([]string, 0, len(params))
				for _, p := range params {
					names = append(names, p.Name)
				}
				signatures[paramKey(names)] = struct{}{}
			}
		}

		return signatures
	}

	requireDeclaredSignatures := func(
		ctx *framework.TestContext,
		doc *metadataDocument,
		name string,
		bound bool,
		kind string,
		required [][]string,
	) (map[string]struct{}, error) {
		signatures := operationSignatures(doc, name, bound, kind)
		if len(signatures) == 0 {
			scope := "unbound"
			if bound {
				scope = "bound"
			}
			return nil, ctx.Skip(fmt.Sprintf("$metadata does not declare %s %s operation '%s'", scope, strings.ToLower(kind), name))
		}

		for _, req := range required {
			if _, ok := signatures[paramKey(req)]; !ok {
				scope := "unbound"
				if bound {
					scope = "bound"
				}
				return nil, ctx.Skip(fmt.Sprintf("$metadata declares %s %s '%s' but not signature with parameters [%s]", scope, strings.ToLower(kind), name, strings.Join(sortedCopy(req), ", ")))
			}
		}

		return signatures, nil
	}

	assertSuccessNoErrorPayload := func(resp *framework.HTTPResponse, context string) error {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return framework.NewError("%s: expected 2xx success, got %d", context, resp.StatusCode)
		}
		trimmed := strings.TrimSpace(string(resp.Body))
		if trimmed == "" {
			return nil
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(resp.Body, &payload); err != nil {
			return framework.NewError("%s: expected JSON payload, got unmarshal error: %v", context, err)
		}
		if _, hasError := payload["error"]; hasError {
			return framework.NewError("%s: successful response must not contain an error object", context)
		}
		return nil
	}

	assertClientError := func(resp *framework.HTTPResponse, context string) error {
		if resp.StatusCode < 400 || resp.StatusCode >= 500 {
			return framework.NewError("%s: expected 4xx client error, got %d", context, resp.StatusCode)
		}
		return nil
	}

	// Test 1: Function overload with different parameter counts
	suite.AddTest(
		"test_function_overload_param_count",
		"Function overload with different parameter counts",
		func(ctx *framework.TestContext) error {
			doc, err := getMetadata(ctx)
			if err != nil {
				return err
			}

			_, err = requireDeclaredSignatures(ctx, doc, "GetTopProducts", false, "Function", [][]string{{}, {"count"}})
			if err != nil {
				return err
			}

			resp1, err := ctx.GET("/GetTopProducts()")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp1, "GetTopProducts() overload without parameters"); err != nil {
				return err
			}

			resp2, err := ctx.GET("/GetTopProducts()?count=5")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp2, "GetTopProducts() overload with count"); err != nil {
				return err
			}

			invalidResp, err := ctx.GET("/GetTopProducts()?__invalid=1")
			if err != nil {
				return err
			}
			return assertClientError(invalidResp, "GetTopProducts() invalid signature")
		},
	)

	// Test 2: Function overload with different parameter types
	suite.AddTest(
		"test_function_overload_param_types",
		"Function overload with different parameter types",
		func(ctx *framework.TestContext) error {
			doc, err := getMetadata(ctx)
			if err != nil {
				return err
			}
			_, err = requireDeclaredSignatures(ctx, doc, "Convert", false, "Function", [][]string{{"input"}, {"number"}})
			if err != nil {
				return err
			}

			resp1, err := ctx.GET("/Convert()?input=hello")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp1, "Convert() overload with input"); err != nil {
				return err
			}

			resp2, err := ctx.GET("/Convert()?number=5")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp2, "Convert() overload with number"); err != nil {
				return err
			}

			invalidResp, err := ctx.GET("/Convert()?input=hello&number=5")
			if err != nil {
				return err
			}
			return assertClientError(invalidResp, "Convert() ambiguous/invalid signature")
		},
	)

	// Test 3: Function overload resolution based on parameters
	suite.AddTest(
		"test_function_overload_resolution",
		"Function overload resolution based on parameters",
		func(ctx *framework.TestContext) error {
			doc, err := getMetadata(ctx)
			if err != nil {
				return err
			}
			_, err = requireDeclaredSignatures(ctx, doc, "Calculate", false, "Function", [][]string{{"value"}, {"a", "b"}})
			if err != nil {
				return err
			}

			resp1, err := ctx.GET("/Calculate()?value=5")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp1, "Calculate() overload with value"); err != nil {
				return err
			}

			resp2, err := ctx.GET("/Calculate()?a=3&b=7")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp2, "Calculate() overload with a,b"); err != nil {
				return err
			}

			invalidResp, err := ctx.GET("/Calculate()?a=3")
			if err != nil {
				return err
			}
			return assertClientError(invalidResp, "Calculate() invalid signature")
		},
	)

	// Test 4: Action overload with different parameter counts
	suite.AddTest(
		"test_action_overload_param_count",
		"Action overload with different parameter counts",
		func(ctx *framework.TestContext) error {
			doc, err := getMetadata(ctx)
			if err != nil {
				return err
			}
			_, err = requireDeclaredSignatures(ctx, doc, "Process", false, "Action", [][]string{{"percentage"}, {"minPrice", "percentage"}})
			if err != nil {
				return err
			}

			resp1, err := ctx.POST("/Process", map[string]interface{}{"percentage": 10.0})
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp1, "Process action overload with one parameter"); err != nil {
				return err
			}

			resp2, err := ctx.POST("/Process", map[string]interface{}{"percentage": 10.0, "minPrice": 100.0})
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp2, "Process action overload with two parameters"); err != nil {
				return err
			}

			invalidResp, err := ctx.POST("/Process", map[string]interface{}{"minPrice": 100.0})
			if err != nil {
				return err
			}
			return assertClientError(invalidResp, "Process action invalid signature")
		},
	)

	var productPath string
	getProductPath := func(ctx *framework.TestContext) (string, error) {
		if productPath != "" {
			return productPath, nil
		}
		resp, err := ctx.GET("/Products?$top=1&$select=ID")
		if err != nil {
			return "", err
		}
		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return "", err
		}
		var body struct {
			Value []map[string]interface{} `json:"value"`
		}
		if err := json.Unmarshal(resp.Body, &body); err != nil {
			return "", err
		}
		if len(body.Value) == 0 {
			return "", framework.NewError("no products available")
		}
		id := body.Value[0]["ID"]
		productPath = fmt.Sprintf("/Products(%v)", id)
		return productPath, nil
	}

	// Test 5: Bound function overload on different entity sets
	suite.AddTest(
		"test_bound_function_overload",
		"Bound function overload on different entity sets",
		func(ctx *framework.TestContext) error {
			doc, err := getMetadata(ctx)
			if err != nil {
				return err
			}
			sigs, err := requireDeclaredSignatures(ctx, doc, "GetInfo", true, "Function", [][]string{{"format"}})
			if err != nil {
				return err
			}

			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}

			resp, err := ctx.GET(path + "/GetInfo?format=json")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp, "bound GetInfo overload with format"); err != nil {
				return err
			}

			invalidParams := []string{"__invalid"}
			if _, exists := sigs[paramKey(invalidParams)]; exists {
				return framework.NewError("test setup error: invalid signature marker unexpectedly declared")
			}
			invalidResp, err := ctx.GET(path + "/GetInfo?__invalid=1")
			if err != nil {
				return err
			}
			return assertClientError(invalidResp, "bound GetInfo invalid signature")
		},
	)

	// Test 6: Reject duplicate overloads
	suite.AddTest(
		"test_reject_duplicate_overload",
		"Verify duplicate function signatures validation",
		func(ctx *framework.TestContext) error {
			doc, err := getMetadata(ctx)
			if err != nil {
				return err
			}

			seen := map[string]struct{}{}
			for _, schema := range doc.DataServices.Schemas {
				for _, fn := range schema.Functions {
					params := fn.Parameters
					if fn.IsBound && len(params) > 0 {
						params = params[1:]
					}
					names := make([]string, 0, len(params))
					for _, p := range params {
						names = append(names, p.Name)
					}
					sig := fmt.Sprintf("Function|%t|%s|%s", fn.IsBound, fn.Name, paramKey(names))
					if _, exists := seen[sig]; exists {
						return framework.NewError("$metadata contains duplicate function overload signature for %s", fn.Name)
					}
					seen[sig] = struct{}{}
				}

				for _, action := range schema.Actions {
					params := action.Parameters
					if action.IsBound && len(params) > 0 {
						params = params[1:]
					}
					names := make([]string, 0, len(params))
					for _, p := range params {
						names = append(names, p.Name)
					}
					sig := fmt.Sprintf("Action|%t|%s|%s", action.IsBound, action.Name, paramKey(names))
					if _, exists := seen[sig]; exists {
						return framework.NewError("$metadata contains duplicate action overload signature for %s", action.Name)
					}
					seen[sig] = struct{}{}
				}
			}

			return nil
		},
	)

	// Test 7: Function overload with additional parameter
	suite.AddTest(
		"test_function_overload_additional_param",
		"Function overload with additional parameter",
		func(ctx *framework.TestContext) error {
			doc, err := getMetadata(ctx)
			if err != nil {
				return err
			}
			_, err = requireDeclaredSignatures(ctx, doc, "GetTopProducts", false, "Function", [][]string{{"count"}, {"category", "count"}})
			if err != nil {
				return err
			}

			resp1, err := ctx.GET("/GetTopProducts()?count=5")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp1, "GetTopProducts overload with count"); err != nil {
				return err
			}

			resp2, err := ctx.GET("/GetTopProducts()?count=5&category=Electronics")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp2, "GetTopProducts overload with count and category"); err != nil {
				return err
			}

			invalidResp, err := ctx.GET("/GetTopProducts()?count=5&category=Electronics&__invalid=1")
			if err != nil {
				return err
			}
			return assertClientError(invalidResp, "GetTopProducts invalid extended signature")
		},
	)

	// Test 8: Bound function overload with different parameter counts
	suite.AddTest(
		"test_bound_function_param_overload",
		"Bound function overload with different parameter counts",
		func(ctx *framework.TestContext) error {
			doc, err := getMetadata(ctx)
			if err != nil {
				return err
			}
			_, err = requireDeclaredSignatures(ctx, doc, "CalculatePrice", true, "Function", [][]string{{"discount"}, {"discount", "tax"}})
			if err != nil {
				return err
			}

			path, err := getProductPath(ctx)
			if err != nil {
				return err
			}

			resp1, err := ctx.GET(path + "/CalculatePrice?discount=10")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp1, "CalculatePrice bound overload with discount"); err != nil {
				return err
			}

			resp2, err := ctx.GET(path + "/CalculatePrice?discount=10&tax=8")
			if err != nil {
				return err
			}
			if err := assertSuccessNoErrorPayload(resp2, "CalculatePrice bound overload with discount and tax"); err != nil {
				return err
			}

			invalidResp, err := ctx.GET(path + "/CalculatePrice?tax=8")
			if err != nil {
				return err
			}
			return assertClientError(invalidResp, "CalculatePrice invalid bound signature")
		},
	)

	return suite
}

func paramKey(params []string) string {
	sorted := sortedCopy(params)
	return strings.Join(sorted, "|")
}

func sortedCopy(values []string) []string {
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return copied
}
