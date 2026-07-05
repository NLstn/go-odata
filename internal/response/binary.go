package response

import "encoding/base64"

// EncodeEdmBinary converts an Edm.Binary property value ([]byte) into the string
// representation required by the OData JSON Format spec for binary values
// (OData JSON Format v4.0 §7.1; RFC 4648 §5 - the URL and filename safe alphabet,
// without padding).
//
// Go's encoding/json marshals []byte using the standard base64 alphabet
// (base64.StdEncoding) with padding, which is not compliant with the OData JSON
// Format spec. Call this on every property value immediately before it is placed
// into a JSON response so that []byte values are pre-encoded as compliant strings
// instead of being left for encoding/json to encode incorrectly.
//
// Non-[]byte values (and nil) are returned unchanged, so this can be called
// unconditionally on any property value.
func EncodeEdmBinary(value interface{}) interface{} {
	b, ok := value.([]byte)
	if !ok {
		return value
	}
	if b == nil {
		return nil
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
