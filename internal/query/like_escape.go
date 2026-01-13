package query

import (
	"fmt"
	"strings"
)

const likeEscapeClause = "ESCAPE '\\'"

func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(value)
}

func buildLikeComparison(columnName string, value interface{}, prefixWildcard bool, suffixWildcard bool) (string, []interface{}) {
	pattern := escapeLikePattern(fmt.Sprint(value))
	if prefixWildcard {
		pattern = "%" + pattern
	}
	if suffixWildcard {
		pattern = pattern + "%"
	}

	return fmt.Sprintf("%s LIKE ? %s", columnName, likeEscapeClause), []interface{}{pattern}
}
