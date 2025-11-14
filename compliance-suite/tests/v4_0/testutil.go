package v4_0

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

const nonExistingUUID = "00000000-0000-0000-0000-000000000000"

// firstEntityPath returns a canonical path to the first entity in the given entity set.
// It requests one entity and extracts the ID to build "/EntitySet(<key>)".
func firstEntityPath(ctx *framework.TestContext, entitySet string) (string, error) {
	// Select only ID for minimal payload and avoid encoding issues
	qp := url.Values{}
	qp.Set("$top", "1")
	qp.Set("$select", "ID")

	resp, err := ctx.GET("/" + entitySet + "?" + qp.Encode())
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
	id, ok := body.Value[0]["ID"]
	if !ok || id == nil {
		return "", fmt.Errorf("entity in %s missing ID", entitySet)
	}

	// Build key segment as unquoted literal — server accepts GUIDs and numerics as-is
	return fmt.Sprintf("/%s(%v)", entitySet, id), nil
}

// firstEntityID returns the ID value of the first entity in the set as a string.
// Useful for creating related entities that require a foreign key.
func firstEntityID(ctx *framework.TestContext, entitySet string) (string, error) {
	qp := url.Values{}
	qp.Set("$top", "1")
	qp.Set("$select", "ID")

	resp, err := ctx.GET("/" + entitySet + "?" + qp.Encode())
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
	id, ok := body.Value[0]["ID"]
	if !ok || id == nil {
		return "", fmt.Errorf("entity in %s missing ID", entitySet)
	}
	return fmt.Sprintf("%v", id), nil
}

func parseEntityID(value interface{}) (string, error) {
	if value == nil {
		return "", fmt.Errorf("entity ID is nil")
	}
	switch v := value.(type) {
	case string:
		if v == "" {
			return "", fmt.Errorf("entity ID is empty")
		}
		return v, nil
	case float64:
		if math.Abs(v-math.Round(v)) > 1e-9 {
			return "", fmt.Errorf("entity ID must be an integer, got %v", v)
		}
		return strconv.FormatInt(int64(math.Round(v)), 10), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func entityPathByFilter(ctx *framework.TestContext, entitySet, filter string) (string, error) {
	qp := url.Values{}
	qp.Set("$top", "1")
	qp.Set("$select", "ID")
	qp.Set("$filter", filter)

	resp, err := ctx.GET("/" + entitySet + "?" + qp.Encode())
	if err != nil {
		return "", err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return "", fmt.Errorf("list %s with filter: %w", entitySet, err)
	}

	var body struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return "", fmt.Errorf("parse %s filtered list: %w", entitySet, err)
	}
	if len(body.Value) == 0 {
		return "", fmt.Errorf("no entities found in %s for filter %q", entitySet, filter)
	}
	id, err := parseEntityID(body.Value[0]["ID"])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/%s(%s)", entitySet, id), nil
}

func entityIDByFilter(ctx *framework.TestContext, entitySet, filter string) (string, error) {
	path, err := entityPathByFilter(ctx, entitySet, filter)
	if err != nil {
		return "", err
	}
	start := strings.Index(path, "(")
	end := strings.LastIndex(path, ")")
	if start == -1 || end == -1 || end <= start+1 {
		return "", fmt.Errorf("unable to parse entity ID from path %s", path)
	}
	return path[start+1 : end], nil
}

func fetchEntityIDs(ctx *framework.TestContext, entitySet string, top int) ([]string, error) {
	qp := url.Values{}
	qp.Set("$top", strconv.Itoa(top))
	qp.Set("$select", "ID")

	resp, err := ctx.GET("/" + entitySet + "?" + qp.Encode())
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, fmt.Errorf("list %s: %w", entitySet, err)
	}

	var body struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return nil, fmt.Errorf("parse %s list: %w", entitySet, err)
	}
	if len(body.Value) == 0 {
		return nil, fmt.Errorf("no entities in %s", entitySet)
	}
	result := make([]string, 0, len(body.Value))
	for _, entity := range body.Value {
		id, err := parseEntityID(entity["ID"])
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}

func firstEntitySegment(ctx *framework.TestContext, entitySet string) (string, error) {
	path, err := firstEntityPath(ctx, entitySet)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(path, "/"), nil
}

func nonExistingEntityPath(entitySet string) string {
	return fmt.Sprintf("/%s(%s)", entitySet, nonExistingUUID)
}

func nonExistingEntitySegment(entitySet string) string {
	return fmt.Sprintf("%s(%s)", entitySet, nonExistingUUID)
}

func buildProductPayload(ctx *framework.TestContext, name string, price float64) (map[string]interface{}, error) {
	categoryID, err := firstEntityID(ctx, "Categories")
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"Name":       name,
		"Price":      price,
		"CategoryID": categoryID,
		"Status":     1,
	}, nil
}

func createTestProduct(ctx *framework.TestContext, name string, price float64) (string, error) {
	payload, err := buildProductPayload(ctx, name, price)
	if err != nil {
		return "", err
	}
	resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
	if err != nil {
		return "", err
	}
	if err := ctx.AssertStatusCode(resp, 201); err != nil {
		return "", err
	}
	var body map[string]interface{}
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return "", fmt.Errorf("parse product creation response: %w", err)
	}
	return parseEntityID(body["ID"])
}
