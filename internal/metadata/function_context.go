package metadata

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

const defaultNamespace = "ODataService"

var (
	timeType = reflect.TypeOf(time.Time{})
)

// FunctionContextFragment builds the metadata fragment for a function return type.
func FunctionContextFragment(returnType reflect.Type, entities map[string]*EntityMetadata, namespace string) string {
	if returnType == nil {
		return ""
	}

	typ := returnType
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	isCollection := false

	switch typ.Kind() {
	case reflect.Slice:
		if typ.Elem().Kind() != reflect.Uint8 {
			isCollection = true
			typ = typ.Elem()
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
		}
	case reflect.Array:
		if typ.Elem().Kind() != reflect.Uint8 {
			isCollection = true
			typ = typ.Elem()
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
		}
	}

	if edmType, ok := primitiveEdmType(typ); ok {
		if isCollection {
			return fmt.Sprintf("Collection(%s)", edmType)
		}
		return edmType
	}

	if entityMeta := entityMetadataByType(typ, entities); entityMeta != nil {
		if isCollection {
			return entityMeta.EntitySetName
		}
		return fmt.Sprintf("%s/$entity", entityMeta.EntitySetName)
	}

	if typ.Kind() == reflect.Struct {
		qualifiedName := buildQualifiedComplexTypeName(typ, namespace)
		if qualifiedName == "" {
			return ""
		}
		if isCollection {
			return fmt.Sprintf("Collection(%s)", qualifiedName)
		}
		return qualifiedName
	}

	if typ.Kind() == reflect.Map || typ.Kind() == reflect.Interface {
		if isCollection {
			return "Collection(Edm.Untyped)"
		}
		return "Edm.Untyped"
	}

	return ""
}

func entityMetadataByType(goType reflect.Type, entities map[string]*EntityMetadata) *EntityMetadata {
	if goType == nil {
		return nil
	}

	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	for _, meta := range entities {
		if meta == nil {
			continue
		}
		entityType := meta.EntityType
		if entityType.Kind() == reflect.Ptr {
			entityType = entityType.Elem()
		}
		if entityType == goType {
			return meta
		}
	}

	return nil
}

func primitiveEdmType(goType reflect.Type) (string, bool) {
	if goType == nil {
		return "", false
	}

	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	if goType == timeType {
		return "Edm.DateTimeOffset", true
	}

	if goType.Kind() == reflect.Slice && goType.Elem().Kind() == reflect.Uint8 {
		return "Edm.Binary", true
	}

	if goType.Kind() == reflect.Array && goType.Elem().Kind() == reflect.Uint8 {
		return "Edm.Binary", true
	}

	if pkgPath := goType.PkgPath(); pkgPath != "" {
		switch pkgPath + "." + goType.Name() {
		case "github.com/google/uuid.UUID":
			return "Edm.Guid", true
		}
	}

	switch goType.Kind() {
	case reflect.String:
		return "Edm.String", true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "Edm.Int32", true
	case reflect.Int64:
		return "Edm.Int64", true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "Edm.Int32", true
	case reflect.Uint64:
		return "Edm.Int64", true
	case reflect.Float32:
		return "Edm.Single", true
	case reflect.Float64:
		return "Edm.Double", true
	case reflect.Bool:
		return "Edm.Boolean", true
	}

	return "", false
}

func buildQualifiedComplexTypeName(goType reflect.Type, namespace string) string {
	if goType == nil {
		return ""
	}

	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	if goType.Name() == "" {
		return ""
	}

	return fmt.Sprintf("%s.%s", normalizeNamespace(namespace), goType.Name())
}

func normalizeNamespace(namespace string) string {
	trimmed := strings.TrimSpace(namespace)
	if trimmed == "" {
		return defaultNamespace
	}
	return trimmed
}
