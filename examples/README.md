# go-odata Examples

This directory contains example scripts and code demonstrating various features of the go-odata library.

## Running Examples

First, start the development server:

```bash
cd ../cmd/devserver
go run .
```

The server will start on `http://localhost:8080`.

## Available Examples

### ETag Support Example

The `etag_example.sh` script demonstrates the ETag (optimistic concurrency control) feature:

```bash
./etag_example.sh
```

This script demonstrates:
- Retrieving an entity with its ETag header
- Updating an entity with the correct ETag (succeeds)
- Attempting to update with an incorrect ETag (fails with 412 Precondition Failed)
- Updating with wildcard `If-Match: *` (succeeds)
- Creating a new entity that includes an ETag

### Manual Testing with curl

You can also test ETag functionality manually:

```bash
# Get an entity and its ETag
curl -i 'http://localhost:8080/Products(1)'

# Update with ETag validation
curl -X PATCH 'http://localhost:8080/Products(1)' \
  -H 'Content-Type: application/json' \
  -H 'If-Match: W/"<etag-value-here>"' \
  -d '{"Price": 899.99}'

# Test with incorrect ETag (should return 412)
curl -X PATCH 'http://localhost:8080/Products(1)' \
  -H 'Content-Type: application/json' \
  -H 'If-Match: W/"wrongetag"' \
  -d '{"Price": 799.99}'
```

## Further Documentation

For more information, see the main [README.md](../README.md) in the project root.
