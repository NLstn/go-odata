package query

import (
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
)

// ApplySearch filters results based on the $search query parameter
// This is a database-agnostic implementation that filters results in Go code
func ApplySearch(results interface{}, searchQuery string, entityMetadata *metadata.EntityMetadata) interface{} {
	if searchQuery == "" {
		return results
	}

	// Get searchable properties
	searchableProps := getSearchableProperties(entityMetadata)

	// If no properties are marked as searchable, consider all string properties
	if len(searchableProps) == 0 {
		searchableProps = getAllStringProperties(entityMetadata)
	}

	// Get the slice value
	sliceValue := reflect.ValueOf(results)
	if sliceValue.Kind() != reflect.Slice {
		return results
	}

	// Normalize search query for case-insensitive matching
	searchLower := strings.ToLower(searchQuery)

	// Filter results
	filteredSlice := reflect.MakeSlice(sliceValue.Type(), 0, sliceValue.Len())
	for i := 0; i < sliceValue.Len(); i++ {
		entity := sliceValue.Index(i)
		if matchesSearch(entity, searchLower, searchableProps) {
			filteredSlice = reflect.Append(filteredSlice, entity)
		}
	}

	return filteredSlice.Interface()
}

// getSearchableProperties returns all properties marked as searchable
func getSearchableProperties(entityMetadata *metadata.EntityMetadata) []metadata.PropertyMetadata {
	var searchable []metadata.PropertyMetadata
	for _, prop := range entityMetadata.Properties {
		if prop.IsSearchable && !prop.IsNavigationProp {
			searchable = append(searchable, prop)
		}
	}
	return searchable
}

// getAllStringProperties returns all string properties if no searchable properties are defined
func getAllStringProperties(entityMetadata *metadata.EntityMetadata) []metadata.PropertyMetadata {
	var stringProps []metadata.PropertyMetadata
	for _, prop := range entityMetadata.Properties {
		if prop.Type.Kind() == reflect.String && !prop.IsNavigationProp {
			// Set default fuzziness to 1 for string properties
			prop.SearchFuzziness = 1
			stringProps = append(stringProps, prop)
		}
	}
	return stringProps
}

// matchesSearch checks if an entity matches the search query
func matchesSearch(entity reflect.Value, searchQueryLower string, searchableProps []metadata.PropertyMetadata) bool {
	// Handle pointer types
	if entity.Kind() == reflect.Ptr {
		entity = entity.Elem()
	}

	// Check each searchable property
	for _, prop := range searchableProps {
		fieldValue := entity.FieldByName(prop.Name)
		if !fieldValue.IsValid() {
			continue
		}

		// Convert field value to string
		fieldStr := ""
		switch fieldValue.Kind() {
		case reflect.String:
			fieldStr = fieldValue.String()
		default:
			// Skip non-string fields
			continue
		}

		// Normalize for case-insensitive matching
		fieldLower := strings.ToLower(fieldStr)

		// Apply fuzzy matching based on fuzziness level
		fuzziness := prop.SearchFuzziness
		if fuzziness == 0 {
			fuzziness = 1 // Default to exact match
		}

		if fuzzyMatch(fieldLower, searchQueryLower, fuzziness) {
			return true
		}
	}

	return false
}

// fuzzyMatch performs fuzzy matching based on the fuzziness level
// fuzziness=1: exact substring match (contains)
// fuzziness>1: allows for character differences based on Levenshtein-like distance
func fuzzyMatch(text, pattern string, fuzziness int) bool {
	if fuzziness == 1 {
		// Exact substring match (case-insensitive)
		return strings.Contains(text, pattern)
	}

	// For fuzziness > 1, we implement a simple approximate matching
	// that allows for character differences
	return fuzzyContains(text, pattern, fuzziness)
}

// fuzzyContains checks if pattern is approximately contained in text
// allowing for up to (fuzziness-1) character differences
func fuzzyContains(text, pattern string, fuzziness int) bool {
	if len(pattern) == 0 {
		return true
	}
	if len(text) == 0 {
		return false
	}

	maxErrors := fuzziness - 1

	// Try matching at each position in the text
	for i := 0; i <= len(text)-len(pattern); i++ {
		errors := 0
		for j := 0; j < len(pattern) && i+j < len(text); j++ {
			if text[i+j] != pattern[j] {
				errors++
				if errors > maxErrors {
					break
				}
			}
		}
		if errors <= maxErrors {
			return true
		}
	}

	// Also check if pattern matches with insertions/deletions
	// This is a simplified version - for production, you'd want a full Levenshtein implementation
	if len(text) >= len(pattern)-maxErrors && len(text) <= len(pattern)+maxErrors {
		errors := levenshteinDistance(text, pattern)
		if errors <= maxErrors {
			return true
		}
	}

	return false
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	len1 := len(s1)
	len2 := len(s2)

	// Create a 2D array for dynamic programming
	dp := make([][]int, len1+1)
	for i := range dp {
		dp[i] = make([]int, len2+1)
	}

	// Initialize base cases
	for i := 0; i <= len1; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		dp[0][j] = j
	}

	// Fill the DP table
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = min3(
					dp[i-1][j]+1,   // deletion
					dp[i][j-1]+1,   // insertion
					dp[i-1][j-1]+1, // substitution
				)
			}
		}
	}

	return dp[len1][len2]
}

// min3 returns the minimum of three integers
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
