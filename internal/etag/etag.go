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

	// Get the ETag field value
	fieldValue := entityValue.FieldByName(meta.ETagProperty.FieldName)
	if !fieldValue.IsValid() {
		return ""
	}

	// Convert the field value to a string for hashing
	var etagSource string
	switch fieldValue.Kind() {
	case reflect.String:
		etagSource = fieldValue.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		etagSource = strconv.FormatInt(fieldValue.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		etagSource = strconv.FormatUint(fieldValue.Uint(), 10)
	case reflect.Struct:
		// Handle time.Time specially
		if t, ok := fieldValue.Interface().(time.Time); ok {
			etagSource = strconv.FormatInt(t.Unix(), 10)
		} else {
			etagSource = fmt.Sprintf("%v", fieldValue.Interface())
		}
	default:
		etagSource = fmt.Sprintf("%v", fieldValue.Interface())
	}

	// Generate SHA256 hash of the ETag source
	hash := sha256.Sum256([]byte(etagSource))
	hashStr := hex.EncodeToString(hash[:])

	// Return as quoted ETag (weak ETag format: W/"hash")
	return fmt.Sprintf("W/\"%s\"", hashStr)
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
