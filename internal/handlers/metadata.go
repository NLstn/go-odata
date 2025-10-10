package handlers

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

// MetadataHandler handles metadata document requests
type MetadataHandler struct {
	entities map[string]*metadata.EntityMetadata
}

// NewMetadataHandler creates a new metadata handler
func NewMetadataHandler(entities map[string]*metadata.EntityMetadata) *MetadataHandler {
	return &MetadataHandler{
		entities: entities,
	}
}

// HandleMetadata handles the metadata document endpoint
func (h *MetadataHandler) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for metadata document", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)

	metadataDoc := h.buildMetadataDocument()

	if _, err := w.Write([]byte(metadataDoc)); err != nil {
		fmt.Printf("Error writing metadata response: %v\n", err)
	}
}

// buildMetadataDocument builds the complete metadata XML document
func (h *MetadataHandler) buildMetadataDocument() string {
	metadata := `<?xml version="1.0" encoding="UTF-8"?>
<edmx:Edmx xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx" Version="4.0">
  <edmx:DataServices>
    <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="ODataService">
`

	// Add entity types
	metadata += h.buildEntityTypes()

	// Add entity container with entity sets
	metadata += h.buildEntityContainer()

	metadata += `    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	return metadata
}

// buildEntityTypes builds the entity type definitions
func (h *MetadataHandler) buildEntityTypes() string {
	result := ""
	for _, entityMeta := range h.entities {
		result += h.buildEntityType(entityMeta)
	}
	return result
}

// buildEntityType builds a single entity type definition
func (h *MetadataHandler) buildEntityType(entityMeta *metadata.EntityMetadata) string {
	result := fmt.Sprintf(`      <EntityType Name="%s">
        <Key>
`, entityMeta.EntityName)

	// Add all key properties (supports composite keys)
	for _, keyProp := range entityMeta.KeyProperties {
		result += fmt.Sprintf(`          <PropertyRef Name="%s" />
`, keyProp.JsonName)
	}

	result += `        </Key>
`

	// Add regular properties
	result += h.buildRegularProperties(entityMeta)

	// Add navigation properties
	result += h.buildNavigationProperties(entityMeta)

	result += `      </EntityType>
`
	return result
}

// buildRegularProperties builds the regular (non-navigation) properties
func (h *MetadataHandler) buildRegularProperties(entityMeta *metadata.EntityMetadata) string {
	result := ""
	for _, prop := range entityMeta.Properties {
		if prop.IsNavigationProp {
			continue // Handle navigation properties separately
		}

		edmType := getEdmType(prop.Type)
		nullable := "false"
		if !prop.IsRequired && !prop.IsKey {
			nullable = "true"
		}
		result += fmt.Sprintf(`        <Property Name="%s" Type="%s" Nullable="%s" />
`, prop.JsonName, edmType, nullable)
	}
	return result
}

// buildNavigationProperties builds the navigation properties
func (h *MetadataHandler) buildNavigationProperties(entityMeta *metadata.EntityMetadata) string {
	result := ""
	for _, prop := range entityMeta.Properties {
		if !prop.IsNavigationProp {
			continue
		}

		if prop.NavigationIsArray {
			result += fmt.Sprintf(`        <NavigationProperty Name="%s" Type="Collection(ODataService.%s)" />
`, prop.JsonName, prop.NavigationTarget)
		} else {
			result += fmt.Sprintf(`        <NavigationProperty Name="%s" Type="ODataService.%s" />
`, prop.JsonName, prop.NavigationTarget)
		}
	}
	return result
}

// buildEntityContainer builds the entity container with entity sets
func (h *MetadataHandler) buildEntityContainer() string {
	result := `      <EntityContainer Name="Container">
`
	for entitySetName, entityMeta := range h.entities {
		result += fmt.Sprintf(`        <EntitySet Name="%s" EntityType="ODataService.%s">
`, entitySetName, entityMeta.EntityName)

		// Add navigation property bindings
		for _, prop := range entityMeta.Properties {
			if prop.IsNavigationProp {
				targetEntitySet := pluralize(prop.NavigationTarget)
				result += fmt.Sprintf(`          <NavigationPropertyBinding Path="%s" Target="%s" />
`, prop.JsonName, targetEntitySet)
			}
		}

		result += `        </EntitySet>
`
	}

	result += `      </EntityContainer>
`
	return result
}

// getEdmType converts a Go type to an EDM (Entity Data Model) type
func getEdmType(goType reflect.Type) string {
	// Handle pointer types
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	switch goType.Kind() {
	case reflect.String:
		return "Edm.String"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "Edm.Int32"
	case reflect.Int64:
		return "Edm.Int64"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "Edm.Int32"
	case reflect.Uint64:
		return "Edm.Int64"
	case reflect.Float32:
		return "Edm.Single"
	case reflect.Float64:
		return "Edm.Double"
	case reflect.Bool:
		return "Edm.Boolean"
	default:
		// For complex types, return as string
		return "Edm.String"
	}
}

// pluralize creates a simple pluralized form of the entity name
func pluralize(word string) string {
	if word == "" {
		return word
	}

	// Simple pluralization rules
	switch {
	case strings.HasSuffix(word, "y"):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") || strings.HasSuffix(word, "z") ||
		strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh"):
		return word + "es"
	default:
		return word + "s"
	}
}
