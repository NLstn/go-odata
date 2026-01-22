# go-odata Examples

This directory contains example code demonstrating various features of the go-odata library.

**Important Note**: Each example file in this directory is a standalone program with its own `main()` function. They are tagged with the `example` build tag to exclude them from normal builds. To run an individual example, use the `-tags example` flag.

## Available Examples

### 1. Query Types Usage Example (`query_types_usage.go`)

Demonstrates the use of newly exported query types, constants, and functions:

#### Features Demonstrated

- **Parsing Filter Expressions**: Using the `ParseFilter()` function to parse OData filter strings
- **Filter Operator Constants**: Accessing symbolic constants for operators
- **Apply Transformations**: Building complex transformations programmatically
- **Parser Configuration**: Using `ParserConfig` to control parser behavior

#### Running the Example

```bash
# Option 1: Run directly with the example tag
cd documentation/examples
go run -tags example query_types_usage.go

# Option 2: Copy to a temporary directory
mkdir -p /tmp/query-example
cp query_types_usage.go /tmp/query-example/main.go
# Remove the build tag line from main.go
sed -i '/^\/\/ +build example$/d' /tmp/query-example/main.go
cd /tmp/query-example
go mod init example
go get github.com/nlstn/go-odata
go run main.go
```

### 2. Authorization Examples (`auth_examples.go`)

Demonstrates authorization patterns including:

- **Basic Policy Implementation**: Simple role-based authorization
- **Tenant-Based Policy**: Multi-tenant authorization with row-level filtering
- **Resource-Level Policy**: Fine-grained resource-level authorization
- **Scope-Based Policy**: OAuth2 scope-based authorization
- **Auth Context Population**: Extracting authentication from HTTP requests
- **Field-Level Authorization**: Using hooks to redact sensitive fields

#### Running the Example

```bash
# Run with the example tag
cd documentation/examples
go run -tags example auth_examples.go
```

### 3. Overwrite Context Examples (`overwrite_context_examples.go`)

Demonstrates usage of key exported types:

- **OverwriteContext with Composite Keys**: Accessing individual key components
- **QueryFilterProvider**: Implementing row-level security with automatic filtering
- **SliceFilterFunc**: Creating custom filter evaluators for in-memory data
- **ApplyQueryOptionsToSlice**: Applying OData query options to slices

#### Running the Example

```bash
# Run with the example tag
cd documentation/examples
go run -tags example overwrite_context_examples.go
```

## Use Cases

These exported types and functions enable:

- **Custom Query Builders**: Building OData queries programmatically with type safety
- **Filter Expression Inspection**: Analyzing and transforming filter expressions
- **Query Options Overwrites**: Implementing custom logic for query processing
- **Authorization Policies**: Implementing custom authorization logic
- **Virtual Entities**: Exposing non-database data sources through OData
- **Testing**: Writing tests that verify query parsing and structure
- **Dynamic Query Generation**: Generating OData queries based on runtime conditions
