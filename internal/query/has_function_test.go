package query

import (
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

func TestHasFunctionParsing(t *testing.T) {
	// Test basic has function parsing
	filterStr := "has(Status, 1)"
	
	filter, err := parseFilterWithoutMetadata(filterStr)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Expected filter to be non-nil")
	}

	t.Logf("Parsed filter: Property=%s, Operator=%s, Value=%v", filter.Property, filter.Operator, filter.Value)

	if filter.Operator != OpHas {
		t.Errorf("Expected operator to be OpHas, got %s", filter.Operator)
	}

	if filter.Property != "Status" {
		t.Errorf("Expected property to be 'Status', got '%s'", filter.Property)
	}

	// Value should be the integer 1
	if filter.Value != int64(1) {
		t.Errorf("Expected value to be 1, got %v (type %T)", filter.Value, filter.Value)
	}
}

func TestHasFunctionWithMetadata(t *testing.T) {
	// Create mock metadata
	entityType := &metadata.EntityMetadata{
		EntityName: "Product",
		Properties: []metadata.PropertyMetadata{
			{
				Name:      "Status",
				FieldName: "Status",
				JsonName:  "Status",
				IsEnum:    true,
				IsFlags:   true,
			},
		},
	}

	filterStr := "has(Status, 1)"
	
	filter, err := parseFilter(filterStr, entityType)
	if err != nil {
		t.Fatalf("Failed to parse filter: %v", err)
	}

	if filter == nil {
		t.Fatal("Expected filter to be non-nil")
	}

	t.Logf("Parsed filter: Property=%s, Operator=%s, Value=%v", filter.Property, filter.Operator, filter.Value)

	if filter.Operator != OpHas {
		t.Errorf("Expected operator to be OpHas, got %s", filter.Operator)
	}

	if filter.Property != "Status" {
		t.Errorf("Expected property to be 'Status', got '%s'", filter.Property)
	}
}

func TestHasFunctionSQLGeneration(t *testing.T) {
	filter := &FilterExpression{
		Property: "Status",
		Operator: OpHas,
		Value:    int64(1),
	}

	sql, args := buildSimpleOperatorCondition(filter.Operator, "status", filter.Value)
	
	t.Logf("Generated SQL: %s", sql)
	t.Logf("Args: %v", args)

	expectedSQL := "(status & ?) = ?"
	if sql != expectedSQL {
		t.Errorf("Expected SQL '%s', got '%s'", expectedSQL, sql)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if args[0] != int64(1) || args[1] != int64(1) {
		t.Errorf("Expected args [1, 1], got %v", args)
	}
}
