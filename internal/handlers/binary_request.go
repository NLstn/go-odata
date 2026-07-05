package handlers

import (
	"encoding/base64"
	"fmt"
	"reflect"
)

// binaryBase64Decoders lists the base64 alphabets accepted when decoding an
// Edm.Binary property value supplied by a client. Per the OData JSON Format
// spec (v4.0 §7.1; RFC 4648 §5), binary values must be encoded using the
// URL and filename safe alphabet without padding (base64.RawURLEncoding).
// The remaining alphabets are accepted leniently for interoperability with
// clients that still send standard base64 (with or without padding).
var binaryBase64Decoders = [...]*base64.Encoding{
	base64.RawURLEncoding,
	base64.URLEncoding,
	base64.RawStdEncoding,
	base64.StdEncoding,
}

// decodeBase64Lenient decodes a base64-encoded string, accepting both the
// URL-safe alphabet required by the OData JSON Format spec and the standard
// alphabet (with or without padding).
func decodeBase64Lenient(s string) ([]byte, error) {
	if s == "" {
		return []byte{}, nil
	}
	var lastErr error
	for _, enc := range binaryBase64Decoders {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		} else {
			lastErr = err
		}
	}
	return nil, fmt.Errorf("illegal base64 data: %w", lastErr)
}

// isBinaryFieldType reports whether t (after unwrapping a pointer) is the Go
// representation of an Edm.Binary property, i.e. []byte.
func isBinaryFieldType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8
}

// decodeBinaryPropertiesInPlace scans data for Edm.Binary properties (Go
// []byte fields) and, where the JSON-decoded value is a string, replaces it
// in-place with the decoded []byte value.
//
// Go's encoding/json only unmarshals []byte fields using the standard base64
// alphabet with padding, so without this step base64url-encoded values sent
// per the OData JSON Format spec (§7.1) would be rejected. Normalizing to raw
// []byte here lets every downstream consumer (re-marshal into an entity
// struct for POST/PUT, or GORM's Updates(map) for PATCH) work the same way
// regardless of which base64 alphabet the client used.
func (h *EntityHandler) decodeBinaryPropertiesInPlace(data map[string]interface{}) error {
	for _, prop := range h.metadata.Properties {
		if prop.IsNavigationProp || prop.IsStream || !isBinaryFieldType(prop.Type) {
			continue
		}

		for _, key := range [...]string{prop.JsonName, prop.Name} {
			raw, exists := data[key]
			if !exists {
				continue
			}
			s, ok := raw.(string)
			if !ok {
				// Not a string (nil, or already []byte) - leave as-is for
				// existing validation/type-checking to handle.
				continue
			}
			decoded, err := decodeBase64Lenient(s)
			if err != nil {
				return fmt.Errorf("property '%s': invalid base64-encoded binary value: %w", prop.JsonName, err)
			}
			data[key] = decoded
		}
	}
	return nil
}
