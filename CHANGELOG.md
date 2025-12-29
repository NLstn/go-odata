# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project will adhere to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Release tags will follow the `vMAJOR.MINOR.PATCH` pattern so downstream users can
rely on version numbers to reason about compatibility.

## [Unreleased]

### Breaking Changes
- **Hook method names renamed with "OData" prefix**: All EntityHook interface methods have been renamed to avoid conflicts with GORM's hook detection logic
  - `BeforeCreate` → `ODataBeforeCreate`
  - `AfterCreate` → `ODataAfterCreate`
  - `BeforeUpdate` → `ODataBeforeUpdate`
  - `AfterUpdate` → `ODataAfterUpdate`
  - `BeforeDelete` → `ODataBeforeDelete`
  - `AfterDelete` → `ODataAfterDelete`
  - `BeforeReadCollection` → `ODataBeforeReadCollection`
  - `AfterReadCollection` → `ODataAfterReadCollection`
  - `BeforeReadEntity` → `ODataBeforeReadEntity`
  - `AfterReadEntity` → `ODataAfterReadEntity`
  - **Migration Required**: Existing hook implementations must be renamed to use the new "OData" prefix
  - **Benefit**: Models can now safely implement both GORM hooks (e.g., `BeforeCreate(*gorm.DB) error`) and OData hooks (e.g., `ODataBeforeCreate(context.Context, *http.Request) error`) without GORM warnings
  - See migration guide for detailed update instructions

### Added
- PostgreSQL is now fully supported alongside SQLite with all 105 compliance test suites passing on both databases
- MariaDB is now fully supported with all compliance test suites passing on MariaDB 11
- MySQL is now fully supported with all compliance test suites passing on MySQL 8
- CI/CD pipeline now runs compliance tests on SQLite, PostgreSQL, MariaDB, and MySQL to ensure cross-database compatibility
- Authorization policy scaffolding with new auth types, operations, decisions, and service registration hook.
- Authorization enforcement in entity, metadata, and service document handlers with request context/resource descriptors and OData-compliant 401/403 responses.
- Added `ApplyQueryOptionsToSlice` helper for applying `$orderby`, `$skip`, and `$top` to in-memory slices with a `$filter` evaluation hook, along with public query option type aliases to simplify handler usage.
- Authorization policies can now provide query filters to constrain result sets, and navigation property access authorizes both source entities and target sets.
- **Public hook interfaces**: `EntityHook` and `ReadHook` are now exported in the public API, making hooks discoverable via `go doc` and `pkg.go.dev`
  - Hook interface documentation includes comprehensive examples for lifecycle hooks (BeforeCreate, AfterCreate, etc.)
  - Read hook documentation with examples for tenant filtering and data redaction
  - Added "Hooks: Inject Custom Logic" section to README with quick-start examples
  - Improved discoverability: hooks are now prominent in main package documentation
- **Type inheritance support (OData v4 spec 11.2.13)**: Implemented type discriminator detection and entity type filtering
  - Auto-detection of type discriminator properties (`ProductType`, `EntityType`, `Type`, etc.)
  - `isof('Namespace.EntityType')` filter function now correctly filters by entity type using the discriminator column
  - Type casting in URL paths (`/Products/Namespace.SpecialProduct`) for collection filtering
  - Type casting on single entities (`/Products(id)/Namespace.SpecialProduct`) for type verification
  - Access to derived type properties through type cast (`/Products(id)/Namespace.SpecialProduct/SpecialProperty`)
  - Type cast with navigation properties (`/Products(id)/Namespace.SpecialProduct/Category`)

### Fixed
- **Custom entity set names with $search**: Fixed incorrect table name resolution when entities use custom `EntitySetName()` methods. The code now properly uses the pre-computed `TableName` from metadata instead of deriving it from `EntitySetName`, which respects GORM's `TableName()` method and prevents "missing FROM-clause entry" SQL errors
- Authorization checks in entity mutation handlers now use entity-set level create validation and include key-based descriptors with original entity data for update/delete decisions, returning OData-compliant 401/403 errors on failure.
- **MySQL/MariaDB compatibility for OData query functions**: Added database-specific SQL generation for date extraction functions (YEAR, MONTH, DAY, HOUR, MINUTE, SECOND), arithmetic functions (CEILING, FLOOR), and the NOW function. MySQL compliance tests improved from 95% to 97% pass rate (21 failures reduced to 7).
  - Date extraction functions now use MySQL's native YEAR(), MONTH(), etc. instead of PostgreSQL's EXTRACT()
  - CEILING and FLOOR use MySQL's native functions instead of SQLite's CASE expressions
  - Type conversion functions (CAST, ISOF) now use MySQL-appropriate type names (SIGNED, CHAR, DATETIME, etc.)
- **Ambiguous column reference error when combining `$select` with navigation filters**: Fixed PostgreSQL error "column reference is ambiguous" that occurred when using `$select` with `$filter` on navigation properties. The `applySelect` function now qualifies column names with table names (e.g., `members.id` instead of `id`) to prevent ambiguity when JOINs are present. This fix ensures compatibility with both PostgreSQL and SQLite.
- **Dialect-aware quoting for `$apply` aggregations (issue #343)**: `groupby` and `aggregate` SQL builders now qualify and quote identifiers using the active database dialect, preventing case-folding and reserved-word conflicts in PostgreSQL and ensuring compatibility across SQLite/MySQL. `GetColumnName` continues to return unquoted names by design; callers that generate raw SQL now apply proper quoting.
- **PostgreSQL multi-property `$orderby` fix**: Fixed issue where multiple ORDER BY clauses were not preserved correctly in PostgreSQL due to multiple `db.Clauses()` calls. Now builds all ORDER BY expressions in a single `Clauses()` call for PostgreSQL while maintaining the simpler approach for other databases. This ensures proper multi-column sorting (e.g., `$orderby=Name,Price desc`) works correctly across all supported databases.
- Data race in async monitor configuration resolved by synchronizing access in the router, fixing `-race` CI test failures in `internal/service/runtime.TestServiceRespondAsyncFlow`.
- Compliance test flakiness eliminated by implementing per-test database reseeding instead of per-suite reseeding
  - Async processing tests now pass consistently (previously failed 5/6 tests on first run, 0/6 on second run)
  - Test isolation improved: each test starts with clean database state
  - Async table lifecycle fixed: explicit cleanup and verification of `_odata_async_jobs` table after reseed
  - Test results now consistent across runs: ~13 deterministic failures instead of 10-14 flaky failures
- Compliance tests now use file-based SQLite database (`/tmp/go-odata-compliance.db`) instead of in-memory database to prevent flakiness in CI environments
- Database reseeding in compliance server now handles PostgreSQL foreign key constraints correctly, ensuring cross-database compatibility between SQLite and PostgreSQL without requiring users to handle database-specific cleanup logic
- Removed SQLite-specific GORM blob type specification for binary content, allowing GORM to use appropriate database-specific types (BLOB for SQLite, BYTEA for PostgreSQL)
- Compliance server entity table names now match OData entity set names (Products, Categories, ProductDescriptions, MediaItems, Company) fixing 300+ test failures
- Registered all 105 test suites in compliance test runner (previously only 45 were registered)
- Compliance test pass rate improved from 30% to 75% (501 of 666 tests now passing)
- Added skip logic to tests requiring unimplemented features (complex property ordering, UUID validation)
- Remaining 128 failures mostly due to: schema mismatches (UUID vs int IDs), missing response headers, and unimplemented optional features

- Go-based compliance test suite (`compliance-suite/`) for OData v4 specification validation
  - Ported 34 test suites from bash to Go with 242 individual tests
  - Tests cover: JSON format, introduction, conformance, EDMX elements, HTTP headers (Content-Type, Accept, OData-MaxVersion, OData-Version),
    error responses, service/metadata documents, entity addressing, canonical URLs, property access, metadata levels, 
    resource paths, collection operations, requesting individual entities, lambda operators (any/all), geospatial functions, 
    filtering on expanded properties, string function edge cases, and query options ($filter with logical/comparison/string/date/arithmetic/type functions, 
    $select, $orderby, $top, $skip, $count, $expand, $format)
  - Custom test framework with HTTP utilities, proper URL encoding, and detailed reporting
  - **Known Issue**: 1 test failing - empty path segment handling (`/Products//`) returns 200 instead of 404/400/301 per OData spec
  
- Entity handlers now expose `NavigationTargetSet` so the router can resolve bound actions and functions after renamed navigation properties.

- Lifecycle hooks now expose the active GORM transaction through `odata.TransactionFromContext`, enabling user code to perform
  additional queries that participate in the same commit.
- `AsyncConfig.DisableRetention` allows services to opt out of automatic async
  job cleanup when stricter audit retention is required.
- Support deriving action/function parameters from a struct by setting
  `ParameterStructType`, and expose an `actions.BindParams` helper in the public
  `github.com/nlstn/go-odata/actions` package so handlers can consume strongly
  typed inputs without manual map assertions.
- Public `Service.Close` helper to stop async processing and release resources.
- Service-level key generator registry with metadata validation powers server-generated keys
  (including built-in UUIDs) and generalized entity key initialization.
- Documentation for configuring server-generated keys plus sample API key entities in the
  development and performance servers that exercise UUID-based generation.

### Changed

- Moved service routing and operation handling into internal packages to reduce
  root-level surface area while keeping exported APIs unchanged.
- Async job managers now apply a 24-hour default retention window when no
  duration is provided and continue purging expired rows in the background.
- Compliance suite default output is now concise (overall progress and summary only); use `-verbose` for per-suite and per-test details.
- Documentation now clarifies full-text search support for SQLite/PostgreSQL, along with fallback behavior and operational constraints.

### Fixed

- Ensure function context URLs honor the configured service namespace when returning complex types.
- NewService constructors now return a clear error when given a nil database
  handle, preventing later panics from misconfigured callers.

## [v0.4.0] - 2025-11-08

### Added

- Core OData v4 service capable of registering entities, exposing metadata, and
  handling CRUD requests through the HTTP service entrypoints.
- Comprehensive query option support, including `$filter`, `$select`, `$expand`,
  `$orderby`, `$top`, `$skip`, `$count`, `$search`, and `$apply` aggregations.
- Change-tracking infrastructure with `Service.EnableChangeTracking` and
  delta-token responses for clients that need incremental synchronization.
- Persistent change-tracking storage backed by GORM, including a `_odata_change_log` table and
  `ServiceConfig.PersistentChangeTracking` for restart-safe delta tokens.
- Asynchronous request processing via `Service.EnableAsyncProcessing` with job
  monitoring endpoints for long-running operations.
- Hooks, lifecycle callbacks, and custom action/function registration so
  applications can inject business logic and expose bespoke operations.
- Geospatial query support that integrates with GORM-backed entity models.
- Integration tests that recreate the service from stored change history to ensure delta tokens survive restarts.
- Regression tests covering transactional rollbacks for navigation binding failures to ensure no phantom change-tracking events are emitted.

### Changed

- Entity create/update/delete handlers now execute within database transactions so navigation binding errors roll back persisted rows and change-tracking events are recorded only after successful commits.
- `Service.RegisterEntity` and `Service.RegisterSingleton` now return descriptive errors when duplicate names are registered
  instead of overwriting existing metadata.

### Fixed

- Navigation property pagination now emits `$skiptoken` next links and documents the ordering requirements for deterministic paging.
- Preserve entity handler configuration when executing transactional batch requests so navigation property
  handling and change tracking continue to work inside changesets.

### Notes

- This is the first documented release in the changelog. Subsequent releases
  will increment the version following semantic versioning rules and will be
  recorded in this changelog alongside Git tags.
