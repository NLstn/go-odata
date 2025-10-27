package query

import (
	"net/url"
	"testing"
)

// TestCompute_ArithmeticOperations tests $compute with arithmetic operations
func TestCompute_ArithmeticOperations(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		compute   string
		expectErr bool
	}{
		{
			name:      "simple multiplication",
			compute:   "Price mul 1.1 as PriceWithTax",
			expectErr: false,
		},
		{
			name:      "simple division",
			compute:   "Price div 2 as HalfPrice",
			expectErr: false,
		},
		{
			name:      "simple addition",
			compute:   "Price add 10 as IncreasedPrice",
			expectErr: false,
		},
		{
			name:      "simple subtraction",
			compute:   "Price sub 5 as DiscountedPrice",
			expectErr: false,
		},
		{
			name:      "modulo operation",
			compute:   "Price mod 10 as PriceRemainder",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCompute("compute("+tt.compute+")", meta)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil || result.Compute == nil {
				t.Error("Expected non-nil compute transformation")
				return
			}

			if len(result.Compute.Expressions) == 0 {
				t.Error("Expected at least one compute expression")
			}
		})
	}
}

// TestCompute_StringFunctions tests $compute with string functions
func TestCompute_StringFunctions(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		compute   string
		expectErr bool
	}{
		{
			name:      "toupper function",
			compute:   "toupper(Name) as UpperName",
			expectErr: false,
		},
		{
			name:      "tolower function",
			compute:   "tolower(Name) as LowerName",
			expectErr: false,
		},
		{
			name:      "trim function",
			compute:   "trim(Name) as TrimmedName",
			expectErr: false,
		},
		{
			name:      "length function",
			compute:   "length(Name) as NameLength",
			expectErr: false,
		},
		{
			name:      "concat function",
			compute:   "concat(Name,'_suffix') as ExtendedName",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCompute("compute("+tt.compute+")", meta)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil || result.Compute == nil {
				t.Error("Expected non-nil compute transformation")
				return
			}

			if len(result.Compute.Expressions) == 0 {
				t.Error("Expected at least one compute expression")
			}
		})
	}
}

// TestCompute_MultipleExpressions tests $compute with multiple computed properties
func TestCompute_MultipleExpressions(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name          string
		compute       string
		expectErr     bool
		expectedCount int
	}{
		{
			name:          "two computed properties",
			compute:       "Price mul 1.1 as WithTax,Price mul 0.9 as Discounted",
			expectErr:     false,
			expectedCount: 2,
		},
		{
			name:          "three computed properties",
			compute:       "Price mul 1.1 as WithTax,Price mul 0.9 as Discounted,Price div 2 as HalfPrice",
			expectErr:     false,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCompute("compute("+tt.compute+")", meta)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil || result.Compute == nil {
				t.Error("Expected non-nil compute transformation")
				return
			}

			if len(result.Compute.Expressions) != tt.expectedCount {
				t.Errorf("Expected %d expressions, got %d",
					tt.expectedCount, len(result.Compute.Expressions))
			}
		})
	}
}

// TestCompute_WithSelect tests $compute combined with $select
func TestCompute_WithSelect(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		compute   string
		select_   string
		expectErr bool
	}{
		{
			name:      "select computed property",
			compute:   "Price mul 2 as DoublePrice",
			select_:   "Name,DoublePrice",
			expectErr: false,
		},
		{
			name:      "select multiple including computed",
			compute:   "Price mul 1.1 as WithTax,Price mul 0.9 as Discounted",
			select_:   "Name,WithTax,Discounted",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams := url.Values{}
			queryParams.Set("$compute", tt.compute)
			queryParams.Set("$select", tt.select_)

			options, err := ParseQueryOptions(queryParams, meta)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if options.Compute == nil {
				t.Error("Expected non-nil compute option")
			}

			if len(options.Select) == 0 {
				t.Error("Expected non-empty select option")
			}
		})
	}
}

// TestCompute_WithFilter tests $compute combined with $filter
func TestCompute_WithFilter(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		compute   string
		filter    string
		expectErr bool
	}{
		{
			name:      "filter on base property",
			compute:   "Price mul 1.1 as PriceWithTax",
			filter:    "Price gt 100",
			expectErr: false,
		},
		// Note: Filtering on computed properties is not yet supported in this implementation
		// as the computed property only exists in the SELECT clause, not in the WHERE clause
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams := url.Values{}
			queryParams.Set("$compute", tt.compute)
			queryParams.Set("$filter", tt.filter)

			options, err := ParseQueryOptions(queryParams, meta)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if options.Compute == nil {
				t.Error("Expected non-nil compute option")
			}

			if options.Filter == nil {
				t.Error("Expected non-nil filter option")
			}
		})
	}
}

// TestCompute_WithOrderBy tests $compute combined with $orderby
func TestCompute_WithOrderBy(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		compute   string
		orderby   string
		expectErr bool
	}{
		{
			name:      "orderby base property",
			compute:   "Price div 2 as HalfPrice",
			orderby:   "Price",
			expectErr: false,
		},
		// Note: Ordering by computed properties is not yet supported in this implementation
		// as the computed property only exists in the SELECT clause
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams := url.Values{}
			queryParams.Set("$compute", tt.compute)
			queryParams.Set("$orderby", tt.orderby)

			options, err := ParseQueryOptions(queryParams, meta)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if options.Compute == nil {
				t.Error("Expected non-nil compute option")
			}

			if len(options.OrderBy) == 0 {
				t.Error("Expected non-empty orderby option")
			}
		})
	}
}

// TestCompute_InvalidSyntax tests $compute with invalid syntax
func TestCompute_InvalidSyntax(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		compute   string
		expectErr bool
	}{
		{
			name:      "missing alias",
			compute:   "Price mul 1.1",
			expectErr: true,
		},
		{
			name:      "invalid expression",
			compute:   "InvalidSyntax",
			expectErr: true,
		},
		{
			name:      "missing as keyword",
			compute:   "Price mul 1.1 PriceWithTax",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCompute("compute("+tt.compute+")", meta)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestCompute_ParseFromQueryOptions tests parsing $compute from query parameters
func TestCompute_ParseFromQueryOptions(t *testing.T) {
	meta := getTestMetadata(t)

	tests := []struct {
		name      string
		compute   string
		expectErr bool
	}{
		{
			name:      "valid arithmetic compute",
			compute:   "Price mul 1.1 as PriceWithTax",
			expectErr: false,
		},
		{
			name:      "valid string function compute",
			compute:   "toupper(Name) as UpperName",
			expectErr: false,
		},
		{
			name:      "invalid syntax",
			compute:   "InvalidSyntax",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParams := url.Values{}
			queryParams.Set("$compute", tt.compute)

			options, err := ParseQueryOptions(queryParams, meta)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if options.Compute == nil {
				t.Error("Expected non-nil compute option")
			}
		})
	}
}
