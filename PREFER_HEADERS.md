# OData Prefer Header Support

This document describes the implementation of the `Prefer` and `Preference-Applied` headers according to the OData v4 specification.

## Overview

The OData `Prefer` header allows clients to specify preferences for how the server should handle the request. This implementation supports the `return` preference with two values:

- `return=representation` - Requests the service to return the created/updated entity in the response
- `return=minimal` - Requests the service to return minimal or no content in the response

When a preference is honored, the service returns a `Preference-Applied` header indicating which preference was applied.

## Default Behavior

### POST Operations
- **Default**: Returns the created entity with status `201 Created`
- **With `Prefer: return=minimal`**: Returns no content with status `204 No Content`
- **With `Prefer: return=representation`**: Returns the created entity with status `201 Created` (explicit)

### PATCH/PUT Operations
- **Default**: Returns no content with status `204 No Content`
- **With `Prefer: return=representation`**: Returns the updated entity with status `200 OK`
- **With `Prefer: return=minimal`**: Returns no content with status `204 No Content` (explicit)

## Request Examples

### POST with return=minimal

```http
POST /Products HTTP/1.1
Content-Type: application/json
Prefer: return=minimal

{
  "name": "Laptop",
  "price": 999.99
}
```

**Response:**
```http
HTTP/1.1 204 No Content
OData-Version: 4.0
Location: http://localhost:8080/Products(1)
Preference-Applied: return=minimal
```

### POST with return=representation (or default)

```http
POST /Products HTTP/1.1
Content-Type: application/json
Prefer: return=representation

{
  "name": "Laptop",
  "price": 999.99
}
```

**Response:**
```http
HTTP/1.1 201 Created
OData-Version: 4.0
Location: http://localhost:8080/Products(1)
Preference-Applied: return=representation
Content-Type: application/json;odata.metadata=minimal

{
  "@odata.context": "http://localhost:8080/$metadata#Products/$entity",
  "id": 1,
  "name": "Laptop",
  "price": 999.99
}
```

### PATCH with return=representation

```http
PATCH /Products(1) HTTP/1.1
Content-Type: application/json
Prefer: return=representation

{
  "price": 899.99
}
```

**Response:**
```http
HTTP/1.1 200 OK
OData-Version: 4.0
Preference-Applied: return=representation
Content-Type: application/json;odata.metadata=minimal

{
  "@odata.context": "http://localhost:8080/$metadata#Products/$entity",
  "id": 1,
  "name": "Laptop",
  "price": 899.99
}
```

### PATCH without Prefer header (default)

```http
PATCH /Products(1) HTTP/1.1
Content-Type: application/json

{
  "price": 899.99
}
```

**Response:**
```http
HTTP/1.1 204 No Content
OData-Version: 4.0
```

## Implementation Details

### New Package: `internal/preference`

The `preference` package provides functionality to parse and handle the `Prefer` header:

- **`ParsePrefer(r *http.Request) *Preference`** - Parses the Prefer header from a request
- **`ShouldReturnContent(isPostOperation bool) bool`** - Determines if content should be returned
- **`GetPreferenceApplied() string`** - Returns the value for the Preference-Applied header

### Modified Handlers

The following handlers in `internal/handlers/entity.go` were updated to support preferences:

1. **`handlePostEntity`** - Supports `return=minimal` for POST operations
2. **`handlePatchEntity`** - Supports `return=representation` for PATCH operations
3. **`handlePutEntity`** - Supports `return=representation` for PUT operations

## Features

- ✅ Case-insensitive parsing of Prefer header values
- ✅ Support for comma-separated multiple preferences
- ✅ Automatic `Preference-Applied` header when preference is honored
- ✅ Full OData v4 compliance for return preferences
- ✅ Backward compatible (default behavior unchanged)
- ✅ Comprehensive test coverage

## Testing

The implementation includes extensive test coverage:

- Unit tests for preference parsing (14 tests in `internal/preference/preference_test.go`)
- Integration tests for all scenarios (13 tests in `prefer_test.go`)
- Tests for case-insensitivity and multiple preferences
- Tests for default behavior and explicit preferences
- All existing tests continue to pass

Run tests with:
```bash
go test ./...
```

Run only preference tests:
```bash
go test -v -run ".*Prefer.*"
```

## Code Quality

The implementation passes all quality checks:

- ✅ All tests pass
- ✅ `golangci-lint` reports 0 issues
- ✅ Code follows Go best practices
- ✅ Properly formatted with `gofmt`

## References

- [OData Version 4.0 Part 1: Protocol](https://docs.oasis-open.org/odata/odata/v4.0/os/part1-protocol/odata-v4.0-os-part1-protocol.html#_Toc372793752)
- [RFC 7240 - Prefer Header for HTTP](https://tools.ietf.org/html/rfc7240)
