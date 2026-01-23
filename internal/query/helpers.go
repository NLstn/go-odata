package query

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nlstn/go-odata/internal/metadata"
)

// defaultCacheCapacity is the initial capacity for the parser cache map.
// This value was chosen based on profiling data showing that pre-allocating
// the map reduces allocations by ~4% (propertyExistsWithCache was 210 MB).
// Most OData queries reference fewer than 16 properties.
const defaultCacheCapacity = 16

// navPathCacheEntry stores cached navigation path resolution results
type navPathCacheEntry struct {
	targetMetadata     interface{} // *metadata.EntityMetadata stored as interface to avoid import cycle
	navigationSegments []string
	remainingPath      string
	err                error
}

// parserCache provides per-request caching for expensive operations
type parserCache struct {
	resolvedPaths map[string]bool
	navPathCache  map[string]*navPathCacheEntry // Cache for navigation path resolution
	mu            sync.RWMutex
}

// parserCachePool is a sync.Pool for reusing parserCache instances.
// This reduces allocations by ~15-18% by avoiding map creation per parse operation.
var parserCachePool = sync.Pool{
	New: func() interface{} {
		return &parserCache{
			resolvedPaths: make(map[string]bool, defaultCacheCapacity),
			navPathCache:  make(map[string]*navPathCacheEntry, defaultCacheCapacity),
		}
	},
}

// acquireParserCache gets a parserCache from the pool
func acquireParserCache() *parserCache {
	if v := parserCachePool.Get(); v != nil {
		if c, ok := v.(*parserCache); ok {
			return c
		}
	}
	return &parserCache{
		resolvedPaths: make(map[string]bool, defaultCacheCapacity),
		navPathCache:  make(map[string]*navPathCacheEntry, defaultCacheCapacity),
	}
}

// releaseParserCache returns a parserCache to the pool after clearing it
func releaseParserCache(c *parserCache) {
	if c == nil {
		return
	}
	// Clear the maps for reuse
	for k := range c.resolvedPaths {
		delete(c.resolvedPaths, k)
	}
	for k := range c.navPathCache {
		delete(c.navPathCache, k)
	}
	parserCachePool.Put(c)
}

// newParserCache creates a new parser cache (now uses pool internally)
func newParserCache() *parserCache {
	return acquireParserCache()
}

// propertyExistsWithCache checks if a property exists using cache
func (c *parserCache) propertyExistsWithCache(propertyName string, entityMetadata *metadata.EntityMetadata) bool {
	if c == nil {
		return propertyExists(propertyName, entityMetadata)
	}

	// Try to get from cache first
	c.mu.RLock()
	if exists, cached := c.resolvedPaths[propertyName]; cached {
		c.mu.RUnlock()
		return exists
	}
	c.mu.RUnlock()

	// Not in cache, compute it
	exists := propertyExistsWithNavCache(propertyName, entityMetadata, c)

	// Store in cache
	c.mu.Lock()
	c.resolvedPaths[propertyName] = exists
	c.mu.Unlock()

	return exists
}

// resolveSingleEntityNavPathWithCache resolves a navigation path using cache
func (c *parserCache) resolveSingleEntityNavPathWithCache(path string, entityMetadata *metadata.EntityMetadata) (*metadata.EntityMetadata, []string, string, error) {
	if c == nil || entityMetadata == nil {
		return entityMetadata.ResolveSingleEntityNavigationPath(path)
	}

	// Create cache key combining entity name and path
	cacheKey := entityMetadata.EntityName + ":" + path

	// Try to get from cache first
	c.mu.RLock()
	if entry, cached := c.navPathCache[cacheKey]; cached {
		c.mu.RUnlock()
		if entry.err != nil {
			return nil, nil, "", entry.err
		}
		targetMeta, ok := entry.targetMetadata.(*metadata.EntityMetadata)
		if !ok {
			return nil, nil, "", nil
		}
		return targetMeta, entry.navigationSegments, entry.remainingPath, nil
	}
	c.mu.RUnlock()

	// Not in cache, compute it
	targetMeta, navSegments, remaining, err := entityMetadata.ResolveSingleEntityNavigationPath(path)

	// Store in cache
	c.mu.Lock()
	c.navPathCache[cacheKey] = &navPathCacheEntry{
		targetMetadata:     targetMeta,
		navigationSegments: navSegments,
		remainingPath:      remaining,
		err:                err,
	}
	c.mu.Unlock()

	return targetMeta, navSegments, remaining, err
}

// propertyExistsWithNavCache checks if a property exists, using navigation path cache when available
func propertyExistsWithNavCache(propertyName string, entityMetadata *metadata.EntityMetadata, cache *parserCache) bool {
	if entityMetadata == nil {
		return false
	}

	// Check if this is a single-entity navigation property path using cache
	if cache != nil {
		targetMeta, _, remainingPath, err := cache.resolveSingleEntityNavPathWithCache(propertyName, entityMetadata)
		if err == nil && targetMeta != nil && remainingPath != "" {
			prop, _, err := targetMeta.ResolvePropertyPath(remainingPath)
			if err == nil && prop != nil && !prop.IsNavigationProp {
				return true
			}
		}
	} else if entityMetadata.IsSingleEntityNavigationPath(propertyName) {
		return true
	}

	_, _, err := entityMetadata.ResolvePropertyPath(propertyName)
	return err == nil
}

// parseSelect parses the $select query option
func parseSelect(selectStr string) []string {
	parts := strings.Split(selectStr, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// propertyExists checks if a property exists in the entity metadata
func propertyExists(propertyName string, entityMetadata *metadata.EntityMetadata) bool {
	if entityMetadata == nil {
		return false
	}

	// Check if this is a single-entity navigation property path
	// Per OData v4 spec 5.1.1.15, single-entity navigation properties can be accessed directly
	if entityMetadata.IsSingleEntityNavigationPath(propertyName) {
		return true
	}

	_, _, err := entityMetadata.ResolvePropertyPath(propertyName)
	return err == nil
}

func resolveNavigationPropertyPath(propertyName string, entityMetadata *metadata.EntityMetadata) (*metadata.EntityMetadata, []string, *metadata.PropertyMetadata, string, error) {
	if entityMetadata == nil {
		return nil, nil, nil, "", errEntityMetadataIsNil
	}

	targetMetadata, navSegments, remainingPath, err := entityMetadata.ResolveSingleEntityNavigationPath(propertyName)
	if err != nil {
		return nil, nil, nil, "", err
	}

	if targetMetadata == nil || remainingPath == "" {
		return nil, nil, nil, "", errNavPathNoRemainingProperty
	}

	prop, prefix, err := targetMetadata.ResolvePropertyPath(remainingPath)
	if err != nil || prop == nil {
		return nil, nil, nil, "", fmt.Errorf("property '%s' not found", remainingPath)
	}

	if prop.IsNavigationProp {
		return nil, nil, nil, "", errNavPathEndsWithNavProp
	}

	return targetMetadata, navSegments, prop, prefix, nil
}

func navigationAliasForPath(segments []string) string {
	if len(segments) == 0 {
		return ""
	}

	aliasSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" {
			continue
		}
		aliasSegments = append(aliasSegments, toSnakeCase(trimmed))
	}

	if len(aliasSegments) == 0 {
		return ""
	}

	return "nav_" + strings.Join(aliasSegments, "_")
}

// isNavigationProperty checks if a property is a navigation property
func isNavigationProperty(propName string, entityMetadata *metadata.EntityMetadata) bool {
	for _, prop := range entityMetadata.Properties {
		if (prop.JsonName == propName || prop.Name == propName) && prop.IsNavigationProp {
			return true
		}
	}
	return false
}

// GetPropertyFieldName returns the struct field name for a given JSON property name
// This returns the actual Go struct field name, not the JSON name
func GetPropertyFieldName(propertyName string, entityMetadata *metadata.EntityMetadata) string {
	for _, prop := range entityMetadata.Properties {
		if prop.JsonName == propertyName || prop.Name == propertyName {
			return prop.Name // Return the struct field name
		}
	}
	return propertyName
}

// GetColumnName returns the database column name (snake_case) for a property
func GetColumnName(propertyName string, entityMetadata *metadata.EntityMetadata) string {
	// Handle $it - refers to the current instance (OData v4 spec 5.1.1.11.4)
	// Used in isof() function to check the type of the current entity
	if propertyName == "$it" {
		return "$it"
	}

	if entityMetadata == nil {
		return toSnakeCase(propertyName)
	}

	if entityMetadata != nil {
		if _, navSegments, prop, prefix, err := resolveNavigationPropertyPath(propertyName, entityMetadata); err == nil {
			columnName := prefix + prop.ColumnName
			alias := navigationAliasForPath(navSegments)
			if alias != "" {
				return alias + "." + columnName
			}
		}
	}

	prop, prefix, err := entityMetadata.ResolvePropertyPath(propertyName)
	if err != nil || prop == nil {
		// Fallback to the last segment when metadata cannot resolve the path
		if strings.Contains(propertyName, "/") {
			parts := strings.Split(propertyName, "/")
			propertyName = parts[len(parts)-1]
		}
		return toSnakeCase(propertyName)
	}

	// Use cached column name from metadata
	return prefix + prop.ColumnName
}

// findNavigationProperty finds a navigation property in the entity metadata
func findNavigationProperty(propName string, entityMetadata *metadata.EntityMetadata) *metadata.PropertyMetadata {
	if entityMetadata == nil {
		return nil
	}
	return entityMetadata.FindNavigationProperty(propName)
}

// toSnakeCase converts a camelCase or PascalCase string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if the previous character was lowercase or if this is the start of a new word
			// For "ProductID", we want "product_id" not "product_i_d"
			prevRune := rune(s[i-1])
			if prevRune >= 'a' && prevRune <= 'z' {
				result.WriteRune('_')
			} else if i < len(s)-1 {
				// Check if next character is lowercase (e.g., "XMLParser" -> "xml_parser")
				nextRune := rune(s[i+1])
				if nextRune >= 'a' && nextRune <= 'z' {
					result.WriteRune('_')
				}
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// MergeFilterExpressions combines two filter expressions using a logical AND.
func MergeFilterExpressions(left *FilterExpression, right *FilterExpression) *FilterExpression {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	expr := acquireFilterExpression()
	expr.Left = left
	expr.Right = right
	expr.Logical = LogicalAnd
	return expr
}

// ParseFilterExpression parses a raw filter string into a filter expression with metadata validation.
// This helper enforces the DefaultMaxInClauseSize limit for security.
// For custom limits, use ParseFilterExpressionWithConfig instead.
func ParseFilterExpression(filterStr string, entityMetadata *metadata.EntityMetadata) (*FilterExpression, error) {
	// Use default limit for security - prevent DoS via large IN clauses
	return parseFilter(filterStr, entityMetadata, map[string]bool{}, 1000) // DefaultMaxInClauseSize
}

// ParseFilterExpressionWithConfig parses a raw filter string with custom configuration.
func ParseFilterExpressionWithConfig(filterStr string, entityMetadata *metadata.EntityMetadata, maxInClauseSize int) (*FilterExpression, error) {
	return parseFilter(filterStr, entityMetadata, map[string]bool{}, maxInClauseSize)
}
