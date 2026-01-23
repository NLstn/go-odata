package query

import (
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
)

// TestEntity with date field for date function tests
type TestEntityWithDate struct {
	ID        int       `json:"ID" odata:"key"`
	Name      string    `json:"Name"`
	CreatedAt time.Time `json:"CreatedAt"`
}

func getTestMetadataWithDate(t *testing.T) *metadata.EntityMetadata {
	meta, err := metadata.AnalyzeEntity(TestEntityWithDate{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}
	return meta
}

func TestDateFunctions_Year(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "year simple",
			filter:    "year(CreatedAt) eq 2024",
			expectErr: false,
		},
		{
			name:      "year with comparison",
			filter:    "year(CreatedAt) gt 2020",
			expectErr: false,
		},
		{
			name:      "year in complex expression",
			filter:    "year(CreatedAt) eq 2024 and Name eq 'Test'",
			expectErr: false,
		},
		{
			name:      "year wrong argument count",
			filter:    "year(CreatedAt, 2024) eq 2024",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestDateFunctions_Month(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "month simple",
			filter:    "month(CreatedAt) eq 12",
			expectErr: false,
		},
		{
			name:      "month with comparison",
			filter:    "month(CreatedAt) ge 6",
			expectErr: false,
		},
		{
			name:      "month in complex expression",
			filter:    "month(CreatedAt) eq 1 or Name eq 'Test'",
			expectErr: false,
		},
		{
			name:      "month wrong argument count",
			filter:    "month() eq 12",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestDateFunctions_Day(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "day simple",
			filter:    "day(CreatedAt) eq 15",
			expectErr: false,
		},
		{
			name:      "day with comparison",
			filter:    "day(CreatedAt) le 31",
			expectErr: false,
		},
		{
			name:      "day in complex expression",
			filter:    "day(CreatedAt) gt 20 and month(CreatedAt) eq 12",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestDateFunctions_Hour(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "hour simple",
			filter:    "hour(CreatedAt) eq 14",
			expectErr: false,
		},
		{
			name:      "hour with comparison",
			filter:    "hour(CreatedAt) lt 12",
			expectErr: false,
		},
		{
			name:      "hour in complex expression",
			filter:    "hour(CreatedAt) ge 9 and hour(CreatedAt) le 17",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestDateFunctions_Minute(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "minute simple",
			filter:    "minute(CreatedAt) eq 30",
			expectErr: false,
		},
		{
			name:      "minute with comparison",
			filter:    "minute(CreatedAt) ne 0",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestDateFunctions_Second(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "second simple",
			filter:    "second(CreatedAt) eq 45",
			expectErr: false,
		},
		{
			name:      "second with comparison",
			filter:    "second(CreatedAt) lt 60",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestDateFunctions_Date(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "date simple",
			filter:    "date(CreatedAt) eq '2024-12-25'",
			expectErr: false,
		},
		{
			name:      "date with comparison",
			filter:    "date(CreatedAt) ge '2024-01-01'",
			expectErr: false,
		},
		{
			name:      "date in complex expression",
			filter:    "date(CreatedAt) eq '2024-06-15' and Name ne ''",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestDateFunctions_Time(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "time simple",
			filter:    "time(CreatedAt) eq '14:30:00'",
			expectErr: false,
		},
		{
			name:      "time with comparison",
			filter:    "time(CreatedAt) lt '12:00:00'",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestDateFunctions_Combined(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "multiple date functions",
			filter:    "year(CreatedAt) eq 2024 and month(CreatedAt) eq 12",
			expectErr: false,
		},
		{
			name:      "date and time functions",
			filter:    "date(CreatedAt) eq '2024-12-25' and hour(CreatedAt) ge 9",
			expectErr: false,
		},
		{
			name:      "all component functions",
			filter:    "year(CreatedAt) eq 2024 and month(CreatedAt) eq 12 and day(CreatedAt) eq 25",
			expectErr: false,
		},
		{
			name:      "time components",
			filter:    "hour(CreatedAt) eq 14 and minute(CreatedAt) eq 30 and second(CreatedAt) eq 0",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if (err != nil) != tt.expectErr {
				t.Errorf("Expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr && filterExpr == nil {
				t.Error("Expected non-nil FilterExpression")
			}
		})
	}
}

func TestDateFunctions_SQLGeneration(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name           string
		filter         string
		expectErr      bool
		expectedSQL    string
		expectedArgsNo int
	}{
		{
			name:           "year SQL",
			filter:         "year(CreatedAt) eq 2024",
			expectErr:      false,
			expectedSQL:    "CAST(strftime('%Y', created_at) AS INTEGER) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "month SQL",
			filter:         "month(CreatedAt) eq 12",
			expectErr:      false,
			expectedSQL:    "CAST(strftime('%m', created_at) AS INTEGER) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "day SQL",
			filter:         "day(CreatedAt) eq 25",
			expectErr:      false,
			expectedSQL:    "CAST(strftime('%d', created_at) AS INTEGER) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "hour SQL",
			filter:         "hour(CreatedAt) eq 14",
			expectErr:      false,
			expectedSQL:    "CAST(strftime('%H', created_at) AS INTEGER) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "minute SQL",
			filter:         "minute(CreatedAt) eq 30",
			expectErr:      false,
			expectedSQL:    "CAST(strftime('%M', created_at) AS INTEGER) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "second SQL",
			filter:         "second(CreatedAt) eq 45",
			expectErr:      false,
			expectedSQL:    "CAST(strftime('%S', created_at) AS INTEGER) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "date SQL",
			filter:         "date(CreatedAt) eq '2024-12-25'",
			expectErr:      false,
			expectedSQL:    "DATE(created_at) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "time SQL",
			filter:         "time(CreatedAt) eq '14:30:00'",
			expectErr:      false,
			expectedSQL:    "TIME(created_at) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "year with greater than",
			filter:         "year(CreatedAt) gt 2020",
			expectErr:      false,
			expectedSQL:    "CAST(strftime('%Y', created_at) AS INTEGER) > ?",
			expectedArgsNo: 1,
		},
		{
			name:           "month with less than or equal",
			filter:         "month(CreatedAt) le 6",
			expectErr:      false,
			expectedSQL:    "CAST(strftime('%m', created_at) AS INTEGER) <= ?",
			expectedArgsNo: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Tokenization failed: %v", err)
				}
				return
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("Parsing failed: %v", err)
				}
				return
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if err != nil {
				if !tt.expectErr {
					t.Fatalf("AST to FilterExpression failed: %v", err)
				}
				return
			}

			if tt.expectErr {
				return
			}

			sql, args := buildFilterCondition("sqlite", filterExpr, meta)
			if sql != tt.expectedSQL {
				t.Errorf("Expected SQL: %s, got: %s", tt.expectedSQL, sql)
			}
			if len(args) != tt.expectedArgsNo {
				t.Errorf("Expected %d args, got %d", tt.expectedArgsNo, len(args))
			}
		})
	}
}
