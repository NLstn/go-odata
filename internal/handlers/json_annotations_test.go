package handlers

import (
	"testing"

	"github.com/nlstn/go-odata/internal/version"
)

func TestValidateJSONRequestAnnotations(t *testing.T) {
	v40 := version.Version{Major: 4, Minor: 0}
	v401 := version.Version{Major: 4, Minor: 1}

	if err := validateJSONRequestAnnotations(map[string]interface{}{"@odata.type": "#Demo.Product"}, v40); err != nil {
		t.Fatalf("valid OData 4.0 annotation rejected: %v", err)
	}
	if err := validateJSONRequestAnnotations(map[string]interface{}{"@type": "#Demo.Product"}, v401); err != nil {
		t.Fatalf("valid OData 4.01 annotation rejected: %v", err)
	}
	if err := validateJSONRequestAnnotations(map[string]interface{}{
		"nested": []interface{}{map[string]interface{}{"@type": "#Demo.Product"}},
	}, v40); err == nil {
		t.Fatal("expected compact @type annotation to be rejected for OData 4.0")
	}
}
