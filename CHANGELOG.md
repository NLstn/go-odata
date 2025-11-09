# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project will adhere to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Release tags will follow the `vMAJOR.MINOR.PATCH` pattern so downstream users can
rely on version numbers to reason about compatibility.

## [Unreleased]

### Added
- Entity handlers now expose `NavigationTargetSet` so the router can resolve bound actions and functions after renamed navigation properties.

- Lifecycle hooks now expose the active GORM transaction through `odata.TransactionFromContext`, enabling user code to perform
  additional queries that participate in the same commit.
- `AsyncConfig.DisableRetention` allows services to opt out of automatic async
  job cleanup when stricter audit retention is required.

### Changed

- Moved service routing and operation handling into internal packages to reduce
  root-level surface area while keeping exported APIs unchanged.
- Async job managers now apply a 24-hour default retention window when no
  duration is provided and continue purging expired rows in the background.

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

