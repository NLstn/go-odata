# EDM Package

Internal package providing type-safe implementations of OData Entity Data Model (EDM) primitive types.

## Status

**Internal, standalone implementation** - Does not affect existing go-odata functionality.

Provides EDM type validation, facets support (precision, scale, maxLength), JSON marshaling, and OData literal format for core numeric and string types.

## Basic Usage

```go
import "github.com/nlstn/go-odata/internal/edm"

// Create basic types
str, _ := edm.NewString("Hello")
num, _ := edm.NewInt32(42)

// With facets
maxLen := 50
str, _ := edm.NewString("value", edm.Facets{MaxLength: &maxLen})

// Decimal with precision/scale
precision, scale := 18, 4
dec, _ := edm.NewDecimal(decimal.NewFromFloat(123.45), 
    edm.Facets{Precision: &precision, Scale: &scale})
```

## Struct Tags

```go
type Product struct {
    Name  string          `odata:"type=Edm.String,maxLength=50"`
    Price decimal.Decimal `odata:"type=Edm.Decimal,precision=18,scale=4"`
}
```

Tag options: `type`, `precision`, `scale`, `maxLength`, `unicode`, `srid`, `nullable`

## Implemented Types

| EDM Type    | Go Type         | Notes                     |
|-------------|-----------------|---------------------------|
| Edm.String  | string          | maxLength, unicode facets |
| Edm.Boolean | bool            |                           |
| Edm.Int32   | int32           |                           |
| Edm.Int64   | int64           |                           |
| Edm.Int16   | int16           |                           |
| Edm.Byte    | uint8           |                           |
| Edm.SByte   | int8            |                           |
| Edm.Double  | float64         | INF, -INF, NaN            |
| Edm.Single  | float32         | INF, -INF, NaN            |
| Edm.Decimal | decimal.Decimal | precision, scale facets   |

Additional OData types (Date, TimeOfDay, DateTimeOffset, Duration, GUID, Binary, geospatial) are not yet implemented.

## Important Notes

- **Dependency**: Uses `github.com/shopspring/decimal` for Edm.Decimal
- All types support null values via `IsNull()` method
- Types validate facets constraints on creation
- JSON marshaling follows OData format conventions

## Potential Integration

If maintainers choose to integrate more broadly:
- Could be used in metadata generation for type-safe facet handling
- Could support query parameter type conversion and validation  
- Would require backward-compatible migration strategy
