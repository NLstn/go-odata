package query

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// ApplySearch filters results based on the $search query parameter.
// This is a database-agnostic implementation that filters results in Go code.
// The query string is parsed into a boolean expression tree that supports:
//   - Single terms:   laptop
//   - Quoted phrases: "high performance"
//   - Boolean AND:    laptop AND wireless  (or implicit: laptop wireless)
//   - Boolean OR:     laptop OR phone
//   - Boolean NOT:    NOT phone
//   - Grouping:       (laptop OR phone) AND wireless
func ApplySearch(results interface{}, searchQuery string, entityMetadata *metadata.EntityMetadata) interface{} {
	if searchQuery == "" {
		return results
	}

	searchableProps := SearchableProperties(entityMetadata)

	// Get the slice value
	sliceValue := reflect.ValueOf(results)
	if sliceValue.Kind() != reflect.Slice {
		return results
	}

	// Parse the search query into an expression tree for boolean operator support
	expr := ParseSearchExpression(searchQuery)
	if expr == nil {
		return results
	}

	// Filter results
	filteredSlice := reflect.MakeSlice(sliceValue.Type(), 0, sliceValue.Len())
	for i := 0; i < sliceValue.Len(); i++ {
		entity := sliceValue.Index(i)
		if matchesSearchExpr(entity, expr, searchableProps) {
			filteredSlice = reflect.Append(filteredSlice, entity)
		}
	}

	return filteredSlice.Interface()
}

// SearchableProperties returns the properties used for $search.
func SearchableProperties(entityMetadata *metadata.EntityMetadata) []metadata.PropertyMetadata {
	if entityMetadata == nil {
		return nil
	}

	searchableProps := getSearchableProperties(entityMetadata)
	if len(searchableProps) == 0 {
		searchableProps = getAllStringProperties(entityMetadata)
	}

	return searchableProps
}

// getSearchableProperties returns all properties marked as searchable.
func getSearchableProperties(entityMetadata *metadata.EntityMetadata) []metadata.PropertyMetadata {
	var searchable []metadata.PropertyMetadata
	for _, prop := range entityMetadata.Properties {
		if prop.IsSearchable && !prop.IsNavigationProp {
			searchable = append(searchable, prop)
		}
	}
	return searchable
}

// getAllStringProperties returns all string properties when no searchable properties are explicitly defined.
func getAllStringProperties(entityMetadata *metadata.EntityMetadata) []metadata.PropertyMetadata {
	var stringProps []metadata.PropertyMetadata
	for _, prop := range entityMetadata.Properties {
		if prop.Type.Kind() == reflect.String && !prop.IsNavigationProp {
			// Set default fuzziness to 1 (exact substring match) for implicitly searchable properties
			prop.SearchFuzziness = 1
			stringProps = append(stringProps, prop)
		}
	}
	return stringProps
}

// matchesSearchExpr evaluates a search expression tree against a single entity.
func matchesSearchExpr(entity reflect.Value, node *SearchExprNode, props []metadata.PropertyMetadata) bool {
	if node == nil {
		return false
	}
	switch node.op {
	case searchOpAnd:
		return matchesSearchExpr(entity, node.left, props) && matchesSearchExpr(entity, node.right, props)
	case searchOpOr:
		return matchesSearchExpr(entity, node.left, props) || matchesSearchExpr(entity, node.right, props)
	case searchOpNot:
		return !matchesSearchExpr(entity, node.left, props)
	case searchOpTerm, searchOpPhrase:
		return matchesTermInEntity(entity, strings.ToLower(node.term), props)
	}
	return false
}

// matchesTermInEntity checks if a term or phrase appears in any searchable property of an entity.
func matchesTermInEntity(entity reflect.Value, termLower string, searchableProps []metadata.PropertyMetadata) bool {
	// Handle pointer types
	if entity.Kind() == reflect.Ptr {
		entity = entity.Elem()
	}

	for _, prop := range searchableProps {
		fieldValue := entity.FieldByName(prop.Name)
		if !fieldValue.IsValid() {
			continue
		}

		// Convert field value to a searchable string.
		// Non-string fields (int, float, bool, etc.) are formatted with %v so that
		// a field tagged `odata:"searchable"` actually participates in search regardless
		// of its Go type.
		var fieldStr string
		switch fieldValue.Kind() {
		case reflect.String:
			fieldStr = fieldValue.String()
		case reflect.Invalid:
			continue
		default:
			// Convert non-string primitive values to their default string representation
			fieldStr = fmt.Sprintf("%v", fieldValue.Interface())
		}

		fieldLower := strings.ToLower(fieldStr)

		// Use similarity score if defined, otherwise use fuzziness
		if prop.SearchSimilarity > 0 {
			if similarityMatch(fieldLower, termLower, prop.SearchSimilarity) {
				return true
			}
		} else {
			fuzziness := prop.SearchFuzziness
			if fuzziness == 0 {
				fuzziness = 1
			}
			if fuzzyMatch(fieldLower, termLower, fuzziness) {
				return true
			}
		}
	}

	return false
}

// fuzzyMatch performs matching based on the fuzziness level.
//
// Fuzziness controls how strictly the term must appear in the text:
//   - fuzziness=1 (default): exact substring match — the term must appear verbatim inside the field value.
//   - fuzziness=2: allows up to 1 character difference (insertion, deletion, or substitution) when
//     comparing the pattern against a sliding window of the same length in the text.
//   - fuzziness=N: allows up to N-1 character differences.
//
// Higher fuzziness numbers are more permissive. Use fuzziness=1 for exact matching.
func fuzzyMatch(text, pattern string, fuzziness int) bool {
	if fuzziness == 1 {
		// Exact substring match (case-insensitive — caller lowercases both sides)
		return strings.Contains(text, pattern)
	}

	// For fuzziness > 1, use approximate sliding-window matching
	return fuzzyContains(text, pattern, fuzziness)
}

// fuzzyContains checks if pattern is approximately contained in text,
// allowing up to (fuzziness-1) character differences.
//
// The comparison is rune-aware so multi-byte UTF-8 characters (e.g. accented
// letters, CJK) are counted as single characters, not as multiple bytes.
func fuzzyContains(text, pattern string, fuzziness int) bool {
	if len(pattern) == 0 {
		return true
	}
	if len(text) == 0 {
		return false
	}

	textRunes := []rune(text)
	patternRunes := []rune(pattern)
	maxErrors := fuzziness - 1

	// Sliding-window: compare pattern against every window of the same length in text
	for i := 0; i <= len(textRunes)-len(patternRunes); i++ {
		errors := 0
		matched := true
		for j := 0; j < len(patternRunes); j++ {
			if textRunes[i+j] != patternRunes[j] {
				errors++
				if errors > maxErrors {
					matched = false
					break
				}
			}
		}
		if matched {
			return true
		}
	}

	// Also check whole-string Levenshtein distance (handles insertions/deletions
	// that would shift the window beyond the above loop's reach)
	if len(textRunes) >= len(patternRunes)-maxErrors && len(textRunes) <= len(patternRunes)+maxErrors {
		errors := levenshteinDistance(text, pattern)
		if errors <= maxErrors {
			return true
		}
	}

	return false
}

// levenshteinDistance calculates the Levenshtein distance between two strings.
// Optimized to use only two rows instead of a full 2D matrix, reducing memory
// allocations from O(m*n) to O(n) where n is the length of the shorter string.
func levenshteinDistance(s1, s2 string) int {
	len1 := len(s1)
	len2 := len(s2)

	// Ensure s1 is the shorter string to minimize memory usage
	if len1 > len2 {
		s1, s2 = s2, s1
		len1, len2 = len2, len1
	}

	// Only need two rows: previous and current
	// This reduces memory from O(m*n) to O(min(m,n))
	prev := make([]int, len1+1)
	curr := make([]int, len1+1)

	// Initialize the first row (base case: distance from empty string)
	for i := 0; i <= len1; i++ {
		prev[i] = i
	}

	// Fill the DP table row by row
	for j := 1; j <= len2; j++ {
		curr[0] = j // Base case: distance to empty string
		for i := 1; i <= len1; i++ {
			if s1[i-1] == s2[j-1] {
				curr[i] = prev[i-1]
			} else {
				curr[i] = min3(
					prev[i]+1,   // deletion
					curr[i-1]+1, // insertion
					prev[i-1]+1, // substitution
				)
			}
		}
		// Swap rows: current becomes previous for next iteration
		prev, curr = curr, prev
	}

	return prev[len1]
}

// min3 returns the minimum of three integers.
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// similarityMatch performs similarity-based matching using a normalized Levenshtein similarity.
// similarity: a value between 0.0 and 1.0, where 1.0 means exact match.
// For example, similarity=0.95 means the field must be at least 95% similar to the search term.
func similarityMatch(text, pattern string, minSimilarity float64) bool {
	if len(pattern) == 0 {
		return true
	}
	if len(text) == 0 {
		return false
	}

	// Check if pattern is contained in text (exact substring match)
	if strings.Contains(text, pattern) {
		return true
	}

	// For similarity matching, we compare the entire field to the pattern.
	// Calculate the Levenshtein distance.
	distance := levenshteinDistance(text, pattern)

	// Normalize the distance based on the longer string length
	maxLen := len(text)
	if len(pattern) > maxLen {
		maxLen = len(pattern)
	}

	// Calculate similarity score (1.0 = exact match, 0.0 = completely different)
	similarity := 1.0 - (float64(distance) / float64(maxLen))

	return similarity >= minSimilarity
}
