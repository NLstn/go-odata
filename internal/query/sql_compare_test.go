package query

import "strings"

func normalizeSQLForComparison(sql string) string {
	sql = strings.ReplaceAll(sql, "\"", "")
	sql = strings.ReplaceAll(sql, "`", "")
	return strings.Join(strings.Fields(sql), " ")
}

func sqlEquivalent(expected, actual string) bool {
	return normalizeSQLForComparison(expected) == normalizeSQLForComparison(actual)
}
