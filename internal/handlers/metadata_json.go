package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/version"
)

// handleMetadataJSON handles JSON metadata format (CSDL JSON) with version-specific caching
func (h *MetadataHandler) handleMetadataJSON(w http.ResponseWriter, r *http.Request) {
	// Get the negotiated OData version from the request context
	ver := version.GetVersion(r.Context())
	versionKey := ver.String()

	// Lock-free cache lookup (fast path - common case)
	if cached, ok := h.cachedJSON.Load(versionKey); ok {
		cachedBytes, ok := cached.([]byte)
		if !ok {
			// Cache corruption - rebuild
			h.cachedJSON.Delete(versionKey)
			h.cacheSizeJSON.Add(-1)
			h.logger.Warn("Invalid cache entry, rebuilding", "version", versionKey)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(cachedBytes)))

			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(cachedBytes); err != nil {
				h.logger.Error("Error writing JSON metadata response", "error", err)
			}
			return
		}
	}

	// Cache miss - build metadata (slow path)
	model := h.newMetadataModel()
	cached := h.buildMetadataJSON(model, ver)

	// Store in cache using LoadOrStore for thread safety
	// (another goroutine might have built it while we were building)
	actual, loaded := h.cachedJSON.LoadOrStore(versionKey, cached)
	if !loaded {
		// We stored our version, increment counter and check for eviction
		newSize := h.cacheSizeJSON.Add(1)
		if newSize > maxCacheEntries {
			h.evictOldCacheEntriesJSON()
		}
	}

	// Use the actual cached value (ours or the one that was stored by another goroutine)
	cachedBytes, ok := actual.([]byte)
	if !ok {
		// Should never happen, but handle gracefully
		h.logger.Error("Invalid cache entry type", "version", versionKey)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(cachedBytes)))

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(cachedBytes); err != nil {
		h.logger.Error("Error writing JSON metadata response", "error", err)
	}
}

func (h *MetadataHandler) buildMetadataJSON(model metadataModel, ver version.Version) []byte {
	odataService := make(map[string]interface{})
	csdl := map[string]interface{}{
		"$Version":         ver.String(),
		"$EntityContainer": fmt.Sprintf("%s.Container", model.namespace),
	}
	usedVocabularies := model.collectUsedVocabularies()
	if len(usedVocabularies) > 0 {
		vocabularyAliases := metadata.VocabularyAliasMap()
		references := make(map[string]interface{}, len(usedVocabularies))
		for _, ns := range usedVocabularies {
			alias := vocabularyAliases[ns]
			if alias == "" {
				parts := strings.Split(ns, ".")
				alias = parts[len(parts)-1]
			}
			uri := vocabularyURI(ns)
			references[uri] = map[string]interface{}{
				"$Include": []map[string]interface{}{
					{
						"$Namespace": ns,
						"$Alias":     alias,
					},
				},
			}
		}
		csdl["$Reference"] = references
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

	// Add entity-level annotations
	if entityMeta.Annotations != nil {
		for _, annotation := range entityMeta.Annotations.Get() {
			annotationKey := "@" + annotation.QualifiedTerm()
			entityType[annotationKey] = h.annotationJSONValue(annotation.Value)
		}
	}

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

	// Add property-level annotations
	if prop.Annotations != nil {
		for _, annotation := range prop.Annotations.Get() {
			annotationKey := "@" + annotation.QualifiedTerm()
			propDef[annotationKey] = h.annotationJSONValue(annotation.Value)
		}
	}

	return propDef
}

func (h *MetadataHandler) addJSONPropertyFacets(propDef map[string]interface{}, prop *metadata.PropertyMetadata) {
	edmType, ok := propDef["$Type"].(string)
	if !ok {
		return
	}

	// MaxLength is valid for Edm.String and Edm.Binary
	if prop.MaxLength > 0 && (edmType == "Edm.String" || edmType == "Edm.Binary") {
		propDef["$MaxLength"] = prop.MaxLength
	}
	// Precision and Scale are ONLY valid for Edm.Decimal per OData CSDL spec
	if edmType == "Edm.Decimal" {
		if prop.Precision > 0 {
			propDef["$Precision"] = prop.Precision
		}
		if prop.Scale > 0 {
			propDef["$Scale"] = prop.Scale
		}
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

func (h *MetadataHandler) annotationJSONValue(value interface{}) interface{} {
	if collectionValues, ok := annotationCollectionValues(value); ok {
		collection := make([]interface{}, 0, len(collectionValues))
		for _, item := range collectionValues {
			collection = append(collection, h.annotationJSONValue(item))
		}
		return map[string]interface{}{
			"$Collection": collection,
		}
	}

	if recordValues, ok := annotationRecordValues(value); ok {
		record := make(map[string]interface{}, len(recordValues))
		for _, key := range sortedAnnotationKeys(recordValues) {
			record[key] = h.annotationJSONValue(recordValues[key])
		}
		return map[string]interface{}{
			"$Record": record,
		}
	}

	return value
}

func (h *MetadataHandler) buildJSONEntityContainer(model metadataModel) map[string]interface{} {
	container := map[string]interface{}{
		"$Kind": "EntityContainer",
	}

	if model.containerAnnotations != nil {
		for _, annotation := range model.containerAnnotations.Get() {
			annotationKey := "@" + annotation.QualifiedTerm()
			container[annotationKey] = h.annotationJSONValue(annotation.Value)
		}
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

			if entityMeta.SingletonAnnotations != nil {
				for _, annotation := range entityMeta.SingletonAnnotations.Get() {
					annotationKey := "@" + annotation.QualifiedTerm()
					singleton[annotationKey] = h.annotationJSONValue(annotation.Value)
				}
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

			if entityMeta.EntitySetAnnotations != nil {
				for _, annotation := range entityMeta.EntitySetAnnotations.Get() {
					annotationKey := "@" + annotation.QualifiedTerm()
					entitySet[annotationKey] = h.annotationJSONValue(annotation.Value)
				}
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
