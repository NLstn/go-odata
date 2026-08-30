package metadata

import (
	"fmt"
	"reflect"
)

// PrimitiveType identifies an EDM primitive type.
type PrimitiveType string

const (
	PrimitiveTypeBinary                   PrimitiveType = "Edm.Binary"
	PrimitiveTypeBoolean                  PrimitiveType = "Edm.Boolean"
	PrimitiveTypeByte                     PrimitiveType = "Edm.Byte"
	PrimitiveTypeDate                     PrimitiveType = "Edm.Date"
	PrimitiveTypeDateTimeOffset           PrimitiveType = "Edm.DateTimeOffset"
	PrimitiveTypeDecimal                  PrimitiveType = "Edm.Decimal"
	PrimitiveTypeDouble                   PrimitiveType = "Edm.Double"
	PrimitiveTypeDuration                 PrimitiveType = "Edm.Duration"
	PrimitiveTypeGeography                PrimitiveType = "Edm.Geography"
	PrimitiveTypeGeographyCollection      PrimitiveType = "Edm.GeographyCollection"
	PrimitiveTypeGeographyLineString      PrimitiveType = "Edm.GeographyLineString"
	PrimitiveTypeGeographyMultiLineString PrimitiveType = "Edm.GeographyMultiLineString"
	PrimitiveTypeGeographyMultiPoint      PrimitiveType = "Edm.GeographyMultiPoint"
	PrimitiveTypeGeographyMultiPolygon    PrimitiveType = "Edm.GeographyMultiPolygon"
	PrimitiveTypeGeographyPoint           PrimitiveType = "Edm.GeographyPoint"
	PrimitiveTypeGeographyPolygon         PrimitiveType = "Edm.GeographyPolygon"
	PrimitiveTypeGeometry                 PrimitiveType = "Edm.Geometry"
	PrimitiveTypeGeometryCollection       PrimitiveType = "Edm.GeometryCollection"
	PrimitiveTypeGeometryLineString       PrimitiveType = "Edm.GeometryLineString"
	PrimitiveTypeGeometryMultiLineString  PrimitiveType = "Edm.GeometryMultiLineString"
	PrimitiveTypeGeometryMultiPoint       PrimitiveType = "Edm.GeometryMultiPoint"
	PrimitiveTypeGeometryMultiPolygon     PrimitiveType = "Edm.GeometryMultiPolygon"
	PrimitiveTypeGeometryPoint            PrimitiveType = "Edm.GeometryPoint"
	PrimitiveTypeGeometryPolygon          PrimitiveType = "Edm.GeometryPolygon"
	PrimitiveTypeGuid                     PrimitiveType = "Edm.Guid"
	PrimitiveTypeInt16                    PrimitiveType = "Edm.Int16"
	PrimitiveTypeInt32                    PrimitiveType = "Edm.Int32"
	PrimitiveTypeInt64                    PrimitiveType = "Edm.Int64"
	PrimitiveTypeSByte                    PrimitiveType = "Edm.SByte"
	PrimitiveTypeSingle                   PrimitiveType = "Edm.Single"
	PrimitiveTypeStream                   PrimitiveType = "Edm.Stream"
	PrimitiveTypeString                   PrimitiveType = "Edm.String"
	PrimitiveTypeTimeOfDay                PrimitiveType = "Edm.TimeOfDay"
	PrimitiveTypeUntyped                  PrimitiveType = "Edm.Untyped"
)

var validPrimitiveTypes = map[PrimitiveType]struct{}{
	PrimitiveTypeBinary:                   {},
	PrimitiveTypeBoolean:                  {},
	PrimitiveTypeByte:                     {},
	PrimitiveTypeDate:                     {},
	PrimitiveTypeDateTimeOffset:           {},
	PrimitiveTypeDecimal:                  {},
	PrimitiveTypeDouble:                   {},
	PrimitiveTypeDuration:                 {},
	PrimitiveTypeGeography:                {},
	PrimitiveTypeGeographyCollection:      {},
	PrimitiveTypeGeographyLineString:      {},
	PrimitiveTypeGeographyMultiLineString: {},
	PrimitiveTypeGeographyMultiPoint:      {},
	PrimitiveTypeGeographyMultiPolygon:    {},
	PrimitiveTypeGeographyPoint:           {},
	PrimitiveTypeGeographyPolygon:         {},
	PrimitiveTypeGeometry:                 {},
	PrimitiveTypeGeometryCollection:       {},
	PrimitiveTypeGeometryLineString:       {},
	PrimitiveTypeGeometryMultiLineString:  {},
	PrimitiveTypeGeometryMultiPoint:       {},
	PrimitiveTypeGeometryMultiPolygon:     {},
	PrimitiveTypeGeometryPoint:            {},
	PrimitiveTypeGeometryPolygon:          {},
	PrimitiveTypeGuid:                     {},
	PrimitiveTypeInt16:                    {},
	PrimitiveTypeInt32:                    {},
	PrimitiveTypeInt64:                    {},
	PrimitiveTypeSByte:                    {},
	PrimitiveTypeSingle:                   {},
	PrimitiveTypeStream:                   {},
	PrimitiveTypeString:                   {},
	PrimitiveTypeTimeOfDay:                {},
	PrimitiveTypeUntyped:                  {},
}

// ParsePrimitiveType validates and returns an EDM primitive type name.
func ParsePrimitiveType(name string) (PrimitiveType, error) {
	primitiveType := PrimitiveType(name)
	if _, ok := validPrimitiveTypes[primitiveType]; !ok {
		return "", fmt.Errorf("unsupported EDM primitive type %q", name)
	}
	return primitiveType, nil
}

// EffectivePrimitiveType returns the declared EDM type, falling back to the Go
// type for metadata created by legacy internal callers.
func (p *PropertyMetadata) EffectivePrimitiveType() (PrimitiveType, bool) {
	if p == nil {
		return "", false
	}
	if p.EdmType != "" {
		return p.EdmType, true
	}
	primitiveType, err := PrimitiveTypeFromGoType(p.Type)
	return primitiveType, err == nil
}

// PrimitiveTypeFromGoType infers the EDM primitive type represented by a Go type.
func PrimitiveTypeFromGoType(goType reflect.Type) (PrimitiveType, error) {
	if goType == nil {
		return "", fmt.Errorf("cannot infer EDM primitive type from nil Go type")
	}
	for goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	switch goType.String() {
	case "time.Time":
		return PrimitiveTypeDateTimeOffset, nil
	case "uuid.UUID", "github.com/google/uuid.UUID":
		return PrimitiveTypeGuid, nil
	case "decimal.Decimal", "github.com/shopspring/decimal.Decimal":
		return PrimitiveTypeDecimal, nil
	case "json.RawMessage", "encoding/json.RawMessage":
		return PrimitiveTypeUntyped, nil
	}

	if goType.Kind() == reflect.Interface {
		return PrimitiveTypeUntyped, nil
	}
	if (goType.Kind() == reflect.Slice || goType.Kind() == reflect.Array) && goType.Elem().Kind() == reflect.Uint8 {
		return PrimitiveTypeBinary, nil
	}

	switch goType.Kind() {
	case reflect.String:
		return PrimitiveTypeString, nil
	case reflect.Bool:
		return PrimitiveTypeBoolean, nil
	case reflect.Int8:
		return PrimitiveTypeSByte, nil
	case reflect.Int16:
		return PrimitiveTypeInt16, nil
	case reflect.Int, reflect.Int32:
		return PrimitiveTypeInt32, nil
	case reflect.Int64:
		return PrimitiveTypeInt64, nil
	case reflect.Uint8:
		return PrimitiveTypeByte, nil
	case reflect.Uint16:
		return PrimitiveTypeInt32, nil
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		return PrimitiveTypeInt64, nil
	case reflect.Float32:
		return PrimitiveTypeSingle, nil
	case reflect.Float64:
		return PrimitiveTypeDouble, nil
	default:
		return "", fmt.Errorf("unsupported Go type %s", goType)
	}
}
