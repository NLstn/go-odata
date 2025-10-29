# go-odata Documentation

Welcome to the go-odata documentation! This directory contains detailed guides for using the library.

## Table of Contents

### Getting Started

- **[Entity Definition](entities.md)** - Learn how to define entities with Go structs, including tags, relationships, and metadata
- **[End-to-End Tutorial](tutorial.md)** - Build a Products/Orders/Customers sample backend with migrations, seeding, and custom logic
- **[Server Configuration](server-configuration.md)** - Set up your OData service, configure middleware, and integrate with routers like Chi, Gin, and Echo

### Advanced Usage

- **[Actions and Functions](actions-and-functions.md)** - Implement custom OData operations beyond standard CRUD
- **[Advanced Features](advanced-features.md)** - Use singletons, ETags for concurrency control, lifecycle hooks, and read hooks for authorization/redaction
- **[Geospatial Functions](geospatial.md)** - Query geographic data with geo.distance, geo.length, and geo.intersects

### Testing & Development

- **[Testing](testing.md)** - Unit tests, OData v4 compliance tests, performance profiling, and SQL query tracing

## Quick Links

- [Main README](../README.md) - Library overview and quick start
- [OData v4 Specification](https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html) - Official specification
- [GitHub Repository](https://github.com/NLstn/go-odata) - Source code and issues

## Documentation Summary

### Entity Definition
Define your data model with Go structs and OData tags. Learn about:
- Basic entity structure
- Rich metadata facets (maxlength, precision, scale, default values)
- Relationships and navigation properties
- Composite keys
- Type mappings from Go to EDM
- Configuring search behavior

### Server Configuration
Set up and configure your OData service. Topics include:
- Basic server setup
- Using the service as an HTTP handler
- Mounting at custom paths
- Adding middleware (auth, logging, CORS)
- Integrating with Chi, Gin, and Echo routers
- Combining with other HTTP handlers
- Database configuration
- Production considerations

### Actions and Functions
Extend your service with custom operations. Learn about:
- The difference between actions and functions
- Registering unbound and bound operations
- Parameter types and handling
- Error handling in custom operations
- Best practices for implementation

### Advanced Features
Leverage powerful OData v4 features:
- **Singletons**: Single-instance entities accessible by name
- **ETags**: Optimistic concurrency control for safe updates
- **Lifecycle & Read Hooks**: Execute custom logic at specific points in entity lifecycle, add tenant filters, or redact responses before returning data
- **Geospatial Functions**: Query geographic data using geo.distance, geo.length, and geo.intersects

### Testing
Ensure your OData service works correctly:
- Unit testing strategies
- Integration tests with GORM
- Running the OData v4 compliance test suite (85+ tests)
- Performance profiling with CPU profiles
- SQL query tracing to identify N+1 queries and bottlenecks

## Contributing

If you find issues with the documentation or have suggestions for improvements, please:
1. Open an issue on [GitHub](https://github.com/NLstn/go-odata/issues)
2. Submit a pull request with improvements

## License

This documentation is part of the go-odata project and is licensed under the MIT License.
