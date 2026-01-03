package odata

// Package odata provides functionality for building OData services in Go.
// This library implements the OData v4.01 specification, allowing you to define
// Go structs representing entities and automatically handling the necessary OData
// protocol logic.
//
// # Hooks
//
// Entity types can optionally implement hook methods to inject custom business logic
// at specific points in the request lifecycle. All hook methods are optional and are
// automatically discovered via reflection.
//
// ## Lifecycle Hooks
//
// Implement these methods on your entity type to handle lifecycle events:
//
//	func (p *Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error
//	func (p *Product) ODataAfterCreate(ctx context.Context, r *http.Request) error
//	func (p *Product) ODataBeforeUpdate(ctx context.Context, r *http.Request) error
//	func (p *Product) ODataAfterUpdate(ctx context.Context, r *http.Request) error
//	func (p *Product) ODataBeforeDelete(ctx context.Context, r *http.Request) error
//	func (p *Product) ODataAfterDelete(ctx context.Context, r *http.Request) error
//
// Hooks have access to the request context and HTTP request, allowing you to validate
// data, apply business rules, log changes, or check authentication/authorization.
// Returning an error from a Before* hook aborts the operation and returns the error
// to the client. Errors from After* hooks are logged but don't affect the response.
//
// ## Read Hooks
//
// Implement these methods to customize query behavior and response data:
//
//	func (p Product) ODataBeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)
//	func (p Product) ODataAfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error)
//	func (p Product) ODataBeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)
//	func (p Product) ODataAfterReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, error)
//
// Before* read hooks return GORM scopes that are applied before OData query options
// ($filter, $orderby, $top, $skip). Use them for authorization filters and eager-loading.
// After* read hooks receive the final results after all query processing and can redact
// sensitive data or append computed fields.
//
// See the EntityHook interface documentation for detailed hook descriptions and
// the documentation directory for comprehensive examples and use cases.

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"sync"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/async"
	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/observability"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/service/operations"
	servrouter "github.com/nlstn/go-odata/internal/service/router"
	servruntime "github.com/nlstn/go-odata/internal/service/runtime"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// KeyGenerator describes a function that produces a key value for new entities.
type KeyGenerator func(context.Context) (interface{}, error)

// EntityHook defines optional lifecycle hooks that entity types can implement to inject
// custom business logic at specific points in the request lifecycle.
//
// All hook methods are optional. If a method exists on an entity type, it will be
// automatically detected and called at the appropriate time.
//
// Hook methods should be defined on the entity type. While you can use pointer or
// value receivers, ensure consistency with how your entity is used elsewhere.
//
// # Lifecycle Hooks
//
// ODataBeforeCreate is called before a new entity is created via POST request. Return an
// error to prevent creation; the error will be returned to the client. Use this hook
// for validation, authorization checks, or setting default values.
//
//	func (p *Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
//	    if p.Price < 0 {
//	        return fmt.Errorf("price cannot be negative")
//	    }
//	    p.CreatedAt = time.Now()
//	    return nil
//	}
//
// ODataAfterCreate is called after a new entity has been successfully created and persisted
// to the database. Errors returned are logged but do not affect the response to the client.
// Use this hook for audit logging or triggering external notifications.
//
//	func (p *Product) ODataAfterCreate(ctx context.Context, r *http.Request) error {
//	    log.Printf("Product created: %s", p.Name)
//	    return nil
//	}
//
// ODataBeforeUpdate is called before an entity is updated via PATCH or PUT request. Return
// an error to prevent the update. Use this for validation, authorization, or tracking
// what changed.
//
//	func (p *Product) ODataBeforeUpdate(ctx context.Context, r *http.Request) error {
//	    if p.Price < 0 {
//	        return fmt.Errorf("price cannot be negative")
//	    }
//	    return nil
//	}
//
// ODataAfterUpdate is called after an entity has been successfully updated. Errors are logged
// but do not affect the response. Use this for audit logging.
//
// ODataBeforeDelete is called before an entity is deleted via DELETE request. Return an error
// to prevent deletion. Use this for authorization checks or protecting special entities.
//
//	func (p *Product) ODataBeforeDelete(ctx context.Context, r *http.Request) error {
//	    if p.IsProtected {
//	        return fmt.Errorf("cannot delete protected products")
//	    }
//	    return nil
//	}
//
// ODataAfterDelete is called after an entity has been successfully deleted. Errors are logged
// but do not affect the response. Use this for audit logging or cleanup.
//
// # Accessing the Transaction
//
// Write hooks (Before/After Create/Update/Delete) execute inside a shared GORM transaction.
// Use TransactionFromContext to participate in the same transaction:
//
//	func (p *Product) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
//	    tx, ok := TransactionFromContext(ctx)
//	    if !ok {
//	        return fmt.Errorf("transaction unavailable")
//	    }
//
//	    audit := AuditLog{ProductID: p.ID, Action: "CREATE"}
//	    if err := tx.Create(&audit).Error; err != nil {
//	        return err
//	    }
//	    return nil
//	}
//
// Any error returned aborts the operation and rolls back all changes made via
// the shared transaction.
type EntityHook interface {
	// ODataBeforeCreate is called before a new entity is created via POST.
	// Return an error to prevent the creation and return that error to the client.
	ODataBeforeCreate(ctx context.Context, r *http.Request) error

	// ODataAfterCreate is called after a new entity has been successfully created.
	// Any error returned will be logged but won't affect the response to the client.
	ODataAfterCreate(ctx context.Context, r *http.Request) error

	// ODataBeforeUpdate is called before an entity is updated via PATCH or PUT.
	// Return an error to prevent the update and return that error to the client.
	ODataBeforeUpdate(ctx context.Context, r *http.Request) error

	// ODataAfterUpdate is called after an entity has been successfully updated.
	// Any error returned will be logged but won't affect the response to the client.
	ODataAfterUpdate(ctx context.Context, r *http.Request) error

	// ODataBeforeDelete is called before an entity is deleted via DELETE.
	// Return an error to prevent the deletion and return that error to the client.
	ODataBeforeDelete(ctx context.Context, r *http.Request) error

	// ODataAfterDelete is called after an entity has been successfully deleted.
	// Any error returned will be logged but won't affect the response to the client.
	ODataAfterDelete(ctx context.Context, r *http.Request) error
}

// ReadHook defines optional read hooks that entity types can implement to customize
// query behavior and response data.
//
// All read hook methods are optional. Unlike EntityHook which is an interface,
// these are standalone methods that are discovered via reflection on your entity type.
//
// # Before Read Hooks
//
// ODataBeforeReadCollection is called before fetching a collection. It returns GORM scopes
// that are applied before OData query options ($filter, $orderby, $top, $skip).
//
//	func (p Product) ODataBeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
//	    // Apply tenant filter
//	    tenantID := r.Header.Get("X-Tenant-ID")
//	    if tenantID == "" {
//	        return nil, fmt.Errorf("missing tenant header")
//	    }
//	    return []func(*gorm.DB) *gorm.DB{
//	        func(db *gorm.DB) *gorm.DB { return db.Where("tenant_id = ?", tenantID) },
//	    }, nil
//	}
//
// ODataBeforeReadEntity is called before fetching a single entity. It works the same as
// ODataBeforeReadCollection but for individual entity reads. Return scopes for authorization
// filters or eager-loading related data.
//
// # After Read Hooks
//
// ODataAfterReadCollection is called after fetching a collection. It receives the results
// after all query processing and can redact sensitive data or transform the response.
// Return nil, nil to keep the original response.
//
//	func (p Product) ODataAfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error) {
//	    products, ok := results.([]Product)
//	    if !ok {
//	        return results, nil
//	    }
//	    // Redact cost price for non-privileged users
//	    if !isPrivileged(r) {
//	        for i := range products {
//	            products[i].CostPrice = 0
//	        }
//	    }
//	    return products, nil
//	}
//
// ODataAfterReadEntity is called after fetching a single entity. It works the same as
// ODataAfterReadCollection but for individual entity reads.
//
// # Hook Execution Order
//
// For any operation, hooks execute in this order:
//
//	Create: ODataBeforeCreate -> INSERT -> ODataAfterCreate
//	Update: ODataBeforeUpdate -> UPDATE -> ODataAfterUpdate
//	Delete: ODataBeforeDelete -> DELETE -> ODataAfterDelete
//	Read:   ODataBeforeReadCollection/ODataBeforeReadEntity -> SELECT + OData options -> ODataAfterReadCollection/ODataAfterReadEntity
type ReadHook interface {
	// ODataBeforeReadCollection is called before fetching a collection.
	// Return GORM scopes to apply before OData options ($filter, $orderby, etc).
	// These scopes are ideal for authorization filters and eager-loading.
	ODataBeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)

	// ODataAfterReadCollection is called after fetching a collection.
	// It receives the results after all query processing and can redact or transform them.
	// Return nil, nil to keep the original response.
	ODataAfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error)

	// ODataBeforeReadEntity is called before fetching a single entity.
	// Return GORM scopes to apply before OData options. Ideal for authorization and eager-loading.
	ODataBeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error)

	// ODataAfterReadEntity is called after fetching a single entity.
	// It receives the entity after all query processing and can redact or transform it.
	// Return nil, nil to keep the original response.
	ODataAfterReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, error)
}

// ServiceConfig controls optional service behaviours.
type ServiceConfig struct {
	// PersistentChangeTracking enables database-backed change tracking history.
	PersistentChangeTracking bool
}

// DefaultNamespace is used when no explicit namespace is configured for the service.
const DefaultNamespace = "ODataService"

// Service represents an OData service that can handle multiple entities.
type Service struct {
	// db holds the GORM database connection
	db *gorm.DB
	// entities holds registered entity metadata keyed by entity set name
	entities map[string]*metadata.EntityMetadata
	// handlers holds entity handlers keyed by entity set name
	handlers map[string]*handlers.EntityHandler
	// metadataHandler handles metadata document requests
	metadataHandler *handlers.MetadataHandler
	// serviceDocumentHandler handles service document requests
	serviceDocumentHandler *handlers.ServiceDocumentHandler
	// batchHandler handles batch requests
	batchHandler *handlers.BatchHandler
	// actions holds registered actions keyed by action name (supports overloads)
	actions map[string][]*actions.ActionDefinition
	// functions holds registered functions keyed by function name (supports overloads)
	functions map[string][]*actions.FunctionDefinition
	// namespace used for metadata generation
	namespace string
	// deltaTracker tracks entity changes for change tracking requests
	deltaTracker *trackchanges.Tracker
	// changeTrackingPersistent indicates whether tracker state is backed by the database
	changeTrackingPersistent bool
	// router handles HTTP routing for the service
	router *servrouter.Router
	// operationsHandler orchestrates action and function execution
	operationsHandler *operations.Handler
	// runtime coordinates HTTP handling and async dispatch
	runtime *servruntime.Runtime
	// asyncManager manages asynchronous requests when enabled
	asyncManager *async.Manager
	// asyncConfig stores the configuration for async processing
	asyncConfig *AsyncConfig
	// asyncQueue limits concurrent async jobs when configured
	asyncQueue chan struct{}
	// asyncMonitorPrefix is the normalized monitor path prefix
	asyncMonitorPrefix string
	// logger is used for structured logging throughout the service
	logger *slog.Logger
	// policy handles authorization decisions when configured
	policy auth.Policy
	// ftsManager manages full-text search functionality for SQLite
	ftsManager *query.FTSManager
	// keyGenerators maintains registered key generator functions by name
	keyGenerators   map[string]KeyGenerator
	keyGeneratorsMu sync.RWMutex
	// defaultMaxTop is the default maximum number of results to return if no explicit $top is set
	defaultMaxTop *int
	// observability holds the observability configuration (tracing, metrics)
	observability *observability.Config
}

// NewService creates a new OData service instance with database connection.
func NewService(db *gorm.DB) *Service {
	service, err := NewServiceWithConfig(db, ServiceConfig{})
	if err != nil {
		panic(err)
	}
	return service
}

// NewServiceWithConfig creates a new OData service instance with additional configuration.
func NewServiceWithConfig(db *gorm.DB, cfg ServiceConfig) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("odata: database handle is required")
	}

	entities := make(map[string]*metadata.EntityMetadata)
	handlersMap := make(map[string]*handlers.EntityHandler)
	logger := slog.Default()

	var (
		tracker *trackchanges.Tracker
		err     error
	)
	if cfg.PersistentChangeTracking {
		tracker, err = trackchanges.NewTrackerWithDB(db)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize persistent change tracker: %w", err)
		}
	} else {
		tracker = trackchanges.NewTracker()
	}

	// Initialize FTS manager for SQLite full-text search
	ftsManager := query.NewFTSManager(db)

	s := &Service{
		db:                       db,
		entities:                 entities,
		handlers:                 handlersMap,
		metadataHandler:          handlers.NewMetadataHandler(entities),
		serviceDocumentHandler:   handlers.NewServiceDocumentHandler(entities, logger),
		actions:                  make(map[string][]*actions.ActionDefinition),
		functions:                make(map[string][]*actions.FunctionDefinition),
		namespace:                DefaultNamespace,
		deltaTracker:             tracker,
		changeTrackingPersistent: cfg.PersistentChangeTracking,
		logger:                   logger,
		ftsManager:               ftsManager,
		keyGenerators:            make(map[string]KeyGenerator),
	}
	s.metadataHandler.SetNamespace(DefaultNamespace)
	s.metadataHandler.SetPolicy(s.policy)
	s.serviceDocumentHandler.SetPolicy(s.policy)
	s.operationsHandler = operations.NewHandler(s.actions, s.functions, s.handlers, s.entities, s.namespace, logger)
	// Initialize batch handler with reference to service
	s.batchHandler = handlers.NewBatchHandler(db, handlersMap, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.serveHTTP(w, r, false)
	}))
	s.router = servrouter.NewRouter(
		func(name string) (servrouter.EntityHandler, bool) {
			handler, ok := s.handlers[name]
			if !ok {
				return nil, false
			}
			return handler, true
		},
		s.serviceDocumentHandler.HandleServiceDocument,
		s.metadataHandler.HandleMetadata,
		s.batchHandler.HandleBatch,
		s.actions,
		s.functions,
		s.operationsHandler.HandleActionOrFunction,
		logger,
	)
	s.router.SetAsyncMonitor(s.asyncMonitorPrefix, s.asyncManager)
	s.runtime = servruntime.New(s.router, logger)

	if err := s.RegisterKeyGenerator("uuid", func(context.Context) (interface{}, error) {
		uuid, err := generateUUIDBytes()
		if err != nil {
			return nil, err
		}
		return uuid, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to register default key generator: %w", err)
	}

	return s, nil
}

func generateUUIDBytes() ([16]byte, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return [16]byte{}, err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return b, nil
}

// SetLogger sets a custom logger for the service.
// If not called, slog.Default() is used.
func (s *Service) SetLogger(logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	s.logger = logger
	s.router.SetLogger(logger)
	s.serviceDocumentHandler.SetLogger(logger)
	if s.operationsHandler != nil {
		s.operationsHandler.SetLogger(logger)
	}
	if s.runtime != nil {
		s.runtime.SetLogger(logger)
	}
	// Update logger for all existing handlers
	for _, handler := range s.handlers {
		handler.SetLogger(logger)
	}
}

// ObservabilityConfig configures observability features (tracing, metrics) for the service.
// All providers are optional; when nil, the corresponding feature is disabled with zero overhead.
type ObservabilityConfig struct {
	// TracerProvider provides the OpenTelemetry tracer for distributed tracing.
	// If nil, tracing is disabled.
	TracerProvider trace.TracerProvider

	// MeterProvider provides the OpenTelemetry meter for metrics collection.
	// If nil, metrics collection is disabled.
	MeterProvider metric.MeterProvider

	// ServiceName identifies this service in telemetry data.
	// Defaults to "odata-service" if not specified.
	ServiceName string

	// ServiceVersion is reported in telemetry attributes.
	ServiceVersion string

	// EnableDetailedDBTracing enables per-query database spans.
	// This can generate significant trace data; disabled by default.
	EnableDetailedDBTracing bool

	// EnableQueryOptionTracing adds spans for individual query option processing.
	// Note: This feature is not yet implemented and is reserved for future use.
	EnableQueryOptionTracing bool

	// EnableServerTiming enables the Server-Timing HTTP response header.
	// When enabled, timing metrics are added to responses for debugging in browser dev tools.
	// This uses the mitchellh/go-server-timing library to add timing information to HTTP responses.
	EnableServerTiming bool
}

// SetObservability configures OpenTelemetry-based observability for the service.
// This enables distributed tracing, metrics collection, and enhanced structured logging.
//
// When observability is configured:
//   - HTTP requests are automatically instrumented with traces and metrics
//   - Entity operations (CRUD) create spans with relevant attributes
//   - Database queries can optionally be traced (when EnableDetailedDBTracing is true)
//   - Batch operations and async jobs are instrumented
//   - Errors are recorded on spans and counted in metrics
//
// Example:
//
//	import (
//	    "go.opentelemetry.io/otel"
//	    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
//	    sdktrace "go.opentelemetry.io/otel/sdk/trace"
//	)
//
//	// Set up tracing
//	exporter, _ := otlptracehttp.New(ctx)
//	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
//	defer tp.Shutdown(ctx)
//
//	// Configure OData service
//	service := odata.NewService(db)
//	service.SetObservability(odata.ObservabilityConfig{
//	    TracerProvider: tp,
//	    ServiceName:    "my-odata-api",
//	    ServiceVersion: "1.0.0",
//	})
func (s *Service) SetObservability(cfg ObservabilityConfig) error {
	opts := []observability.Option{}

	if cfg.TracerProvider != nil {
		opts = append(opts, observability.WithTracerProvider(cfg.TracerProvider))
	}
	if cfg.MeterProvider != nil {
		opts = append(opts, observability.WithMeterProvider(cfg.MeterProvider))
	}
	if cfg.ServiceName != "" {
		opts = append(opts, observability.WithServiceName(cfg.ServiceName))
	}
	if cfg.ServiceVersion != "" {
		opts = append(opts, observability.WithServiceVersion(cfg.ServiceVersion))
	}
	if s.logger != nil {
		opts = append(opts, observability.WithLogger(s.logger))
	}
	if cfg.EnableDetailedDBTracing {
		opts = append(opts, observability.WithDetailedDBTracing())
	}
	if cfg.EnableQueryOptionTracing {
		opts = append(opts, observability.WithQueryOptionTracing())
	}
	if cfg.EnableServerTiming {
		opts = append(opts, observability.WithServerTiming())
	}

	obsCfg := observability.NewConfig(opts...)
	if err := obsCfg.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize observability: %w", err)
	}

	s.observability = obsCfg

	// Configure runtime with observability
	if s.runtime != nil {
		s.runtime.SetObservability(obsCfg)
	}

	// Configure handlers with observability
	for _, handler := range s.handlers {
		handler.SetObservability(obsCfg)
	}

	// Configure batch handler with observability
	if s.batchHandler != nil {
		s.batchHandler.SetObservability(obsCfg)
	}

	// Register GORM callbacks for detailed DB tracing if enabled
	if cfg.EnableDetailedDBTracing {
		if err := observability.RegisterGORMCallbacks(s.db, obsCfg); err != nil {
			return fmt.Errorf("failed to register GORM callbacks: %w", err)
		}
	}

	if s.logger != nil {
		s.logger.Info("Observability configured",
			"tracing_enabled", cfg.TracerProvider != nil,
			"metrics_enabled", cfg.MeterProvider != nil,
			"server_timing_enabled", cfg.EnableServerTiming,
			"service_name", cfg.ServiceName,
		)
	}

	return nil
}

// Observability returns the current observability configuration.
// Returns nil if observability is not configured.
func (s *Service) Observability() *observability.Config {
	return s.observability
}

// ServerTimingMetric represents a Server-Timing metric that tracks the duration
// of an operation for the Server-Timing HTTP response header.
// Use StartServerTiming or StartServerTimingWithDesc to create metrics.
type ServerTimingMetric = observability.ServerTimingMetric

// StartServerTiming starts a Server-Timing metric with the given name.
// The metric tracks the duration until Stop() is called, and appears in
// the Server-Timing HTTP response header when EnableServerTiming is true.
//
// Returns a metric that should be stopped when the timed operation completes.
// If server timing is not enabled or the context doesn't contain timing info,
// returns a no-op metric that is safe to call Stop() on.
//
// Example:
//
//	func myHandler(ctx context.Context) {
//	    metric := odata.StartServerTiming(ctx, "db-query")
//	    defer metric.Stop()
//	    // perform database operation
//	}
func StartServerTiming(ctx context.Context, name string) *ServerTimingMetric {
	return observability.StartServerTiming(ctx, name)
}

// StartServerTimingWithDesc starts a Server-Timing metric with a name and description.
// The description provides additional context in browser developer tools.
//
// Returns a metric that should be stopped when the timed operation completes.
// If server timing is not enabled or the context doesn't contain timing info,
// returns a no-op metric that is safe to call Stop() on.
//
// Example:
//
//	func myHandler(ctx context.Context) {
//	    metric := odata.StartServerTimingWithDesc(ctx, "cache", "Cache lookup")
//	    defer metric.Stop()
//	    // perform cache operation
//	}
func StartServerTimingWithDesc(ctx context.Context, name, description string) *ServerTimingMetric {
	return observability.StartServerTimingWithDesc(ctx, name, description)
}

// AsyncConfig controls asynchronous request processing behaviour for a Service.
type AsyncConfig struct {
	// MonitorPathPrefix is the URL path prefix where async job monitors are exposed.
	MonitorPathPrefix string
	// DefaultRetryInterval configures the Retry-After header returned for pending jobs.
	DefaultRetryInterval time.Duration
	// MaxQueueSize limits concurrently executing async jobs. Zero disables the limit.
	MaxQueueSize int
	// JobRetention controls how long completed jobs are retained for polling clients.
	// A zero duration applies async.DefaultJobRetention unless DisableRetention is true.
	JobRetention time.Duration
	// DisableRetention disables automatic removal of completed jobs from the backing store.
	DisableRetention bool
}

// EnableAsyncProcessing configures asynchronous request handling for the service.
func (s *Service) EnableAsyncProcessing(cfg AsyncConfig) error {
	normalized := cfg
	if normalized.MonitorPathPrefix == "" {
		normalized.MonitorPathPrefix = "/$async/jobs/"
	}
	if !strings.HasPrefix(normalized.MonitorPathPrefix, "/") {
		normalized.MonitorPathPrefix = "/" + normalized.MonitorPathPrefix
	}
	if !strings.HasSuffix(normalized.MonitorPathPrefix, "/") {
		normalized.MonitorPathPrefix += "/"
	}

	if s.asyncManager != nil {
		s.asyncManager.Close()
		s.asyncManager = nil
	}

	if s.runtime != nil {
		s.runtime.ConfigureAsync(nil, nil, "", 0)
	}

	managerOptions := make([]async.ManagerOption, 0, 1)
	if normalized.DisableRetention {
		managerOptions = append(managerOptions, async.WithRetentionDisabled())
	}

	mgr, err := async.NewManager(s.db, normalized.JobRetention, managerOptions...)
	if err != nil {
		return fmt.Errorf("failed to configure async processing: %w", err)
	}

	s.asyncManager = mgr
	s.asyncMonitorPrefix = normalized.MonitorPathPrefix
	cfgCopy := normalized
	s.asyncConfig = &cfgCopy

	if s.router != nil {
		s.router.SetAsyncMonitor(s.asyncMonitorPrefix, s.asyncManager)
	}

	if normalized.MaxQueueSize > 0 {
		s.asyncQueue = make(chan struct{}, normalized.MaxQueueSize)
	} else {
		s.asyncQueue = nil
	}

	if s.runtime != nil {
		s.runtime.ConfigureAsync(s.asyncManager, s.asyncQueue, s.asyncMonitorPrefix, s.asyncConfig.DefaultRetryInterval)
	}

	return nil
}

// AsyncManager exposes the current async manager instance for testing and monitoring.
func (s *Service) AsyncManager() *async.Manager {
	return s.asyncManager
}

// AsyncMonitorPrefix returns the configured monitor path prefix.
func (s *Service) AsyncMonitorPrefix() string {
	return s.asyncMonitorPrefix
}

// Close releases resources held by the service, including background managers.
// It is safe to call multiple times; subsequent calls have no effect.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}

	if s.asyncManager != nil {
		s.asyncManager.Close()
	}

	if s.router != nil {
		s.router.SetAsyncMonitor("", nil)
	}

	s.asyncManager = nil
	s.asyncConfig = nil
	s.asyncQueue = nil
	s.asyncMonitorPrefix = ""

	return nil
}

// RegisterKeyGenerator registers a key generator function under the provided name.
// Existing generators with the same name will be replaced.
func (s *Service) RegisterKeyGenerator(name string, generator KeyGenerator) error {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "" {
		return fmt.Errorf("key generator name cannot be empty")
	}
	if generator == nil {
		return fmt.Errorf("key generator '%s' cannot be nil", trimmed)
	}

	s.keyGeneratorsMu.Lock()
	if s.keyGenerators == nil {
		s.keyGenerators = make(map[string]KeyGenerator)
	}
	s.keyGenerators[trimmed] = generator
	s.keyGeneratorsMu.Unlock()

	metadata.RegisterKeyGeneratorName(trimmed)
	return nil
}

func (s *Service) resolveKeyGenerator(name string) (KeyGenerator, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	s.keyGeneratorsMu.RLock()
	defer s.keyGeneratorsMu.RUnlock()
	generator, ok := s.keyGenerators[trimmed]
	return generator, ok
}

// RegisterEntity registers an entity type with the OData service.
func (s *Service) RegisterEntity(entity interface{}) error {
	// Analyze the entity structure
	entityMetadata, err := metadata.AnalyzeEntity(entity)
	if err != nil {
		return fmt.Errorf("failed to analyze entity: %w", err)
	}

	if _, exists := s.entities[entityMetadata.EntitySetName]; exists {
		return fmt.Errorf("entity set '%s' is already registered", entityMetadata.EntitySetName)
	}
	if _, exists := s.handlers[entityMetadata.EntitySetName]; exists {
		return fmt.Errorf("entity handler for '%s' is already registered", entityMetadata.EntitySetName)
	}

	// Store the metadata
	s.entities[entityMetadata.EntitySetName] = entityMetadata

	// Create and store the handler
	handler := handlers.NewEntityHandler(s.db, entityMetadata, s.logger)
	handler.SetNamespace(s.namespace)
	handler.SetEntitiesMetadata(s.entities)
	handler.SetDeltaTracker(s.deltaTracker)
	handler.SetFTSManager(s.ftsManager)
	handler.SetPolicy(s.policy)
	handler.SetKeyGeneratorResolver(func(name string) (func(context.Context) (interface{}, error), bool) {
		generator, ok := s.resolveKeyGenerator(name)
		if !ok {
			return nil, false
		}
		return generator, true
	})
	// Set service-level default max top if configured
	if s.defaultMaxTop != nil {
		handler.SetDefaultMaxTop(s.defaultMaxTop)
	}
	// Set observability configuration if enabled
	if s.observability != nil {
		handler.SetObservability(s.observability)
	}
	s.handlers[entityMetadata.EntitySetName] = handler

	s.logger.Debug("Registered entity",
		"entity", entityMetadata.EntityName,
		"entitySet", entityMetadata.EntitySetName)
	return nil
}

// EnableChangeTracking enables OData change tracking for the specified entity set.
// When enabled, the service will issue delta tokens and record entity changes.
func (s *Service) EnableChangeTracking(entitySetName string) error {
	handler, exists := s.handlers[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' is not registered", entitySetName)
	}

	if handler == nil {
		return fmt.Errorf("entity handler for '%s' is not initialized", entitySetName)
	}

	if err := handler.EnableChangeTracking(); err != nil {
		return err
	}

	return nil
}

// RegisterSingleton registers a singleton type with the OData service.
// Singletons are single instances of an entity type that can be accessed directly by name.
// For example, RegisterSingleton(&MyCompany{}, "Company") allows access via /Company instead of /Companies(1)
func (s *Service) RegisterSingleton(entity interface{}, singletonName string) error {
	// Analyze the singleton structure
	singletonMetadata, err := metadata.AnalyzeSingleton(entity, singletonName)
	if err != nil {
		return fmt.Errorf("failed to analyze singleton: %w", err)
	}

	if _, exists := s.entities[singletonName]; exists {
		return fmt.Errorf("singleton '%s' is already registered", singletonName)
	}
	if _, exists := s.handlers[singletonName]; exists {
		return fmt.Errorf("singleton handler for '%s' is already registered", singletonName)
	}

	// Store the metadata using singleton name as key
	s.entities[singletonName] = singletonMetadata

	// Create and store the handler (same handler type works for both entities and singletons)
	handler := handlers.NewEntityHandler(s.db, singletonMetadata, s.logger)
	handler.SetNamespace(s.namespace)
	handler.SetEntitiesMetadata(s.entities)
	handler.SetFTSManager(s.ftsManager)
	handler.SetPolicy(s.policy)
	handler.SetKeyGeneratorResolver(func(name string) (func(context.Context) (interface{}, error), bool) {
		generator, ok := s.resolveKeyGenerator(name)
		if !ok {
			return nil, false
		}
		return generator, true
	})
	// Set service-level default max top if configured
	if s.defaultMaxTop != nil {
		handler.SetDefaultMaxTop(s.defaultMaxTop)
	}
	// Set observability configuration if enabled
	if s.observability != nil {
		handler.SetObservability(s.observability)
	}
	s.handlers[singletonName] = handler

	s.logger.Debug("Registered singleton",
		"entity", singletonMetadata.EntityName,
		"singleton", singletonName)
	return nil
}

// RegisterVirtualEntity registers a virtual entity type with the OData service.
// Virtual entities have no database backing store and require overwrite handlers for all operations.
// Operations without overwrite handlers will return HTTP 405 Method Not Allowed.
//
// Virtual entities are useful for:
//   - Exposing data from external APIs or services
//   - Creating computed or aggregated views
//   - Implementing custom business logic without database persistence
//
// Example:
//
//	type ExternalProduct struct {
//	    ID   int    `json:"id" odata:"key"`
//	    Name string `json:"name"`
//	}
//
//	service.RegisterVirtualEntity(&ExternalProduct{})
//	service.SetEntityOverwrite("ExternalProducts", &EntityOverwrite{
//	    GetCollection: func(ctx *OverwriteContext) (*CollectionResult, error) {
//	        // Fetch from external API
//	        products := fetchFromExternalAPI()
//	        return &CollectionResult{Items: products}, nil
//	    },
//	    GetEntity: func(ctx *OverwriteContext) (interface{}, error) {
//	        // Fetch single entity from external API
//	        return fetchFromExternalAPI(ctx.EntityKey)
//	    },
//	})
func (s *Service) RegisterVirtualEntity(entity interface{}) error {
	// Analyze the entity structure
	entityMetadata, err := metadata.AnalyzeVirtualEntity(entity)
	if err != nil {
		return fmt.Errorf("failed to analyze virtual entity: %w", err)
	}

	if _, exists := s.entities[entityMetadata.EntitySetName]; exists {
		return fmt.Errorf("entity set '%s' is already registered", entityMetadata.EntitySetName)
	}
	if _, exists := s.handlers[entityMetadata.EntitySetName]; exists {
		return fmt.Errorf("entity handler for '%s' is already registered", entityMetadata.EntitySetName)
	}

	// Store the metadata
	s.entities[entityMetadata.EntitySetName] = entityMetadata

	// Create and store the handler (no database operations will be performed)
	handler := handlers.NewEntityHandler(s.db, entityMetadata, s.logger)
	handler.SetNamespace(s.namespace)
	handler.SetEntitiesMetadata(s.entities)
	handler.SetFTSManager(s.ftsManager)
	handler.SetPolicy(s.policy)
	handler.SetKeyGeneratorResolver(func(name string) (func(context.Context) (interface{}, error), bool) {
		generator, ok := s.resolveKeyGenerator(name)
		if !ok {
			return nil, false
		}
		return generator, true
	})
	// Set service-level default max top if configured
	if s.defaultMaxTop != nil {
		handler.SetDefaultMaxTop(s.defaultMaxTop)
	}
	// Set observability configuration if enabled
	if s.observability != nil {
		handler.SetObservability(s.observability)
	}
	s.handlers[entityMetadata.EntitySetName] = handler

	s.logger.Debug("Registered virtual entity",
		"entity", entityMetadata.EntityName,
		"entitySet", entityMetadata.EntitySetName)
	return nil
}

// Types for registering custom OData actions and functions.
//
// The following types are re-exported from internal/actions package to provide a public API
// for defining and registering custom operations. See RegisterAction and RegisterFunction
// methods for usage examples, or refer to documentation/actions-and-functions.md for
// comprehensive guides.
type (
	// ParameterDefinition describes a single parameter for an action or function.
	// Use reflect.TypeOf to specify the expected Go type for the parameter.
	// See the full documentation on this type for field descriptions and examples.
	ParameterDefinition = actions.ParameterDefinition

	// ActionDefinition defines an OData action that can modify data (invoked with POST).
	// Actions can be bound to entities or unbound (service-level).
	// See the full documentation on this type for field descriptions and examples.
	ActionDefinition = actions.ActionDefinition

	// FunctionDefinition defines an OData function that computes values (invoked with GET).
	// Functions must be side-effect-free and can be bound to entities or unbound.
	// See the full documentation on this type for field descriptions and examples.
	FunctionDefinition = actions.FunctionDefinition

	// ActionHandler is the function signature for implementing action logic.
	// Return an error to abort the action; return nil after writing the HTTP response.
	// See the full documentation on this type for parameter descriptions and examples.
	ActionHandler = actions.ActionHandler

	// FunctionHandler is the function signature for implementing function logic.
	// Return the computed value and nil error on success; return nil and error on failure.
	// See the full documentation on this type for parameter descriptions and examples.
	FunctionHandler = actions.FunctionHandler

	// OverwriteContext provides context information for overwrite handlers.
	// It contains the parsed OData query options and entity key (for single entity operations).
	OverwriteContext = handlers.OverwriteContext

	// CollectionResult represents the result from a GetCollection overwrite handler.
	CollectionResult = handlers.CollectionResult

	// GetCollectionHandler is the function signature for overwriting the GetCollection operation.
	// It receives the overwrite context and should return the collection result or an error.
	GetCollectionHandler = handlers.GetCollectionHandler

	// GetEntityHandler is the function signature for overwriting the GetEntity operation.
	// It receives the overwrite context (including EntityKey) and should return the entity or an error.
	GetEntityHandler = handlers.GetEntityHandler

	// CreateHandler is the function signature for overwriting the Create operation.
	// It receives the overwrite context and the parsed entity data, and should return the created entity or an error.
	CreateHandler = handlers.CreateHandler

	// UpdateHandler is the function signature for overwriting the Update operation.
	// It receives the overwrite context (including EntityKey), update data, and whether it's a full replacement (PUT).
	// It should return the updated entity or an error.
	UpdateHandler = handlers.UpdateHandler

	// DeleteHandler is the function signature for overwriting the Delete operation.
	// It receives the overwrite context (including EntityKey) and should return an error if deletion fails.
	DeleteHandler = handlers.DeleteHandler

	// GetCountHandler is the function signature for overwriting the GetCount operation.
	// It receives the overwrite context and should return the count or an error.
	GetCountHandler = handlers.GetCountHandler

	// EntityOverwrite contains all overwrite handlers for an entity set.
	// Any handler that is nil will use the default GORM-based implementation.
	EntityOverwrite = handlers.EntityOverwrite
)

// RegisterAction registers a custom OData action with the service.
//
// Actions are operations that modify data and are invoked via HTTP POST. They can be bound to
// specific entities or unbound (accessible at the service root).
//
// # Required Fields
//
//   - Name: The action name (e.g., "ApplyDiscount")
//   - Handler: The function implementing the action logic (see ActionHandler)
//
// # Bound vs Unbound Actions
//
// Bound actions (IsBound = true):
//   - Must specify EntitySet (e.g., "Products")
//   - Invoked on entity instances: POST /Products(1)/ApplyDiscount
//   - The ctx parameter in the handler contains the entity instance
//   - EntitySet must match a registered entity set
//
// Unbound actions (IsBound = false):
//   - Invoked at service root: POST /ResetAllPrices
//   - The ctx parameter in the handler is nil
//
// # Parameters
//
// Parameters can be defined in two ways:
//
//  1. Explicitly using Parameters field:
//     Parameters: []ParameterDefinition{
//         {Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
//     }
//
//  2. Automatically via ParameterStructType field:
//     ParameterStructType: reflect.TypeOf(MyParamsStruct{})
//     The framework derives parameters from struct fields
//
// Parameters are passed in the JSON request body for actions.
//
// # Return Values
//
//   - Set ReturnType to nil for actions that return no value (should return HTTP 204 No Content)
//   - Set ReturnType to reflect.TypeOf(MyType{}) for actions that return values
//
// # Error Handling
//
// Return an error to abort the action and send an error response to the client.
// The handler must write the HTTP response (status, headers, body) before returning nil.
//
// # Example - Bound Action with Return Value
//
//	err := service.RegisterAction(ActionDefinition{
//	    Name:      "ApplyDiscount",
//	    IsBound:   true,
//	    EntitySet: "Products",
//	    Parameters: []ParameterDefinition{
//	        {Name: "percentage", Type: reflect.TypeOf(float64(0)), Required: true},
//	    },
//	    ReturnType: reflect.TypeOf(Product{}),
//	    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
//	        product := ctx.(*Product)
//	        percentage := params["percentage"].(float64)
//	        product.Price = product.Price * (1 - percentage/100)
//	        if err := db.Save(product).Error; err != nil {
//	            return err
//	        }
//	        w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
//	        w.WriteHeader(http.StatusOK)
//	        response := map[string]interface{}{
//	            "@odata.context": "$metadata#Products/$entity",
//	            "value":          product,
//	        }
//	        return json.NewEncoder(w).Encode(response)
//	    },
//	})
//
// Invoke: POST /Products(1)/ApplyDiscount with body {"percentage": 10}
//
// # Example - Unbound Action with No Return Value
//
//	err := service.RegisterAction(ActionDefinition{
//	    Name:       "ResetAllPrices",
//	    IsBound:    false,
//	    Parameters: []ParameterDefinition{},
//	    ReturnType: nil,
//	    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) error {
//	        if err := db.Model(&Product{}).Update("Price", 0).Error; err != nil {
//	            return err
//	        }
//	        w.WriteHeader(http.StatusNoContent)
//	        return nil
//	    },
//	})
//
// Invoke: POST /ResetAllPrices
//
// # Overloading
//
// Multiple actions with the same name but different parameter signatures can be registered
// (action overloading). The framework selects the appropriate overload based on the
// parameters provided in the request.
//
// See documentation/actions-and-functions.md for comprehensive examples and best practices.

func (s *Service) RegisterAction(action actions.ActionDefinition) error {
	if action.Name == "" {
		return fmt.Errorf("action name cannot be empty")
	}
	if action.Handler == nil {
		return fmt.Errorf("action handler cannot be nil")
	}
	if action.ParameterStructType != nil {
		derived, err := actions.ParameterDefinitionsFromStruct(action.ParameterStructType)
		if err != nil {
			return fmt.Errorf("invalid parameter struct for action '%s': %w", action.Name, err)
		}
		if len(action.Parameters) == 0 {
			action.Parameters = derived
		} else if !parameterDefinitionsCompatible(action.Parameters, derived) {
			return fmt.Errorf("parameter definitions do not match struct type for action '%s'", action.Name)
		}
	}
	if action.IsBound && action.EntitySet == "" {
		return fmt.Errorf("bound action must specify entity set")
	}
	if action.IsBound {
		// Verify entity set exists
		if _, exists := s.entities[action.EntitySet]; !exists {
			return fmt.Errorf("entity set '%s' not found", action.EntitySet)
		}
	}

	// Check for duplicate overloads (same name, binding, entity set, and parameters)
	existingActions := s.actions[action.Name]
	for _, existing := range existingActions {
		if actions.ActionSignaturesMatch(existing, &action) {
			return fmt.Errorf("action '%s' with this signature is already registered", action.Name)
		}
	}

	// Add to the list of overloads
	s.actions[action.Name] = append(s.actions[action.Name], &action)
	s.logger.Debug("Registered action",
		"name", action.Name,
		"bound", action.IsBound,
		"entitySet", action.EntitySet,
		"parameters", len(action.Parameters))
	return nil
}

// RegisterFunction registers a custom OData function with the service.
//
// Functions are side-effect-free operations that compute and return values. They are invoked via
// HTTP GET and must not modify data. Functions can be bound to specific entities or unbound
// (accessible at the service root).
//
// # Required Fields
//
//   - Name: The function name (e.g., "GetTotalPrice")
//   - Handler: The function implementing the logic (see FunctionHandler)
//   - ReturnType: The Go type of the return value (e.g., reflect.TypeOf(float64(0)))
//
// # Bound vs Unbound Functions
//
// Bound functions (IsBound = true):
//   - Must specify EntitySet (e.g., "Products")
//   - Invoked on entity instances: GET /Products(1)/GetTotalPrice(taxRate=0.08)
//   - The ctx parameter in the handler contains the entity instance
//   - EntitySet must match a registered entity set
//
// Unbound functions (IsBound = false):
//   - Invoked at service root: GET /GetTopProducts(count=10)
//   - The ctx parameter in the handler is nil
//
// # Parameters
//
// Parameters can be defined in two ways:
//
//  1. Explicitly using Parameters field:
//     Parameters: []ParameterDefinition{
//     {Name: "taxRate", Type: reflect.TypeOf(float64(0)), Required: true},
//     }
//
//  2. Automatically via ParameterStructType field:
//     ParameterStructType: reflect.TypeOf(MyParamsStruct{})
//     The framework derives parameters from struct fields
//
// Parameters are passed in the URL query string or using OData function call syntax:
//   - Query string: GET /GetTopProducts?count=10
//   - Function call: GET /GetTopProducts(count=10)
//
// # Return Values
//
// Functions must always return a value. The ReturnType field is required and specifies
// the Go type of the return value:
//   - Primitive types: reflect.TypeOf(float64(0)), reflect.TypeOf(""), etc.
//   - Complex types: reflect.TypeOf(Product{})
//   - Collections: reflect.TypeOf([]Product{})
//
// The framework automatically serializes the return value to JSON with appropriate OData annotations.
//
// # Error Handling
//
// Return (nil, error) to abort the function and send an error response to the client.
// Return (value, nil) on success. The framework handles response formatting automatically.
//
// # Example - Bound Function
//
//	err := service.RegisterFunction(FunctionDefinition{
//	    Name:      "GetTotalPrice",
//	    IsBound:   true,
//	    EntitySet: "Products",
//	    Parameters: []ParameterDefinition{
//	        {Name: "taxRate", Type: reflect.TypeOf(float64(0)), Required: true},
//	    },
//	    ReturnType: reflect.TypeOf(float64(0)),
//	    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
//	        product := ctx.(*Product)
//	        taxRate := params["taxRate"].(float64)
//	        totalPrice := product.Price * (1 + taxRate)
//	        return totalPrice, nil
//	    },
//	})
//
// Invoke: GET /Products(1)/GetTotalPrice(taxRate=0.08)
//
// Response:
//
//	{
//	  "@odata.context": "$metadata#Edm.Double",
//	  "value": 1079.99
//	}
//
// # Example - Unbound Function Returning Collection
//
//	err := service.RegisterFunction(FunctionDefinition{
//	    Name:    "GetTopProducts",
//	    IsBound: false,
//	    Parameters: []ParameterDefinition{
//	        {Name: "count", Type: reflect.TypeOf(int64(0)), Required: true},
//	    },
//	    ReturnType: reflect.TypeOf([]Product{}),
//	    Handler: func(w http.ResponseWriter, r *http.Request, ctx interface{}, params map[string]interface{}) (interface{}, error) {
//	        count := params["count"].(int64)
//	        var products []Product
//	        if err := db.Order("price DESC").Limit(int(count)).Find(&products).Error; err != nil {
//	            return nil, err
//	        }
//	        return products, nil
//	    },
//	})
//
// Invoke: GET /GetTopProducts(count=10)
//
// Response:
//
//	{
//	  "@odata.context": "$metadata#Products",
//	  "value": [
//	    {"ID": 1, "Name": "Laptop", "Price": 999.99},
//	    ...
//	  ]
//	}
//
// # Overloading
//
// Multiple functions with the same name but different parameter signatures can be registered
// (function overloading). The framework selects the appropriate overload based on the
// parameters provided in the request.
//
// See documentation/actions-and-functions.md for comprehensive examples and best practices.
func (s *Service) RegisterFunction(function actions.FunctionDefinition) error {
	if function.Name == "" {
		return fmt.Errorf("function name cannot be empty")
	}
	if function.Handler == nil {
		return fmt.Errorf("function handler cannot be nil")
	}
	if function.ReturnType == nil {
		return fmt.Errorf("function must have a return type")
	}
	if function.ParameterStructType != nil {
		derived, err := actions.ParameterDefinitionsFromStruct(function.ParameterStructType)
		if err != nil {
			return fmt.Errorf("invalid parameter struct for function '%s': %w", function.Name, err)
		}
		if len(function.Parameters) == 0 {
			function.Parameters = derived
		} else if !parameterDefinitionsCompatible(function.Parameters, derived) {
			return fmt.Errorf("parameter definitions do not match struct type for function '%s'", function.Name)
		}
	}
	if function.IsBound && function.EntitySet == "" {
		return fmt.Errorf("bound function must specify entity set")
	}
	if function.IsBound {
		// Verify entity set exists
		if _, exists := s.entities[function.EntitySet]; !exists {
			return fmt.Errorf("entity set '%s' not found", function.EntitySet)
		}
	}

	// Check for duplicate overloads (same name, binding, entity set, and parameters)
	existingFunctions := s.functions[function.Name]
	for _, existing := range existingFunctions {
		if actions.FunctionSignaturesMatch(existing, &function) {
			return fmt.Errorf("function '%s' with this signature is already registered", function.Name)
		}
	}

	// Add to the list of overloads
	s.functions[function.Name] = append(s.functions[function.Name], &function)
	s.logger.Debug("Registered function",
		"name", function.Name,
		"bound", function.IsBound,
		"entitySet", function.EntitySet,
		"parameters", len(function.Parameters))
	return nil
}

// SetNamespace updates the namespace used for metadata generation and @odata.type annotations.
func (s *Service) SetNamespace(namespace string) error {
	trimmed := strings.TrimSpace(namespace)
	if trimmed == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	if trimmed == s.namespace {
		return nil
	}

	s.namespace = trimmed
	s.metadataHandler.SetNamespace(trimmed)
	s.operationsHandler.SetNamespace(trimmed)
	for _, handler := range s.handlers {
		handler.SetNamespace(trimmed)
	}
	return nil
}

// SetEntityOverwrite configures all overwrite handlers for the specified entity set.
//
// Overwrite handlers completely replace the default GORM-based data access for entity operations.
// The library still validates OData query syntax before calling overwrite handlers, and handlers
// receive parsed OData query options that they can apply to their custom data sources.
//
// # Example
//
//	err := service.SetEntityOverwrite("Products", &EntityOverwrite{
//	    GetCollection: func(ctx *OverwriteContext) (*CollectionResult, error) {
//	        // Fetch products from external API instead of database
//	        products, err := externalAPI.GetProducts()
//	        if err != nil {
//	            return nil, err
//	        }
//	        return &CollectionResult{Items: products}, nil
//	    },
//	    GetEntity: func(ctx *OverwriteContext) (interface{}, error) {
//	        // Fetch single product by key from external API
//	        return externalAPI.GetProduct(ctx.EntityKey)
//	    },
//	})
//
// Any handler that is nil will use the default GORM-based implementation.
func (s *Service) SetEntityOverwrite(entitySetName string, overwrite *EntityOverwrite) error {
	handler, exists := s.handlers[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' is not registered", entitySetName)
	}

	handler.SetOverwrite(overwrite)
	s.logger.Debug("Set entity overwrite",
		"entitySet", entitySetName,
		"hasGetCollection", overwrite != nil && overwrite.GetCollection != nil,
		"hasGetEntity", overwrite != nil && overwrite.GetEntity != nil,
		"hasCreate", overwrite != nil && overwrite.Create != nil,
		"hasUpdate", overwrite != nil && overwrite.Update != nil,
		"hasDelete", overwrite != nil && overwrite.Delete != nil,
		"hasGetCount", overwrite != nil && overwrite.GetCount != nil)
	return nil
}

// SetGetCollectionOverwrite configures the overwrite handler for the GetCollection operation.
//
// The handler replaces the default collection retrieval logic (GET /EntitySet).
// OData query options ($filter, $select, $expand, $orderby, $top, $skip, $search, $count)
// are parsed and validated before the handler is called, and are available in the context.
//
// # Example
//
//	err := service.SetGetCollectionOverwrite("Products", func(ctx *OverwriteContext) (*CollectionResult, error) {
//	    products, err := customDB.Query(ctx.QueryOptions.Filter, ctx.QueryOptions.Top)
//	    if err != nil {
//	        return nil, err
//	    }
//	    return &CollectionResult{Items: products}, nil
//	})
func (s *Service) SetGetCollectionOverwrite(entitySetName string, handler GetCollectionHandler) error {
	h, exists := s.handlers[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' is not registered", entitySetName)
	}

	h.SetGetCollectionOverwrite(handler)
	return nil
}

// SetGetEntityOverwrite configures the overwrite handler for the GetEntity operation.
//
// The handler replaces the default single entity retrieval logic (GET /EntitySet(key)).
// The entity key is available in ctx.EntityKey.
//
// # Example
//
//	err := service.SetGetEntityOverwrite("Products", func(ctx *OverwriteContext) (interface{}, error) {
//	    return customDB.GetById(ctx.EntityKey)
//	})
func (s *Service) SetGetEntityOverwrite(entitySetName string, handler GetEntityHandler) error {
	h, exists := s.handlers[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' is not registered", entitySetName)
	}

	h.SetGetEntityOverwrite(handler)
	return nil
}

// SetCreateOverwrite configures the overwrite handler for the Create operation.
//
// The handler replaces the default entity creation logic (POST /EntitySet).
// The entity parameter contains the parsed entity from the request body.
// The handler should return the created entity with any server-generated values.
//
// # Example
//
//	err := service.SetCreateOverwrite("Products", func(ctx *OverwriteContext, entity interface{}) (interface{}, error) {
//	    product := entity.(*Product)
//	    product.ID = generateID()
//	    if err := customDB.Insert(product); err != nil {
//	        return nil, err
//	    }
//	    return product, nil
//	})
func (s *Service) SetCreateOverwrite(entitySetName string, handler CreateHandler) error {
	h, exists := s.handlers[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' is not registered", entitySetName)
	}

	h.SetCreateOverwrite(handler)
	return nil
}

// SetUpdateOverwrite configures the overwrite handler for the Update operation.
//
// The handler replaces the default entity update logic (PATCH/PUT /EntitySet(key)).
// The entity key is available in ctx.EntityKey. The updateData contains the properties to update.
// The isFullReplace parameter is true for PUT requests (complete replacement) and false for PATCH (partial update).
// The handler should return the updated entity.
//
// # Example
//
//	err := service.SetUpdateOverwrite("Products", func(ctx *OverwriteContext, updateData map[string]interface{}, isFullReplace bool) (interface{}, error) {
//	    if isFullReplace {
//	        return customDB.Replace(ctx.EntityKey, updateData)
//	    }
//	    return customDB.PartialUpdate(ctx.EntityKey, updateData)
//	})
func (s *Service) SetUpdateOverwrite(entitySetName string, handler UpdateHandler) error {
	h, exists := s.handlers[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' is not registered", entitySetName)
	}

	h.SetUpdateOverwrite(handler)
	return nil
}

// SetDeleteOverwrite configures the overwrite handler for the Delete operation.
//
// The handler replaces the default entity deletion logic (DELETE /EntitySet(key)).
// The entity key is available in ctx.EntityKey.
// Return nil to indicate successful deletion, or an error to abort.
//
// # Example
//
//	err := service.SetDeleteOverwrite("Products", func(ctx *OverwriteContext) error {
//	    return customDB.Delete(ctx.EntityKey)
//	})
func (s *Service) SetDeleteOverwrite(entitySetName string, handler DeleteHandler) error {
	h, exists := s.handlers[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' is not registered", entitySetName)
	}

	h.SetDeleteOverwrite(handler)
	return nil
}

// SetGetCountOverwrite configures the overwrite handler for the GetCount operation.
//
// The handler replaces the default count retrieval logic (GET /EntitySet/$count).
// OData $filter query option is parsed and available in ctx.QueryOptions for filtering the count.
//
// # Example
//
//	err := service.SetGetCountOverwrite("Products", func(ctx *OverwriteContext) (int64, error) {
//	    return customDB.Count(ctx.QueryOptions.Filter)
//	})
func (s *Service) SetGetCountOverwrite(entitySetName string, handler GetCountHandler) error {
	h, exists := s.handlers[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' is not registered", entitySetName)
	}

	h.SetGetCountOverwrite(handler)
	return nil
}

// DisableHTTPMethods disables specific HTTP methods for an entity set.
// This allows you to restrict certain operations on entities without needing hooks.
//
// Supported methods: GET, POST, PUT, PATCH, DELETE
//
// # Example - Disable POST for Users
//
//	err := service.DisableHTTPMethods("Users", "POST")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// After this, POST requests to /Users will return HTTP 405 Method Not Allowed.
// Other methods like GET, PUT, PATCH, DELETE will continue to work normally.
//
// # Example - Disable Multiple Methods
//
//	err := service.DisableHTTPMethods("Products", "POST", "DELETE")
//
// This disables both creating and deleting products while allowing read and update operations.
func (s *Service) DisableHTTPMethods(entitySetName string, methods ...string) error {
	_, exists := s.handlers[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' is not registered", entitySetName)
	}

	metadata, exists := s.entities[entitySetName]
	if !exists {
		return fmt.Errorf("entity metadata for '%s' not found", entitySetName)
	}

	if metadata.DisabledMethods == nil {
		metadata.DisabledMethods = make(map[string]bool)
	}

	// Validate and normalize methods
	for _, method := range methods {
		normalized := strings.ToUpper(strings.TrimSpace(method))
		switch normalized {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			metadata.DisabledMethods[normalized] = true
		default:
			return fmt.Errorf("unsupported HTTP method '%s'; supported methods are GET, POST, PUT, PATCH, DELETE", method)
		}
	}

	s.logger.Debug("Disabled HTTP methods",
		"entitySet", entitySetName,
		"methods", methods)

	return nil
}

// SetDefaultMaxTop sets the default maximum number of results to return when no explicit $top is provided.
// This applies to all entity sets in the service unless overridden at the entity level.
// Pass 0 or a negative value to remove the default limit.
//
// # Example
//
//	service.SetDefaultMaxTop(100) // Default limit of 100 results for all entities
func (s *Service) SetDefaultMaxTop(maxTop int) {
	if maxTop <= 0 {
		s.defaultMaxTop = nil
		s.logger.Debug("Removed default max top for service")
	} else {
		s.defaultMaxTop = &maxTop
		s.logger.Debug("Set default max top for service", "maxTop", maxTop)
	}
	// Update all existing handlers that don't have entity-level defaults
	s.updateHandlersDefaultMaxTop()
}

// updateHandlersDefaultMaxTop updates all handlers with the service-level default,
// unless they have an entity-level default set
func (s *Service) updateHandlersDefaultMaxTop() {
	for _, handler := range s.handlers {
		// Only update if there's no entity-level default set
		if !handler.HasEntityLevelDefaultMaxTop() {
			handler.SetDefaultMaxTop(s.defaultMaxTop)
		}
	}
}

// SetEntityDefaultMaxTop sets the default maximum number of results for a specific entity set.
// This overrides the service-level default for this entity set.
// Pass 0 or a negative value to remove the entity-level default (falls back to service default).
//
// # Example
//
//	service.SetEntityDefaultMaxTop("Products", 50) // Products limited to 50 by default
//	service.SetEntityDefaultMaxTop("Orders", 200)  // Orders limited to 200 by default
func (s *Service) SetEntityDefaultMaxTop(entitySetName string, maxTop int) error {
	handler, exists := s.handlers[entitySetName]
	if !exists {
		return fmt.Errorf("entity set '%s' is not registered", entitySetName)
	}

	metadata, exists := s.entities[entitySetName]
	if !exists {
		return fmt.Errorf("entity metadata for '%s' not found", entitySetName)
	}

	if maxTop <= 0 {
		// Remove entity-level default and fall back to service default
		metadata.DefaultMaxTop = nil
		handler.SetDefaultMaxTop(s.defaultMaxTop)
		s.logger.Debug("Removed default max top for entity", "entitySet", entitySetName)
	} else {
		// Store entity-level default in metadata
		// Note: handler.defaultMaxTop continues to hold the service-level default as fallback
		metadata.DefaultMaxTop = &maxTop
		s.logger.Debug("Set default max top for entity", "entitySet", entitySetName, "maxTop", maxTop)
	}

	return nil
}

func parameterDefinitionsCompatible(existing, derived []actions.ParameterDefinition) bool {
	if len(existing) != len(derived) {
		return false
	}

	expected := make(map[string]actions.ParameterDefinition, len(derived))
	for _, def := range derived {
		expected[def.Name] = def
	}

	for _, def := range existing {
		if match, ok := expected[def.Name]; !ok || match.Type != def.Type || match.Required != def.Required {
			return false
		}
	}

	return true
}
