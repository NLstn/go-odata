package metadata

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PropertyDefinition describes a primitive structural property without a Go struct field.
type PropertyDefinition struct {
	Name     string
	Type     PrimitiveType
	Nullable *bool
}

// EntityDefinition describes a map-backed entity type registered at runtime.
type EntityDefinition struct {
	Name            string
	EntitySetName   string
	Properties      []PropertyDefinition
	Keys            []string
	DisabledMethods []string
}

// Validate checks whether the definition can represent a top-level OData entity set.
func (definition EntityDefinition) Validate() error {
	if err := validateDefinitionIdentifier("entity name", definition.Name); err != nil {
		return err
	}
	if err := validateDefinitionIdentifier("entity set name", definition.EntitySetName); err != nil {
		return err
	}
	if len(definition.Properties) == 0 {
		return fmt.Errorf("entity definition must contain at least one property")
	}

	properties := make(map[string]PropertyDefinition, len(definition.Properties))
	for _, property := range definition.Properties {
		if err := validateDefinitionIdentifier("property name", property.Name); err != nil {
			return err
		}
		if _, exists := properties[property.Name]; exists {
			return fmt.Errorf("property %q is defined more than once", property.Name)
		}
		if _, err := ParsePrimitiveType(string(property.Type)); err != nil {
			return fmt.Errorf("property %q: %w", property.Name, err)
		}
		properties[property.Name] = property
	}

	if len(definition.Keys) == 0 {
		return fmt.Errorf("entity %q must have at least one key property", definition.Name)
	}
	keys := make(map[string]struct{}, len(definition.Keys))
	for _, keyName := range definition.Keys {
		if _, exists := keys[keyName]; exists {
			return fmt.Errorf("key property %q is listed more than once", keyName)
		}
		keys[keyName] = struct{}{}

		property, exists := properties[keyName]
		if !exists {
			return fmt.Errorf("key property %q is not defined", keyName)
		}
		if property.Nullable != nil && *property.Nullable {
			return fmt.Errorf("key property %q must not be nullable", keyName)
		}
		if !isValidPrimitiveKeyType(property.Type) {
			return fmt.Errorf("key property %q cannot use %s", keyName, property.Type)
		}
	}

	disabledMethods := make(map[string]struct{}, len(definition.DisabledMethods))
	for _, method := range definition.DisabledMethods {
		normalized := strings.ToUpper(strings.TrimSpace(method))
		switch normalized {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			return fmt.Errorf("unsupported HTTP method %q; supported methods are GET, POST, PUT, PATCH, DELETE", method)
		}
		if _, exists := disabledMethods[normalized]; exists {
			return fmt.Errorf("HTTP method %q is disabled more than once", normalized)
		}
		disabledMethods[normalized] = struct{}{}
	}

	return nil
}

// AnalyzeEntityDefinition validates a runtime definition and converts it to entity metadata.
func AnalyzeEntityDefinition(definition EntityDefinition) (*EntityMetadata, error) {
	if err := definition.Validate(); err != nil {
		return nil, err
	}

	keyNames := make(map[string]struct{}, len(definition.Keys))
	for _, keyName := range definition.Keys {
		keyNames[keyName] = struct{}{}
	}

	entityMetadata := &EntityMetadata{
		EntityName:      definition.Name,
		EntitySetName:   definition.EntitySetName,
		TableName:       definition.EntitySetName,
		IsVirtual:       true,
		Properties:      make([]PropertyMetadata, 0, len(definition.Properties)),
		KeyProperties:   make([]PropertyMetadata, 0, len(definition.Keys)),
		DisabledMethods: make(map[string]bool, len(definition.DisabledMethods)),
	}
	properties := make(map[string]PropertyMetadata, len(definition.Properties))
	for _, definitionProperty := range definition.Properties {
		_, isKey := keyNames[definitionProperty.Name]
		nullable := !isKey
		if definitionProperty.Nullable != nil {
			nullable = *definitionProperty.Nullable
		}
		property := PropertyMetadata{
			Name:       definitionProperty.Name,
			FieldName:  definitionProperty.Name,
			ColumnName: definitionProperty.Name,
			JsonName:   definitionProperty.Name,
			EdmType:    definitionProperty.Type,
			IsKey:      isKey,
			IsRequired: !nullable,
			Nullable:   boolPointer(nullable),
		}
		entityMetadata.Properties = append(entityMetadata.Properties, property)
		properties[property.Name] = property
	}
	for _, keyName := range definition.Keys {
		entityMetadata.KeyProperties = append(entityMetadata.KeyProperties, properties[keyName])
	}
	if len(entityMetadata.KeyProperties) == 1 {
		entityMetadata.KeyProperty = &entityMetadata.KeyProperties[0]
	}
	for _, method := range definition.DisabledMethods {
		entityMetadata.DisabledMethods[strings.ToUpper(strings.TrimSpace(method))] = true
	}

	return entityMetadata, nil
}

func validateDefinitionIdentifier(kind, identifier string) error {
	if identifier == "" {
		return fmt.Errorf("%s cannot be empty", kind)
	}
	if !utf8.ValidString(identifier) || utf8.RuneCountInString(identifier) > 128 {
		return fmt.Errorf("%s %q is not a valid OData identifier", kind, identifier)
	}
	for index, character := range []rune(identifier) {
		if index == 0 {
			if character != '_' && !unicode.IsLetter(character) {
				return fmt.Errorf("%s %q is not a valid OData identifier", kind, identifier)
			}
			continue
		}
		if character != '_' && !unicode.IsLetter(character) && !unicode.IsDigit(character) &&
			!unicode.In(character, unicode.Mn, unicode.Mc, unicode.Pc, unicode.Cf) {
			return fmt.Errorf("%s %q is not a valid OData identifier", kind, identifier)
		}
	}
	return nil
}

func isValidPrimitiveKeyType(primitiveType PrimitiveType) bool {
	switch primitiveType {
	case PrimitiveTypeBoolean, PrimitiveTypeByte, PrimitiveTypeDate, PrimitiveTypeDateTimeOffset,
		PrimitiveTypeDecimal, PrimitiveTypeDuration, PrimitiveTypeGuid, PrimitiveTypeInt16,
		PrimitiveTypeInt32, PrimitiveTypeInt64, PrimitiveTypeSByte, PrimitiveTypeString,
		PrimitiveTypeTimeOfDay:
		return true
	default:
		return false
	}
}

func boolPointer(value bool) *bool {
	return &value
}
