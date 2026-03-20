package handlers

import (
	"testing"

	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/version"
)

func TestValidateQueryOptionsForNegotiatedVersion_Rejects401FeaturesIn40(t *testing.T) {
	tests := []struct {
		name    string
		options *query.QueryOptions
	}{
		{
			name: "in operator",
			options: &query.QueryOptions{
				Filter: &query.FilterExpression{Operator: query.OpIn},
			},
		},
		{
			name: "compute",
			options: &query.QueryOptions{
				Compute: &query.ComputeTransformation{},
			},
		},
		{
			name: "index",
			options: &query.QueryOptions{
				Index: true,
			},
		},
		{
			name: "schemaversion",
			options: &query.QueryOptions{
				SchemaVersion: ptr("v1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQueryOptionsForNegotiatedVersion(tt.options, version.Version{Major: 4, Minor: 0})
			if err == nil {
				t.Fatalf("expected feature %s to be rejected for OData 4.0", tt.name)
			}
		})
	}
}

func TestValidateQueryOptionsForNegotiatedVersion_Allows401FeaturesIn401(t *testing.T) {
	opts := &query.QueryOptions{
		Filter:        &query.FilterExpression{Operator: query.OpIn},
		Compute:       &query.ComputeTransformation{},
		Index:         true,
		SchemaVersion: ptr("v1"),
	}

	if err := validateQueryOptionsForNegotiatedVersion(opts, version.Version{Major: 4, Minor: 1}); err != nil {
		t.Fatalf("expected OData 4.01 to allow 4.01 features, got error: %v", err)
	}
}

func ptr(v string) *string {
	return &v
}
