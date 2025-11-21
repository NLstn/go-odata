package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

type collectionResponse struct {
	Value []map[string]interface{} `json:"value"`
}

func requireStatusOK(resp *framework.HTTPResponse) error {
	if resp.StatusCode != http.StatusOK {
		return framework.NewError(fmt.Sprintf("expected HTTP 200 OK but got %d", resp.StatusCode))
	}
	return nil
}

func decodeCollection(resp *framework.HTTPResponse) ([]map[string]interface{}, error) {
	var payload collectionResponse
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return nil, framework.NewError(fmt.Sprintf("failed to parse JSON response: %v", err))
	}

	if payload.Value == nil {
		return nil, framework.NewError("response body does not contain 'value' collection")
	}

	if len(payload.Value) == 0 {
		return nil, framework.NewError("response collection is empty; cannot validate computed properties")
	}

	return payload.Value, nil
}

func ensureComputedProperties(entities []map[string]interface{}, aliases ...string) error {
	for i, entity := range entities {
		for _, alias := range aliases {
			if _, ok := entity[alias]; !ok {
				return framework.NewError(fmt.Sprintf("entity %d missing computed property %q", i, alias))
			}
		}
	}

	return nil
}
