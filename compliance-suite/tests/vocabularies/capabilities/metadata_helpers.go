package capabilities

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/compliance-suite/framework"
)

type capabilitiesMetadata struct {
	insertRestricted []entitySetInfo
	updateRestricted []entitySetInfo
	deleteRestricted []entitySetInfo
}

type entitySetInfo struct {
	name     string
	keyProps []keyProperty
}

type keyProperty struct {
	name string
	typ  string
}

type metadataDocument struct {
	DataServices dataServices `xml:"DataServices"`
}

type dataServices struct {
	Schemas []schema `xml:"Schema"`
}

type schema struct {
	Namespace       string          `xml:"Namespace,attr"`
	EntityTypes     []entityType    `xml:"EntityType"`
	EntityContainer entityContainer `xml:"EntityContainer"`
	Annotations     []annotations   `xml:"Annotations"`
}

type entityType struct {
	Name       string     `xml:"Name,attr"`
	Key        entityKey  `xml:"Key"`
	Properties []property `xml:"Property"`
}

type entityKey struct {
	PropertyRefs []propertyRef `xml:"PropertyRef"`
}

type propertyRef struct {
	Name string `xml:"Name,attr"`
}

type property struct {
	Name string `xml:"Name,attr"`
	Type string `xml:"Type,attr"`
}

type entityContainer struct {
	EntitySets []entitySet `xml:"EntitySet"`
}

type entitySet struct {
	Name       string `xml:"Name,attr"`
	EntityType string `xml:"EntityType,attr"`
}

type annotations struct {
	Target      string       `xml:"Target,attr"`
	Annotations []annotation `xml:"Annotation"`
}

type annotation struct {
	Term   string  `xml:"Term,attr"`
	Record *record `xml:"Record"`
}

type record struct {
	PropertyValues []propertyValue `xml:"PropertyValue"`
}

type propertyValue struct {
	Property string `xml:"Property,attr"`
	Bool     *bool  `xml:"Bool,attr"`
}

type entityTypeInfo struct {
	keys         []string
	propertyType map[string]string
}

func parseCapabilitiesMetadata(metadataXML []byte) (capabilitiesMetadata, error) {
	var doc metadataDocument
	if err := xml.Unmarshal(metadataXML, &doc); err != nil {
		return capabilitiesMetadata{}, fmt.Errorf("failed to parse metadata XML: %w", err)
	}

	entityTypeMap := make(map[string]entityTypeInfo)
	entitySetTypes := make(map[string]string)
	restrictions := make(map[string]map[string]bool)

	for _, schema := range doc.DataServices.Schemas {
		for _, entityType := range schema.EntityTypes {
			fullName := fmt.Sprintf("%s.%s", schema.Namespace, entityType.Name)
			info := entityTypeInfo{
				keys:         make([]string, 0, len(entityType.Key.PropertyRefs)),
				propertyType: make(map[string]string),
			}
			for _, prop := range entityType.Properties {
				info.propertyType[prop.Name] = prop.Type
			}
			for _, ref := range entityType.Key.PropertyRefs {
				info.keys = append(info.keys, ref.Name)
			}
			entityTypeMap[fullName] = info
		}

		for _, set := range schema.EntityContainer.EntitySets {
			entitySetTypes[set.Name] = set.EntityType
		}

		for _, block := range schema.Annotations {
			prefix := fmt.Sprintf("%s.Container/", schema.Namespace)
			if !strings.HasPrefix(block.Target, prefix) {
				continue
			}
			setName := strings.TrimPrefix(block.Target, prefix)
			for _, ann := range block.Annotations {
				if ann.Record == nil {
					continue
				}
				values := make(map[string]bool)
				for _, prop := range ann.Record.PropertyValues {
					if prop.Bool == nil {
						continue
					}
					values[prop.Property] = *prop.Bool
				}
				if len(values) == 0 {
					continue
				}
				if restrictions[setName] == nil {
					restrictions[setName] = make(map[string]bool)
				}
				for key, value := range values {
					restrictions[setName][fmt.Sprintf("%s:%s", ann.Term, key)] = value
				}
			}
		}
	}

	capability := capabilitiesMetadata{}
	for setName, record := range restrictions {
		setInfo, err := buildEntitySetInfo(setName, entitySetTypes, entityTypeMap)
		if err != nil {
			return capabilitiesMetadata{}, err
		}
		if record["Org.OData.Capabilities.V1.InsertRestrictions:Insertable"] == false {
			capability.insertRestricted = append(capability.insertRestricted, setInfo)
		}
		if record["Org.OData.Capabilities.V1.UpdateRestrictions:Updatable"] == false {
			capability.updateRestricted = append(capability.updateRestricted, setInfo)
		}
		if record["Org.OData.Capabilities.V1.DeleteRestrictions:Deletable"] == false {
			capability.deleteRestricted = append(capability.deleteRestricted, setInfo)
		}
	}

	return capability, nil
}

func buildEntitySetInfo(setName string, entitySetTypes map[string]string, entityTypeMap map[string]entityTypeInfo) (entitySetInfo, error) {
	entityType, ok := entitySetTypes[setName]
	if !ok {
		return entitySetInfo{}, fmt.Errorf("entity set %q missing entity type", setName)
	}
	info, ok := entityTypeMap[entityType]
	if !ok {
		return entitySetInfo{}, fmt.Errorf("entity type %q not found for set %q", entityType, setName)
	}

	keyProps := make([]keyProperty, 0, len(info.keys))
	for _, key := range info.keys {
		keyProps = append(keyProps, keyProperty{name: key, typ: info.propertyType[key]})
	}

	return entitySetInfo{name: setName, keyProps: keyProps}, nil
}

func fetchMetadata(ctx *framework.TestContext) ([]byte, error) {
	resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func fetchFirstEntity(ctx *framework.TestContext, entitySetName string) (map[string]interface{}, error) {
	resp, err := ctx.GET(fmt.Sprintf("/%s?$top=1", entitySetName))
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}

	var payload map[string]interface{}
	if err := ctx.GetJSON(resp, &payload); err != nil {
		return nil, err
	}
	items, ok := payload["value"].([]interface{})
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("no entities returned for %s", entitySetName)
	}
	entity, ok := items[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected entity shape for %s", entitySetName)
	}
	return entity, nil
}

func buildEntityKey(entity map[string]interface{}, keyProps []keyProperty) (string, error) {
	if len(keyProps) == 0 {
		return "", fmt.Errorf("missing key properties")
	}

	parts := make([]string, 0, len(keyProps))
	for _, key := range keyProps {
		value, ok := entity[key.name]
		if !ok {
			return "", fmt.Errorf("missing key %s in entity", key.name)
		}
		formatted, err := formatKeyValue(value, key.typ)
		if err != nil {
			return "", err
		}
		if len(keyProps) == 1 {
			parts = append(parts, formatted)
		} else {
			parts = append(parts, fmt.Sprintf("%s=%s", key.name, formatted))
		}
	}

	if len(keyProps) == 1 {
		return fmt.Sprintf("(%s)", parts[0]), nil
	}
	return fmt.Sprintf("(%s)", strings.Join(parts, ",")), nil
}

func formatKeyValue(value interface{}, edmType string) (string, error) {
	switch edmType {
	case "Edm.String":
		strValue, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("expected string key value, got %T", value)
		}
		escaped := strings.ReplaceAll(strValue, "'", "''")
		return fmt.Sprintf("'%s'", escaped), nil
	case "Edm.Guid":
		return fmt.Sprintf("%v", value), nil
	case "Edm.Int16", "Edm.Int32", "Edm.Int64", "Edm.Byte", "Edm.SByte":
		return fmt.Sprintf("%v", value), nil
	case "Edm.Boolean":
		if boolValue, ok := value.(bool); ok {
			if boolValue {
				return "true", nil
			}
			return "false", nil
		}
		return "", fmt.Errorf("expected boolean key value, got %T", value)
	default:
		return fmt.Sprintf("%v", value), nil
	}
}
