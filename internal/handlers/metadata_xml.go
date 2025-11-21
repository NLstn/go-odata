package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

// handleMetadataXML handles XML metadata format (existing implementation)
func (h *MetadataHandler) handleMetadataXML(w http.ResponseWriter, r *http.Request) {
	model := h.newMetadataModel()
	h.onceXML.Do(func() {
		h.cachedXML = h.buildMetadataDocument(model)
	})

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(h.cachedXML)))
		return
	}

	if _, err := w.Write([]byte(h.cachedXML)); err != nil {
		h.logger.Error("Error writing metadata response", "error", err)
	}
}

func (h *MetadataHandler) buildMetadataDocument(model metadataModel) string {
	metadata := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<edmx:Edmx xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx" Version="%s">
  <edmx:DataServices>
    <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="%s">
`, response.ODataVersionValue, model.namespace)

	metadata += h.buildEnumTypes(model)
	metadata += h.buildEntityTypes(model)
	metadata += h.buildEntityContainer(model)

	metadata += `    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	return metadata
}

func (h *MetadataHandler) buildEnumTypes(model metadataModel) string {
	enumDefinitions := model.collectEnumDefinitions()
	if len(enumDefinitions) == 0 {
		return ""
	}

	enumNames := make([]string, 0, len(enumDefinitions))
	for name := range enumDefinitions {
		enumNames = append(enumNames, name)
	}
	sort.Strings(enumNames)

	var builder strings.Builder
	for _, name := range enumNames {
		info := enumDefinitions[name]
		if info == nil || len(info.Members) == 0 {
			continue
		}
		builder.WriteString(h.buildEnumType(name, info))
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
	result := ""
	for _, entityMeta := range model.entities {
		result += h.buildEntityType(model, entityMeta)
	}
	return result
}

func (h *MetadataHandler) buildEntityType(model metadataModel, entityMeta *metadata.EntityMetadata) string {
	hasStreamAttr := ""
	if entityMeta.HasStream {
		hasStreamAttr = ` HasStream="true"`
	}

	result := fmt.Sprintf(`      <EntityType Name="%s"%s>
        <Key>
`, entityMeta.EntityName, hasStreamAttr)

	for _, keyProp := range entityMeta.KeyProperties {
		result += fmt.Sprintf(`          <PropertyRef Name="%s" />
`, keyProp.JsonName)
	}

	result += `        </Key>
`

	result += h.buildRegularProperties(model, entityMeta)
	result += h.buildNavigationProperties(model, entityMeta)

	result += `      </EntityType>
`
	return result
}

func (h *MetadataHandler) buildRegularProperties(model metadataModel, entityMeta *metadata.EntityMetadata) string {
	result := ""
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

		var edmType string
		if prop.IsEnum && prop.EnumTypeName != "" {
			edmType = model.qualifiedTypeName(prop.EnumTypeName)
		} else {
			edmType = getEdmType(prop.Type)
		}

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

		result += fmt.Sprintf(`        <Property %s />
`, attrs)
	}

	for _, streamProp := range entityMeta.StreamProperties {
		result += fmt.Sprintf(`        <Property Name="%s" Type="Edm.Stream" />
`, streamProp.Name)
	}

	return result
}

func (h *MetadataHandler) buildNavigationProperties(model metadataModel, entityMeta *metadata.EntityMetadata) string {
	result := ""
	for _, prop := range entityMeta.Properties {
		if !prop.IsNavigationProp {
			continue
		}

		typeName := model.qualifiedTypeName(prop.NavigationTarget)
		if prop.NavigationIsArray {
			typeName = fmt.Sprintf("Collection(%s)", typeName)
		}

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

func (h *MetadataHandler) buildEntityContainer(model metadataModel) string {
	result := `      <EntityContainer Name="Container">
`
	for entitySetName, entityMeta := range model.entities {
		if entityMeta.IsSingleton {
			result += fmt.Sprintf(`        <Singleton Name="%s" Type="%s">
`, entityMeta.SingletonName, model.qualifiedTypeName(entityMeta.EntityName))
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
			result += fmt.Sprintf(`        <EntitySet Name="%s" EntityType="%s">
`, entitySetName, model.qualifiedTypeName(entityMeta.EntityName))
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
