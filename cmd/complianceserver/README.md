# OData Compliance Server

This is a simplified OData server specifically designed for running compliance tests. Unlike the development server (`cmd/devserver`), this server:

- **No custom middleware**: Runs without authentication or other middleware to ensure pure OData compliance
- **No lifecycle hooks**: Excludes BeforeCreate/BeforeUpdate hooks
- **Minimal entities**: Only includes the entities required for compliance tests:
  - `Products` - Main entity for most tests
  - `ProductDescriptions` - Entity with composite keys
  - `Categories` - Related entity for navigation properties
  - `Company` - Singleton entity
- **Standard actions/functions**: Includes actions and functions needed for compliance testing:
  - Functions: `GetTopProducts`, `GetTotalPrice`, `GetProductStats`
  - Actions: `ApplyDiscount`, `ResetAllPrices`, `IncreasePrice`, `Reseed`
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

## Example Primitive Queries

The compliance sample model includes primitive-focused fields on `Products` to exercise literal parsing and filtering:

- Byte/SByte/Int16/Single:
  - `GET /Products?$filter=Rating gt 100`
  - `GET /Products?$filter=Temperature lt 0`
  - `GET /Products?$filter=Quantity le 32767`
  - `GET /Products?$filter=Weight eq 3.14f`
- Binary literal:
  - `GET /Products?$filter=Data eq binary'dGVzdA=='`
- Date and Time-of-day shaped values:
  - `GET /Products?$filter=ReleaseDate eq 2024-01-15`
  - `GET /Products?$filter=OpenTime eq 09:30:00`
- Duration literal:
  - `GET /Products?$filter=ShippingTime eq duration'P1D'`

## Testing

The compliance suite lives in the external
[NLstn/odata-compliance-suite](https://github.com/NLstn/odata-compliance-suite)
repository. It is a black-box client and does not start this server.

Start the server from the repository root:

```bash
go run ./cmd/complianceserver -db sqlite
```

Then run the suite from another terminal:

```bash
go run github.com/nlstn/odata-compliance-suite@latest -server http://localhost:9090

# Run a focused subset
go run github.com/nlstn/odata-compliance-suite@latest \
  -server http://localhost:9090 -pattern service_document
```

Use a dedicated database because compliance tests can create, update, delete, and reseed data.

## Differences from Development Server

| Feature | Compliance Server | Development Server |
|---------|------------------|-------------------|
| Port | 9090 | 8080 |
| Authentication | None | Dummy middleware |
| Actions | Standard (for compliance) | Demo examples |
| Functions | Standard (for compliance) | Demo examples |
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
