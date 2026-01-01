package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

// handleMetadataJSON handles JSON metadata format (CSDL JSON)
func (h *MetadataHandler) handleMetadataJSON(w http.ResponseWriter, r *http.Request) {
	model := h.newMetadataModel()
	h.onceJSON.Do(func() {
		h.cachedJSON = h.buildMetadataJSON(model)
	})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(h.cachedJSON)))

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(h.cachedJSON); err != nil {
		h.logger.Error("Error writing JSON metadata response", "error", err)
	}
}

func (h *MetadataHandler) buildMetadataJSON(model metadataModel) []byte {
	odataService := make(map[string]interface{})
	csdl := map[string]interface{}{
		"$Version":         response.ODataVersionValue,
		"$EntityContainer": fmt.Sprintf("%s.Container", model.namespace),
	}
	csdl[model.namespace] = odataService

	h.addJSONEnumTypes(model, odataService)

	for _, entityMeta := range model.entities {
		entityType := h.buildJSONEntityType(model, entityMeta)
		odataService[entityMeta.EntityName] = entityType
	}

	container := h.buildJSONEntityContainer(model)
	odataService["Container"] = container

	jsonBytes, err := json.MarshalIndent(csdl, "", "  ")
	if err != nil {
		h.logger.Error("Error marshaling JSON metadata", "error", err)
		return []byte("{}")
	}

	return jsonBytes
}

func (h *MetadataHandler) addJSONEnumTypes(model metadataModel, odataService map[string]interface{}) {
	enumDefinitions := h.sortedEnumDefinitions(model)
	for _, definition := range enumDefinitions {
		enumType := h.buildJSONEnumType(definition.info)
		odataService[definition.name] = enumType
	}
}

func (h *MetadataHandler) buildJSONEnumType(info *enumTypeInfo) map[string]interface{} {
	if info == nil {
		return nil
	}

	enumType := map[string]interface{}{
		"$Kind": "EnumType",
	}

	underlyingType := "Edm.Int32"
	if info.UnderlyingType != "" {
		underlyingType = info.UnderlyingType
	}
	enumType["$UnderlyingType"] = underlyingType

	if info.IsFlags {
		enumType["$IsFlags"] = true
	}

	for _, member := range info.Members {
		enumType[member.Name] = member.Value
	}

	return enumType
}

func (h *MetadataHandler) buildJSONEntityType(model metadataModel, entityMeta *metadata.EntityMetadata) map[string]interface{} {
	entityType := make(map[string]interface{})
	entityType["$Kind"] = "EntityType"

	keyNames := make([]string, 0, len(entityMeta.KeyProperties))
	for _, keyProp := range entityMeta.KeyProperties {
		keyNames = append(keyNames, keyProp.JsonName)
	}
	entityType["$Key"] = keyNames

	h.addJSONRegularProperties(model, entityType, entityMeta)
	h.addJSONNavigationProperties(model, entityType, entityMeta)

	return entityType
}

func (h *MetadataHandler) addJSONRegularProperties(model metadataModel, entityType map[string]interface{}, entityMeta *metadata.EntityMetadata) {
	for _, prop := range entityMeta.Properties {
		if prop.IsNavigationProp {
			continue
		}

		propDef := h.buildJSONPropertyDefinition(model, &prop)
		entityType[prop.JsonName] = propDef
	}
}

func (h *MetadataHandler) buildJSONPropertyDefinition(model metadataModel, prop *metadata.PropertyMetadata) map[string]interface{} {
	propDef := make(map[string]interface{})

	propDef["$Type"] = h.propertyEdmType(model, prop)

	if value, include := h.propertyNullable(prop); include {
		propDef["$Nullable"] = value
	}

	h.addJSONPropertyFacets(propDef, prop)

	return propDef
}

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

func (h *MetadataHandler) addJSONNavigationProperties(model metadataModel, entityType map[string]interface{}, entityMeta *metadata.EntityMetadata) {
	for _, prop := range entityMeta.Properties {
		if !prop.IsNavigationProp {
			continue
		}

		navProp := h.buildJSONNavigationProperty(model, &prop)
		entityType[prop.JsonName] = navProp
	}
}

func (h *MetadataHandler) buildJSONNavigationProperty(model metadataModel, prop *metadata.PropertyMetadata) map[string]interface{} {
	navProp := make(map[string]interface{})
	navProp["$Kind"] = "NavigationProperty"

	if prop.NavigationIsArray {
		navProp["$Collection"] = true
		navProp["$Type"] = model.qualifiedTypeName(prop.NavigationTarget)
	} else {
		navProp["$Type"] = model.qualifiedTypeName(prop.NavigationTarget)
	}

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

func (h *MetadataHandler) buildJSONEntityContainer(model metadataModel) map[string]interface{} {
	container := map[string]interface{}{
		"$Kind": "EntityContainer",
	}

	for entitySetName, entityMeta := range model.entities {
		if entityMeta.IsSingleton {
			singleton := map[string]interface{}{
				"$Type": model.qualifiedTypeName(entityMeta.EntityName),
			}

			navigationBindings := h.navigationBindings(model, entityMeta)
			if len(navigationBindings) > 0 {
				singleton["$NavigationPropertyBinding"] = h.navigationBindingsMap(navigationBindings)
			}

			container[entityMeta.SingletonName] = singleton
		} else {
			entitySet := map[string]interface{}{
				"$Collection": true,
				"$Type":       model.qualifiedTypeName(entityMeta.EntityName),
			}

			navigationBindings := h.navigationBindings(model, entityMeta)
			if len(navigationBindings) > 0 {
				entitySet["$NavigationPropertyBinding"] = h.navigationBindingsMap(navigationBindings)
			}

			container[entitySetName] = entitySet
		}
	}

	return container
}

func (h *MetadataHandler) navigationBindingsMap(bindings []navigationBinding) map[string]string {
	navigationBindings := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		navigationBindings[binding.path] = binding.target
	}
	return navigationBindings
}
