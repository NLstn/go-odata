package odata

import "github.com/nlstn/go-odata/internal/metadata"

// PrimitiveType identifies an EDM primitive type.
type PrimitiveType = metadata.PrimitiveType

const (
	EdmBinary                   = metadata.PrimitiveTypeBinary
	EdmBoolean                  = metadata.PrimitiveTypeBoolean
	EdmByte                     = metadata.PrimitiveTypeByte
	EdmDate                     = metadata.PrimitiveTypeDate
	EdmDateTimeOffset           = metadata.PrimitiveTypeDateTimeOffset
	EdmDecimal                  = metadata.PrimitiveTypeDecimal
	EdmDouble                   = metadata.PrimitiveTypeDouble
	EdmDuration                 = metadata.PrimitiveTypeDuration
	EdmGeography                = metadata.PrimitiveTypeGeography
	EdmGeographyCollection      = metadata.PrimitiveTypeGeographyCollection
	EdmGeographyLineString      = metadata.PrimitiveTypeGeographyLineString
	EdmGeographyMultiLineString = metadata.PrimitiveTypeGeographyMultiLineString
	EdmGeographyMultiPoint      = metadata.PrimitiveTypeGeographyMultiPoint
	EdmGeographyMultiPolygon    = metadata.PrimitiveTypeGeographyMultiPolygon
	EdmGeographyPoint           = metadata.PrimitiveTypeGeographyPoint
	EdmGeographyPolygon         = metadata.PrimitiveTypeGeographyPolygon
	EdmGeometry                 = metadata.PrimitiveTypeGeometry
	EdmGeometryCollection       = metadata.PrimitiveTypeGeometryCollection
	EdmGeometryLineString       = metadata.PrimitiveTypeGeometryLineString
	EdmGeometryMultiLineString  = metadata.PrimitiveTypeGeometryMultiLineString
	EdmGeometryMultiPoint       = metadata.PrimitiveTypeGeometryMultiPoint
	EdmGeometryMultiPolygon     = metadata.PrimitiveTypeGeometryMultiPolygon
	EdmGeometryPoint            = metadata.PrimitiveTypeGeometryPoint
	EdmGeometryPolygon          = metadata.PrimitiveTypeGeometryPolygon
	EdmGuid                     = metadata.PrimitiveTypeGuid
	EdmInt16                    = metadata.PrimitiveTypeInt16
	EdmInt32                    = metadata.PrimitiveTypeInt32
	EdmInt64                    = metadata.PrimitiveTypeInt64
	EdmSByte                    = metadata.PrimitiveTypeSByte
	EdmSingle                   = metadata.PrimitiveTypeSingle
	EdmStream                   = metadata.PrimitiveTypeStream
	EdmString                   = metadata.PrimitiveTypeString
	EdmTimeOfDay                = metadata.PrimitiveTypeTimeOfDay
	EdmUntyped                  = metadata.PrimitiveTypeUntyped
)

// ParsePrimitiveType validates an EDM primitive type name.
func ParsePrimitiveType(name string) (PrimitiveType, error) {
	return metadata.ParsePrimitiveType(name)
}
