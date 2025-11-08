# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project will adhere to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Release tags will follow the `vMAJOR.MINOR.PATCH` pattern so downstream users can
rely on version numbers to reason about compatibility.

## [Unreleased]

### Added

- Persistent change-tracking storage backed by GORM, including a `_odata_change_log` table and
  `ServiceConfig.PersistentChangeTracking` for restart-safe delta tokens.
- Integration tests that recreate the service from stored change history to ensure delta tokens survive restarts.

## [v0.1.0] - 2025-11-07 _(planned)_

### Added

- Core OData v4 service capable of registering entities, exposing metadata, and
  handling CRUD requests through the HTTP service entrypoints.
- Comprehensive query option support, including `$filter`, `$select`, `$expand`,
  `$orderby`, `$top`, `$skip`, `$count`, `$search`, and `$apply` aggregations.
- Change-tracking infrastructure with `Service.EnableChangeTracking` and
  delta-token responses for clients that need incremental synchronization.
- Asynchronous request processing via `Service.EnableAsyncProcessing` with job
  monitoring endpoints for long-running operations.
- Hooks, lifecycle callbacks, and custom action/function registration so
  applications can inject business logic and expose bespoke operations.
- Geospatial query support that integrates with GORM-backed entity models.

### Notes

- This will be the first published tag for the repository. Subsequent releases
  will increment the version following semantic versioning rules and will be
  recorded in this changelog alongside Git tags.

