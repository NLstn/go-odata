package etag

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
)

// Generate creates an ETag value for an entity based on its ETag property
// Returns an empty string if no ETag property is defined
func Generate(entity interface{}, meta *metadata.EntityMetadata) string {
	if meta.ETagProperty == nil {
		return ""
	}

	// Get the entity value
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Handle map[string]interface{} (from $select queries)
	if entityValue.Kind() == reflect.Map {
		entityMap, ok := entity.(map[string]interface{})
		if !ok {
			return ""
		}
		return generateFromMap(entityMap, meta.ETagProperty)
	}

	// Get the ETag field value from struct
	fieldValue := entityValue.FieldByName(meta.ETagProperty.FieldName)
	if !fieldValue.IsValid() {
		return ""
	}

	// Convert the field value to a string for hashing
	etagSource := convertToETagSource(fieldValue)

	// Generate SHA256 hash of the ETag source
	hash := sha256.Sum256([]byte(etagSource))
	hashStr := hex.EncodeToString(hash[:])

	// Return as quoted ETag (weak ETag format: W/"hash")
	return fmt.Sprintf("W/\"%s\"", hashStr)
}

// generateFromMap generates an ETag from a map[string]interface{} entity
func generateFromMap(entityMap map[string]interface{}, etagProp *metadata.PropertyMetadata) string {
	// Try to get the ETag value using both FieldName and JsonName
	var etagValue interface{}
	var found bool

	// First try JsonName (most common in maps from $select)
	if etagProp.JsonName != "" {
		etagValue, found = entityMap[etagProp.JsonName]
	}

	// If not found, try FieldName
	if !found && etagProp.FieldName != "" {
		etagValue, found = entityMap[etagProp.FieldName]
	}

	if !found {
		return ""
	}

	// Convert the value to string for hashing
	etagSource := convertInterfaceToETagSource(etagValue)

	// Generate SHA256 hash of the ETag source
	hash := sha256.Sum256([]byte(etagSource))
	hashStr := hex.EncodeToString(hash[:])

	// Return as quoted ETag (weak ETag format: W/"hash")
	return fmt.Sprintf("W/\"%s\"", hashStr)
}

// convertToETagSource converts a reflect.Value to a string for ETag generation
func convertToETagSource(fieldValue reflect.Value) string {
	switch fieldValue.Kind() {
	case reflect.String:
		return fieldValue.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(fieldValue.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(fieldValue.Uint(), 10)
	case reflect.Struct:
		// Handle time.Time specially
		if t, ok := fieldValue.Interface().(time.Time); ok {
			return strconv.FormatInt(t.Unix(), 10)
		}
		return fmt.Sprintf("%v", fieldValue.Interface())
	default:
		return fmt.Sprintf("%v", fieldValue.Interface())
	}
}

// convertInterfaceToETagSource converts an interface{} to a string for ETag generation
func convertInterfaceToETagSource(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case time.Time:
		return strconv.FormatInt(v.Unix(), 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Parse extracts the ETag value from a quoted ETag string
// Handles both strong ("value") and weak (W/"value") ETags
func Parse(etagHeader string) string {
	if etagHeader == "" {
		return ""
	}

	// Remove W/ prefix if present (weak ETag)
	if len(etagHeader) > 2 && etagHeader[:2] == "W/" {
		etagHeader = etagHeader[2:]
	}

	// Remove quotes
	if len(etagHeader) >= 2 && etagHeader[0] == '"' && etagHeader[len(etagHeader)-1] == '"' {
		return etagHeader[1 : len(etagHeader)-1]
	}

	return etagHeader
}

// Match checks if the provided If-Match header value matches the current ETag
// Returns true if they match or if ifMatch is "*" (match any)
func Match(ifMatch string, currentETag string) bool {
	if ifMatch == "" {
		return true // No If-Match header means no precondition
	}

	// "*" matches any ETag (entity must exist)
	if ifMatch == "*" {
		return currentETag != ""
	}

	// Parse both ETags and compare
	parsedIfMatch := Parse(ifMatch)
	parsedCurrent := Parse(currentETag)

	return parsedIfMatch == parsedCurrent
}

// NoneMatch checks if the provided If-None-Match header value does NOT match the current ETag
// Returns true if they don't match or if ifNoneMatch is empty (no condition)
// Returns false if they match (meaning resource hasn't changed - should return 304)
// The "*" wildcard means "match if entity exists" for If-None-Match
func NoneMatch(ifNoneMatch string, currentETag string) bool {
	if ifNoneMatch == "" {
		return true // No If-None-Match header means no condition, proceed normally
	}

	// "*" matches any existing entity, so none-match is false if entity exists
	if ifNoneMatch == "*" {
		return currentETag == ""
	}

	// Parse both ETags and compare
	parsedIfNoneMatch := Parse(ifNoneMatch)
	parsedCurrent := Parse(currentETag)

	// Return true if they DON'T match (proceed with normal response)
	// Return false if they DO match (should return 304)
	return parsedIfNoneMatch != parsedCurrent
}
