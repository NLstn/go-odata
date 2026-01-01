package handlers

import (
	"sort"

	"github.com/nlstn/go-odata/internal/metadata"
)

type navigationBinding struct {
	path   string
	target string
}

type namedEnumDefinition struct {
	name string
	info *enumTypeInfo
}

func (h *MetadataHandler) navigationBindings(model metadataModel, entityMeta *metadata.EntityMetadata) []navigationBinding {
	bindings := make([]navigationBinding, 0, len(entityMeta.Properties))
	for _, prop := range entityMeta.Properties {
		if prop.IsNavigationProp {
			targetEntitySet := model.getEntitySetNameForType(prop.NavigationTarget)
			bindings = append(bindings, navigationBinding{
				path:   prop.JsonName,
				target: targetEntitySet,
			})
		}
	}
	return bindings
}

func (h *MetadataHandler) sortedEnumDefinitions(model metadataModel) []namedEnumDefinition {
	enumDefinitions := model.collectEnumDefinitions()
	if len(enumDefinitions) == 0 {
		return nil
	}

	enumNames := make([]string, 0, len(enumDefinitions))
	for name := range enumDefinitions {
		enumNames = append(enumNames, name)
	}
	sort.Strings(enumNames)

	sorted := make([]namedEnumDefinition, 0, len(enumNames))
	for _, name := range enumNames {
		info := enumDefinitions[name]
		if info == nil || len(info.Members) == 0 {
			continue
		}
		sorted = append(sorted, namedEnumDefinition{
			name: name,
			info: info,
		})
	}

	return sorted
}

func (h *MetadataHandler) propertyEdmType(model metadataModel, prop *metadata.PropertyMetadata) string {
	if prop.IsEnum && prop.EnumTypeName != "" {
		return model.qualifiedTypeName(prop.EnumTypeName)
	}
	return getEdmType(prop.Type)
}

func (h *MetadataHandler) propertyNullable(prop *metadata.PropertyMetadata) (value bool, include bool) {
	if prop.Nullable != nil {
		return *prop.Nullable, true
	}
	if !prop.IsRequired && !prop.IsKey {
		return true, true
	}
	return false, false
}
