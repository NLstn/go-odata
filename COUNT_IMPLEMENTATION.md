# $count Endpoint Implementation - OData v4 Specification Compliance

## Overview

This document describes the implementation of the `$count` endpoint in the go-odata library and its compliance with the OData v4 specification.

## OData v4 Specification Requirements

According to the [OData v4 specification](http://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_count), the `$count` system query option:

1. **Returns a plain text value**: The response MUST be a plain text integer value, not a JSON object
2. **Content-Type**: The response MUST have `Content-Type: text/plain`
3. **Applies filtering**: The `$filter` system query option MUST be applied to the count
4. **Ignores pagination**: The `$top` and `$skip` system query options MUST NOT affect the count
5. **Ignores ordering**: The `$orderby` system query option MUST NOT affect the count
6. **Ignores selection**: The `$select` system query option MUST NOT affect the count
7. **HTTP Method**: Only GET requests are allowed

## Implementation Details

### Endpoint URL Pattern

```
GET /EntitySet/$count
GET /EntitySet/$count?$filter=<expression>
```

### Response Format

**Success Response (200 OK)**:
```
Content-Type: text/plain
OData-Version: 4.0

5
```

The response body contains only the integer count value.

**Error Response (4xx/5xx)**:
```json
{
  "error": {
    "code": "400",
    "message": "Invalid query options",
    "details": [...]
  }
}
```

### Implementation Location

- **Handler**: `internal/handlers/collection.go` - `HandleCount()` method
- **Routing**: `server.go` - `routeRequest()` method
- **URL Parsing**: `internal/response/odata.go` - `ParseODataURLComponents()` function

### Key Implementation Features

1. **Plain Text Response**: Returns only the count as a plain text integer
   ```go
   w.Header().Set(HeaderContentType, "text/plain")
   w.Header().Set(HeaderODataVersion, "4.0")
   w.WriteHeader(http.StatusOK)
   fmt.Fprintf(w, "%d", count)
   ```

2. **Filter Support**: Applies `$filter` query option to the count query
   ```go
   if queryOptions.Filter != nil {
       countDB = query.ApplyFilterOnly(countDB, queryOptions.Filter, h.metadata)
   }
   ```

3. **Query Option Handling**: Only `$filter` is applied; other query options like `$top`, `$skip`, `$orderby`, and `$select` are parsed but not applied to the count operation

4. **Method Validation**: Only GET requests are allowed; other HTTP methods return 405 Method Not Allowed

5. **Empty Collection Handling**: Returns "0" for empty collections or filters with no matches

## Test Coverage

### Unit Tests (`internal/handlers/count_test.go`)

- `TestEntityHandlerCount`: Basic count functionality with various filters
- `TestEntityHandlerCountInvalidMethod`: Validates HTTP method restrictions
- `TestEntityHandlerCountInvalidFilter`: Tests error handling for invalid filters
- `TestEntityHandlerCountEmptyCollection`: Tests empty collection behavior
- `TestEntityHandlerCountReturnsPlainText`: Validates plain text response format
- `TestEntityHandlerCountComplexFilter`: Tests string functions (contains, startswith, endswith)
- `TestEntityHandlerCountIgnoresOtherQueryOptions`: Validates that non-filter query options are ignored

### Integration Tests (`test/count_integration_test.go`)

- `TestIntegrationCountEndpoint`: End-to-end tests with various filters
- `TestIntegrationCountEndpointVerifyCollectionStillWorks`: Ensures regular collection endpoint is not affected
- `TestIntegrationCountEndpointODataV4Compliance`: Comprehensive OData v4 compliance tests

## Examples

### Basic Count
```
GET /Products/$count
Response: 5
```

### Count with Filter
```
GET /Products/$count?$filter=Price gt 100
Response: 3
```

### Count with Filter (No Matches)
```
GET /Products/$count?$filter=Category eq 'NonExistent'
Response: 0
```

### Count Ignores $top
```
GET /Products/$count?$top=2
Response: 5  (returns total count, not limited by $top)
```

### Count with String Function
```
GET /Products/$count?$filter=contains(Name,'Laptop')
Response: 2
```

## Compliance Checklist

- [x] Returns plain text integer value
- [x] Sets Content-Type: text/plain
- [x] Sets OData-Version: 4.0
- [x] Applies $filter system query option
- [x] Ignores $top system query option
- [x] Ignores $skip system query option
- [x] Ignores $orderby system query option
- [x] Ignores $select system query option
- [x] Supports string functions in filters (contains, startswith, endswith)
- [x] Supports comparison operators in filters (eq, ne, gt, lt, ge, le)
- [x] Supports logical operators in filters (and, or, not)
- [x] Returns 0 for empty collections
- [x] Returns 405 Method Not Allowed for non-GET requests
- [x] Returns 400 Bad Request for invalid filters
- [x] Returns 404 Not Found for non-existent entity sets

## References

- [OData v4 URL Conventions - $count](http://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part2-url-conventions.html#sec_count)
- [OData v4 Protocol - System Query Options](http://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptions)
