# @odata.id Field Implementation

This document describes the implementation of the `@odata.id` field in the go-odata library according to the OData v4 specification.

## Overview

The `@odata.id` field is a control information element that specifies the unique identifier (canonical URL) for an entity. Its inclusion in responses depends on the metadata level requested by the client.

## OData v4 Specification

According to the OData v4 JSON Format specification:

- **Full metadata (`odata.metadata=full`)**: The `@odata.id` field is **always** included
- **Minimal metadata (`odata.metadata=minimal`)**: The `@odata.id` field is included only if:
  - Any of the entity's key fields are omitted from the response, OR
  - The entity-id is not identical to the canonical URL of the entity
- **No metadata (`odata.metadata=none`)**: The `@odata.id` field is **never** included

## Format

The `@odata.id` field value follows the canonical URL format:

### Single Key
```
http://host/service/EntitySet(KeyValue)
```

Example:
```json
{
  "@odata.id": "http://localhost:8080/Products(1)"
}
```

### Composite Keys
```
http://host/service/EntitySet(Key1=Value1,Key2=Value2)
```

Example:
```json
{
  "@odata.id": "http://localhost:8080/ProductTranslations(productId=1,languageKey='EN')"
}
```

Note: String values in composite keys are quoted with single quotes.

## Implementation Details

### Library Behavior

This go-odata library automatically includes key properties in responses even when not explicitly selected via `$select`. This is per the OData specification requirement. As a result:

- When using `$select`, key fields are automatically added to the response
- Since key fields are always present, `@odata.id` is typically not needed in minimal metadata mode
- `@odata.id` is primarily useful in full metadata mode where it provides explicit entity identification

### Code Changes

The implementation adds `@odata.id` generation in two main areas:

1. **Collection responses** (`internal/response/odata.go`):
   - `processMapEntity()`: Handles entities that are maps (e.g., from `$select`)
   - `processStructEntityOrdered()`: Handles entities that are structs

2. **Single entity responses** (`internal/handlers/helpers.go`):
   - `buildOrderedEntityResponseWithMetadata()`: Handles both map and struct responses

### Helper Functions

New helper functions were added:

- `buildEntityIDFromValue()`: Builds @odata.id from an entity value
- `buildEntityIDFromMap()`: Builds @odata.id from a map entity
- `allKeyFieldsPresent()`: Checks if all key fields are present in a map
- `allKeyFieldsPresentInOrderedMap()`: Checks if all key fields are present in an OrderedMap

## Examples

### Full Metadata - Collection

**Request:**
```
GET /Products
Accept: application/json;odata.metadata=full
```

**Response:**
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products",
  "value": [
    {
      "@odata.id": "http://localhost:8080/Products(1)",
      "@odata.type": "#ODataService.Product",
      "id": 1,
      "name": "Laptop",
      "price": 999.99
    },
    {
      "@odata.id": "http://localhost:8080/Products(2)",
      "@odata.type": "#ODataService.Product",
      "id": 2,
      "name": "Mouse",
      "price": 29.99
    }
  ]
}
```

### Full Metadata - Single Entity

**Request:**
```
GET /Products(1)
Accept: application/json;odata.metadata=full
```

**Response:**
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products/$entity",
  "@odata.id": "http://localhost:8080/Products(1)",
  "@odata.type": "#ODataService.Product",
  "id": 1,
  "name": "Laptop",
  "price": 999.99
}
```

### Minimal Metadata - Collection

**Request:**
```
GET /Products
Accept: application/json;odata.metadata=minimal
```

**Response:**
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products",
  "value": [
    {
      "id": 1,
      "name": "Laptop",
      "price": 999.99
    },
    {
      "id": 2,
      "name": "Mouse",
      "price": 29.99
    }
  ]
}
```

Note: No `@odata.id` because key fields are present in the response.

### Composite Keys - Full Metadata

**Request:**
```
GET /ProductTranslations
Accept: application/json;odata.metadata=full
```

**Response:**
```json
{
  "@odata.context": "http://localhost:8080/$metadata#ProductTranslations",
  "value": [
    {
      "@odata.id": "http://localhost:8080/ProductTranslations(productId=1,languageKey='EN')",
      "@odata.type": "#ODataService.ProductTranslation",
      "productId": 1,
      "languageKey": "EN",
      "description": "Laptop computer"
    }
  ]
}
```

## Testing

Comprehensive tests have been added in `test/odata_id_test.go`:

- `TestODataIDFieldIntegration`: Tests @odata.id with different metadata levels for both collections and single entities
- `TestODataIDFieldWithCompositeKeys`: Tests @odata.id with composite key entities
- `TestODataIDFieldOrdering`: Tests that @odata.id appears in the correct position in the response

All tests pass and verify:
- @odata.id is present in full metadata mode
- @odata.id is absent in minimal metadata mode (when keys are present)
- @odata.id is absent in none metadata mode
- @odata.id format is correct for both single and composite keys
- Field ordering is maintained (OData control fields come before data properties)

## References

- [OData JSON Format Version 4.0 - OASIS Standard](https://docs.oasis-open.org/odata/odata-json-format/v4.0/odata-json-format-v4.0.html)
- [OData Version 4.0 Part 1: Protocol](https://docs.oasis-open.org/odata/odata/v4.0/os/part1-protocol/odata-v4.0-os-part1-protocol.html)
