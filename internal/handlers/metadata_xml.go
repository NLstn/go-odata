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

	// Check if we have a cached version for this OData version
	h.onceMutex.Lock()
	cached, exists := h.cachedXML[versionKey]
	if !exists {
		// Build and cache the metadata document for this version
		model := h.newMetadataModel()
		cached = h.buildMetadataDocument(model, ver)
		h.cachedXML[versionKey] = cached
	}
	h.onceMutex.Unlock()

	w.Header().Set("Content-Type", "application/xml")

	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(cached)))
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(cached)); err != nil {
		h.logger.Error("Error writing metadata response", "error", err)
	}
}

func (h *MetadataHandler) buildMetadataDocument(model metadataModel, ver version.Version) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<edmx:Edmx xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx" Version="%s">
  <edmx:DataServices>
    <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="%s">
`, ver.String(), model.namespace))

	builder.WriteString(h.buildEnumTypes(model))
	builder.WriteString(h.buildEntityTypes(model))
	builder.WriteString(h.buildEntityContainer(model))

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
