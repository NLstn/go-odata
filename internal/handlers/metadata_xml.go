package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/version"
)

// handleMetadataXML handles XML metadata format with version-specific caching
func (h *MetadataHandler) handleMetadataXML(w http.ResponseWriter, r *http.Request) {
	// Get the negotiated OData version from the request context
	ver := version.GetVersion(r.Context())
	versionKey := ver.String()

	// Lock-free cache lookup (fast path - common case)
	if cached, ok := h.cachedXML.Load(versionKey); ok {
		cachedStr, ok := cached.(string)
		if !ok {
			// Cache corruption - rebuild
			h.cachedXML.Delete(versionKey)
			h.cacheSizeXML.Add(-1)
			h.logger.Warn("Invalid cache entry, rebuilding", "version", versionKey)
		} else {
			w.Header().Set("Content-Type", "application/xml")

			if r.Method == http.MethodHead {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(cachedStr)))
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(cachedStr)); err != nil {
				h.logger.Error("Error writing metadata response", "error", err)
			}
			return
		}
	}

	// Cache miss - build metadata (slow path)
	model := h.newMetadataModel()
	cached := h.buildMetadataDocument(model, ver)

	// Store in cache using LoadOrStore for thread safety
	// (another goroutine might have built it while we were building)
	actual, loaded := h.cachedXML.LoadOrStore(versionKey, cached)
	if !loaded {
		// We stored our version, increment counter and check for eviction
		newSize := h.cacheSizeXML.Add(1)
		if newSize > maxCacheEntries {
			h.evictOldCacheEntriesXML()
		}
	}

	// Use the actual cached value (ours or the one that was stored by another goroutine)
	cachedStr, ok := actual.(string)
	if !ok {
		// Should never happen, but handle gracefully
		h.logger.Error("Invalid cache entry type", "version", versionKey)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")

	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(cachedStr)))
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(cachedStr)); err != nil {
		h.logger.Error("Error writing metadata response", "error", err)
	}
}

func (h *MetadataHandler) buildMetadataDocument(model metadataModel, ver version.Version) string {
	var builder strings.Builder

	// Collect all vocabulary namespaces used in annotations
	usedVocabularies := model.collectUsedVocabularies()
	vocabularyAliases := metadata.VocabularyAliasMap()

	builder.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<edmx:Edmx xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx" Version="%s">
`, ver.String()))

	// Add Reference elements for used vocabularies
	for _, ns := range usedVocabularies {
		alias := vocabularyAliases[ns]
		if alias == "" {
			// Use the last part of namespace as alias if not in map
			parts := strings.Split(ns, ".")
			alias = parts[len(parts)-1]
		}
		uri := vocabularyURI(ns)
		builder.WriteString(fmt.Sprintf(`  <edmx:Reference Uri="%s">
    <edmx:Include Namespace="%s" Alias="%s" />
  </edmx:Reference>
`, uri, ns, alias))
	}

	builder.WriteString(fmt.Sprintf(`  <edmx:DataServices>
    <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="%s">
`, model.namespace))

	builder.WriteString(h.buildEnumTypes(model))
	builder.WriteString(h.buildEntityTypes(model))
	builder.WriteString(h.buildEntityContainer(model))
	builder.WriteString(h.buildAnnotations(model))

	builder.WriteString(`    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`)

	return builder.String()
}

func (h *MetadataHandler) buildEnumTypes(model metadataModel) string {
	enumDefinitions := h.sortedEnumDefinitions(model)
	if len(enumDefinitions) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, definition := range enumDefinitions {
		builder.WriteString(h.buildEnumType(definition.name, definition.info))
	}

	return builder.String()
}

func (h *MetadataHandler) buildEnumType(enumTypeName string, info *enumTypeInfo) string {
	if info == nil {
		return ""
	}

	flagsAttr := ""
	if info.IsFlags {
		flagsAttr = ` IsFlags="true"`
	}

	underlyingType := "Edm.Int32"
	if info.UnderlyingType != "" {
		underlyingType = info.UnderlyingType
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`      <EnumType Name="%s" UnderlyingType="%s"%s>`, enumTypeName, underlyingType, flagsAttr))
	builder.WriteString("\n")

	for _, member := range info.Members {
		builder.WriteString(fmt.Sprintf(`        <Member Name="%s" Value="%d" />`, member.Name, member.Value))
		builder.WriteString("\n")
	}

	builder.WriteString("      </EnumType>\n")
	return builder.String()
}

func (h *MetadataHandler) buildEntityTypes(model metadataModel) string {
	var builder strings.Builder
	for _, entityMeta := range model.entities {
		builder.WriteString(h.buildEntityType(model, entityMeta))
	}
	return builder.String()
}

func (h *MetadataHandler) buildEntityType(model metadataModel, entityMeta *metadata.EntityMetadata) string {
	hasStreamAttr := ""
	if entityMeta.HasStream {
		hasStreamAttr = ` HasStream="true"`
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`      <EntityType Name="%s"%s>
        <Key>
`, entityMeta.EntityName, hasStreamAttr))

	for _, keyProp := range entityMeta.KeyProperties {
		builder.WriteString(fmt.Sprintf(`          <PropertyRef Name="%s" />
`, keyProp.JsonName))
	}

	builder.WriteString(`        </Key>
`)

	builder.WriteString(h.buildRegularProperties(model, entityMeta))
	builder.WriteString(h.buildNavigationProperties(model, entityMeta))

	builder.WriteString(`      </EntityType>
`)
	return builder.String()
}

func (h *MetadataHandler) buildRegularProperties(model metadataModel, entityMeta *metadata.EntityMetadata) string {
	var builder strings.Builder
	for _, prop := range entityMeta.Properties {
		if prop.IsNavigationProp {
			continue
		}

		if prop.IsStream || strings.HasSuffix(prop.FieldName, "ContentType") || strings.HasSuffix(prop.FieldName, "Content") {
			isStreamField := false
			for _, streamProp := range entityMeta.StreamProperties {
				if prop.FieldName == streamProp.StreamContentTypeField || prop.FieldName == streamProp.StreamContentField {
					isStreamField = true
					break
				}
			}
			if isStreamField {
				continue
			}
		}

		edmType := h.propertyEdmType(model, &prop)
		nullableValue, _ := h.propertyNullable(&prop)
		nullable := "false"
		if nullableValue {
			nullable = "true"
		}

		attrs := fmt.Sprintf(`Name="%s" Type="%s" Nullable="%s"`, prop.JsonName, edmType, nullable)

		// MaxLength is valid for Edm.String and Edm.Binary
		if prop.MaxLength > 0 && (edmType == "Edm.String" || edmType == "Edm.Binary") {
			attrs += fmt.Sprintf(` MaxLength="%d"`, prop.MaxLength)
		}
		// Precision and Scale are ONLY valid for Edm.Decimal per OData CSDL spec
		if edmType == "Edm.Decimal" {
			if prop.Precision > 0 {
				attrs += fmt.Sprintf(` Precision="%d"`, prop.Precision)
			}
			if prop.Scale > 0 {
				attrs += fmt.Sprintf(` Scale="%d"`, prop.Scale)
			}
		}
		if prop.DefaultValue != "" {
			attrs += fmt.Sprintf(` DefaultValue="%s"`, prop.DefaultValue)
		}

		builder.WriteString(fmt.Sprintf(`        <Property %s />
`, attrs))
	}

	for _, streamProp := range entityMeta.StreamProperties {
		builder.WriteString(fmt.Sprintf(`        <Property Name="%s" Type="Edm.Stream" />
`, streamProp.Name))
	}

	return builder.String()
}

func (h *MetadataHandler) buildNavigationProperties(model metadataModel, entityMeta *metadata.EntityMetadata) string {
	var builder strings.Builder
	for _, prop := range entityMeta.Properties {
		if !prop.IsNavigationProp {
			continue
		}

		typeName := model.qualifiedTypeName(prop.NavigationTarget)
		if prop.NavigationIsArray {
			typeName = fmt.Sprintf("Collection(%s)", typeName)
		}

		if len(prop.ReferentialConstraints) > 0 {
			builder.WriteString(fmt.Sprintf(`        <NavigationProperty Name="%s" Type="%s">
`, prop.JsonName, typeName))
			builder.WriteString(`          <ReferentialConstraint>
`)
			for dependent, principal := range prop.ReferentialConstraints {
				builder.WriteString(fmt.Sprintf(`            <Property Name="%s" ReferencedProperty="%s" />
`, dependent, principal))
			}
			builder.WriteString(`          </ReferentialConstraint>
        </NavigationProperty>
`)
		} else {
			builder.WriteString(fmt.Sprintf(`        <NavigationProperty Name="%s" Type="%s" />
`, prop.JsonName, typeName))
		}
	}
	return builder.String()
}

func (h *MetadataHandler) buildEntityContainer(model metadataModel) string {
	var builder strings.Builder
	builder.WriteString(`      <EntityContainer Name="Container">
`)
	for entitySetName, entityMeta := range model.entities {
		navigationBindings := h.navigationBindings(model, entityMeta)
		if entityMeta.IsSingleton {
			builder.WriteString(fmt.Sprintf(`        <Singleton Name="%s" Type="%s">
`, entityMeta.SingletonName, model.qualifiedTypeName(entityMeta.EntityName)))
			for _, binding := range navigationBindings {
				builder.WriteString(fmt.Sprintf(`          <NavigationPropertyBinding Path="%s" Target="%s" />
`, binding.path, binding.target))
			}
			builder.WriteString(`        </Singleton>
`)
		} else {
			builder.WriteString(fmt.Sprintf(`        <EntitySet Name="%s" EntityType="%s">
`, entitySetName, model.qualifiedTypeName(entityMeta.EntityName)))
			for _, binding := range navigationBindings {
				builder.WriteString(fmt.Sprintf(`          <NavigationPropertyBinding Path="%s" Target="%s" />
`, binding.path, binding.target))
			}
			builder.WriteString(`        </EntitySet>
`)
		}
	}

	builder.WriteString(`      </EntityContainer>
`)
	return builder.String()
}

// buildAnnotations builds the Annotations sections for all annotated targets
func (h *MetadataHandler) buildAnnotations(model metadataModel) string {
	var builder strings.Builder

	// Build annotations for each entity type and its properties
	for _, entityMeta := range model.entities {
		// Entity type annotations
		if entityMeta.Annotations != nil && entityMeta.Annotations.Len() > 0 {
			target := model.qualifiedTypeName(entityMeta.EntityName)
			builder.WriteString(fmt.Sprintf(`      <Annotations Target="%s">
`, target))
			for _, annotation := range entityMeta.Annotations.Get() {
				builder.WriteString(h.buildAnnotationXML(annotation, 8))
			}
			builder.WriteString(`      </Annotations>
`)
		}

		// Property annotations
		for _, prop := range entityMeta.Properties {
			if prop.Annotations != nil && prop.Annotations.Len() > 0 {
				target := fmt.Sprintf("%s/%s", model.qualifiedTypeName(entityMeta.EntityName), prop.JsonName)
				builder.WriteString(fmt.Sprintf(`      <Annotations Target="%s">
`, target))
				for _, annotation := range prop.Annotations.Get() {
					builder.WriteString(h.buildAnnotationXML(annotation, 8))
				}
				builder.WriteString(`      </Annotations>
`)
			}
		}
	}

	return builder.String()
}

// buildAnnotationXML builds the XML representation of a single annotation
func (h *MetadataHandler) buildAnnotationXML(annotation metadata.Annotation, indent int) string {
	indentStr := strings.Repeat(" ", indent)

	escapedTerm := escapeXML(annotation.Term)

	// Handle boolean values inline
	if boolVal, ok := annotation.Value.(bool); ok {
		return fmt.Sprintf(`%s<Annotation Term="%s" Bool="%t" />
`, indentStr, escapedTerm, boolVal)
	}

	// Handle string values
	if strVal, ok := annotation.Value.(string); ok {
		return fmt.Sprintf(`%s<Annotation Term="%s" String="%s" />
`, indentStr, escapedTerm, escapeXML(strVal))
	}

	// Handle integer values (all signed integer types)
	switch intVal := annotation.Value.(type) {
	case int:
		return fmt.Sprintf(`%s<Annotation Term="%s" Int="%d" />
`, indentStr, escapedTerm, intVal)
	case int8:
		return fmt.Sprintf(`%s<Annotation Term="%s" Int="%d" />
`, indentStr, escapedTerm, intVal)
	case int16:
		return fmt.Sprintf(`%s<Annotation Term="%s" Int="%d" />
`, indentStr, escapedTerm, intVal)
	case int32:
		return fmt.Sprintf(`%s<Annotation Term="%s" Int="%d" />
`, indentStr, escapedTerm, intVal)
	case int64:
		return fmt.Sprintf(`%s<Annotation Term="%s" Int="%d" />
`, indentStr, escapedTerm, intVal)
	case uint:
		return fmt.Sprintf(`%s<Annotation Term="%s" Int="%d" />
`, indentStr, escapedTerm, intVal)
	case uint8:
		return fmt.Sprintf(`%s<Annotation Term="%s" Int="%d" />
`, indentStr, escapedTerm, intVal)
	case uint16:
		return fmt.Sprintf(`%s<Annotation Term="%s" Int="%d" />
`, indentStr, escapedTerm, intVal)
	case uint32:
		return fmt.Sprintf(`%s<Annotation Term="%s" Int="%d" />
`, indentStr, escapedTerm, intVal)
	case uint64:
		return fmt.Sprintf(`%s<Annotation Term="%s" Int="%d" />
`, indentStr, escapedTerm, intVal)
	}

	// Handle float values (both float32 and float64)
	if floatVal, ok := annotation.Value.(float64); ok {
		return fmt.Sprintf(`%s<Annotation Term="%s" Float="%g" />
`, indentStr, escapedTerm, floatVal)
	}
	if floatVal, ok := annotation.Value.(float32); ok {
		return fmt.Sprintf(`%s<Annotation Term="%s" Float="%g" />
`, indentStr, escapedTerm, float64(floatVal))
	}

	// Default: treat as string with XML escaping
	escapedValue := escapeXML(fmt.Sprintf("%v", annotation.Value))
	return fmt.Sprintf(`%s<Annotation Term="%s" String="%s" />
`, indentStr, escapedTerm, escapedValue)
}

// escapeXML escapes special characters for XML output
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// vocabularyURI returns the canonical URI for a vocabulary namespace
func vocabularyURI(namespace string) string {
	// Standard OData vocabularies
	standardVocabularyURIs := map[string]string{
		"Org.OData.Core.V1":         "https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Core.V1.xml",
		"Org.OData.Capabilities.V1": "https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Capabilities.V1.xml",
		"Org.OData.Validation.V1":   "https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Validation.V1.xml",
		"Org.OData.Measures.V1":     "https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Measures.V1.xml",
		"Org.OData.Authorization.V1": "https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Authorization.V1.xml",
	}

	if uri, ok := standardVocabularyURIs[namespace]; ok {
		return uri
	}

	// For custom vocabularies, use a generic pattern
	// Replace dots with slashes to create a reasonable URI
	return "urn:custom:vocabulary:" + namespace
}
