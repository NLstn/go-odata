# POST Operation Compliance with OData v4 Specification

## Overview

This document describes the POST operation handling in go-odata and verifies compliance with the OData v4.01 specification.

## OData v4 Specification Requirements

According to the [OData v4.01 specification](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_CreateanEntity):

- POST requests are used to create new entities
- POST is only allowed on **entity collections**, not on individual entities
- Successful POST requests return **201 Created** status code
- The response should include:
  - A `Location` header with the URL of the created entity
  - An `OData-Version: 4.0` header
  - The created entity in the response body (unless `Prefer: return=minimal` is specified)

## Implementation Status

### ✅ Correctly Implemented

1. **POST to Collection Endpoints** (`/Products`)
   - Returns `201 Created` with the created entity
   - Includes proper `Location` header
   - Includes `OData-Version: 4.0` header
   - Supports `ETag` generation for entities with ETag properties

2. **POST to Individual Entity Endpoints** (`/Products(1)`)
   - Returns `405 Method Not Allowed`
   - Includes proper error message per OData v4 spec

3. **Validation**
   - Missing required fields return `400 Bad Request`
   - Invalid JSON returns `400 Bad Request`
   - Proper error messages in OData v4 format

4. **Preference Header Support**
   - `Prefer: return=minimal` returns `204 No Content` with `Location` header
   - `Prefer: return=representation` returns `201 Created` with entity body
   - `Preference-Applied` header is included in responses

## Test Coverage

The following test suite verifies POST operation compliance:

### Test File: `test/products_post_integration_test.go`

1. **TestProductsPOST_ToCollectionEndpoint**
   - Verifies POST to `/Products` returns 201 Created
   - Validates response headers and body structure

2. **TestProductsPOST_ToIndividualEntityEndpoint**
   - Verifies POST to `/Products(1)` returns 405 Method Not Allowed

3. **TestProductsPOST_WithTrailingSlash**
   - Verifies POST to `/Products/` works correctly

4. **TestProductsPOST_WithMissingRequiredField**
   - Verifies validation returns 400 Bad Request

5. **TestProductsPOST_WithInvalidJSON**
   - Verifies invalid JSON returns 400 Bad Request

6. **TestProductsPOST_WithETagField**
   - Verifies ETag header generation for entities with ETag properties

7. **TestProductsPOST_AndVerifyCreation**
   - Verifies created entity can be retrieved via GET

8. **TestProductsPOST_MultipleEntities**
   - Verifies multiple sequential POST requests work correctly

9. **TestProductsGET_VerifyCollectionEndpoint**
   - Verifies GET requests still work correctly

10. **TestProductsPOST_WithPreferReturnMinimal**
    - Verifies `Prefer: return=minimal` header support

11. **TestProductsPOST_WithPreferReturnRepresentation**
    - Verifies `Prefer: return=representation` header support

All tests pass successfully.

## Example Usage

### Creating a New Entity

```bash
curl -X POST http://localhost:8080/Products \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "New Laptop",
    "Price": 1299.99,
    "Category": "Electronics",
    "Version": 1
  }'
```

**Response (201 Created):**
```json
{
  "@odata.context": "http://localhost:8080/$metadata#Products/$entity",
  "ID": 6,
  "Name": "New Laptop",
  "Price": 1299.99,
  "Category": "Electronics",
  "Version": 1,
  "Descriptions": null
}
```

**Response Headers:**
```
HTTP/1.1 201 Created
Content-Type: application/json;odata.metadata=minimal
ETag: W/"hash-value"
Location: http://localhost:8080/Products(6)
OData-Version: 4.0
```

### Attempting POST to Individual Entity (Not Allowed)

```bash
curl -X POST http://localhost:8080/Products(1) \
  -H "Content-Type: application/json" \
  -d '{"Name": "Should Fail", "Price": 99.99}'
```

**Response (405 Method Not Allowed):**
```json
{
  "error": {
    "code": "405",
    "message": "Method not allowed",
    "details": [
      {
        "message": "Method POST is not supported for individual entities"
      }
    ]
  }
}
```

### Using Prefer Header for Minimal Response

```bash
curl -X POST http://localhost:8080/Products \
  -H "Content-Type: application/json" \
  -H "Prefer: return=minimal" \
  -d '{
    "Name": "Test Product",
    "Price": 99.99,
    "Category": "Test",
    "Version": 1
  }'
```

**Response (204 No Content):**
```
HTTP/1.1 204 No Content
Location: http://localhost:8080/Products(7)
OData-Version: 4.0
Preference-Applied: return=minimal
```

## Code Implementation

The POST handling is implemented in:

- **`server.go`**: Routes POST requests to collection handlers
- **`internal/handlers/collection.go`**: Implements `HandleCollection` and `handlePostEntity`
- **`internal/handlers/entity_crud.go`**: Rejects POST to individual entities with 405

## Conclusion

The go-odata library correctly implements POST operation handling according to the OData v4 specification. All required functionality is present and tested.
