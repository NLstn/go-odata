package query

import (
"testing"
)

func TestIsOfFunction_EntityTypeWithAnd(t *testing.T) {
meta := getTestMetadata(t)

tests := []struct {
name           string
filter         string
expectErr      bool
expectedSQL    string
expectedArgsNo int
}{
{
name:           "isof entity type with and",
filter:         "isof('Namespace.SpecialProduct') and Price gt 100",
expectErr:      false,
expectedSQL:    "(1 = ?) AND (price > ?)",
expectedArgsNo: 2,
},
{
name:           "isof entity type eq true with and",
filter:         "isof('Namespace.SpecialProduct') eq true and Price gt 100",
expectErr:      false,
expectedSQL:    "(1 = ?) AND (price > ?)",
expectedArgsNo: 2,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
filterExpr, err := parseFilter(tt.filter, meta)
if (err != nil) != tt.expectErr {
t.Fatalf("Expected error: %v, got: %v", tt.expectErr, err)
}

if tt.expectErr {
return
}

sql, args := buildFilterCondition(filterExpr, meta)
t.Logf("OData: %s", tt.filter)
t.Logf("SQL:   %s", sql)
t.Logf("Args:  %v", args)

if sql != tt.expectedSQL {
t.Errorf("Expected SQL:\n%s\nGot:\n%s", tt.expectedSQL, sql)
}

if len(args) != tt.expectedArgsNo {
t.Errorf("Expected %d args, got %d", tt.expectedArgsNo, len(args))
}
})
}
}
