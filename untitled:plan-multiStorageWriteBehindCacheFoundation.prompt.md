## Plan: Multi-Storage Write-Behind Cache Foundation

## Implementation Progress
- [x] Phase 0 scope/invariants documented in plan and kept explicit
- [x] Phase 1 storage abstraction introduced with no behavior change
- [x] Handler read/write flow refactored to use storage abstraction
- [x] Service wiring injects storage implementation with backward-compatible defaults
- [x] Phase 1 verification gates run (`gofmt`, `golangci-lint`, `go test`, `go build`)

### Progress Notes
- 2026-03-06: Started Phase 1 implementation with storage abstraction as first code change.
- 2026-03-06: Added `handlers.Storage` + `DBStorage`, routed entity/collection reads and CRUD primitives through storage seam, and validated with lint/test/build.

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