package handlers

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/version"
)

// validateJSONRequestAnnotations rejects compact OData 4.01 control annotations
// when the request declares OData 4.0. OData 4.0 requires the @odata.* form.
func validateJSONRequestAnnotations(value interface{}, requestVersion version.Version) error {
	if requestVersion != (version.Version{Major: 4, Minor: 0}) {
		return nil
	}
	return validateJSONRequestAnnotationsValue(value)
}

func validateJSONRequestAnnotationsValue(value interface{}) error {
	switch value := value.(type) {
	case map[string]interface{}:
		for key, child := range value {
			if isCompactODataControlAnnotation(key) {
				return fmt.Errorf("compact OData 4.01 control annotation %q is not valid for OData 4.0; use the @odata.* form", key)
			}
			if err := validateJSONRequestAnnotationsValue(child); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, child := range value {
			if err := validateJSONRequestAnnotationsValue(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func isCompactODataControlAnnotation(key string) bool {
	if !strings.HasPrefix(key, "@") || strings.HasPrefix(key, "@odata.") {
		return false
	}
	switch key {
	case "@context", "@metadataEtag", "@type", "@id", "@etag", "@editLink",
		"@readLink", "@navigationLink", "@associationLink", "@mediaReadLink",
		"@mediaEditLink", "@mediaEtag", "@mediaContentType", "@removed",
		"@count", "@nextLink", "@delta", "@deltaLink", "@bind":
		return true
	default:
		return false
	}
}
