package handlers

import (
	"encoding/json"
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
	switch r.Method {
	case http.MethodGet:
		h.handleGetMetadata(w, r)
	case http.MethodOptions:
		h.handleOptionsMetadata(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for metadata document", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
	}
}

// handleGetMetadata handles GET requests for metadata document
func (h *MetadataHandler) handleGetMetadata(w http.ResponseWriter, r *http.Request) {
	// Check if JSON format is requested via $format query parameter or Accept header
	useJSON := shouldReturnJSON(r)

	if useJSON {
		h.handleMetadataJSON(w)
	} else {
		h.handleMetadataXML(w)
	}
}

// handleOptionsMetadata handles OPTIONS requests for metadata document
func (h *MetadataHandler) handleOptionsMetadata(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, OPTIONS")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)
}

// shouldReturnJSON determines if JSON format should be returned based on request
func shouldReturnJSON(r *http.Request) bool {
	// Check $format query parameter first
	format := r.URL.Query().Get("$format")
	if format == "json" || format == "application/json" {
		return true
	}

	// Check Accept header
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

// handleMetadataXML handles XML metadata format (existing implementation)
func (h *MetadataHandler) handleMetadataXML(w http.ResponseWriter) {
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

		// Determine nullable
		nullable := "false"
		if prop.Nullable != nil {
			if *prop.Nullable {
				nullable = "true"
			} else {
				nullable = "false"
			}
		} else if !prop.IsRequired && !prop.IsKey {
			nullable = "true"
		}

		// Build property attributes
		attrs := fmt.Sprintf(`Name="%s" Type="%s" Nullable="%s"`, prop.JsonName, edmType, nullable)

		// Add facets
		if prop.MaxLength > 0 {
			attrs += fmt.Sprintf(` MaxLength="%d"`, prop.MaxLength)
		}
		if prop.Precision > 0 {
			attrs += fmt.Sprintf(` Precision="%d"`, prop.Precision)
		}
		if prop.Scale > 0 {
			attrs += fmt.Sprintf(` Scale="%d"`, prop.Scale)
		}
		if prop.DefaultValue != "" {
			attrs += fmt.Sprintf(` DefaultValue="%s"`, prop.DefaultValue)
		}

		result += fmt.Sprintf(`        <Property %s />
`, attrs)
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

		typeName := fmt.Sprintf("ODataService.%s", prop.NavigationTarget)
		if prop.NavigationIsArray {
			typeName = fmt.Sprintf("Collection(%s)", typeName)
		}

		// Check if we have referential constraints
		if len(prop.ReferentialConstraints) > 0 {
			result += fmt.Sprintf(`        <NavigationProperty Name="%s" Type="%s">
`, prop.JsonName, typeName)
			result += `          <ReferentialConstraint>
`
			for dependent, principal := range prop.ReferentialConstraints {
				result += fmt.Sprintf(`            <Property Name="%s" ReferencedProperty="%s" />
`, dependent, principal)
			}
			result += `          </ReferentialConstraint>
        </NavigationProperty>
`
		} else {
			result += fmt.Sprintf(`        <NavigationProperty Name="%s" Type="%s" />
`, prop.JsonName, typeName)
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

// handleMetadataJSON handles JSON metadata format (CSDL JSON)
func (h *MetadataHandler) handleMetadataJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	// Build CSDL JSON structure
	odataService := make(map[string]interface{})
	csdl := map[string]interface{}{
		"$Version":     "4.0",
		"ODataService": odataService,
	}

	// Add entity types
	for _, entityMeta := range h.entities {
		entityType := h.buildJSONEntityType(entityMeta)
		odataService[entityMeta.EntityName] = entityType
	}

	// Add entity container
	container := h.buildJSONEntityContainer()
	odataService["Container"] = container

	// Encode and write JSON
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(csdl); err != nil {
		fmt.Printf("Error writing JSON metadata response: %v\n", err)
	}
}

// buildJSONEntityType builds a JSON entity type definition
func (h *MetadataHandler) buildJSONEntityType(entityMeta *metadata.EntityMetadata) map[string]interface{} {
	entityType := make(map[string]interface{})
	entityType["$Kind"] = "EntityType"

	// Add key(s) - supports both single and composite keys
	keyNames := make([]string, 0, len(entityMeta.KeyProperties))
	for _, keyProp := range entityMeta.KeyProperties {
		keyNames = append(keyNames, keyProp.JsonName)
	}
	entityType["$Key"] = keyNames

	// Add regular properties
	h.addJSONRegularProperties(entityType, entityMeta)

	// Add navigation properties
	h.addJSONNavigationProperties(entityType, entityMeta)

	return entityType
}

// addJSONRegularProperties adds regular properties to a JSON entity type
func (h *MetadataHandler) addJSONRegularProperties(entityType map[string]interface{}, entityMeta *metadata.EntityMetadata) {
	for _, prop := range entityMeta.Properties {
		if prop.IsNavigationProp {
			continue // Handle navigation properties separately
		}

		propDef := h.buildJSONPropertyDefinition(&prop)
		entityType[prop.JsonName] = propDef
	}
}

// buildJSONPropertyDefinition builds a JSON property definition
func (h *MetadataHandler) buildJSONPropertyDefinition(prop *metadata.PropertyMetadata) map[string]interface{} {
	propDef := make(map[string]interface{})
	propDef["$Type"] = getEdmType(prop.Type)

	// Set nullable
	if prop.Nullable != nil {
		propDef["$Nullable"] = *prop.Nullable
	} else if !prop.IsRequired && !prop.IsKey {
		propDef["$Nullable"] = true
	}

	// Add facets
	h.addJSONPropertyFacets(propDef, prop)

	return propDef
}

// addJSONPropertyFacets adds facets to a JSON property definition
func (h *MetadataHandler) addJSONPropertyFacets(propDef map[string]interface{}, prop *metadata.PropertyMetadata) {
	if prop.MaxLength > 0 {
		propDef["$MaxLength"] = prop.MaxLength
	}
	if prop.Precision > 0 {
		propDef["$Precision"] = prop.Precision
	}
	if prop.Scale > 0 {
		propDef["$Scale"] = prop.Scale
	}
	if prop.DefaultValue != "" {
		propDef["$DefaultValue"] = prop.DefaultValue
	}
}

// addJSONNavigationProperties adds navigation properties to a JSON entity type
func (h *MetadataHandler) addJSONNavigationProperties(entityType map[string]interface{}, entityMeta *metadata.EntityMetadata) {
	for _, prop := range entityMeta.Properties {
		if !prop.IsNavigationProp {
			continue
		}

		navProp := h.buildJSONNavigationProperty(&prop)
		entityType[prop.JsonName] = navProp
	}
}

// buildJSONNavigationProperty builds a JSON navigation property definition
func (h *MetadataHandler) buildJSONNavigationProperty(prop *metadata.PropertyMetadata) map[string]interface{} {
	navProp := make(map[string]interface{})
	navProp["$Kind"] = "NavigationProperty"

	if prop.NavigationIsArray {
		navProp["$Collection"] = true
		navProp["$Type"] = fmt.Sprintf("ODataService.%s", prop.NavigationTarget)
	} else {
		navProp["$Type"] = fmt.Sprintf("ODataService.%s", prop.NavigationTarget)
	}

	// Add referential constraints if present
	if len(prop.ReferentialConstraints) > 0 {
		constraints := make([]map[string]string, 0, len(prop.ReferentialConstraints))
		for dependent, principal := range prop.ReferentialConstraints {
			constraints = append(constraints, map[string]string{
				"Property":           dependent,
				"ReferencedProperty": principal,
			})
		}
		navProp["$ReferentialConstraint"] = constraints
	}

	return navProp
}

// buildJSONEntityContainer builds the JSON entity container
func (h *MetadataHandler) buildJSONEntityContainer() map[string]interface{} {
	container := map[string]interface{}{
		"$Kind": "EntityContainer",
	}

	for entitySetName, entityMeta := range h.entities {
		entitySet := map[string]interface{}{
			"$Collection": true,
			"$Type":       fmt.Sprintf("ODataService.%s", entityMeta.EntityName),
		}

		// Add navigation property bindings
		navigationBindings := h.buildNavigationBindings(entityMeta)
		if len(navigationBindings) > 0 {
			entitySet["$NavigationPropertyBinding"] = navigationBindings
		}

		container[entitySetName] = entitySet
	}

	return container
}

// buildNavigationBindings builds navigation property bindings for an entity
func (h *MetadataHandler) buildNavigationBindings(entityMeta *metadata.EntityMetadata) map[string]string {
	navigationBindings := make(map[string]string)
	for _, prop := range entityMeta.Properties {
		if prop.IsNavigationProp {
			targetEntitySet := pluralize(prop.NavigationTarget)
			navigationBindings[prop.JsonName] = targetEntitySet
		}
	}
	return navigationBindings
}

// getEdmType converts a Go type to an EDM (Entity Data Model) type
func getEdmType(goType reflect.Type) string {
	// Handle pointer types
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	// Check for specific types by name
	typeName := goType.String()
	switch typeName {
	case "time.Time":
		return "Edm.DateTimeOffset"
	case "uuid.UUID", "github.com/google/uuid.UUID":
		return "Edm.Guid"
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
		// Check for arrays of bytes (often used for binary data)
		if goType.Kind() == reflect.Slice && goType.Elem().Kind() == reflect.Uint8 {
			return "Edm.Binary"
		}
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
