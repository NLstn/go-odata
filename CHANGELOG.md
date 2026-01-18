# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project will adhere to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Release tags will follow the `vMAJOR.MINOR.PATCH` pattern so downstream users can
rely on version numbers to reason about compatibility.

## [Unreleased]

### Added
- **Batch request size limits to prevent DoS attacks**: Added `MaxBatchSize` field to `ServiceConfig` to limit the maximum number of sub-requests in a batch request. Default limit is 100 sub-requests, preventing DoS attacks via large batch payloads. When exceeded, the service returns HTTP 413 Request Entity Too Large. The limit applies to both individual requests and changeset operations (which are checked before transaction commit to ensure proper rollback). Configurable via `ServiceConfig{MaxBatchSize: N}` where N is the maximum number of sub-requests allowed.
- Added table-driven tests for EDM Go type inference, value parsing, and struct tag handling.
- Added a unit test for the async job record table name mapping.
- Added expand parser tests covering depth limits and quoted string handling in split expand parsing.
- Added unit tests for delta collection entries across OData metadata levels.
- Added unit tests covering enum registry error handling, enum member resolution, and underlying type detection.
- Added unit tests for expand annotations to validate nested expand handling and count emission.
- **Base path mounting with automatic URL generation**: Added `SetBasePath()` method to mount the OData service at a custom path (e.g., `/api/odata`). The service automatically strips the base path from incoming requests and includes it in all generated URLs (`@odata.id`, `@odata.nextLink`, etc.). This eliminates the need for `http.StripPrefix` middleware and ensures all OData URLs are correctly qualified. Validation prevents path traversal (`..`), trailing slashes, and other invalid patterns. Thread-safe configuration using `sync.RWMutex` allows concurrent requests while the base path is being set. Each service instance maintains its own independent base path, allowing multiple services with different mount points to run in the same process.
- **$count and $levels support in nested $expand options**: Added full support for `$count=true` and `$levels` within `$expand` clauses. Expanded collections now emit `Nav@odata.count` annotations when requested, and `$levels` recursively expands navigation properties with a safe maximum depth (including `$levels=max`). `$count=false` remains a no-op.
- **Compliance suite $count segment coverage**: Added OData v4.0 tests for the `$count` path segment, validating text/plain responses and filtered count parity with `@odata.count`.
- **Compliance suite nested $expand coverage**: Added OData v4.01 compliance tests for `$expand` with nested `$count` and `$levels`.
- **OData version negotiation (4.0 / 4.01) with context-aware handling**: Added full support for OData version negotiation per OData v4 spec §8.2.6. The service now negotiates version based on client's `OData-MaxVersion` header and stores the negotiated version in request context via `version.GetVersion(ctx)`. Metadata documents are now version-specific and cached per version. New features include:
  - Context-aware version handling with `version.WithVersion()` and `version.GetVersion()`
  - Version-specific metadata caching with automatic eviction (max 10 entries)
  - Lock-free metadata cache reads using `sync.Map` (30-70% performance improvement: 368K ops/sec)
  - Router middleware automatically sets `OData-Version` response header based on negotiation
  - Comprehensive edge case tests with race detector validation (550+ concurrent goroutines tested)
  - Version feature detection via `version.Supports()` method for conditional functionality
- Version parsing and negotiation unit test coverage for supported protocol versions.
- **Geospatial feature support with database compatibility checking**: Added `EnableGeospatial()` method to enable geospatial operations (geo.distance, geo.length, geo.intersects). The service now validates database support for spatial features on startup and returns HTTP 501 Not Implemented when geospatial operations are attempted without enablement. Includes detection for SQLite (SpatiaLite), PostgreSQL (PostGIS), MySQL/MariaDB (spatial functions), and SQL Server (spatial types) with detailed error messages for missing extensions.
- **Increased test coverage with meaningful unit tests**: Added comprehensive unit tests across multiple packages, improving overall code coverage from 52.8% to 53.7%. Key improvements include:
  - `internal/auth` package: 0% → 100% coverage (tests for AuthContext, Policy interface, Decision functions)
  - `internal/hookerrors` package: 0% → 100% coverage (tests for HookError, error wrapping, error interface compliance)
  - `internal/query` package: 56.3% → 57.3% coverage (tests for query applier functions)
  - `internal/handlers` package: 42.9% → 43.1% coverage (tests for hook error extraction)
  - `internal/response` package: 53.6% → 60.1% coverage (tests for field caching, entity key extraction, format helpers)
- **Expanded service routing and operation handler coverage**: Added tests for OData version handling, async monitor path normalization, key serialization, function response formatting, action signature matching, and geospatial capability checks, and stabilized async runtime tests by using a file-backed SQLite database.
- Added compliance coverage for parameter aliases in system query options ($filter/$top).
- **Parameter alias support**: Added full support for OData v4.0 parameter aliases (section 11.2.5.8), allowing query options to reference aliases defined as query parameters (e.g., `$filter=Price gt @p&@p=10`). Parameter aliases can be used in $filter, $orderby, $top, $skip, and other query options. This enables more flexible and readable queries, especially when using the same value multiple times.
- Added collection executor tests covering error handling and hook overrides.

### Changed
- **BREAKING: NewService now returns error instead of panicking**: The `NewService` function signature changed from `func NewService(db *gorm.DB) *Service` to `func NewService(db *gorm.DB) (*Service, error)`. This provides better error handling and prevents panics in production code. Code that calls `NewService` must now handle the error return value. Example: `service, err := odata.NewService(db); if err != nil { log.Fatal(err) }`
- **Metadata cache now uses sync.Map for lock-free reads**: Converted metadata handler from `map[string]string` with `sync.RWMutex` to `sync.Map` for both XML and JSON caches, eliminating lock contention on cache hits (99%+ of requests). Benchmarks show 30% improvement in concurrent scenarios.
- **Version headers now set automatically in router middleware**: The router now automatically sets the `OData-Version` response header based on client negotiation, eliminating the need for manual header management in most cases.
- **Version parsing now returns explicit errors**: `parseVersion()` function signature changed from `(int, int)` to `(int, int, error)` for better error handling. Invalid version strings are now validated and rejected with HTTP 400, and versions < 4.0 return HTTP 406 (Not Acceptable).
- **Compliance suite now enforces optional features**: Removed skip-based leniency in compliance tests so optional OData features (lambda operators, geospatial functions, stream properties, etc.) must be implemented to pass.
- **Policy filters now apply to expanded navigation results**: Authorization policy query filters are merged into `$expand` filters to ensure expanded navigation properties are filtered the same way as direct navigation queries.
- **Navigation $orderby now validates and joins single-entity paths**: `$orderby` expressions like `Nav/Field` now validate the target property and add the required JOINs with qualified column references for correct SQL generation.
- **Multi-segment navigation path validation and joins**: `$filter` and `$orderby` now validate multi-hop single-entity navigation paths and generate JOINs with stable aliases to avoid table name collisions when chaining navigation segments.
- **Navigation links included for selected navigation properties in minimal metadata**: `$select=NavProp` now emits `NavProp@odata.navigationLink` without requiring `$expand` when minimal metadata is requested.
- Lambda filter navigation targets now resolve through the cached entity registry, reducing repeated metadata analysis during lambda predicate evaluation.

### Deprecated
- `handlers.SetODataVersionHeader()` - Use `response.SetODataVersionHeaderFromRequest(w, r)` instead for context-aware version handling. The router middleware handles this automatically in most cases.
- `response.SetODataVersionHeader()` - Use `response.SetODataVersionHeaderFromRequest(w, r)` instead. This function always returns "4.01" and does not respect client version negotiation.
- `response.ODataVersionValue` constant - Use `version.GetVersion(ctx).String()` to get the negotiated version from request context. This constant always returns "4.01" and does not support version negotiation.

### Performance
- **30-70% improvement in concurrent metadata requests**: Lock-free cache reads using sync.Map provide significant performance gains under high concurrency. Benchmarks show 368K ops/sec for concurrent cache hits vs 256K ops/sec sequential.
- **Cache eviction prevents unbounded memory growth**: Metadata cache now limited to 10 entries with automatic eviction keeping 5 most common versions (4.0, 4.01 prioritized).

### Fixed
- **Thread-safe geospatial flag access**: Fixed potential data race in `geospatialEnabled` flag by adding mutex protection (`geospatialMu`). The flag is now safely readable by concurrent calls to `IsGeospatialEnabled()` and during entity registration (`RegisterEntity`, `RegisterSingleton`, `RegisterVirtualEntity`) while being writable by `EnableGeospatial()`. This ensures race detector compliance and prevents undefined behavior when geospatial features are toggled during concurrent operations.
- **Schema-qualified table name quoting in SQL generation**: Fixed SQL generation for entities with schema-qualified table names (e.g., `"mart.loads"`). Previously, `quoteIdent()` treated the entire table name as a single identifier, generating invalid SQL like `[mart.loads].[column]` for SQL Server. Added `quoteTableName()` helper that splits on dots and quotes each part separately, producing correct SQL like `[mart].[loads].[column]`. This aligns column reference quoting with GORM's FROM clause handling and prevents "multi-part identifier could not be bound" errors.
- `$expand` parsing now ignores commas inside single-quoted string literals, preventing incorrect splitting of expand items when nested filters include commas.
- Nested `$expand` options now reject negative `$top` and `$skip` values using the same non-negative validation as top-level query options.
- Collection `$expand` now applies `$top/$skip/$orderby` per parent instead of using global preload limits, ensuring consistent pagination semantics across supported dialects.
- Unbalanced parentheses in `$expand` now return clear parsing errors instead of being silently accepted.
- Per-parent `$expand` now batches child queries and supports composite referential constraints, avoiding N+1 queries when expanding collections with composite keys.
- `$orderby` parsing now rejects extra tokens beyond the optional direction keyword and reports the offending token alongside the property name for clearer client errors.
- **Lambda any/all operators now support composite key relationships**: Fixed lambda operators (any/all) to properly join on all key properties when filtering by collection navigation properties. Previously, only the first key property was used in the join condition, which caused incorrect results for entities with composite keys. The fix handles comma-separated foreign keys in GORM tags and builds proper join predicates across all key columns.
- **Lambda any/all joins honor key column mappings**: Fixed lambda join SQL to use the key property column names from metadata, ensuring composite keys with custom `gorm:"column:..."` mappings generate correct parent-side join predicates.
- **`isof()` fallback behavior without discriminator**: Fixed spec deviation where `isof('EntityType')` returned `1 = 1` (matching all entities) when no type discriminator was configured. Now returns `1 = 0` (matching no entities) since correct type filtering cannot be performed without a discriminator. This prevents incorrect results in inheritance scenarios and ensures spec compliance.
- **Property-to-property comparisons in $filter**: Fixed SQL generation for property-to-property comparisons (e.g., `$filter=Price gt Cost`). Previously, the parser correctly identified property names on the right-hand side of comparisons, but the SQL generator treated them as string literals, producing invalid SQL like `price > 'Cost'` instead of `price > cost`. The SQL generator now detects when a value is a property identifier and generates correct column-vs-column comparisons.
- **Navigation joins now use target entity's actual primary key**: Fixed issue where navigation property filters (e.g., `$filter=Department/Name eq 'IT'`) would fail with "no such column" errors when the target entity has a custom primary key (not "id"). The `addNavigationJoin` function now queries the target entity's metadata to determine the actual primary key column instead of defaulting to "id". For composite keys, the first key component is used unless an explicit `references:` tag is specified. This fix ensures correct JOIN generation for entities with custom primary keys like `Code`, `LanguageKey`, etc.
- **Navigation filters honor target column mappings**: Filters that reference single-entity navigation paths now resolve the target entity's property metadata, ensuring SQL uses custom `gorm:"column:..."` names instead of snake-cased field names.
- Nested `$expand` options now validate `$select`, `$filter`, `$orderby`, and `$compute` against the expanded entity metadata, and expanded order-by/filter SQL uses metadata-aware column resolution with dialect quoting to avoid invalid property use and SQL mismatches.
- **Escaped wildcard handling for string filters**: contains/startswith/endswith now escape `%`, `_`, and the escape character itself when generating SQL LIKE patterns with database-specific ESCAPE clauses (MySQL/MariaDB use `ESCAPE '\\\\'`, others use `ESCAPE '\\'`), ensuring literal matching across all supported databases.
- **OData-MaxVersion header now correctly negotiates response version**: Per OData v4 spec section 8.2.6, the service now responds with the maximum supported version that is less than or equal to the requested `OData-MaxVersion` header. When a client sends `OData-MaxVersion: 4.0`, the response now correctly includes `OData-Version: 4.0` instead of the hardcoded `4.01`. This fixes compatibility with clients like Excel that only support OData v4.0.
- **Error responses now respect version negotiation**: Fixed issue where error responses always emitted `OData-Version: 4.01` regardless of the client's `OData-MaxVersion` header. Error responses now correctly use `SetODataVersionHeaderFromRequest` to respect the negotiated OData version (4.0 or 4.01) based on the client's request, ensuring strict OData v4 spec compliance for error scenarios.
- **Batch responses now echo Content-ID headers**: Per OData v4 spec section 11.7.4, Content-ID headers from batch request parts are now properly echoed back in the corresponding response parts. This enables clients to correlate batch responses with their requests, which is essential for batch request processing and changeset referencing.
- **$count results now honor $search**: Count endpoints and `$count=true` responses now apply `$search` (and `$filter`) consistently, using FTS when available with an in-memory fallback, matching OData specification requirements.
- **FTS cache now properly reset after database reseeding**: Fixed compliance test failures where FTS (Full-Text Search) tables were dropped during database reseeding but the FTS manager's internal cache was not cleared, causing subsequent search queries to fail. Added `ClearFTSCache()` method to `FTSManager` and `ResetFTS()` method to `Service` to enable proper cache clearing after FTS table drops.
- Pre-request hook failures now return OData-formatted error responses for non-batch requests.
- **Observability documentation and implementation cleanup**: 
  - Added nil check for logger before calling Info in SetObservability to prevent panic
  - Marked `EnableQueryOptionTracing` as not yet implemented with clear documentation
  - Fixed ServiceName default documentation to match actual value ("odata-service")
  - Updated span hierarchy documentation to reflect actual implementation (removed non-existent spans)
  - Clarified that `odata.db.query.duration` metric requires detailed DB tracing to be enabled
  - Updated database span attributes documentation to include `db.system` and reorder attributes correctly
  - Removed `EnableQueryOptionTracing` from documentation examples since the feature is not implemented
- **Compliance test suite fixes**: Resolved missing go.sum entries in compliance-suite and complianceserver modules that prevented tests from running
- **Linting errors**: Fixed ineffectual variable assignments in observability_test.go
- **Code safety documentation**: Added clarifying comments to request path extraction functions to document that extracted values are used only in metrics/logging contexts and do not require HTML escaping
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
- Ensure function context URLs honor the configured service namespace when returning complex types.
- NewService constructors now return a clear error when given a nil database handle, preventing later panics from misconfigured callers.
- **Hook method names renamed with "OData" prefix**: All EntityHook interface methods have been renamed to avoid conflicts with GORM's hook detection logic
- **Lambda filters now respect navigation target column mappings**: any/all predicates now resolve target entity column names via metadata, honoring custom GORM `column:` tags in navigation collections.
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
- **PreRequestHook for unified request preprocessing**: Introduced a service-level hook that is called before each request is processed
  - Works uniformly for both single requests and batch sub-requests (including changesets)
  - Allows authentication, context enrichment, logging, and other preprocessing tasks
  - Hook can return a modified context to pass values to downstream handlers
  - Hook can return an error to abort the request with HTTP 403 Forbidden
  - See `SetPreRequestHook` documentation for detailed usage examples

### Removed
- **SetBatchSubRequestHandler**: Removed in favor of the simpler and more comprehensive `SetPreRequestHook` mechanism

- **Server-Timing database time metric**: When `EnableServerTiming` is enabled, the Server-Timing HTTP response header now includes a `db` metric that reports the total time spent in database queries during the request
  - The `db` metric shows accumulated time from all GORM database operations (queries, creates, updates, deletes)
  - Combined with the existing `total` metric, you can calculate server processing time (total - db) vs database time
  - Example header: `Server-Timing: total;desc="Total request duration";dur=15.5, db;desc="Database queries";dur=3.2`
  - Useful for identifying whether performance issues are database-related or in application code
- **Server-Timing HTTP response header support**: Optional Server-Timing header for performance debugging in browser dev tools
  - Enabled via `EnableServerTiming: true` in `ObservabilityConfig`
  - Uses the [mitchellh/go-server-timing](https://github.com/mitchellh/go-server-timing) library
  - Adds timing metrics to HTTP responses that are visible in browser developer tools (Chrome 65+, Firefox 71+)
  - Records total request duration automatically
  - Public helper functions `StartServerTiming()` and `StartServerTimingWithDesc()` for custom timing metrics in hooks and handlers
  - Zero overhead when disabled (middleware is not applied)
- **OpenTelemetry-based Observability Support**: Comprehensive observability infrastructure using OpenTelemetry standards
  - **Tracing**: Full distributed tracing with proper span hierarchy for request lifecycle
    - HTTP request spans with method, path, status code attributes
    - Entity operation spans (Read, Create, Update, Delete) with entity set and key info
    - Batch operation spans with changeset correlation
    - Database query tracing via GORM callbacks (opt-in)
    - Query option tracing for $filter, $select, $expand, etc. (opt-in)
  - **Metrics**: OData-specific metrics for monitoring and alerting
    - `odata.request.duration` - Request duration histogram by entity set, operation, status
    - `odata.request.count` - Request counter by entity set, operation, status
    - `odata.result.count` - Result set size histogram for collection queries
    - `odata.db.query.duration` - Database query duration histogram
    - `odata.batch.size` - Batch request size histogram
    - `odata.error.count` - Error counter by type
  - **Zero-overhead when disabled**: No-op implementations ensure zero performance impact when observability is not configured
  - **Flexible configuration**: Functional options pattern for easy setup with any OpenTelemetry-compatible backend (Jaeger, Tempo, Datadog, AWS X-Ray, etc.)
  - See [documentation/observability.md](documentation/observability.md) for detailed usage guide
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
- Authorization checks for CREATE operations use entity-set descriptors, while UPDATE/DELETE operations now include fetched entity data and derived key values in key-based resource descriptors for more granular authorization decisions, returning OData-compliant 401/403 errors on failure.
- **MySQL/MariaDB compatibility for OData query functions**: Added database-specific SQL generation for date extraction functions (YEAR, MONTH, DAY, HOUR, MINUTE, SECOND), arithmetic functions (CEILING, FLOOR), and the NOW function. MySQL compliance tests improved from 95% to 97% pass rate (21 failures reduced to 7).
  - Date extraction functions now use MySQL's native YEAR(), MONTH(), etc. instead of PostgreSQL's EXTRACT()
  - CEILING and FLOOR use MySQL's native functions instead of SQLite's CASE expressions
  - Type conversion functions (CAST, ISOF) now use MySQL-appropriate type names (SIGNED, CHAR, DATETIME, etc.)
- Single-entity query option validation now shares a single helper to parse and enforce disallowed `$top`, `$skip`, and `$index` options, ensuring consistent error responses.
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

### Added

- **Go-based compliance test suite (`compliance-suite/`)** for OData v4 specification validation
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
- Metadata generation now shares common logic between JSON and XML outputs, and XML assembly uses builders to reduce allocations.

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
