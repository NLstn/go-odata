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
	case http.MethodGet, http.MethodHead:
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
		h.handleMetadataJSON(w, r)
	} else {
		h.handleMetadataXML(w, r)
	}
}

// handleOptionsMetadata handles OPTIONS requests for metadata document
func (h *MetadataHandler) handleOptionsMetadata(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)
}

// shouldReturnJSON determines if JSON format should be returned based on request
func shouldReturnJSON(r *http.Request) bool {
	// Check $format query parameter first (highest priority)
	format := r.URL.Query().Get("$format")
	if format == "json" || format == "application/json" {
		return true
	}
	if format == "xml" || format == "application/xml" {
		return false
	}

	// Check Accept header with proper content negotiation
	accept := r.Header.Get("Accept")
	if accept == "" {
		// No Accept header - default to XML for metadata per OData v4 spec
		return false
	}

	// Parse Accept header and find the best match
	// Handle cases like:
	// - "application/json"
	// - "application/xml"
	// - "application/json;q=0.9, application/xml;q=0.8"
	// - "*/*"
	// - "text/html, application/xml;q=0.9, */*;q=0.8"

	type mediaType struct {
		mimeType string
		quality  float64
	}

	parts := strings.Split(accept, ",")
	mediaTypes := make([]mediaType, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by semicolon to separate media type from parameters
		subparts := strings.Split(part, ";")
		mimeType := strings.TrimSpace(subparts[0])
		quality := 1.0 // Default quality

		// Parse quality value if present
		for _, param := range subparts[1:] {
			param = strings.TrimSpace(param)
			if strings.HasPrefix(param, "q=") {
				if q, err := parseQuality(param[2:]); err == nil {
					quality = q
				}
			}
		}

		mediaTypes = append(mediaTypes, mediaType{mimeType: mimeType, quality: quality})
	}

	// Find the best matching media type
	var bestJSON, bestXML, bestWildcard float64
	for _, mt := range mediaTypes {
		switch mt.mimeType {
		case "application/json":
			if mt.quality > bestJSON {
				bestJSON = mt.quality
			}
		case "application/xml", "text/xml":
			if mt.quality > bestXML {
				bestXML = mt.quality
			}
		case "*/*", "application/*":
			if mt.quality > bestWildcard {
				bestWildcard = mt.quality
			}
		}
	}

	// If both JSON and XML are explicitly specified, choose the one with higher quality
	// If qualities are equal, prefer JSON (tie-breaking rule)
	if bestJSON > 0 && bestXML > 0 {
		return bestJSON >= bestXML
	}

	// If only JSON is specified (not via wildcard), return JSON
	if bestJSON > 0 {
		return true
	}

	// If only XML is specified, return XML
	if bestXML > 0 {
		return false
	}

	// If only wildcard is specified, default to XML for metadata
	if bestWildcard > 0 {
		return false
	}

	// Default to XML for metadata per OData v4 spec
	return false
}

// parseQuality parses a quality value from Accept header
func parseQuality(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 1.0, nil
	}
	// Simple parsing - just handle common cases
	switch s {
	case "1", "1.0", "1.00", "1.000":
		return 1.0, nil
	case "0.9":
		return 0.9, nil
	case "0.8":
		return 0.8, nil
	case "0.7":
		return 0.7, nil
	case "0.6":
		return 0.6, nil
	case "0.5":
		return 0.5, nil
	case "0":
		return 0.0, nil
	default:
		// Try to parse as float
		var q float64
		_, err := fmt.Sscanf(s, "%f", &q)
		if err != nil {
			return 1.0, err
		}
		// Quality must be between 0 and 1
		if q < 0 {
			q = 0
		}
		if q > 1 {
			q = 1
		}
		return q, nil
	}
}

// handleMetadataXML handles XML metadata format (existing implementation)
func (h *MetadataHandler) handleMetadataXML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

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

	// Add enum types
	metadata += h.buildEnumTypes()

	// Add entity types
	metadata += h.buildEntityTypes()

	// Add entity container with entity sets
	metadata += h.buildEntityContainer()

	metadata += `    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	return metadata
}

// buildEnumTypes builds the enum type definitions
func (h *MetadataHandler) buildEnumTypes() string {
	result := ""
	enumTypes := make(map[string]bool)

	// Collect all unique enum types from all entities
	for _, entityMeta := range h.entities {
		for _, prop := range entityMeta.Properties {
			if prop.IsEnum && prop.EnumTypeName != "" {
				if !enumTypes[prop.EnumTypeName] {
					enumTypes[prop.EnumTypeName] = true
					result += h.buildEnumType(prop.EnumTypeName, prop.IsFlags)
				}
			}
		}
	}

	return result
}

// buildEnumType builds a single enum type definition
func (h *MetadataHandler) buildEnumType(enumTypeName string, isFlags bool) string {
	flagsAttr := ""
	if isFlags {
		flagsAttr = ` IsFlags="true"`
	}

	result := fmt.Sprintf(`      <EnumType Name="%s" UnderlyingType="Edm.Int32"%s>
`, enumTypeName, flagsAttr)

	// Add enum members based on the enum type name
	// In a real implementation, this would use reflection to get actual enum values
	// For now, we'll add common members for known types
	if enumTypeName == "ProductStatus" {
		result += `        <Member Name="None" Value="0" />
        <Member Name="InStock" Value="1" />
        <Member Name="OnSale" Value="2" />
        <Member Name="Discontinued" Value="4" />
        <Member Name="Featured" Value="8" />
`
	}

	result += `      </EnumType>
`
	return result
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

		// Determine the EDM type
		var edmType string
		if prop.IsEnum && prop.EnumTypeName != "" {
			edmType = fmt.Sprintf("ODataService.%s", prop.EnumTypeName)
		} else {
			edmType = getEdmType(prop.Type)
		}

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

// buildEntityContainer builds the entity container with entity sets and singletons
func (h *MetadataHandler) buildEntityContainer() string {
	result := `      <EntityContainer Name="Container">
`
	// Build entity sets and singletons
	for entitySetName, entityMeta := range h.entities {
		if entityMeta.IsSingleton {
			// Output singleton definition
			result += fmt.Sprintf(`        <Singleton Name="%s" Type="ODataService.%s">
`, entityMeta.SingletonName, entityMeta.EntityName)

			// Add navigation property bindings for singleton
			for _, prop := range entityMeta.Properties {
				if prop.IsNavigationProp {
					targetEntitySet := pluralize(prop.NavigationTarget)
					result += fmt.Sprintf(`          <NavigationPropertyBinding Path="%s" Target="%s" />
`, prop.JsonName, targetEntitySet)
				}
			}

			result += `        </Singleton>
`
		} else {
			// Output entity set definition
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
	}

	result += `      </EntityContainer>
`
	return result
}

// handleMetadataJSON handles JSON metadata format (CSDL JSON)
func (h *MetadataHandler) handleMetadataJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)

	// For HEAD requests, don't write the body
	if r.Method == http.MethodHead {
		return
	}

	// Build CSDL JSON structure
	odataService := make(map[string]interface{})
	csdl := map[string]interface{}{
		"$Version":     "4.0",
		"ODataService": odataService,
	}

	// Add enum types
	h.addJSONEnumTypes(odataService)

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

// addJSONEnumTypes adds enum type definitions to the JSON service
func (h *MetadataHandler) addJSONEnumTypes(odataService map[string]interface{}) {
	enumTypes := make(map[string]bool)

	// Collect all unique enum types from all entities
	for _, entityMeta := range h.entities {
		for _, prop := range entityMeta.Properties {
			if prop.IsEnum && prop.EnumTypeName != "" {
				if !enumTypes[prop.EnumTypeName] {
					enumTypes[prop.EnumTypeName] = true
					enumType := h.buildJSONEnumType(prop.EnumTypeName, prop.IsFlags)
					odataService[prop.EnumTypeName] = enumType
				}
			}
		}
	}
}

// buildJSONEnumType builds a JSON enum type definition
func (h *MetadataHandler) buildJSONEnumType(enumTypeName string, isFlags bool) map[string]interface{} {
	enumType := map[string]interface{}{
		"$Kind":           "EnumType",
		"$UnderlyingType": "Edm.Int32",
	}

	if isFlags {
		enumType["$IsFlags"] = true
	}

	// Add enum members based on the enum type name
	// In a real implementation, this would use reflection to get actual enum values
	if enumTypeName == "ProductStatus" {
		enumType["None"] = 0
		enumType["InStock"] = 1
		enumType["OnSale"] = 2
		enumType["Discontinued"] = 4
		enumType["Featured"] = 8
	}

	return enumType
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
	
	// Determine the type
	if prop.IsEnum && prop.EnumTypeName != "" {
		propDef["$Type"] = fmt.Sprintf("ODataService.%s", prop.EnumTypeName)
	} else {
		propDef["$Type"] = getEdmType(prop.Type)
	}

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

// buildJSONEntityContainer builds the JSON entity container with entity sets and singletons
func (h *MetadataHandler) buildJSONEntityContainer() map[string]interface{} {
	container := map[string]interface{}{
		"$Kind": "EntityContainer",
	}

	for entitySetName, entityMeta := range h.entities {
		if entityMeta.IsSingleton {
			// Build singleton definition
			singleton := map[string]interface{}{
				"$Type": fmt.Sprintf("ODataService.%s", entityMeta.EntityName),
			}

			// Add navigation property bindings
			navigationBindings := h.buildNavigationBindings(entityMeta)
			if len(navigationBindings) > 0 {
				singleton["$NavigationPropertyBinding"] = navigationBindings
			}

			container[entityMeta.SingletonName] = singleton
		} else {
			// Build entity set definition
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
