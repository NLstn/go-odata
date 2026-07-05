package query

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ISO8601Duration is a string alias registered as an OData TypeDefinition with
// underlying type Edm.Duration, so $filter type-checking recognizes properties
// backed by ISO-8601 duration strings (e.g. "P1D", "PT2H") as genuinely
// Edm.Duration rather than Edm.String.
type ISO8601Duration string

func init() {
	if err := metadata.RegisterTypeDefinition(reflect.TypeOf(ISO8601Duration("")), metadata.TypeDefinitionInfo{
		Name:           "ISO8601Duration",
		UnderlyingType: "Edm.Duration",
	}); err != nil {
		panic(err)
	}
}

// TestEntity with date field for date function tests
type TestEntityWithDate struct {
	ID               int             `json:"ID" odata:"key"`
	Name             string          `json:"Name"`
	CreatedAt        time.Time       `json:"CreatedAt"`
	ShippingDuration ISO8601Duration `json:"ShippingDuration"`
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
		{
			// Regression test for issue #800: year() must be rejected against a
			// genuinely Edm.String property instead of silently matching nothing.
			name:      "year against Edm.String property is rejected",
			filter:    "year(Name) eq 2024",
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

// TestDateLiteralAgainstStringProperty is a regression test for issue #800: an
// unquoted date-shaped literal (parsed via the dedicated ABNF date production, as
// opposed to a quoted string literal) must be rejected when compared to a property
// genuinely declared Edm.String, instead of silently matching nothing.
func TestDateLiteralAgainstStringProperty(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tokenizer := NewTokenizer("Name eq 2024-01-15")
	tokens, err := tokenizer.TokenizeAll()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	parser := NewASTParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parsing failed: %v", err)
	}
	defer ReleaseASTNode(ast)

	if _, err := ASTToFilterExpression(ast, meta); err == nil {
		t.Error("expected type mismatch error comparing a date literal to an Edm.String property, got nil")
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
		{
			name:           "fractionalseconds SQL",
			filter:         "fractionalseconds(CreatedAt) gt 0.5",
			expectErr:      false,
			expectedSQL:    "(CAST(strftime('%f', created_at) AS REAL) - CAST(strftime('%S', created_at) AS INTEGER)) > ?",
			expectedArgsNo: 1,
		},
		{
			name:           "totaloffsetminutes SQL",
			filter:         "totaloffsetminutes(CreatedAt) eq -300",
			expectErr:      false,
			expectedSQL:    "0 = ?",
			expectedArgsNo: 1,
		},
		{
			name:      "totalseconds SQL",
			filter:    "totalseconds(ShippingDuration) gt 3600",
			expectErr: false,
			// The SQLite expression parses ISO 8601 duration strings into total seconds.
			// 3-branch CASE: NULL, positive 'P%', negative '-P%'.
			expectedSQL: "CASE WHEN \"shipping_duration\" IS NULL THEN NULL WHEN \"shipping_duration\" LIKE 'P%' THEN (" +
				"CAST(CASE WHEN INSTR(\"shipping_duration\",'D')>0 AND (INSTR(\"shipping_duration\",'T')=0 OR INSTR(\"shipping_duration\",'D')<INSTR(\"shipping_duration\",'T'))" +
				" THEN SUBSTR(\"shipping_duration\",2,INSTR(\"shipping_duration\",'D')-2) ELSE 0 END AS INTEGER)*86400" +
				"+CAST(CASE WHEN INSTR(\"shipping_duration\",'T')>0 AND INSTR(\"shipping_duration\",'H')>INSTR(\"shipping_duration\",'T')" +
				" THEN SUBSTR(\"shipping_duration\",INSTR(\"shipping_duration\",'T')+1,INSTR(\"shipping_duration\",'H')-INSTR(\"shipping_duration\",'T')-1) ELSE 0 END AS INTEGER)*3600" +
				"+CAST(CASE WHEN INSTR(\"shipping_duration\",'T')>0 AND INSTR(SUBSTR(\"shipping_duration\",INSTR(\"shipping_duration\",'T')+1),'M')>0" +
				" THEN SUBSTR(\"shipping_duration\"," +
				"CASE WHEN INSTR(\"shipping_duration\",'H')>INSTR(\"shipping_duration\",'T') THEN INSTR(\"shipping_duration\",'H')+1 ELSE INSTR(\"shipping_duration\",'T')+1 END," +
				"INSTR(\"shipping_duration\",'T')+INSTR(SUBSTR(\"shipping_duration\",INSTR(\"shipping_duration\",'T')+1),'M')-1" +
				"-CASE WHEN INSTR(\"shipping_duration\",'H')>INSTR(\"shipping_duration\",'T') THEN INSTR(\"shipping_duration\",'H') ELSE INSTR(\"shipping_duration\",'T') END)" +
				" ELSE 0 END AS INTEGER)*60" +
				"+CAST(CASE WHEN INSTR(\"shipping_duration\",'T')>0 AND INSTR(\"shipping_duration\",'S')>INSTR(\"shipping_duration\",'T')" +
				" THEN SUBSTR(\"shipping_duration\"," +
				"CASE WHEN INSTR(SUBSTR(\"shipping_duration\",INSTR(\"shipping_duration\",'T')+1),'M')>0 THEN INSTR(\"shipping_duration\",'T')+INSTR(SUBSTR(\"shipping_duration\",INSTR(\"shipping_duration\",'T')+1),'M')+1" +
				" WHEN INSTR(\"shipping_duration\",'H')>INSTR(\"shipping_duration\",'T') THEN INSTR(\"shipping_duration\",'H')+1 ELSE INSTR(\"shipping_duration\",'T')+1 END," +
				"INSTR(\"shipping_duration\",'S')" +
				"-CASE WHEN INSTR(SUBSTR(\"shipping_duration\",INSTR(\"shipping_duration\",'T')+1),'M')>0 THEN INSTR(\"shipping_duration\",'T')+INSTR(SUBSTR(\"shipping_duration\",INSTR(\"shipping_duration\",'T')+1),'M')" +
				" WHEN INSTR(\"shipping_duration\",'H')>INSTR(\"shipping_duration\",'T') THEN INSTR(\"shipping_duration\",'H') ELSE INSTR(\"shipping_duration\",'T') END-1)" +
				" ELSE '0' END AS REAL))" +
				" WHEN \"shipping_duration\" LIKE '-P%' THEN -1.0*(" +
				"CAST(CASE WHEN INSTR(SUBSTR(\"shipping_duration\",2),'D')>0 AND (INSTR(SUBSTR(\"shipping_duration\",2),'T')=0 OR INSTR(SUBSTR(\"shipping_duration\",2),'D')<INSTR(SUBSTR(\"shipping_duration\",2),'T'))" +
				" THEN SUBSTR(SUBSTR(\"shipping_duration\",2),2,INSTR(SUBSTR(\"shipping_duration\",2),'D')-2) ELSE 0 END AS INTEGER)*86400" +
				"+CAST(CASE WHEN INSTR(SUBSTR(\"shipping_duration\",2),'T')>0 AND INSTR(SUBSTR(\"shipping_duration\",2),'H')>INSTR(SUBSTR(\"shipping_duration\",2),'T')" +
				" THEN SUBSTR(SUBSTR(\"shipping_duration\",2),INSTR(SUBSTR(\"shipping_duration\",2),'T')+1,INSTR(SUBSTR(\"shipping_duration\",2),'H')-INSTR(SUBSTR(\"shipping_duration\",2),'T')-1) ELSE 0 END AS INTEGER)*3600" +
				"+CAST(CASE WHEN INSTR(SUBSTR(\"shipping_duration\",2),'T')>0 AND INSTR(SUBSTR(SUBSTR(\"shipping_duration\",2),INSTR(SUBSTR(\"shipping_duration\",2),'T')+1),'M')>0" +
				" THEN SUBSTR(SUBSTR(\"shipping_duration\",2)," +
				"CASE WHEN INSTR(SUBSTR(\"shipping_duration\",2),'H')>INSTR(SUBSTR(\"shipping_duration\",2),'T') THEN INSTR(SUBSTR(\"shipping_duration\",2),'H')+1 ELSE INSTR(SUBSTR(\"shipping_duration\",2),'T')+1 END," +
				"INSTR(SUBSTR(\"shipping_duration\",2),'T')+INSTR(SUBSTR(SUBSTR(\"shipping_duration\",2),INSTR(SUBSTR(\"shipping_duration\",2),'T')+1),'M')-1" +
				"-CASE WHEN INSTR(SUBSTR(\"shipping_duration\",2),'H')>INSTR(SUBSTR(\"shipping_duration\",2),'T') THEN INSTR(SUBSTR(\"shipping_duration\",2),'H') ELSE INSTR(SUBSTR(\"shipping_duration\",2),'T') END)" +
				" ELSE 0 END AS INTEGER)*60" +
				"+CAST(CASE WHEN INSTR(SUBSTR(\"shipping_duration\",2),'T')>0 AND INSTR(SUBSTR(\"shipping_duration\",2),'S')>INSTR(SUBSTR(\"shipping_duration\",2),'T')" +
				" THEN SUBSTR(SUBSTR(\"shipping_duration\",2)," +
				"CASE WHEN INSTR(SUBSTR(SUBSTR(\"shipping_duration\",2),INSTR(SUBSTR(\"shipping_duration\",2),'T')+1),'M')>0 THEN INSTR(SUBSTR(\"shipping_duration\",2),'T')+INSTR(SUBSTR(SUBSTR(\"shipping_duration\",2),INSTR(SUBSTR(\"shipping_duration\",2),'T')+1),'M')+1" +
				" WHEN INSTR(SUBSTR(\"shipping_duration\",2),'H')>INSTR(SUBSTR(\"shipping_duration\",2),'T') THEN INSTR(SUBSTR(\"shipping_duration\",2),'H')+1 ELSE INSTR(SUBSTR(\"shipping_duration\",2),'T')+1 END," +
				"INSTR(SUBSTR(\"shipping_duration\",2),'S')" +
				"-CASE WHEN INSTR(SUBSTR(SUBSTR(\"shipping_duration\",2),INSTR(SUBSTR(\"shipping_duration\",2),'T')+1),'M')>0 THEN INSTR(SUBSTR(\"shipping_duration\",2),'T')+INSTR(SUBSTR(SUBSTR(\"shipping_duration\",2),INSTR(SUBSTR(\"shipping_duration\",2),'T')+1),'M')" +
				" WHEN INSTR(SUBSTR(\"shipping_duration\",2),'H')>INSTR(SUBSTR(\"shipping_duration\",2),'T') THEN INSTR(SUBSTR(\"shipping_duration\",2),'H') ELSE INSTR(SUBSTR(\"shipping_duration\",2),'T') END-1)" +
				" ELSE '0' END AS REAL)) ELSE NULL END > ?",
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
			if !sqlEquivalent(tt.expectedSQL, sql) {
				t.Errorf("Expected SQL: %s, got: %s", tt.expectedSQL, sql)
			}
			if len(args) != tt.expectedArgsNo {
				t.Errorf("Expected %d args, got %d", tt.expectedArgsNo, len(args))
			}
		})
	}
}

func TestDateFunctions_SQLGenerationSQLServer(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name           string
		filter         string
		expectedSQL    string
		expectedArgsNo int
	}{
		{
			name:           "year SQL Server",
			filter:         "year(CreatedAt) eq 2024",
			expectedSQL:    "DATEPART(YEAR, TRY_CONVERT(datetime2, [created_at])) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "time SQL Server",
			filter:         "time(CreatedAt) eq '14:30:00'",
			expectedSQL:    "CAST(TRY_CONVERT(time, [created_at]) AS TIME) = ?",
			expectedArgsNo: 1,
		},
		{
			name:           "fractionalseconds SQL Server",
			filter:         "fractionalseconds(CreatedAt) gt 0.5",
			expectedSQL:    "(DATEPART(MICROSECOND, TRY_CONVERT(datetime2, [created_at])) / 1000000.0) > ?",
			expectedArgsNo: 1,
		},
		{
			name:   "totalseconds SQL Server",
			filter: "totalseconds(ShippingDuration) gt 3600",
			// Edm.Duration is an ISO-8601 string; SQL Server parses it with CHARINDEX/SUBSTRING.
			// 3-branch CASE: NULL, positive 'P%', negative '-P%'.
			expectedSQL:    "CASE WHEN [shipping_duration] IS NULL THEN NULL WHEN [shipping_duration] LIKE 'P%' THEN (CAST(CASE WHEN CHARINDEX('D',[shipping_duration])>0 AND (CHARINDEX('T',[shipping_duration])=0 OR CHARINDEX('D',[shipping_duration])<CHARINDEX('T',[shipping_duration])) THEN SUBSTRING([shipping_duration],2,CHARINDEX('D',[shipping_duration])-2) ELSE 0 END AS INT)*86400+CAST(CASE WHEN CHARINDEX('T',[shipping_duration])>0 AND CHARINDEX('H',[shipping_duration])>CHARINDEX('T',[shipping_duration]) THEN SUBSTRING([shipping_duration],CHARINDEX('T',[shipping_duration])+1,CHARINDEX('H',[shipping_duration])-CHARINDEX('T',[shipping_duration])-1) ELSE 0 END AS INT)*3600+CAST(CASE WHEN CHARINDEX('T',[shipping_duration])>0 AND CHARINDEX('M',SUBSTRING([shipping_duration],CHARINDEX('T',[shipping_duration])+1,8000))>0 THEN SUBSTRING([shipping_duration],CASE WHEN CHARINDEX('H',[shipping_duration])>CHARINDEX('T',[shipping_duration]) THEN CHARINDEX('H',[shipping_duration])+1 ELSE CHARINDEX('T',[shipping_duration])+1 END,CHARINDEX('T',[shipping_duration])+CHARINDEX('M',SUBSTRING([shipping_duration],CHARINDEX('T',[shipping_duration])+1,8000))-1-CASE WHEN CHARINDEX('H',[shipping_duration])>CHARINDEX('T',[shipping_duration]) THEN CHARINDEX('H',[shipping_duration]) ELSE CHARINDEX('T',[shipping_duration]) END) ELSE 0 END AS INT)*60+CAST(CASE WHEN CHARINDEX('T',[shipping_duration])>0 AND CHARINDEX('S',[shipping_duration])>CHARINDEX('T',[shipping_duration]) THEN SUBSTRING([shipping_duration],CASE WHEN CHARINDEX('M',SUBSTRING([shipping_duration],CHARINDEX('T',[shipping_duration])+1,8000))>0 THEN CHARINDEX('T',[shipping_duration])+CHARINDEX('M',SUBSTRING([shipping_duration],CHARINDEX('T',[shipping_duration])+1,8000))+1 WHEN CHARINDEX('H',[shipping_duration])>CHARINDEX('T',[shipping_duration]) THEN CHARINDEX('H',[shipping_duration])+1 ELSE CHARINDEX('T',[shipping_duration])+1 END,CHARINDEX('S',[shipping_duration])-CASE WHEN CHARINDEX('M',SUBSTRING([shipping_duration],CHARINDEX('T',[shipping_duration])+1,8000))>0 THEN CHARINDEX('T',[shipping_duration])+CHARINDEX('M',SUBSTRING([shipping_duration],CHARINDEX('T',[shipping_duration])+1,8000)) WHEN CHARINDEX('H',[shipping_duration])>CHARINDEX('T',[shipping_duration]) THEN CHARINDEX('H',[shipping_duration]) ELSE CHARINDEX('T',[shipping_duration]) END-1) ELSE '0' END AS FLOAT)) WHEN [shipping_duration] LIKE '-P%' THEN -1.0*(CAST(CASE WHEN CHARINDEX('D',SUBSTRING([shipping_duration],2,8000))>0 AND (CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))=0 OR CHARINDEX('D',SUBSTRING([shipping_duration],2,8000))<CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))) THEN SUBSTRING(SUBSTRING([shipping_duration],2,8000),2,CHARINDEX('D',SUBSTRING([shipping_duration],2,8000))-2) ELSE 0 END AS INT)*86400+CAST(CASE WHEN CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))>0 AND CHARINDEX('H',SUBSTRING([shipping_duration],2,8000))>CHARINDEX('T',SUBSTRING([shipping_duration],2,8000)) THEN SUBSTRING(SUBSTRING([shipping_duration],2,8000),CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+1,CHARINDEX('H',SUBSTRING([shipping_duration],2,8000))-CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))-1) ELSE 0 END AS INT)*3600+CAST(CASE WHEN CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))>0 AND CHARINDEX('M',SUBSTRING(SUBSTRING([shipping_duration],2,8000),CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+1,8000))>0 THEN SUBSTRING(SUBSTRING([shipping_duration],2,8000),CASE WHEN CHARINDEX('H',SUBSTRING([shipping_duration],2,8000))>CHARINDEX('T',SUBSTRING([shipping_duration],2,8000)) THEN CHARINDEX('H',SUBSTRING([shipping_duration],2,8000))+1 ELSE CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+1 END,CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+CHARINDEX('M',SUBSTRING(SUBSTRING([shipping_duration],2,8000),CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+1,8000))-1-CASE WHEN CHARINDEX('H',SUBSTRING([shipping_duration],2,8000))>CHARINDEX('T',SUBSTRING([shipping_duration],2,8000)) THEN CHARINDEX('H',SUBSTRING([shipping_duration],2,8000)) ELSE CHARINDEX('T',SUBSTRING([shipping_duration],2,8000)) END) ELSE 0 END AS INT)*60+CAST(CASE WHEN CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))>0 AND CHARINDEX('S',SUBSTRING([shipping_duration],2,8000))>CHARINDEX('T',SUBSTRING([shipping_duration],2,8000)) THEN SUBSTRING(SUBSTRING([shipping_duration],2,8000),CASE WHEN CHARINDEX('M',SUBSTRING(SUBSTRING([shipping_duration],2,8000),CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+1,8000))>0 THEN CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+CHARINDEX('M',SUBSTRING(SUBSTRING([shipping_duration],2,8000),CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+1,8000))+1 WHEN CHARINDEX('H',SUBSTRING([shipping_duration],2,8000))>CHARINDEX('T',SUBSTRING([shipping_duration],2,8000)) THEN CHARINDEX('H',SUBSTRING([shipping_duration],2,8000))+1 ELSE CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+1 END,CHARINDEX('S',SUBSTRING([shipping_duration],2,8000))-CASE WHEN CHARINDEX('M',SUBSTRING(SUBSTRING([shipping_duration],2,8000),CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+1,8000))>0 THEN CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+CHARINDEX('M',SUBSTRING(SUBSTRING([shipping_duration],2,8000),CHARINDEX('T',SUBSTRING([shipping_duration],2,8000))+1,8000)) WHEN CHARINDEX('H',SUBSTRING([shipping_duration],2,8000))>CHARINDEX('T',SUBSTRING([shipping_duration],2,8000)) THEN CHARINDEX('H',SUBSTRING([shipping_duration],2,8000)) ELSE CHARINDEX('T',SUBSTRING([shipping_duration],2,8000)) END-1) ELSE '0' END AS FLOAT)) ELSE NULL END > ?",
			expectedArgsNo: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenizer := NewTokenizer(tt.filter)
			tokens, err := tokenizer.TokenizeAll()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			parser := NewASTParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			defer ReleaseASTNode(ast)

			filterExpr, err := ASTToFilterExpression(ast, meta)
			if err != nil {
				t.Fatalf("AST to FilterExpression failed: %v", err)
			}

			sql, args := buildFilterCondition("sqlserver", filterExpr, meta)
			if sql != tt.expectedSQL {
				t.Errorf("Expected SQL %q, got %q", tt.expectedSQL, sql)
			}
			if len(args) != tt.expectedArgsNo {
				t.Errorf("Expected %d args, got %d", tt.expectedArgsNo, len(args))
			}
		})
	}
}

func TestDateFunctions_FractionalSeconds(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "fractionalseconds simple",
			filter:    "fractionalseconds(CreatedAt) gt 0.5",
			expectErr: false,
		},
		{
			name:      "fractionalseconds eq zero",
			filter:    "fractionalseconds(CreatedAt) eq 0",
			expectErr: false,
		},
		{
			name:      "fractionalseconds wrong argument count",
			filter:    "fractionalseconds(CreatedAt, 2024) gt 0",
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

func TestDateFunctions_TotalOffsetMinutes(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "totaloffsetminutes simple",
			filter:    "totaloffsetminutes(CreatedAt) eq -300",
			expectErr: false,
		},
		{
			name:      "totaloffsetminutes zero offset",
			filter:    "totaloffsetminutes(CreatedAt) eq 0",
			expectErr: false,
		},
		{
			name:      "totaloffsetminutes wrong argument count",
			filter:    "totaloffsetminutes(CreatedAt, 2024) eq 0",
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

func TestDateFunctions_TotalSeconds(t *testing.T) {
	// totalseconds() per OData spec operates on Edm.Duration values, so the test
	// uses the ShippingDuration field (declared Edm.Duration via a registered
	// TypeDefinition, see ISO8601Duration above).
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "totalseconds simple",
			filter:    "totalseconds(ShippingDuration) gt 3600",
			expectErr: false,
		},
		{
			name:      "totalseconds eq zero",
			filter:    "totalseconds(ShippingDuration) eq 0",
			expectErr: false,
		},
		{
			name:      "totalseconds wrong argument count",
			filter:    "totalseconds(ShippingDuration, 2024) gt 0",
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

func TestDateFunctions_MinMaxDatetime(t *testing.T) {
	meta := getTestMetadataWithDate(t)

	tests := []struct {
		name      string
		filter    string
		expectErr bool
	}{
		{
			name:      "mindatetime on right side",
			filter:    "CreatedAt ge mindatetime()",
			expectErr: false,
		},
		{
			name:      "maxdatetime on right side",
			filter:    "CreatedAt le maxdatetime()",
			expectErr: false,
		},
		{
			name:      "mindatetime with wrong arg count",
			filter:    "mindatetime(CreatedAt) eq '2024-01-01'",
			expectErr: true,
		},
		{
			name:      "maxdatetime with wrong arg count",
			filter:    "maxdatetime(CreatedAt) eq '9999-12-31'",
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

func TestDateFunctions_MinMaxDatetimeSQL(t *testing.T) {
	tests := []struct {
		name        string
		op          FilterOperator
		expectedSQL string
	}{
		{
			name:        "mindatetime sqlite",
			op:          OpMinDatetime,
			expectedSQL: "datetime('0001-01-01T00:00:00')",
		},
		{
			name:        "maxdatetime sqlite",
			op:          OpMaxDatetime,
			expectedSQL: "datetime('9999-12-31T23:59:59')",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sql, args := buildFunctionSQL("sqlite", tc.op, "", nil)
			if sql != tc.expectedSQL {
				t.Errorf("expected SQL %q, got %q", tc.expectedSQL, sql)
			}
			if len(args) != 0 {
				t.Errorf("expected 0 args, got %d", len(args))
			}
		})
	}
}

// TestTotalSecondsISO8601SQLite verifies that totalseconds() correctly parses ISO 8601
// duration strings stored as text in SQLite (e.g. "P1D", "PT2H", "P1DT2H30M").
func TestTotalSecondsISO8601SQLite(t *testing.T) {
	type DurationRow struct {
		ID           int             `gorm:"primaryKey"`
		ShippingTime ISO8601Duration `gorm:"column:shipping_time"`
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&DurationRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	rows := []DurationRow{
		{ID: 1, ShippingTime: "P1D"},       // 86400 s
		{ID: 2, ShippingTime: "PT2H"},      // 7200 s
		{ID: 3, ShippingTime: "P2D"},       // 172800 s
		{ID: 4, ShippingTime: "P1DT2H30M"}, // 95400 s
		{ID: 5, ShippingTime: ""},          // empty — should be excluded
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	meta, err := metadata.AnalyzeEntity(DurationRow{})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}

	tests := []struct {
		filter        string
		expectedCount int
		desc          string
	}{
		{"totalseconds(ShippingTime) gt 3600", 4, "all four valid durations (86400, 7200, 172800, 95400) > 3600"},
		{"totalseconds(ShippingTime) eq 7200", 1, "only PT2H equals 7200"},
		{"totalseconds(ShippingTime) ge 86400", 3, "P1D(86400), P2D(172800), P1DT2H30M(95400) >= 86400"},
		{"totalseconds(ShippingTime) ge 0", 4, "all four valid durations; empty string excluded"},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, meta, nil, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			var count int64
			if err := db.Model(&DurationRow{}).Scopes(func(d *gorm.DB) *gorm.DB {
				return ApplyFilterOnly(d, filterExpr, meta, nil)
			}).Count(&count).Error; err != nil {
				t.Fatalf("query: %v", err)
			}
			if int(count) != tt.expectedCount {
				t.Errorf("%s: expected %d rows, got %d", tt.desc, tt.expectedCount, count)
			}
		})

	}
}

// TestDurationDirectComparisonSQL verifies that a direct duration literal comparison
// (e.g. $filter=ShippingTime gt duration'PT1H') generates seconds-based SQL
// rather than a raw string comparison (issue #736).
func TestDurationDirectComparisonSQL(t *testing.T) {
	type DurationRow struct {
		ID           int             `gorm:"primaryKey"`
		ShippingTime ISO8601Duration `gorm:"column:shipping_time"`
	}

	meta, err := metadata.AnalyzeEntity(DurationRow{})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}

	tests := []struct {
		filter   string
		wantSecs float64
		wantOp   string
	}{
		{"ShippingTime gt duration'PT1H'", 3600, ">"},
		{"ShippingTime ge duration'P1D'", 86400, ">="},
		{"ShippingTime lt duration'P1DT2H30M'", 95400, "<"},
		{"ShippingTime eq duration'PT2H'", 7200, "="},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, meta, nil, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if filterExpr.ValueType != "duration" {
				t.Errorf("expected ValueType=duration, got %q", filterExpr.ValueType)
			}
			sql, args := buildFilterCondition("sqlite", filterExpr, meta)
			if len(args) != 1 {
				t.Fatalf("expected 1 bind arg, got %d; sql=%s", len(args), sql)
			}
			secs, ok := args[0].(float64)
			if !ok {
				t.Fatalf("expected float64 arg, got %T %v", args[0], args[0])
			}
			if secs != tt.wantSecs {
				t.Errorf("expected %v seconds, got %v", tt.wantSecs, secs)
			}
			// Verify the SQL contains the comparison operator (not a raw string quote)
			if !strings.Contains(sql, tt.wantOp+" ?") {
				t.Errorf("expected SQL to contain %q; got: %s", tt.wantOp+" ?", sql)
			}
			// The SQL must NOT contain a raw string literal like 'PT1H'
			if strings.Contains(sql, "'PT") || strings.Contains(sql, "'P1") || strings.Contains(sql, "'P2") {
				t.Errorf("SQL contains raw duration string literal (string comparison bug): %s", sql)
			}
		})
	}
}

// TestDurationDirectComparisonE2E verifies that direct duration literal filter
// (e.g. $filter=ShippingTime gt duration'PT1H') returns the correct rows from SQLite.
// This is the regression test for issue #736.
func TestDurationDirectComparisonE2E(t *testing.T) {
	type DurationRow struct {
		ID           int             `gorm:"primaryKey"`
		ShippingTime ISO8601Duration `gorm:"column:shipping_time"`
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&DurationRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	rows := []DurationRow{
		{ID: 1, ShippingTime: "P1D"},       // 86400 s
		{ID: 2, ShippingTime: "PT2H"},      // 7200 s
		{ID: 3, ShippingTime: "P2D"},       // 172800 s
		{ID: 4, ShippingTime: "P1DT2H30M"}, // 95400 s
		{ID: 5, ShippingTime: ""},          // empty — should be excluded
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	meta, err := metadata.AnalyzeEntity(DurationRow{})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}

	tests := []struct {
		filter        string
		expectedCount int
		desc          string
	}{
		// These cases fail without the fix because lexicographic comparison gives
		// wrong results: "P1D" < "PT1H" (string), but 86400 > 3600 (numeric).
		{"ShippingTime gt duration'PT1H'", 4, "P1D,PT2H,P2D,P1DT2H30M are all > 1 hour"},
		{"ShippingTime ge duration'P1D'", 3, "P1D,P2D,P1DT2H30M are >= 1 day"},
		{"ShippingTime lt duration'PT3H'", 1, "only PT2H is < 3 hours"},
		{"ShippingTime eq duration'PT2H'", 1, "only PT2H equals 2 hours"},
		{"ShippingTime ne duration'PT2H'", 3, "P1D, P2D, P1DT2H30M are != 2 hours (empty excluded)"},
	}

	for _, tt := range tests {
		t.Run(tt.filter, func(t *testing.T) {
			filterExpr, err := parseFilter(tt.filter, meta, nil, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			var count int64
			if err := db.Model(&DurationRow{}).Scopes(func(d *gorm.DB) *gorm.DB {
				return ApplyFilterOnly(d, filterExpr, meta, nil)
			}).Count(&count).Error; err != nil {
				t.Fatalf("query: %v", err)
			}
			if int(count) != tt.expectedCount {
				t.Errorf("%s: expected %d rows, got %d", tt.desc, tt.expectedCount, count)
			}
		})
	}
}
