# OData Compliance Server

This is a simplified OData server specifically designed for running compliance tests. Unlike the development server (`cmd/devserver`), this server:

- **No custom middleware**: Runs without authentication or other middleware to ensure pure OData compliance
- **No custom actions/functions**: Excludes custom business logic that isn't part of compliance testing
- **Minimal entities**: Only includes the entities required for compliance tests:
  - `Products` - Main entity for most tests
  - `ProductDescriptions` - Entity with composite keys
  - `Categories` - Related entity for navigation properties
  - `Company` - Singleton entity
- **Port 9090**: Runs on port 9090 by default (devserver uses 8080)

## Running the Compliance Server

```bash
# Default: SQLite in-memory database on port 9090
go run .

# Custom port
go run . -port 8080

# SQLite with persistent database
go run . -db sqlite -dsn ./compliance.db

# PostgreSQL database
go run . -db postgres -dsn "postgresql://user:pass@localhost/dbname"
```

## Available Endpoints

- Service Document: `http://localhost:9090/`
- Metadata: `http://localhost:9090/$metadata`
- Products: `http://localhost:9090/Products`
- Categories: `http://localhost:9090/Categories`
- ProductDescriptions: `http://localhost:9090/ProductDescriptions`
- Company (Singleton): `http://localhost:9090/Company`
- Reseed Database: `POST http://localhost:9090/Reseed`

## Testing

The compliance server is used by the compliance test suite in `compliance/v4/`.

The test script automatically starts and stops the compliance server:

```bash
# Run all compliance tests (server auto-starts)
cd compliance/v4
./run_compliance_tests.sh

# Run specific tests (server auto-starts)
./run_compliance_tests.sh 9.1_service_document

# Use an external/manual server
cd cmd/complianceserver
go run .
# In another terminal:
cd compliance/v4
./run_compliance_tests.sh --external-server
```

## Differences from Development Server

| Feature | Compliance Server | Development Server |
|---------|------------------|-------------------|
| Port | 9090 | 8080 |
| Authentication | None | Dummy middleware |
| Custom Actions | Only Reseed | Multiple demo actions |
| Custom Functions | None | Multiple demo functions |
| Entities | 4 (minimal) | 5 (includes User) |
| Lifecycle Hooks | None | BeforeCreate, BeforeUpdate |
| Purpose | Compliance testing | Development & demos |

## Entity Relationships

```
Category (1) ──< (many) Product (many) >── (many) ProductDescription
                           │
                           └─ RelatedProducts (self-referencing many-to-many)

Company (singleton)
```

## Database Seeding

The server automatically seeds the database with sample data on startup:
- 3 categories (Electronics, Kitchen, Furniture)
- 5 products with various properties
- 7 product descriptions in multiple languages (EN, DE, FR, ES)
- 1 company info singleton

The `Reseed` action can be called to reset the database to this initial state during testing.
