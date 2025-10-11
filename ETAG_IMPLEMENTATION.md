# ETag Implementation Summary

## Overview

This document describes the implementation of ETag (Entity Tag) support in the go-odata library, which provides optimistic concurrency control for entity modifications.

## What are ETags?

ETags are a mechanism defined in HTTP/1.1 (RFC 7232) for optimistic concurrency control. They help prevent "lost update" problems where concurrent modifications could overwrite each other.

### How ETags Work

1. **Server generates ETag**: When a client retrieves an entity via GET, the server includes an `ETag` header
2. **Client stores ETag**: The client stores the ETag value along with the entity data
3. **Client sends If-Match**: When updating the entity, the client includes the stored ETag in an `If-Match` header
4. **Server validates**: The server compares the `If-Match` value with the current ETag
   - If they match: Update proceeds (200 OK or 204 No Content)
   - If they don't match: Update rejected (412 Precondition Failed)

## Implementation Details

### Architecture

The implementation consists of three main components:

1. **Metadata Support** (`internal/metadata/analyzer.go`)
   - Added `IsETag` field to `PropertyMetadata`
   - Added `ETagProperty` field to `EntityMetadata`
   - Added support for `odata:"etag"` tag in struct field analysis

2. **ETag Generation** (`internal/etag/etag.go`)
   - SHA-256 hash-based ETag generation
   - Support for multiple field types (int, string, time.Time)
   - Weak ETag format: `W/"hash"`
   - ETag parsing and matching utilities

3. **Handler Integration** (`internal/handlers/entity.go`)
   - GET: Returns `ETag` header
   - POST: Returns `ETag` header when `Prefer: return=representation`
   - PATCH/PUT/DELETE: Validates `If-Match` header before modifying

### ETag Generation Algorithm

1. Extract the value from the field marked with `odata:"etag"`
2. Convert the value to a string representation
3. Generate SHA-256 hash of the string
4. Encode hash as hexadecimal
5. Format as weak ETag: `W/"<hash>"`

Example:
- Field value: `1` (integer)
- String representation: `"1"`
- SHA-256: `6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b`
- ETag: `W/"6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"`

### Supported Field Types

The following field types can be used for ETag generation:

- **Integer types**: `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- **String type**: `string`
- **Time type**: `time.Time` (converted to Unix timestamp)
- **Other types**: Any type that can be converted to string via `fmt.Sprintf("%v", ...)`

### If-Match Header Support

The implementation supports three types of `If-Match` values:

1. **Specific ETag**: `If-Match: W/"abc123..."` - Matches only if ETag is exactly this value
2. **Wildcard**: `If-Match: *` - Matches if the entity exists (any ETag value)
3. **No header**: Update proceeds without validation (backward compatible)

### Error Responses

When an If-Match validation fails, the server returns:

```json
HTTP/1.1 412 Precondition Failed
Content-Type: application/json

{
  "error": {
    "code": "412",
    "message": "Precondition failed",
    "details": [{
      "message": "The entity has been modified. Please refresh and try again."
    }]
  }
}
```

## Usage

### Defining an ETag Property

Add the `odata:"etag"` tag to any field in your entity struct:

```go
type Product struct {
    ID       uint      `json:"ID" gorm:"primaryKey" odata:"key"`
    Name     string    `json:"Name"`
    Price    float64   `json:"Price"`
    Version  int       `json:"Version" odata:"etag"` // Version-based concurrency
}

// Or using a timestamp
type Document struct {
    ID           int       `json:"ID" odata:"key"`
    Content      string    `json:"Content"`
    LastModified time.Time `json:"LastModified" odata:"etag"` // Timestamp-based
}
```

### Client Usage Pattern

```go
// 1. GET to retrieve entity and ETag
GET /Products(1)
Response:
  ETag: W/"6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
  Body: {"ID": 1, "Name": "Laptop", "Version": 1}

// 2. PATCH with If-Match header
PATCH /Products(1)
If-Match: W/"6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"
Content-Type: application/json
Body: {"Price": 899.99}

// 3. Handle success (204 No Content) or failure (412 Precondition Failed)
```

## Testing

### Unit Tests

Located in `internal/etag/etag_test.go`:

- ETag generation consistency
- ETag generation for different field types
- ETag parsing (weak and strong formats)
- ETag matching logic
- Wildcard matching

### Integration Tests

Located in `test/etag_test.go`:

- GET returns ETag header
- POST returns ETag header with `Prefer: return=representation`
- PATCH with matching If-Match succeeds
- PATCH with non-matching If-Match fails (412)
- PATCH without If-Match succeeds (backward compatible)
- PATCH with wildcard If-Match succeeds
- PUT with matching/non-matching If-Match
- DELETE with matching/non-matching If-Match
- Different field types produce different ETags
- Same field values produce same ETags
- Operations with and without ETag properties

### Manual Testing

Run the development server and use the example script:

```bash
cd cmd/devserver
go run . &

cd ../examples
./etag_example.sh
```

## Design Decisions

### Why Weak ETags?

We use weak ETags (`W/"..."`) because:
1. They're suitable for semantic equivalence rather than byte-for-byte equivalence
2. They're appropriate for database-backed content where minor changes (e.g., whitespace) don't affect semantic meaning
3. They're the recommended format for application-level versioning

### Why SHA-256?

SHA-256 provides:
1. Low collision probability (virtually impossible for practical purposes)
2. Consistent hash length (64 hex characters)
3. Standard library support in Go
4. Good performance characteristics

### Why Optional?

ETag support is optional because:
1. Not all applications need optimistic concurrency control
2. Some entities may use other mechanisms (database-level locking)
3. Maintains backward compatibility with existing code
4. Follows the principle of least surprise

## Performance Considerations

### ETag Generation

- SHA-256 hashing is fast (~100 ns per operation)
- Only performed when ETag property is configured
- Only generated for individual entity requests (not collections)

### If-Match Validation

- Simple string comparison after parsing
- Only performed when If-Match header is present
- Happens after entity fetch (already in memory)

### Database Impact

- No additional database queries required
- ETag field is part of normal entity retrieval
- No special indexes needed

## Future Enhancements

Potential future improvements:

1. **If-None-Match Support**: For conditional GET requests (304 Not Modified)
2. **Strong ETags**: Option to generate strong ETags for byte-exact matching
3. **Custom Hash Functions**: Allow users to provide custom ETag generation logic
4. **ETag in Metadata**: Include ETag property information in OData metadata document
5. **Collection ETags**: Generate ETags for entire collections

## References

- [RFC 7232 - HTTP/1.1: Conditional Requests](https://tools.ietf.org/html/rfc7232)
- [OData v4.0 Part 1: Protocol - Section 8.2.8](http://docs.oasis-open.org/odata/odata/v4.0/odata-v4.0-part1-protocol.html)
- [MDN Web Docs - ETag](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/ETag)
- [MDN Web Docs - If-Match](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/If-Match)
