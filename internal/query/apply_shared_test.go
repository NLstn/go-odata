package query

import (
	"reflect"
	"testing"
)

type sharedTestProduct struct {
	ID          int     `json:"ID" odata:"key"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type sharedTestCategory struct {
	ID   int    `json:"ID" odata:"key"`
	Name string `json:"name"`
}

func TestApplySelectToExpandedEntity(t *testing.T) {
	t.Run("nil value returns nil", func(t *testing.T) {
		result := applySelectToExpandedEntity(nil, []string{"name"}, nil)
		if result != nil {
			t.Error("expected nil result")
		}
	})

	t.Run("empty select returns unchanged", func(t *testing.T) {
		entity := sharedTestProduct{ID: 1, Name: "Test"}
		result := applySelectToExpandedEntity(&entity, []string{}, nil)
		if result == nil {
			t.Error("expected non-nil result")
		}
	})

	t.Run("single entity with select", func(t *testing.T) {
		entity := sharedTestProduct{ID: 1, Name: "Product1", Price: 10.5}
		result := applySelectToExpandedEntity(&entity, []string{"name"}, nil)

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("expected result to be map[string]interface{}")
		}

		if _, ok := resultMap["name"]; !ok {
			t.Error("expected 'name' to be in result")
		}
		if _, ok := resultMap["ID"]; !ok {
			t.Error("expected 'ID' (key) to be in result")
		}
		if _, ok := resultMap["description"]; ok {
			t.Error("expected 'description' to NOT be in result")
		}
	})

	t.Run("slice of entities with select", func(t *testing.T) {
		entities := []sharedTestProduct{
			{ID: 1, Name: "Product1", Price: 10.5},
			{ID: 2, Name: "Product2", Price: 20.0},
		}
		result := applySelectToExpandedEntity(entities, []string{"name"}, nil)

		resultSlice, ok := result.([]map[string]interface{})
		if !ok {
			t.Fatal("expected result to be []map[string]interface{}")
		}

		if len(resultSlice) != 2 {
			t.Fatalf("expected 2 items, got %d", len(resultSlice))
		}

		firstItem := resultSlice[0]
		if _, ok := firstItem["name"]; !ok {
			t.Error("expected 'name' to be in result")
		}
		if _, ok := firstItem["ID"]; !ok {
			t.Error("expected 'ID' (key) to be in result")
		}
	})

	t.Run("pointer to nil returns unchanged", func(t *testing.T) {
		var entity *sharedTestProduct
		result := applySelectToExpandedEntity(entity, []string{"name"}, nil)
		if result != entity {
			t.Error("expected result to be unchanged")
		}
	})
}

func TestFilterEntityFields(t *testing.T) {
	product := sharedTestProduct{ID: 1, Name: "Product1", Price: 10.5, Description: "Desc"}
	productVal := reflect.ValueOf(product)

	t.Run("filter with selected fields", func(t *testing.T) {
		selectedMap := map[string]bool{
			"name": true,
		}
		result := filterEntityFields(productVal, selectedMap, nil)

		if _, ok := result["name"]; !ok {
			t.Error("expected 'name' to be in result")
		}
		if _, ok := result["ID"]; !ok {
			t.Error("expected 'ID' (key) to be in result")
		}
		if _, ok := result["description"]; ok {
			t.Error("expected 'description' to NOT be in result")
		}
	})

	t.Run("filter includes key fields automatically", func(t *testing.T) {
		selectedMap := map[string]bool{
			"name": true,
		}
		result := filterEntityFields(productVal, selectedMap, nil)

		if _, ok := result["ID"]; !ok {
			t.Error("expected 'ID' (key) to be included automatically")
		}
	})
}

func TestEvaluateSingleComputeExpression(t *testing.T) {
	product := sharedTestProduct{ID: 1, Name: "Product1", Price: 10.5}
	productVal := reflect.ValueOf(product)

	t.Run("nil expression returns nil", func(t *testing.T) {
		result := evaluateSingleComputeExpression(productVal, nil)
		if result != nil {
			t.Error("expected nil result")
		}
	})

	t.Run("property reference", func(t *testing.T) {
		expr := &FilterExpression{
			Property: "Price",
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != 10.5 {
			t.Errorf("expected 10.5, got %v", result)
		}
	})

	t.Run("literal value", func(t *testing.T) {
		expr := &FilterExpression{
			Value: 42,
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != 42 {
			t.Errorf("expected 42, got %v", result)
		}
	})

	t.Run("multiplication with Logical field", func(t *testing.T) {
		expr := &FilterExpression{
			Logical: "mul",
			Left: &FilterExpression{
				Property: "Price",
			},
			Right: &FilterExpression{
				Value: 2,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != 21.0 {
			t.Errorf("expected 21.0, got %v", result)
		}
	})

	t.Run("division with Logical field", func(t *testing.T) {
		expr := &FilterExpression{
			Logical: "div",
			Left: &FilterExpression{
				Property: "Price",
			},
			Right: &FilterExpression{
				Value: 2,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != 5.25 {
			t.Errorf("expected 5.25, got %v", result)
		}
	})

	t.Run("addition with Logical field", func(t *testing.T) {
		expr := &FilterExpression{
			Logical: "add",
			Left: &FilterExpression{
				Value: 5,
			},
			Right: &FilterExpression{
				Value: 3,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != 8.0 {
			t.Errorf("expected 8.0, got %v", result)
		}
	})

	t.Run("subtraction with Logical field", func(t *testing.T) {
		expr := &FilterExpression{
			Logical: "sub",
			Left: &FilterExpression{
				Value: 10,
			},
			Right: &FilterExpression{
				Value: 3,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != 7.0 {
			t.Errorf("expected 7.0, got %v", result)
		}
	})

	t.Run("modulo with Logical field", func(t *testing.T) {
		expr := &FilterExpression{
			Logical: "mod",
			Left: &FilterExpression{
				Value: 10,
			},
			Right: &FilterExpression{
				Value: 3,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != int64(1) {
			t.Errorf("expected 1, got %v", result)
		}
	})

	t.Run("division by zero returns nil", func(t *testing.T) {
		expr := &FilterExpression{
			Logical: "div",
			Left: &FilterExpression{
				Value: 10,
			},
			Right: &FilterExpression{
				Value: 0,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != nil {
			t.Error("expected nil for division by zero")
		}
	})

	t.Run("multiplication with Operator field", func(t *testing.T) {
		expr := &FilterExpression{
			Operator: OpMul,
			Left: &FilterExpression{
				Property: "Price",
			},
			Right: &FilterExpression{
				Value: 2,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != 21.0 {
			t.Errorf("expected 21.0, got %v", result)
		}
	})

	t.Run("division with Operator field", func(t *testing.T) {
		expr := &FilterExpression{
			Operator: OpDiv,
			Left: &FilterExpression{
				Property: "Price",
			},
			Right: &FilterExpression{
				Value: 2,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != 5.25 {
			t.Errorf("expected 5.25, got %v", result)
		}
	})

	t.Run("addition with Operator field", func(t *testing.T) {
		expr := &FilterExpression{
			Operator: OpAdd,
			Left: &FilterExpression{
				Value: 5,
			},
			Right: &FilterExpression{
				Value: 3,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != 8.0 {
			t.Errorf("expected 8.0, got %v", result)
		}
	})

	t.Run("subtraction with Operator field", func(t *testing.T) {
		expr := &FilterExpression{
			Operator: OpSub,
			Left: &FilterExpression{
				Value: 10,
			},
			Right: &FilterExpression{
				Value: 3,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != 7.0 {
			t.Errorf("expected 7.0, got %v", result)
		}
	})

	t.Run("modulo with Operator field", func(t *testing.T) {
		expr := &FilterExpression{
			Operator: OpMod,
			Left: &FilterExpression{
				Value: 10,
			},
			Right: &FilterExpression{
				Value: 3,
			},
		}
		result := evaluateSingleComputeExpression(productVal, expr)
		if result != int64(1) {
			t.Errorf("expected 1, got %v", result)
		}
	})
}

func TestGetPropertyValue(t *testing.T) {
	product := sharedTestProduct{ID: 1, Name: "Product1", Price: 10.5}

	t.Run("get existing property by field name", func(t *testing.T) {
		productVal := reflect.ValueOf(product)
		result := getPropertyValue(productVal, "Name")
		if result != "Product1" {
			t.Errorf("expected 'Product1', got %v", result)
		}
	})

	t.Run("get existing property by json name", func(t *testing.T) {
		productVal := reflect.ValueOf(product)
		result := getPropertyValue(productVal, "name")
		if result != "Product1" {
			t.Errorf("expected 'Product1', got %v", result)
		}
	})

	t.Run("get non-existent property returns nil", func(t *testing.T) {
		productVal := reflect.ValueOf(product)
		result := getPropertyValue(productVal, "NonExistent")
		if result != nil {
			t.Error("expected nil for non-existent property")
		}
	})

	t.Run("get property from pointer", func(t *testing.T) {
		productVal := reflect.ValueOf(&product)
		result := getPropertyValue(productVal, "Price")
		if result != 10.5 {
			t.Errorf("expected 10.5, got %v", result)
		}
	})

	t.Run("get property from nil pointer returns nil", func(t *testing.T) {
		var product *sharedTestProduct
		productVal := reflect.ValueOf(product)
		result := getPropertyValue(productVal, "Name")
		if result != nil {
			t.Error("expected nil for nil pointer")
		}
	})

	t.Run("invalid value returns nil", func(t *testing.T) {
		var productVal reflect.Value
		result := getPropertyValue(productVal, "Name")
		if result != nil {
			t.Error("expected nil for invalid value")
		}
	})

	t.Run("non-struct value returns nil", func(t *testing.T) {
		stringVal := reflect.ValueOf("not a struct")
		result := getPropertyValue(stringVal, "Name")
		if result != nil {
			t.Error("expected nil for non-struct value")
		}
	})
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"nil value", nil, 0},
		{"float64", float64(10.5), 10.5},
		{"float32", float32(10.5), 10.5},
		{"int", int(10), 10.0},
		{"int64", int64(10), 10.0},
		{"int32", int32(10), 10.0},
		{"int16", int16(10), 10.0},
		{"int8", int8(10), 10.0},
		{"uint", uint(10), 10.0},
		{"uint64", uint64(10), 10.0},
		{"uint32", uint32(10), 10.0},
		{"uint16", uint16(10), 10.0},
		{"uint8", uint8(10), 10.0},
		{"string", "not a number", 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.input)
			if result != tt.expected {
				t.Errorf("toFloat64(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestApplyExpandComputeToResults(t *testing.T) {
	t.Run("nil results returns nil", func(t *testing.T) {
		result := ApplyExpandComputeToResults(nil, []ExpandOption{})
		if result != nil {
			t.Error("expected nil result")
		}
	})

	t.Run("empty expand options returns results unchanged", func(t *testing.T) {
		products := []sharedTestProduct{{ID: 1, Name: "Product1"}}
		result := ApplyExpandComputeToResults(products, []ExpandOption{})
		if result == nil {
			t.Error("expected non-nil result")
		}
	})

	t.Run("no compute in expand options returns results unchanged", func(t *testing.T) {
		products := []sharedTestProduct{{ID: 1, Name: "Product1"}}
		expandOpts := []ExpandOption{
			{NavigationProperty: "category"},
		}
		result := ApplyExpandComputeToResults(products, expandOpts)
		if result == nil {
			t.Error("expected non-nil result")
		}
	})

	t.Run("with compute in expand options", func(t *testing.T) {
		products := []sharedTestProduct{{ID: 1, Name: "Product1", Price: 10.5}}
		expandOpts := []ExpandOption{
			{
				NavigationProperty: "category",
				Compute: &ComputeTransformation{
					Expressions: []ComputeExpression{
						{
							Alias: "doubled",
							Expression: &FilterExpression{
								Value: 2,
							},
						},
					},
				},
			},
		}
		result := ApplyExpandComputeToResults(products, expandOpts)
		resultSlice, ok := result.([]map[string]interface{})
		if !ok {
			t.Fatal("expected result to be []map[string]interface{}")
		}
		if len(resultSlice) != 1 {
			t.Fatalf("expected 1 result, got %d", len(resultSlice))
		}
	})
}

func TestConvertEntityToMapWithCompute(t *testing.T) {
	t.Run("invalid value returns empty map", func(t *testing.T) {
		var val reflect.Value
		result := convertEntityToMapWithCompute(val, []ExpandOption{})
		if len(result) != 0 {
			t.Error("expected empty map")
		}
	})

	t.Run("nil pointer returns empty map", func(t *testing.T) {
		var product *sharedTestProduct
		val := reflect.ValueOf(product)
		result := convertEntityToMapWithCompute(val, []ExpandOption{})
		if len(result) != 0 {
			t.Error("expected empty map")
		}
	})

	t.Run("non-struct returns empty map", func(t *testing.T) {
		val := reflect.ValueOf("not a struct")
		result := convertEntityToMapWithCompute(val, []ExpandOption{})
		if len(result) != 0 {
			t.Error("expected empty map")
		}
	})

	t.Run("struct without compute", func(t *testing.T) {
		product := sharedTestProduct{ID: 1, Name: "Product1", Price: 10.5}
		val := reflect.ValueOf(product)
		result := convertEntityToMapWithCompute(val, []ExpandOption{})
		if len(result) == 0 {
			t.Error("expected non-empty map")
		}
		if result["name"] != "Product1" {
			t.Error("expected 'name' to be in result")
		}
	})
}

func TestApplyComputeToNavPropertyValue(t *testing.T) {
	t.Run("invalid value returns nil", func(t *testing.T) {
		var val reflect.Value
		result := applyComputeToNavPropertyValue(val, nil)
		if result != nil {
			t.Error("expected nil result")
		}
	})

	t.Run("nil compute with valid value", func(t *testing.T) {
		product := sharedTestProduct{ID: 1, Name: "Product1"}
		val := reflect.ValueOf(product)
		result := applyComputeToNavPropertyValue(val, nil)
		if result == nil {
			t.Error("expected non-nil result")
		}
	})

	t.Run("nil pointer returns nil", func(t *testing.T) {
		var product *sharedTestProduct
		val := reflect.ValueOf(product)
		compute := &ComputeTransformation{}
		result := applyComputeToNavPropertyValue(val, compute)
		if result != nil {
			t.Error("expected nil result")
		}
	})

	t.Run("single entity with compute", func(t *testing.T) {
		category := sharedTestCategory{ID: 1, Name: "Category1"}
		val := reflect.ValueOf(category)
		compute := &ComputeTransformation{
			Expressions: []ComputeExpression{
				{
					Alias: "doubled",
					Expression: &FilterExpression{
						Value: 2,
					},
				},
			},
		}
		result := applyComputeToNavPropertyValue(val, compute)
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("expected result to be map[string]interface{}")
		}
		if resultMap["doubled"] != 2 {
			t.Error("expected 'doubled' to be 2")
		}
	})

	t.Run("collection with compute", func(t *testing.T) {
		categories := []sharedTestCategory{
			{ID: 1, Name: "Category1"},
			{ID: 2, Name: "Category2"},
		}
		val := reflect.ValueOf(categories)
		compute := &ComputeTransformation{
			Expressions: []ComputeExpression{
				{
					Alias: "doubled",
					Expression: &FilterExpression{
						Value: 2,
					},
				},
			},
		}
		result := applyComputeToNavPropertyValue(val, compute)
		resultSlice, ok := result.([]map[string]interface{})
		if !ok {
			t.Fatal("expected result to be []map[string]interface{}")
		}
		if len(resultSlice) != 2 {
			t.Fatalf("expected 2 results, got %d", len(resultSlice))
		}
	})
}

func TestConvertNavEntityToMapWithCompute(t *testing.T) {
	t.Run("invalid value returns empty map", func(t *testing.T) {
		var val reflect.Value
		compute := &ComputeTransformation{}
		result := convertNavEntityToMapWithCompute(val, compute)
		if len(result) != 0 {
			t.Error("expected empty map")
		}
	})

	t.Run("nil pointer returns empty map", func(t *testing.T) {
		var category *sharedTestCategory
		val := reflect.ValueOf(category)
		compute := &ComputeTransformation{}
		result := convertNavEntityToMapWithCompute(val, compute)
		if len(result) != 0 {
			t.Error("expected empty map")
		}
	})

	t.Run("non-struct returns empty map", func(t *testing.T) {
		val := reflect.ValueOf("not a struct")
		compute := &ComputeTransformation{}
		result := convertNavEntityToMapWithCompute(val, compute)
		if len(result) != 0 {
			t.Error("expected empty map")
		}
	})

	t.Run("struct with compute", func(t *testing.T) {
		category := sharedTestCategory{ID: 1, Name: "Category1"}
		val := reflect.ValueOf(category)
		compute := &ComputeTransformation{
			Expressions: []ComputeExpression{
				{
					Alias: "doubled",
					Expression: &FilterExpression{
						Value: 2,
					},
				},
			},
		}
		result := convertNavEntityToMapWithCompute(val, compute)
		if len(result) == 0 {
			t.Error("expected non-empty map")
		}
		if result["name"] != "Category1" {
			t.Error("expected 'name' to be 'Category1'")
		}
		if result["doubled"] != 2 {
			t.Error("expected 'doubled' to be 2")
		}
	})
}
