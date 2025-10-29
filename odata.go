package odata

// Package odata provides functionality for building OData services in Go.
// This library allows you to define Go structs representing entities and
// automatically handles the necessary OData protocol logic.

import (
	"fmt"
	"strings"

	"github.com/nlstn/go-odata/internal/actions"
	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
)

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
}

// NewService creates a new OData service instance with database connection.
func NewService(db *gorm.DB) *Service {
	entities := make(map[string]*metadata.EntityMetadata)
	handlersMap := make(map[string]*handlers.EntityHandler)
	s := &Service{
		db:                     db,
		entities:               entities,
		handlers:               handlersMap,
		metadataHandler:        handlers.NewMetadataHandler(entities),
		serviceDocumentHandler: handlers.NewServiceDocumentHandler(entities),
		actions:                make(map[string][]*actions.ActionDefinition),
		functions:              make(map[string][]*actions.FunctionDefinition),
		namespace:              DefaultNamespace,
		deltaTracker:           trackchanges.NewTracker(),
	}
	s.metadataHandler.SetNamespace(DefaultNamespace)
	// Initialize batch handler with reference to service
	s.batchHandler = handlers.NewBatchHandler(db, handlersMap, s)
	return s
}

// RegisterEntity registers an entity type with the OData service.
func (s *Service) RegisterEntity(entity interface{}) error {
	// Analyze the entity structure
	entityMetadata, err := metadata.AnalyzeEntity(entity)
	if err != nil {
		return fmt.Errorf("failed to analyze entity: %w", err)
	}

	// Store the metadata
	s.entities[entityMetadata.EntitySetName] = entityMetadata

	// Create and store the handler
	handler := handlers.NewEntityHandler(s.db, entityMetadata)
	handler.SetNamespace(s.namespace)
	handler.SetEntitiesMetadata(s.entities)
	handler.SetDeltaTracker(s.deltaTracker)
	s.handlers[entityMetadata.EntitySetName] = handler

	s.deltaTracker.RegisterEntity(entityMetadata.EntitySetName)

	fmt.Printf("Registered entity: %s (EntitySet: %s)\n", entityMetadata.EntityName, entityMetadata.EntitySetName)
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

	// Store the metadata using singleton name as key
	s.entities[singletonName] = singletonMetadata

	// Create and store the handler (same handler type works for both entities and singletons)
	handler := handlers.NewEntityHandler(s.db, singletonMetadata)
	handler.SetNamespace(s.namespace)
	handler.SetEntitiesMetadata(s.entities)
	s.handlers[singletonName] = handler

	fmt.Printf("Registered singleton: %s (Singleton: %s)\n", singletonMetadata.EntityName, singletonName)
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
	fmt.Printf("Registered action: %s (Bound: %v, EntitySet: %s, Parameters: %d)\n", 
		action.Name, action.IsBound, action.EntitySet, len(action.Parameters))
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
	fmt.Printf("Registered function: %s (Bound: %v, EntitySet: %s, Parameters: %d)\n", 
		function.Name, function.IsBound, function.EntitySet, len(function.Parameters))
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
	for _, handler := range s.handlers {
		handler.SetNamespace(trimmed)
	}
	return nil
}
