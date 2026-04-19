# go-odata OData Spec Gap Analysis (v4.0 and v4.01)

## Scope
- Library under review: `go-odata` (main branch workspace snapshot)
- Runtime target used for verification: compliance server with SQLite (`cmd/complianceserver`)
- Negotiated protocol behavior observed: service advertises OData 4.01 in responses and metadata (`Version="4.01"`)
- Compliance evidence baseline: full compliance suite run (v4.0 + v4.01) with 0 failures and 8 skips

## Method
1. Ran compliance-suite end-to-end and extracted skipped tests from verbose logs.
2. Mapped each skip to implementation code paths.
3. Performed targeted runtime checks (metadata and API requests) to classify each potential gap.
4. Cross-checked against OASIS normative text.

---

## Finding 1: Geospatial canonical functions are not available by default in the tested service profile

### Scope
- Behavior: `$filter` with `geo.distance`, `geo.length`, `geo.intersects`
- Version context: applies to both OData 4.0 and 4.01 URL semantics

### Spec Evidence
- OData URL Conventions defines geospatial functions under canonical functions:
  - `geo.distance`, `geo.intersects`, `geo.length`
  - https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#_Toc444868733
- OData Protocol conformance (Intermediate 4.0) says services SHOULD support canonical functions and MUST return `501 Not Implemented` for unsupported canonical functions:
  - https://docs.oasis-open.org/odata/odata/v4.01/os/part1-protocol/odata-v4.01-os-part1-protocol.html (Conformance section 13.1.2, item 7.4)

### Implementation Evidence
- Geospatial parsing/SQL generation exists:
  - `internal/query/ast_parser_functions.go` (handles `geo.distance`, `geo.length`, `geo.intersects`)
  - `internal/query/apply_filter.go` (maps to `ST_Distance`, `ST_Length`, `ST_Intersects`)
- Feature-gating exists and returns 501 when disabled:
  - `internal/handlers/collection_read.go`
  - `internal/handlers/geospatial_error.go`
  - `geospatial.go` (`EnableGeospatial()` and backend capability checks)
- Compliance server does not enable geospatial in startup path:
  - `cmd/complianceserver/main.go`

### Runtime Evidence
- Compliance skip reasons from verbose run:
  - `/tmp/compliance_verbose.log` includes skips such as:
    - `Reason: geo.distance not implemented (optional feature)`
    - `Reason: geo.length not implemented (optional feature)`
    - `Reason: geo.intersects not implemented (optional feature)`
- Direct reproducible request against compliance server:
  - `GET /Products?$filter=geo.distance(...) lt 10000`
  - Observed: `HTTP/1.1 501 Not Implemented`
  - Body: `"Geospatial features not enabled"`

### Verdict
- **Compliant (for unsupported feature signaling), feature gap in default profile.**
- The service correctly returns `501` for unsupported canonical geospatial functionality, but geospatial behavior is absent unless explicitly enabled and backed by spatial DB support.

### Gaps/Risks
- Interop risk for clients expecting canonical geospatial filters in the default deployment profile.
- SQLite profile requires extra spatial extension support; otherwise geo remains unavailable.

### Next Actions
- If geospatial support is a product requirement, make it explicit in service bootstrap and deployment docs:
  - call `EnableGeospatial()`
  - ensure backend extensions (e.g., SpatiaLite/PostGIS) are present
- Add a compliance-suite matrix run where geospatial is enabled and verify full geospatial function behavior (not just 501 fallback).

---

## Finding 2: ETag behavior exists, but optimistic concurrency metadata is not auto-declared for `odata:"etag"` properties

### Scope
- Behavior: conditional request readiness and metadata declaration
- Version context: observed under 4.01 responses; spec requirements shared with 4.0 semantics for this area

### Spec Evidence
- Protocol: services MAY require optimistic concurrency; services SHOULD announce this via `Core.OptimisticConcurrency` (and related capability annotations):
  - https://docs.oasis-open.org/odata/odata/v4.01/os/part1-protocol/odata-v4.01-os-part1-protocol.html (11.4.1.1)
- Protocol: presence of ETag alone does not imply optimistic concurrency is required.

### Implementation Evidence
- `odata:"etag"` maps to internal ETag property metadata:
  - `internal/metadata/analyzer.go` (`processODataTagPart`, `etag` case)
- Runtime ETag handling is implemented in read/write handlers:
  - `internal/handlers/entity_read.go`, `internal/handlers/entity_write.go`, `internal/etag/etag.go`
- Annotation constant exists, but no automatic emission was found for entity-level `Core.OptimisticConcurrency` solely from `odata:"etag"`:
  - `internal/metadata/annotations.go`
- API allows manual annotation registration:
  - `Service.RegisterEntityAnnotation(...)` in `odata.go`

### Runtime Evidence
- Compliance-suite skip:
  - `/tmp/compliance_verbose.log`:
    - `Reason: Products entity type does not declare concurrency metadata; skipping ETag requirement`
- Metadata probe:
  - `$metadata` includes `<Property Name="Version" ...>` and `Core.Computed` annotation on `Product/Version`
  - `$metadata` does **not** include `ConcurrencyMode="Fixed"` or `Core.OptimisticConcurrency` for `ComplianceService.Product`
- Entity probe:
  - `GET /Products(<id>)` returns `Etag: W/...` and `@odata.etag`
- Update probe:
  - `PATCH /Products(<id>)` without `If-Match` returns `204 No Content` (resource does not require optimistic concurrency)

### Verdict
- **Compliant, with metadata signaling gap (SHOULD-level interoperability issue).**
- Behavior is internally consistent: ETags are emitted and conditional headers work, but optimistic concurrency is not declared for `Products` in metadata by default.

### Gaps/Risks
- Generic clients relying on metadata to discover concurrency policies may under-utilize conditional request support.
- Compliance tests that gate on metadata declaration may skip stronger ETag assertions.

### Next Actions
- Optionally auto-emit entity-level `Org.OData.Core.V1.OptimisticConcurrency` when an ETag property is tagged.
- Or register it explicitly in service setup for compliance profiles.

---

## Finding 3: Decimal precision skip is caused by test payload/model mismatch, not confirmed library non-compliance

### Scope
- Behavior: numeric boundary test `test_decimal_precision`
- Version context: v4.0 test suite execution

### Spec Evidence
- JSON format numeric representation:
  - Edm.Int64 and Edm.Decimal as strings requires `IEEE754Compatible=true`.
  - Otherwise numbers MUST be JSON numbers.
  - https://docs.oasis-open.org/odata/odata-json-format/v4.01/os/odata-json-format-v4.01-os.html (section 3.2 and 4.1)
- Primitive values section: numeric primitives are JSON numbers (except `-INF`, `INF`, `NaN`).

### Implementation Evidence
- Compliance test posts `Price` as string in `test_decimal_precision`:
  - `compliance-suite/tests/v4_0/5.1.1.5_numeric_boundary_tests.go`
- Compliance server `Product.Price` is `float64`, which maps to `Edm.Double` in metadata:
  - `cmd/complianceserver/entities/product.go`

### Runtime Evidence
- Compliance log:
  - `/tmp/compliance_verbose.log`:
    - `⊘ SKIP: Decimal type preserves precision`
    - `Reason: Product creation failed: 400`
- Direct POST with string `Price`:
  - `HTTP/1.1 400 Bad Request`
  - `json: cannot unmarshal string into Go struct field Product.Price of type float64`
- Direct POST with numeric `Price` succeeds:
  - `HTTP/1.1 201 Created`
- `$metadata` confirms `Price` is `Edm.Double`, not `Edm.Decimal`.

### Verdict
- **Inconclusive for a library Decimal defect; likely compliance-test/model mismatch.**
- The failing shape tests string input against an `Edm.Double` field without IEEE754-compatible string-number negotiation.

### Gaps/Risks
- Current numeric-boundary test may overstate Decimal coverage while actually exercising a Double field.
- Potential blind spot for true Edm.Decimal request/response precision guarantees in compliance profile.

### Next Actions
- Add/convert compliance entity property to true `Edm.Decimal` (e.g., decimal type mapping) and test Decimal precision there.
- If string-encoded decimal is intended, include `IEEE754Compatible=true` in media type and align model/type accordingly.

---

## Consolidated Verdict
- No hard protocol violations were reproduced in the investigated areas.
- The main actionable gaps are:
  - missing out-of-the-box geospatial functionality in default compliance profile (with correct 501 fallback),
  - missing optimistic concurrency metadata declaration for ETag-tagged entities (SHOULD-level interoperability gap),
  - compliance test/model mismatch for Decimal precision scenario.

## Commands Executed (Representative)
- Full/verbose compliance analysis (previous runs):
  - `go run . -verbose` (from `compliance-suite`)
  - log extraction from `/tmp/compliance_verbose.log`
- Targeted rerun:
  - `go run . -version 4.0 -pattern numeric_boundary -verbose`
- Targeted runtime probes (compliance server on port 9095):
  - `GET /$metadata`
  - `GET /Products?$filter=geo.distance(...)`
  - `GET /Products(<id>)`
  - `PATCH /Products(<id>)` with/without `If-Match`
  - `POST /Products` with string vs numeric `Price`
