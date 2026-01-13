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
