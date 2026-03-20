package v4_01

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

func firstEntityPath(ctx *framework.TestContext, entitySet string) (string, error) {
	id, err := firstEntityID(ctx, entitySet)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/%s(%s)", entitySet, id), nil
}

func firstEntityID(ctx *framework.TestContext, entitySet string) (string, error) {
	resp, err := ctx.GET(fmt.Sprintf("/%s?$top=1&$select=ID", entitySet))
	if err != nil {
		return "", err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return "", fmt.Errorf("list %s: %w", entitySet, err)
	}

	var body struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return "", fmt.Errorf("parse %s list: %w", entitySet, err)
	}
	if len(body.Value) == 0 {
		return "", fmt.Errorf("no entities in %s", entitySet)
	}

	id := body.Value[0]["ID"]
	if id == nil {
		return "", fmt.Errorf("entity in %s missing ID", entitySet)
	}
	return fmt.Sprintf("%v", id), nil
}
