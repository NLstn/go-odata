## Plan: Multi-Storage Write-Behind Cache Foundation

## Implementation Progress
- [x] Phase 0 scope/invariants documented in plan and kept explicit
- [x] Phase 1 storage abstraction introduced with no behavior change
- [x] Handler read/write flow refactored to use storage abstraction
- [x] Service wiring injects storage implementation with backward-compatible defaults
- [x] Phase 1 verification gates run (`gofmt`, `golangci-lint`, `go test`, `go build`)
- [x] Phase 2 local in-memory cache backend introduced (`LocalCacheStorage` over `DBStorage`)
- [x] Entity and collection cache-key strategies implemented (canonical entity tuple + normalized query options)
- [x] Lazy cache population on miss implemented for entity/collection/count reads
- [x] Startup warm hook added for configured entity sets (`ServiceConfig.Cache.WarmEntitySets`)
- [x] Post-commit cache convergence hook implemented (coarse collection invalidation + entity upsert/delete)
- [x] Phase 2 verification gates run (`gofmt`, `golangci-lint`, `go test`, `go build`)
- [x] Phase 3 durable write-behind queue added (DB-backed state + retry/backoff worker + poison handling)
- [x] Post-commit enqueue integration added via `recordChange` with resilient failure semantics (logs on enqueue/apply errors)
- [x] Phase 3 verification gates run (`gofmt`, `golangci-lint`, `go test ./...`, `go build ./...`)
- [x] Phase 4 DB-backed invalidation/change-log table added (`_odata_cache_invalidation_events`) with checkpoint table (`_odata_cache_invalidation_checkpoints`)
- [x] Phase 4 per-instance poller added with offset checkpointing and idempotent local cache replay hooks
- [x] Phase 4 reconciliation refresh mode added for healing missed events
- [x] Phase 4 verification gates run (`gofmt`, `golangci-lint`, `go test ./...`, `go build ./...`)
- [x] Phase 5 config surface extended with cache memory limits (`MaxEntityEntries`, `MaxCollectionEntries`, `MaxCountEntries`)
- [x] Phase 5 write-behind queue limit added (`WriteBehind.MaxQueueSize`) and enforced at enqueue time
- [x] Phase 5 configuration validation added for invalid negative limits while preserving disabled-by-default behavior
- [x] Phase 5 docs updated (`documentation/advanced-features.md`, `documentation/testing.md`)
- [x] Phase 5 verification gates run (`gofmt`, `golangci-lint`, `go test ./...`, `go build ./...`)
- [x] Phase 6 integration tests added for queue visibility, multi-instance convergence, and concurrent read/write shutdown behavior (`test/cache_write_behind_integration_test.go`)
- [x] Phase 6 observability/health visibility checks added for queue backlog/completion and invalidation event presence via DB-backed state tables
- [x] Phase 6 verification gates run (`gofmt -w $(git ls-files '*.go')`, `golangci-lint run ./...`, `go test ./...`, `go build ./...`)
- [x] Phase 6 targeted compliance smoke run (`cd compliance-suite && go run . -version 4.0 -pattern content_type`)

### Progress Notes
- 2026-03-06: Started Phase 1 implementation with storage abstraction as first code change.
- 2026-03-06: Added `handlers.Storage` + `DBStorage`, routed entity/collection reads and CRUD primitives through storage seam, and validated with lint/test/build.
- 2026-03-06: Added `handlers.LocalCacheStorage` with entity/collection/count caching, canonical keying, cache metadata (`version marker`, `updated-at`, `dirty`, `pending op ID`), lazy fill, and optional warm-up.
- 2026-03-06: Reused change-event finalization (`recordChange`) as post-commit cache notification point to avoid cache mutation inside transactions.
- 2026-03-06: Added `ServiceConfig.Cache` and service wiring to enable local cache storage with backward-compatible defaults (disabled unless enabled).
- 2026-03-06: Added unit tests for cache reads, collection invalidation/entity upsert behavior, and warm-up path in `internal/handlers/local_cache_storage_test.go`.
- 2026-03-06: Ran quality gates successfully for Phase 2 foundation (`gofmt`, `golangci-lint`, `go test ./...`, `go build ./...`).
- 2026-03-06: Added durable write-behind queue model/worker in `internal/handlers/write_behind_queue.go` with DB persistence table (`_odata_write_behind_queue`), job leasing, exponential backoff retries, idempotency dedupe keying, and poison-item behavior after max retries.
- 2026-03-06: Wired queue lifecycle/config through `ServiceConfig.Cache.WriteBehind` and `Service.Close`, with compatibility validation that write-behind requires cache enablement.
- 2026-03-06: Integrated post-commit enqueue at `EntityHandler.recordChange(ctx, ...)` and added tests for enqueue integration, worker apply path, and retry-to-poison behavior.
- 2026-03-06: Ran quality gates successfully for Phase 3 foundation (`gofmt`, `golangci-lint`, `go test ./...`, `go build ./...`).
- 2026-03-06: Added DB-backed cache invalidation log/poller foundation in `internal/handlers/cache_invalidation.go`, including event append, per-instance checkpointing, and startup/shutdown lifecycle.
- 2026-03-06: Added `StorageChangeReplayer` with `LocalCacheStorage.ReplayEntityChange(...)` and wired poller-driven replay for cross-instance cache convergence.
- 2026-03-06: Wired service config `Cache.Consistency` and handler/apply integration so direct write-through and write-behind DB apply both append invalidation events.
- 2026-03-06: Added tests for poller replay/skip-own-event behavior and write-behind invalidation append path (`internal/handlers/cache_invalidation_test.go`, `internal/handlers/write_behind_queue_test.go`).
- 2026-03-06: Ran quality gates successfully for Phase 4 foundation (`gofmt`, `golangci-lint`, `go test ./...`, `go build ./...`).
- 2026-03-06: Added periodic reconciliation worker wired via `Cache.Consistency.ReconcileInterval` with optional `ReconcileEntitySets` targeting and graceful shutdown in `Service.Close`.
- 2026-03-06: Added `handlers.StorageReconciler` and `LocalCacheStorage.ReconcileEntitySet(...)` for forced refresh independent of startup warm-set filtering.
- 2026-03-06: Extended `ServiceConfig.Cache` with bounded local cache options and added enqueue-cap limit via `ServiceConfig.Cache.WriteBehind.MaxQueueSize`.
- 2026-03-06: Implemented local cache capacity enforcement/eviction for entity, collection, and count caches in `internal/handlers/local_cache_storage.go`.
- 2026-03-06: Added tests for cache limit eviction, queue size enforcement, and config validation in `internal/handlers/local_cache_storage_test.go`, `internal/handlers/write_behind_queue_test.go`, and `service_config_test.go`.
- 2026-03-06: Documented cache/write-behind/consistency configuration and related testing guidance in `documentation/advanced-features.md` and `documentation/testing.md`.
- 2026-03-06: Ran quality gates successfully for Phase 5 foundation (`gofmt`, `golangci-lint`, `go test ./...`, `go build ./...`).
- 2026-03-06: Added Phase 6 integration tests in `test/cache_write_behind_integration_test.go` covering write-behind queue progress visibility, cross-instance cache convergence, and concurrent read/write with shutdown drain.
- 2026-03-06: Validated queue/invalidation operational visibility by asserting DB-backed queue state transitions (`_odata_write_behind_queue`) and invalidation event append behavior (`_odata_cache_invalidation_events`).
- 2026-03-06: Ran Phase 6 verification gates successfully (`gofmt -w $(git ls-files '*.go')`, `golangci-lint run ./...`, `go test ./...`, `go build ./...`).
- 2026-03-06: Ran targeted OData compliance-suite smoke successfully (`go run . -version 4.0 -pattern content_type`, 5/5 tests passing).

Introduce a storage abstraction in front of existing GORM access, then add a local-memory cache backend with async write-behind to DB and DB-backed cross-instance invalidation/event consumption. Phase 1 will include entity GET and collection GET cache reads, local-first writes with durable retry queue, startup warming for selected entity sets, and eventual consistency across instances using DB as source of truth.

**Steps**
1. Phase 0: Baseline and scope lock
2. Confirm and document phase-1 scope boundaries in code comments/docs: include `GET /EntitySet(key)`, collection GET (with query cache keys), and CRUD write-through into local cache with async DB persistence; exclude navigation-property caching and distributed pub/sub transport for now. This avoids hidden scope drift before refactor.
3. Capture explicit invariants: DB remains system of record, cache is authoritative only for serving local reads/writes between sync cycles, and all write-behind operations are idempotent and replay-safe.
4. Phase 1: Introduce storage abstraction without behavior change
5. Add a narrow internal storage interface layer for read/write primitives used by handlers (entity fetch, collection fetch/count, create/update/delete, plus cache invalidation hooks). *blocks steps 6-11*
6. Refactor current direct GORM calls in handler flow to call the interface, keeping a default `DBStorage` implementation that preserves current behavior; no cache enabled yet. *depends on 5*
7. Add constructor wiring in service initialization so storage implementation is injected once and reused by all entity handlers; keep backward-compatible defaults.
8. Phase 2: Add local memory cache backend (entity + query-result)
9. Implement `LocalCacheStorage` composition over `DBStorage` with cache key strategy:
10. Entity key: entity set + canonical key tuple.
11. Collection key: entity set + normalized OData query options + tenant/base-path relevant dimensions.
12. Store payload + metadata (version marker / updated-at / dirty flag / pending op ID) for write-behind tracking.
13. Add startup warming for configured entity sets and lazy population for all others (cache miss -> DB load -> cache fill). *parallel with step 10 once interfaces exist*
14. Add invalidation rules for collection caches on writes affecting an entity set (coarse invalidation first), and direct entity-key updates for point writes.
15. Phase 3: Async write-behind pipeline and durability
16. Reuse transaction/change-event finalization points to enqueue durable sync tasks after request-local cache mutation succeeds.
17. Implement background sync worker with retry/backoff, dedup/idempotency key, poison-item handling, and graceful shutdown drain semantics.
18. Persist write-behind queue state in DB tables to survive restarts and support multiple instances; include `pending`, `in_progress`, `failed`, `retry_at`, `last_error`, and correlation fields.
19. Define failure semantics: request succeeds after local cache write + successful enqueue; DB write failures are retried asynchronously and surfaced via metrics/logging/health endpoints.
20. Phase 4: Cross-instance consistency (DB-driven)
21. Add DB-backed invalidation/change-log table as single source of truth for cache convergence (append event on successful DB apply).
22. Implement per-instance poller/consumer that reads unseen invalidation events and applies local cache updates/invalidation deterministically.
23. Add lease/offset mechanism (or monotonic sequence checkpoint) to avoid duplicate processing and ensure at-least-once handling with idempotent handlers.
24. Provide reconciliation mode: periodic full/targeted refresh for configured sets to heal missed events.
25. Phase 5: Configuration surface and ergonomics
26. Extend service configuration with cache/write-behind/consistency config blocks:
27. Cache enablement, memory limits, TTL, warming targets.
28. Write-behind retry policy and queue limits.
29. Cross-instance poll interval and reconciliation cadence.
30. Validate incompatible combinations and preserve zero-config backward compatibility (feature disabled by default).
31. Phase 6: Verification and hardening
32. Add integration tests for local-first reads/writes, async DB lag behavior, retry recovery, startup warm + lazy fill, and collection invalidation behavior.
33. Add multi-instance integration tests (two service instances + shared DB) proving eventual convergence after writes from either instance.
34. Add race/concurrency tests for simultaneous writes/reads and worker shutdown.
35. Add observability checks (metrics/traces/log fields) and health visibility for queue backlog and sync failures.
36. Run required quality gates (`gofmt -w .`, `golangci-lint run ./...`, `go test ./...`, `go build ./...`) and targeted compliance-suite smoke to confirm no protocol regressions.

**Relevant files**
- `/workspaces/go-odata/odata.go` — Extend `ServiceConfig`, storage wiring in `NewServiceWithConfig`, lifecycle startup/shutdown hooks for workers/pollers.
- `/workspaces/go-odata/internal/handlers/entity.go` — Inject storage abstraction into `EntityHandler`.
- `/workspaces/go-odata/internal/handlers/entity_read.go` — Route entity reads through storage abstraction and cache lookup path.
- `/workspaces/go-odata/internal/handlers/collection_read.go` — Route collection reads through storage abstraction/query-cache path.
- `/workspaces/go-odata/internal/handlers/collection_executor.go` — Hook fetch/count execution to storage abstraction for query caching.
- `/workspaces/go-odata/internal/handlers/collection_write.go` — Enqueue write-behind events after local writes/transaction completion.
- `/workspaces/go-odata/internal/handlers/entity_write.go` — Local cache mutation + enqueue semantics for update/delete paths.
- `/workspaces/go-odata/internal/handlers/helpers.go` — Reuse/extend `finalizeChangeEvents` integration point for async sync enqueue.
- `/workspaces/go-odata/internal/handlers/transaction.go` — Reuse pending-event context mechanics for durable sync task accumulation.
- `/workspaces/go-odata/internal/trackchanges/` — Reuse sequence/change tracking ideas for cross-instance event consumption/checkpointing.
- `/workspaces/go-odata/internal/async/manager.go` — Reference existing retry/lifecycle patterns for background workers.
- `/workspaces/go-odata/test/` — Add integration tests (`cache_storage_test.go`, `write_behind_test.go`, `multi_instance_consistency_test.go`).
- `/workspaces/go-odata/compliance-suite/tests/v4_0/` — Add focused non-regression tests where caching could affect required headers/consistency semantics.
- `/workspaces/go-odata/documentation/advanced-features.md` and `/workspaces/go-odata/documentation/testing.md` — Document configuration and operational behavior.

**Verification**
1. Unit tests: storage interface contract tests, cache key normalization tests, invalidation tests, worker retry/idempotency tests.
2. Integration tests: local-first write/read behavior, delayed DB sync, restart recovery from persisted queue, startup warming + lazy fill.
3. Multi-instance integration: run two service instances against shared DB, verify instance B observes writes from A via DB invalidation log within configured convergence window.
4. Failure-injection tests: force DB outage during write-behind, confirm retries/backoff and eventual recovery without data loss.
5. Performance smoke: compare baseline read latency/QPS with cache enabled for hot entities and hot collection queries.
6. Quality gates: run `gofmt -w .`, `golangci-lint run ./...`, `go test ./...`, `go build ./...`.

**Decisions**
- Chosen consistency approach: DB-driven invalidation/change-log (single source of truth).
- Chosen write semantics: acknowledge after local mutation + durable enqueue, then async DB apply with retries.
- Chosen cache population: warm selected entity sets on startup; lazy-load others.
- Chosen phase-1 scope: include entity and collection query caching; defer navigation-property cache specialization.
- Included in phase 1: local memory cache and DB-backed convergence.
- Excluded from phase 1: direct instance-to-instance pub/sub transport, strict linearizable reads across instances, and advanced per-query fine-grained invalidation.

**Further Considerations**
1. Queue durability location recommendation: reuse primary DB for sync queue/event log to reduce moving parts (Option A) versus dedicated queue DB for isolation (Option B).
2. Collection invalidation strategy recommendation: start coarse (invalidate all cached queries for touched entity set) then optimize to predicate-aware invalidation later.
3. Operational defaults recommendation: conservative poll interval and retry backoff with explicit metrics-driven tuning after load tests.