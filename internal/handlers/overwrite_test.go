package handlers

import (
	"testing"
)

func TestEntityOverwriteHandlers_HasGetCollection(t *testing.T) {
	tests := []struct {
		name     string
		handlers *entityOverwriteHandlers
		want     bool
	}{
		{
			name:     "Nil handlers",
			handlers: nil,
			want:     false,
		},
		{
			name:     "Empty handlers",
			handlers: &entityOverwriteHandlers{},
			want:     false,
		},
		{
			name: "With GetCollection handler",
			handlers: &entityOverwriteHandlers{
				getCollection: func(*OverwriteContext) (*CollectionResult, error) {
					return nil, nil
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.handlers.hasGetCollection(); got != tt.want {
				t.Errorf("hasGetCollection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityOverwriteHandlers_HasGetEntity(t *testing.T) {
	tests := []struct {
		name     string
		handlers *entityOverwriteHandlers
		want     bool
	}{
		{
			name:     "Nil handlers",
			handlers: nil,
			want:     false,
		},
		{
			name:     "Empty handlers",
			handlers: &entityOverwriteHandlers{},
			want:     false,
		},
		{
			name: "With GetEntity handler",
			handlers: &entityOverwriteHandlers{
				getEntity: func(*OverwriteContext) (interface{}, error) {
					return nil, nil
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.handlers.hasGetEntity(); got != tt.want {
				t.Errorf("hasGetEntity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityOverwriteHandlers_HasCreate(t *testing.T) {
	tests := []struct {
		name     string
		handlers *entityOverwriteHandlers
		want     bool
	}{
		{
			name:     "Nil handlers",
			handlers: nil,
			want:     false,
		},
		{
			name:     "Empty handlers",
			handlers: &entityOverwriteHandlers{},
			want:     false,
		},
		{
			name: "With Create handler",
			handlers: &entityOverwriteHandlers{
				create: func(*OverwriteContext, interface{}) (interface{}, error) {
					return nil, nil
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.handlers.hasCreate(); got != tt.want {
				t.Errorf("hasCreate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityOverwriteHandlers_HasUpdate(t *testing.T) {
	tests := []struct {
		name     string
		handlers *entityOverwriteHandlers
		want     bool
	}{
		{
			name:     "Nil handlers",
			handlers: nil,
			want:     false,
		},
		{
			name:     "Empty handlers",
			handlers: &entityOverwriteHandlers{},
			want:     false,
		},
		{
			name: "With Update handler",
			handlers: &entityOverwriteHandlers{
				update: func(*OverwriteContext, map[string]interface{}, bool) (interface{}, error) {
					return nil, nil
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.handlers.hasUpdate(); got != tt.want {
				t.Errorf("hasUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityOverwriteHandlers_HasDelete(t *testing.T) {
	tests := []struct {
		name     string
		handlers *entityOverwriteHandlers
		want     bool
	}{
		{
			name:     "Nil handlers",
			handlers: nil,
			want:     false,
		},
		{
			name:     "Empty handlers",
			handlers: &entityOverwriteHandlers{},
			want:     false,
		},
		{
			name: "With Delete handler",
			handlers: &entityOverwriteHandlers{
				delete: func(*OverwriteContext) error {
					return nil
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.handlers.hasDelete(); got != tt.want {
				t.Errorf("hasDelete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityOverwriteHandlers_HasGetCount(t *testing.T) {
	tests := []struct {
		name     string
		handlers *entityOverwriteHandlers
		want     bool
	}{
		{
			name:     "Nil handlers",
			handlers: nil,
			want:     false,
		},
		{
			name:     "Empty handlers",
			handlers: &entityOverwriteHandlers{},
			want:     false,
		},
		{
			name: "With GetCount handler",
			handlers: &entityOverwriteHandlers{
				getCount: func(*OverwriteContext) (int64, error) {
					return 0, nil
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.handlers.hasGetCount(); got != tt.want {
				t.Errorf("hasGetCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityOverwrite_AllHandlers(t *testing.T) {
	// Test that all handlers can be set on EntityOverwrite
	overwrite := &EntityOverwrite{
		GetCollection: func(*OverwriteContext) (*CollectionResult, error) {
			return &CollectionResult{Items: []string{"item1", "item2"}, Count: nil}, nil
		},
		GetEntity: func(*OverwriteContext) (interface{}, error) {
			return map[string]string{"id": "1"}, nil
		},
		Create: func(*OverwriteContext, interface{}) (interface{}, error) {
			return map[string]string{"id": "new"}, nil
		},
		Update: func(*OverwriteContext, map[string]interface{}, bool) (interface{}, error) {
			return map[string]string{"id": "updated"}, nil
		},
		Delete: func(*OverwriteContext) error {
			return nil
		},
		GetCount: func(*OverwriteContext) (int64, error) {
			return 42, nil
		},
	}

	// Verify all handlers are set
	if overwrite.GetCollection == nil {
		t.Error("GetCollection should be set")
	}
	if overwrite.GetEntity == nil {
		t.Error("GetEntity should be set")
	}
	if overwrite.Create == nil {
		t.Error("Create should be set")
	}
	if overwrite.Update == nil {
		t.Error("Update should be set")
	}
	if overwrite.Delete == nil {
		t.Error("Delete should be set")
	}
	if overwrite.GetCount == nil {
		t.Error("GetCount should be set")
	}
}

func TestCollectionResult(t *testing.T) {
	t.Run("CollectionResult with items and count", func(t *testing.T) {
		count := int64(10)
		result := &CollectionResult{
			Items: []string{"item1", "item2", "item3"},
			Count: &count,
		}

		items, ok := result.Items.([]string)
		if !ok {
			t.Fatal("Items should be []string")
		}
		if len(items) != 3 {
			t.Errorf("Expected 3 items, got %d", len(items))
		}

		if result.Count == nil {
			t.Fatal("Count should not be nil")
		}
		if *result.Count != 10 {
			t.Errorf("Expected count 10, got %d", *result.Count)
		}
	})

	t.Run("CollectionResult without count", func(t *testing.T) {
		result := &CollectionResult{
			Items: []int{1, 2, 3, 4, 5},
			Count: nil,
		}

		items, ok := result.Items.([]int)
		if !ok {
			t.Fatal("Items should be []int")
		}
		if len(items) != 5 {
			t.Errorf("Expected 5 items, got %d", len(items))
		}

		if result.Count != nil {
			t.Error("Count should be nil")
		}
	})

	t.Run("CollectionResult with empty items", func(t *testing.T) {
		result := &CollectionResult{
			Items: []interface{}{},
			Count: nil,
		}

		items, ok := result.Items.([]interface{})
		if !ok {
			t.Fatal("Items should be []interface{}")
		}
		if len(items) != 0 {
			t.Errorf("Expected 0 items, got %d", len(items))
		}
	})
}

func TestOverwriteContext(t *testing.T) {
	t.Run("OverwriteContext with all fields", func(t *testing.T) {
		ctx := &OverwriteContext{
			QueryOptions: nil, // Would normally be query.QueryOptions
			EntityKey:    "123",
			Request:      nil, // Would normally be *http.Request
		}

		if ctx.EntityKey != "123" {
			t.Errorf("EntityKey = %v, want 123", ctx.EntityKey)
		}
	})

	t.Run("OverwriteContext for collection", func(t *testing.T) {
		ctx := &OverwriteContext{
			QueryOptions: nil,
			EntityKey:    "", // Empty for collection operations
			Request:      nil,
		}

		if ctx.EntityKey != "" {
			t.Errorf("EntityKey should be empty for collection, got %v", ctx.EntityKey)
		}
	})

	t.Run("OverwriteContext with EntityKeyValues for single key", func(t *testing.T) {
		keyValues := map[string]interface{}{
			"ID": int64(123),
		}
		ctx := &OverwriteContext{
			QueryOptions:    nil,
			EntityKey:       "123",
			EntityKeyValues: keyValues,
			Request:         nil,
		}

		if ctx.EntityKeyValues == nil {
			t.Fatal("EntityKeyValues should not be nil")
		}
		if len(ctx.EntityKeyValues) != 1 {
			t.Errorf("Expected 1 key-value pair, got %d", len(ctx.EntityKeyValues))
		}
		if val, ok := ctx.EntityKeyValues["ID"]; !ok || val != int64(123) {
			t.Errorf("Expected ID=123, got %v", ctx.EntityKeyValues)
		}
	})

	t.Run("OverwriteContext with EntityKeyValues for composite key", func(t *testing.T) {
		keyValues := map[string]interface{}{
			"OrderID":   int64(1),
			"ProductID": int64(5),
		}
		ctx := &OverwriteContext{
			QueryOptions:    nil,
			EntityKey:       "OrderID=1,ProductID=5",
			EntityKeyValues: keyValues,
			Request:         nil,
		}

		if ctx.EntityKeyValues == nil {
			t.Fatal("EntityKeyValues should not be nil")
		}
		if len(ctx.EntityKeyValues) != 2 {
			t.Errorf("Expected 2 key-value pairs, got %d", len(ctx.EntityKeyValues))
		}
		if val, ok := ctx.EntityKeyValues["OrderID"]; !ok || val != int64(1) {
			t.Errorf("Expected OrderID=1, got %v", ctx.EntityKeyValues["OrderID"])
		}
		if val, ok := ctx.EntityKeyValues["ProductID"]; !ok || val != int64(5) {
			t.Errorf("Expected ProductID=5, got %v", ctx.EntityKeyValues["ProductID"])
		}
	})
}
