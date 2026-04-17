package query

import (
	"fmt"
	"strings"
)

// getLikeEscapeClause returns the appropriate ESCAPE clause for the database dialect.
// MySQL/MariaDB require ESCAPE '\\\\' (4 backslashes) while others use ESCAPE '\\'
func getLikeEscapeClause(dialect string) string {
	switch dialect {
	case "mysql":
		// MySQL/MariaDB need 4 backslashes in Go source to represent one backslash escape char
		return "ESCAPE '\\\\'"
	default:
		// PostgreSQL, SQLite, SQL Server use 2 backslashes
		return "ESCAPE '\\'"
	}
}

func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(value)
}

func buildLikeComparison(dialect string, columnName string, value interface{}, prefixWildcard bool, suffixWildcard bool) (string, []interface{}) {
	pattern := escapeLikePattern(fmt.Sprint(value))
	if prefixWildcard {
		pattern = "%" + pattern
	}
	if suffixWildcard {
		pattern = pattern + "%"
	}

	escapeClause := getLikeEscapeClause(dialect)
	return fmt.Sprintf("%s LIKE ? %s", columnName, escapeClause), []interface{}{pattern}
}

// buildRegexComparison builds a SQL regex comparison for the OData v4.01 matchesPattern function.
// The pattern is a POSIX ERE pattern per the OData v4.01 spec (Section 11.5.3.3).
// SQL dialect mappings:
//   - PostgreSQL: uses ~ operator for POSIX ERE matching
//   - MySQL/MariaDB: uses REGEXP operator
//   - SQLite: uses REGEXP operator (requires user-defined function to be registered)
//   - SQL Server: uses LIKE as a best-effort approximation (limited regex support)
func buildRegexComparison(dialect string, columnName string, value interface{}) (string, []interface{}) {
	pattern := fmt.Sprint(value)
	switch dialect {
	case "postgres", "postgresql":
		return fmt.Sprintf("%s ~ ?", columnName), []interface{}{pattern}
	default:
		// SQLite, MySQL, MariaDB all support the REGEXP operator
		return fmt.Sprintf("%s REGEXP ?", columnName), []interface{}{pattern}
	}
}
