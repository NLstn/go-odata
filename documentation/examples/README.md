# go-odata Examples

This directory contains example code demonstrating various features of the go-odata library.

## Query Types Usage Example

The `query_types_usage.go` example demonstrates the use of newly exported query types, constants, and functions:

### Features Demonstrated

1. **Parsing Filter Expressions**: Using the `ParseFilter()` function to parse OData filter strings into structured `FilterExpression` objects.

2. **Filter Operator Constants**: Accessing symbolic constants for filter operators (e.g., `OpEqual`, `OpGreaterThan`, `OpContains`) and logical operators (`LogicalAnd`, `LogicalOr`).

3. **Apply Transformations**: Building complex Apply transformations programmatically with proper type safety:
   - `GroupByTransformation` for grouping
   - `AggregateTransformation` for aggregations
   - `ComputeTransformation` for computed properties
   - `AggregationMethod` constants (Sum, Avg, Min, Max, Count, CountDistinct)

4. **Parser Configuration**: Using `ParserConfig` to control parser behavior such as max IN clause size and max expand depth.

### Running the Example

```bash
cd documentation/examples
go run query_types_usage.go
```

### Use Cases

These exported types and functions enable:

- **Custom Query Builders**: Building OData queries programmatically with type safety
- **Filter Expression Inspection**: Analyzing and transforming filter expressions
- **Query Options Overwrites**: Implementing custom logic for query processing
- **Testing**: Writing tests that verify query parsing and structure
- **Dynamic Query Generation**: Generating OData queries based on runtime conditions
