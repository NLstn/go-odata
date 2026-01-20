package query

import (
	"testing"
)

func TestBuildSimpleOperatorCondition(t *testing.T) {
	tests := []struct {
		name      string
		op        FilterOperator
		fieldName string
		value     interface{}
		wantSQL   string
		wantArgs  []interface{}
	}{
		{
			name:      "Equal operator",
			op:        OpEqual,
			fieldName: "name",
			value:     "John",
			wantSQL:   "name = ?",
			wantArgs:  []interface{}{"John"},
		},
		{
			name:      "NotEqual operator",
			op:        OpNotEqual,
			fieldName: "status",
			value:     "inactive",
			wantSQL:   "status != ?",
			wantArgs:  []interface{}{"inactive"},
		},
		{
			name:      "GreaterThan operator",
			op:        OpGreaterThan,
			fieldName: "age",
			value:     18,
			wantSQL:   "age > ?",
			wantArgs:  []interface{}{18},
		},
		{
			name:      "GreaterThanOrEqual operator",
			op:        OpGreaterThanOrEqual,
			fieldName: "score",
			value:     75.5,
			wantSQL:   "score >= ?",
			wantArgs:  []interface{}{75.5},
		},
		{
			name:      "LessThan operator",
			op:        OpLessThan,
			fieldName: "count",
			value:     100,
			wantSQL:   "count < ?",
			wantArgs:  []interface{}{100},
		},
		{
			name:      "LessThanOrEqual operator",
			op:        OpLessThanOrEqual,
			fieldName: "price",
			value:     99.99,
			wantSQL:   "price <= ?",
			wantArgs:  []interface{}{99.99},
		},
		{
			name:      "Contains operator",
			op:        OpContains,
			fieldName: "description",
			value:     "test",
			wantSQL:   "description LIKE ?",
			wantArgs:  []interface{}{"%test%"},
		},
		{
			name:      "StartsWith operator",
			op:        OpStartsWith,
			fieldName: "prefix",
			value:     "abc",
			wantSQL:   "prefix LIKE ?",
			wantArgs:  []interface{}{"abc%"},
		},
		{
			name:      "EndsWith operator",
			op:        OpEndsWith,
			fieldName: "suffix",
			value:     "xyz",
			wantSQL:   "suffix LIKE ?",
			wantArgs:  []interface{}{"%xyz"},
		},
		{
			name:      "Has operator",
			op:        OpHas,
			fieldName: "flags",
			value:     4,
			wantSQL:   "(flags & ?) = ?",
			wantArgs:  []interface{}{4, 4},
		},
		{
			name:      "Unknown operator returns empty",
			op:        FilterOperator("unknown"),
			fieldName: "field",
			value:     "value",
			wantSQL:   "",
			wantArgs:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := buildSimpleOperatorCondition(tt.op, tt.fieldName, tt.value)

			if gotSQL != tt.wantSQL {
				t.Errorf("buildSimpleOperatorCondition() SQL = %v, want %v", gotSQL, tt.wantSQL)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("buildSimpleOperatorCondition() args length = %v, want %v", len(gotArgs), len(tt.wantArgs))
			}

			for i := range gotArgs {
				if i < len(tt.wantArgs) && gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("buildSimpleOperatorCondition() args[%d] = %v, want %v", i, gotArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}
