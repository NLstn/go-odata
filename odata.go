package odata

// Package odata provides functionality for building OData services in Go.
// This library allows you to define Go structs representing entities and
// automatically handles the necessary OData protocol logic.

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/async"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"github.com/nlstn/go-odata/internal/service/operations"
	servrouter "github.com/nlstn/go-odata/internal/service/router"
	servruntime "github.com/nlstn/go-odata/internal/service/runtime"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
	"sync"
)

// KeyGenerator describes a function that produces a key value for new entities.
type KeyGenerator func(context.Context) (interface{}, error)

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
	// ftsManager manages full-text search functionality for SQLite
	ftsManager *query.FTSManager
	// keyGenerators maintains registered key generator functions by name
	keyGenerators   map[string]KeyGenerator
	keyGeneratorsMu sync.RWMutex
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
	handler.SetKeyGeneratorResolver(func(name string) (func(context.Context) (interface{}, error), bool) {
		generator, ok := s.resolveKeyGenerator(name)
		if !ok {
			return nil, false
		}
		return generator, true
	})
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
	handler.SetKeyGeneratorResolver(func(name string) (func(context.Context) (interface{}, error), bool) {
		generator, ok := s.resolveKeyGenerator(name)
		if !ok {
			return nil, false
		}
		return generator, true
	})
	s.handlers[singletonName] = handler

	s.logger.Debug("Registered singleton",
		"entity", singletonMetadata.EntityName,
		"singleton", singletonName)
	return nil
}

// Re-export types from internal/actions package for public API
type (
	ParameterDefinition = actions.ParameterDefinition
	ActionDefinition    = actions.ActionDefinition
	FunctionDefinition  = actions.FunctionDefinition
	ActionHandler       = actions.ActionHandler
	FunctionHandler     = actions.FunctionHandler
)

// RegisterAction registers an action with the OData service
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

// RegisterFunction registers a function with the OData service
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
