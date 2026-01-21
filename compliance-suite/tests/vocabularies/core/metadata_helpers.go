package core

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

func metadataNamespace(metadataXML []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(metadataXML))
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to parse metadata XML: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Schema" {
			continue
		}
		for _, attr := range start.Attr {
			if attr.Name.Local == "Namespace" {
				return attr.Value, nil
			}
		}
	}
	return "", fmt.Errorf("metadata namespace not found")
}

func hasAnnotation(metadataXML []byte, target, term string) (bool, error) {
	decoder := xml.NewDecoder(bytes.NewReader(metadataXML))
	inTarget := false
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return false, fmt.Errorf("failed to parse metadata XML: %w", err)
		}
		switch node := token.(type) {
		case xml.StartElement:
			switch node.Name.Local {
			case "Annotations":
				inTarget = false
				for _, attr := range node.Attr {
					if attr.Name.Local == "Target" && attr.Value == target {
						inTarget = true
						break
					}
				}
			case "Annotation":
				if !inTarget {
					continue
				}
				for _, attr := range node.Attr {
					if attr.Name.Local == "Term" && attr.Value == term {
						return true, nil
					}
				}
			}
		case xml.EndElement:
			if node.Name.Local == "Annotations" {
				inTarget = false
			}
		}
	}
	return false, nil
}

func assertODataError(resp *framework.HTTPResponse) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return fmt.Errorf("expected JSON error response, got parse error: %w", err)
	}

	errObjRaw, ok := payload["error"]
	if !ok {
		return fmt.Errorf("missing error object in response")
	}

	errObj, ok := errObjRaw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("error object has unexpected type")
	}

	code, ok := errObj["code"].(string)
	if !ok || code != fmt.Sprintf("%d", resp.StatusCode) {
		return fmt.Errorf("error code mismatch: got %v, expected %d", errObj["code"], resp.StatusCode)
	}

	message, ok := errObj["message"].(string)
	if !ok || strings.TrimSpace(message) == "" {
		return fmt.Errorf("error message is missing or empty")
	}

	return nil
}
