package query

import (
	"testing"
)

func TestQuoteTableName(t *testing.T) {
	tests := []struct {
		name      string
		dialect   string
		tableName string
		expected  string
	}{
		// Simple table names (no schema)
		{
			name:      "simple table - mssql",
			dialect:   "mssql",
			tableName: "users",
			expected:  "[users]",
		},
		{
			name:      "simple table - postgres",
			dialect:   "postgres",
			tableName: "users",
			expected:  `"users"`,
		},
		{
			name:      "simple table - mysql",
			dialect:   "mysql",
			tableName: "users",
			expected:  "`users`",
		},
		{
			name:      "simple table - sqlite",
			dialect:   "sqlite",
			tableName: "users",
			expected:  `"users"`,
		},

		// Schema-qualified table names (schema.table)
		{
			name:      "schema.table - mssql",
			dialect:   "mssql",
			tableName: "dbo.users",
			expected:  "[dbo].[users]",
		},
		{
			name:      "schema.table - postgres",
			dialect:   "postgres",
			tableName: "public.users",
			expected:  `"public"."users"`,
		},
		{
			name:      "schema.table - mysql",
			dialect:   "mysql",
			tableName: "mydb.users",
			expected:  "`mydb`.`users`",
		},
		{
			name:      "schema.table - sqlite",
			dialect:   "sqlite",
			tableName: "main.users",
			expected:  `"main"."users"`,
		},

		// Three-part names (database.schema.table)
		{
			name:      "database.schema.table - mssql",
			dialect:   "mssql",
			tableName: "mydb.dbo.users",
			expected:  "[mydb].[dbo].[users]",
		},
		{
			name:      "database.schema.table - postgres",
			dialect:   "postgres",
			tableName: "db1.public.users",
			expected:  `"db1"."public"."users"`,
		},
		{
			name:      "database.schema.table - mysql",
			dialect:   "mysql",
			tableName: "db1.schema1.users",
			expected:  "`db1`.`schema1`.`users`",
		},

		// Defensive cases - malformed input
		{
			name:      "empty parts - schema..table - mssql",
			dialect:   "mssql",
			tableName: "dbo..users",
			expected:  "[dbo].[users]", // Empty part skipped
		},
		{
			name:      "empty parts - ..table - postgres",
			dialect:   "postgres",
			tableName: "..users",
			expected:  `"users"`, // Leading empty parts skipped
		},
		{
			name:      "empty parts - schema.. - mysql",
			dialect:   "mysql",
			tableName: "dbo..",
			expected:  "`dbo`", // Trailing empty parts skipped
		},
		{
			name:      "all dots - ...",
			dialect:   "mssql",
			tableName: "...",
			expected:  "[...]", // Fallback to quoting the entire string
		},
		{
			name:      "empty string",
			dialect:   "mssql",
			tableName: "",
			expected:  "", // Returns empty string as-is
		},

		// Edge cases with special characters that need escaping
		{
			name:      "table with bracket in name - mssql",
			dialect:   "mssql",
			tableName: "users]table",
			expected:  "[users]]table]", // Escaped bracket
		},
		{
			name:      "schema.table with brackets - mssql",
			dialect:   "mssql",
			tableName: "dbo].users]",
			expected:  "[dbo]]].[users]]]", // Both parts escaped (] becomes ]])
		},
		{
			name:      "table with quote in name - postgres",
			dialect:   "postgres",
			tableName: `users"table`,
			expected:  `"users""table"`, // Escaped quote
		},
		{
			name:      "schema.table with quotes - postgres",
			dialect:   "postgres",
			tableName: `public".users"`,
			expected:  `"public"""."users"""`, // Both parts escaped (" becomes """)
		},
		{
			name:      "table with backtick in name - mysql",
			dialect:   "mysql",
			tableName: "users`table",
			expected:  "`users``table`", // Escaped backtick
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteTableName(tt.dialect, tt.tableName)
			if result != tt.expected {
				t.Errorf("quoteTableName(%q, %q) = %q, want %q",
					tt.dialect, tt.tableName, result, tt.expected)
			}
		})
	}
}

// TestQuoteTableNameConsistencyWithQuoteIdent verifies that simple table names
// produce identical output whether using quoteTableName or quoteIdent
func TestQuoteTableNameConsistencyWithQuoteIdent(t *testing.T) {
	dialects := []string{"mssql", "postgres", "mysql", "sqlite"}
	simpleNames := []string{"users", "products", "orders", "table_name"}

	for _, dialect := range dialects {
		for _, name := range simpleNames {
			quotedByTableName := quoteTableName(dialect, name)
			quotedByIdent := quoteIdent(dialect, name)

			if quotedByTableName != quotedByIdent {
				t.Errorf("Inconsistency for simple table name %q with dialect %q: quoteTableName=%q, quoteIdent=%q",
					name, dialect, quotedByTableName, quotedByIdent)
			}
		}
	}
}
