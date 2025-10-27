# Development Server

This development server demonstrates the go-odata library with support for both SQLite and PostgreSQL databases.

## Database Support

The devserver supports two database backends:

### SQLite (Default)

SQLite is the default database and requires no additional configuration.

**Run with file-based SQLite (default):**
```bash
go run .
```

**Run with custom SQLite file:**
```bash
go run . -db sqlite -dsn /path/to/database.db
```

### PostgreSQL

PostgreSQL support is available for production-like testing scenarios.

**Run with PostgreSQL using DSN flag:**
```bash
go run . -db postgres -dsn "postgresql://user:password@host:port/database?sslmode=require"
```

**Run with PostgreSQL using environment variable:**
```bash
export DATABASE_URL="postgresql://user:password@host:port/database?sslmode=require"
go run . -db postgres
```

## Command-Line Flags

- `-db` - Database type: `sqlite` (default) or `postgres`
- `-dsn` - Database connection string (DSN)
  - For SQLite: file path (defaults to `/tmp/go-odata-dev.db`)
  - For PostgreSQL: full PostgreSQL connection string
  - If not provided for PostgreSQL, falls back to `DATABASE_URL` environment variable

## Examples

### Example 1: File-based SQLite (default)
```bash
go run .
```

### Example 2: Custom SQLite file
```bash
go run . -db sqlite -dsn ./devserver.db
```

### Example 3: PostgreSQL with DSN
```bash
go run . -db postgres -dsn "postgresql://neondb_owner:npg_o7fmilaes0dP@ep-plain-king-a9kjvh93-pooler.gwc.azure.neon.tech/neondb?sslmode=require&channel_binding=require"
```

### Example 4: PostgreSQL with environment variable
```bash
export DATABASE_URL="postgresql://user:password@localhost:5432/odata_dev?sslmode=disable"
go run . -db postgres
```

## Database Agnostic Features

The devserver implementation demonstrates database agnostic patterns:

- **Auto-migration**: GORM's AutoMigrate works across both SQLite and PostgreSQL
- **Sequence/Auto-increment handling**: The seeding function handles database-specific sequence resets
  - SQLite: Resets `sqlite_sequence` table
  - PostgreSQL: Resets sequences using `ALTER SEQUENCE`
- **Standard SQL operations**: All CRUD operations use GORM's database-agnostic API

## Testing Database Agnosticity

To test that the OData service works identically across databases:

1. Start with SQLite:
   ```bash
   go run . -db sqlite
   ```

2. Test your OData queries

3. Start with PostgreSQL:
   ```bash
   go run . -db postgres -dsn "your-postgres-url"
   ```

4. Run the same OData queries and verify identical behavior

## Service Endpoints

Once started, the following endpoints are available:

- Service Document: `http://localhost:8080/`
- Metadata (XML): `http://localhost:8080/$metadata`
- Metadata (JSON): `http://localhost:8080/$metadata?$format=json`
- Products: `http://localhost:8080/Products`
- Single Product: `http://localhost:8080/Products(1)`
- ProductDescriptions: `http://localhost:8080/ProductDescriptions`
- Company (Singleton): `http://localhost:8080/Company`

## Reseed Action

The devserver includes a `Reseed` action to reset the database to its initial state:

```bash
curl -X POST http://localhost:8080/Reseed
```

This is useful when testing to ensure a clean database state.
