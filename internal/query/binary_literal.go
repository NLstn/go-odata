package query

import (
	"encoding/base64"
	"fmt"
)

// binaryLiteralDecoders lists the base64 alphabets accepted when decoding a
// $filter binary literal (binary'...'). Per the OData ABNF (binaryValue) and
// the OData JSON Format spec (v4.0 §7.1; RFC 4648 §5), binary values must be
// encoded using the URL and filename safe alphabet without padding
// (base64.RawURLEncoding). The remaining alphabets are accepted leniently for
// interoperability with clients that still send standard base64 (with or
// without padding).
var binaryLiteralDecoders = [...]*base64.Encoding{
	base64.RawURLEncoding,
	base64.URLEncoding,
	base64.RawStdEncoding,
	base64.StdEncoding,
}

// decodeBase64Lenient decodes a base64-encoded string, accepting both the
// URL-safe alphabet required by the OData spec and the standard alphabet
// (with or without padding).
func decodeBase64Lenient(s string) ([]byte, error) {
	if s == "" {
		return []byte{}, nil
	}
	var lastErr error
	for _, enc := range binaryLiteralDecoders {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		} else {
			lastErr = err
		}
	}
	return nil, fmt.Errorf("illegal base64 data: %w", lastErr)
}
