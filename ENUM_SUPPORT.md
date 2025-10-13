# Enum Support in go-odata

This document describes the enum support implementation in the go-odata library.

## Overview

The library now supports OData enum types with flags, including the `has` function for checking flag combinations using bitwise operations.

## Features

### 1. Enum Type Definition
Define enum types in your Go code:

```go
type ProductStatus int

const (
    ProductStatusNone ProductStatus = 0
    ProductStatusInStock ProductStatus = 1
    ProductStatusOnSale ProductStatus = 2
    ProductStatusDiscontinued ProductStatus = 4
    ProductStatusFeatured ProductStatus = 8
)
```

### 2. Entity Field with Enum
Use the enum type in your entity and mark it with odata tags:

```go
type Product struct {
    ID     uint          `json:"ID" gorm:"primaryKey" odata:"key"`
    Name   string        `json:"Name" gorm:"not null"`
    Status ProductStatus `json:"Status" gorm:"not null" odata:"enum=ProductStatus,flags"`
}
```

### 3. OData Tags
- `enum=<TypeName>`: Marks the field as an enum type and specifies the enum type name
- `flags`: Indicates that the enum supports flag combinations (bitwise operations)

### 4. Metadata Generation

#### XML Metadata
The library generates proper OData v4 XML metadata with EnumType definitions:

```xml
<EnumType Name="ProductStatus" UnderlyingType="Edm.Int32" IsFlags="true">
    <Member Name="None" Value="0" />
    <Member Name="InStock" Value="1" />
    <Member Name="OnSale" Value="2" />
    <Member Name="Discontinued" Value="4" />
    <Member Name="Featured" Value="8" />
</EnumType>
```

#### JSON Metadata (CSDL JSON)
```json
{
    "ProductStatus": {
        "$Kind": "EnumType",
        "$UnderlyingType": "Edm.Int32",
        "$IsFlags": true,
        "None": 0,
        "InStock": 1,
        "OnSale": 2,
        "Discontinued": 4,
        "Featured": 8
    }
}
```

### 5. The `has` Function

The `has` function allows checking if a specific flag is set in an enum field using bitwise AND operations.

#### Syntax
```
has(<property>, <value>)
```

#### Examples

**Filter products that are in stock:**
```
GET /Products?$filter=has(Status, 1)
```

**Filter products that are on sale:**
```
GET /Products?$filter=has(Status, 2)
```

**Filter products that are both in stock AND on sale:**
```
GET /Products?$filter=has(Status, 1) and has(Status, 2)
```

**Filter products that are either on sale OR discontinued:**
```
GET /Products?$filter=has(Status, 2) or has(Status, 4)
```

**Combine with other filters:**
```
GET /Products?$filter=has(Status, 1) and Price gt 100
```

**Use with NOT operator:**
```
GET /Products?$filter=not has(Status, 8)
```

### 6. SQL Generation

The `has` function is translated to SQL using bitwise AND:

```sql
-- has(Status, 1)
WHERE (status & 1) = 1

-- has(Status, 1) and has(Status, 2)  
WHERE ((status & 1) = 1) AND ((status & 2) = 2)

-- not has(Status, 8)
WHERE NOT ((status & 8) = 8)
```

## Implementation Details

### Internal Components Modified

1. **internal/metadata/analyzer.go**
   - Added `IsEnum`, `EnumTypeName`, and `IsFlags` fields to `PropertyMetadata`
   - Updated `processODataTagPart` to handle `enum=<TypeName>` and `flags` tags

2. **internal/handlers/metadata.go**
   - Added `buildEnumTypes()` and `buildEnumType()` for XML metadata generation
   - Added `addJSONEnumTypes()` and `buildJSONEnumType()` for JSON metadata generation
   - Updated property builders to use enum types when appropriate

3. **internal/query/parser.go**
   - Added `OpHas` constant for the has operator

4. **internal/query/ast_parser.go**
   - Added `has` to the list of two-argument functions
   - The existing two-arg function handler properly processes `has` calls

5. **internal/query/applier.go**
   - Added SQL generation for `OpHas` using bitwise AND: `(column & value) = value`
   - Added handling in both `buildComparisonCondition` (with metadata) and `buildSimpleOperatorCondition` (without metadata)

### Testing

Comprehensive tests were added in:
- `test/enum_integration_test.go`: Integration tests for enum support and has function
- `internal/query/has_function_test.go`: Unit tests for has function parsing and SQL generation

All tests include:
- Basic has function filtering
- Combined flag checks with AND/OR operators
- Combining has with other filter operators
- NOT operator with has function
- Metadata validation (both XML and JSON)
- Enum value validation in responses

## Example Usage

See `cmd/devserver/product.go` for a complete example with:
- ProductStatus enum definition
- Product entity with Status field
- Sample data with various flag combinations
- Running devserver to test queries

## Database Seeding Example

```go
products := []Product{
    {ID: 1, Name: "Laptop", Status: ProductStatusInStock | ProductStatusFeatured},  // 9
    {ID: 2, Name: "Mouse", Status: ProductStatusInStock | ProductStatusOnSale},     // 3
    {ID: 3, Name: "Keyboard", Status: ProductStatusInStock},                         // 1
    {ID: 4, Name: "Chair", Status: ProductStatusDiscontinued},                       // 4
}
```

## Code Quality

All changes have been verified with:
- Go unit tests (100% pass rate)
- Integration tests (100% pass rate)
- golangci-lint (0 issues)
