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
	inTargetAnnotations := false
	inTargetElement := false
	targetDepth := 0
	currentDepth := 0
	
	// Parse target to extract entity type and property if present
	// Format: "Namespace.EntityType/PropertyName" or "Namespace.EntityType"
	var targetEntityType, targetProperty string
	if strings.Contains(target, "/") {
		parts := strings.Split(target, "/")
		targetEntityType = parts[0]
		if len(parts) > 1 {
			targetProperty = parts[1]
		}
	} else {
		targetEntityType = target
	}
	
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
			currentDepth++
			switch node.Name.Local {
			case "Annotations":
				// Check for external Annotations block with Target attribute
				inTargetAnnotations = false
				for _, attr := range node.Attr {
					if attr.Name.Local == "Target" && attr.Value == target {
						inTargetAnnotations = true
						break
					}
				}
			case "EntityType", "ComplexType":
				// Check if this is the target entity/complex type
				for _, attr := range node.Attr {
					if attr.Name.Local == "Name" {
						// Construct full name with namespace if target contains namespace
						fullName := attr.Value
						if strings.Contains(targetEntityType, ".") {
							// Target includes namespace, need to match exactly
							// We'll check this when we know the namespace
						}
						if strings.HasSuffix(targetEntityType, "."+attr.Value) || targetEntityType == attr.Value {
							inTargetElement = true
							targetDepth = currentDepth
						}
					}
				}
			case "Property":
				// Check for inline annotations within Property element
				if inTargetElement && targetProperty != "" {
					for _, attr := range node.Attr {
						if attr.Name.Local == "Name" && attr.Value == targetProperty {
							// We're in the target property, check for inline annotations
							// Continue processing to find Annotation child elements
						}
					}
				}
			case "Annotation":
				// Check if this annotation matches our term
				for _, attr := range node.Attr {
					if attr.Name.Local == "Term" && attr.Value == term {
						// Found the annotation in either:
						// 1. External Annotations block (inTargetAnnotations)
						// 2. Inline within the target element (inTargetElement)
						if inTargetAnnotations || (inTargetElement && currentDepth > targetDepth) {
							return true, nil
						}
					}
				}
			}
		case xml.EndElement:
			if node.Name.Local == "Annotations" {
				inTargetAnnotations = false
			}
			if node.Name.Local == "EntityType" || node.Name.Local == "ComplexType" {
				if inTargetElement && currentDepth == targetDepth {
					inTargetElement = false
				}
			}
			currentDepth--
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
