package odata

// Package odata provides functionality for building OData services in Go.
// This library allows you to define Go structs representing entities and
// automatically handles the necessary OData protocol logic.

import (
	"fmt"

	"github.com/nlstn/go-odata/internal/handlers"
	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

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
	// actions holds registered actions keyed by action name
	actions map[string]*ActionDefinition
	// functions holds registered functions keyed by function name
	functions map[string]*FunctionDefinition
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
		actions:                make(map[string]*ActionDefinition),
		functions:              make(map[string]*FunctionDefinition),
	}
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
	s.handlers[entityMetadata.EntitySetName] = handler

	fmt.Printf("Registered entity: %s (EntitySet: %s)\n", entityMetadata.EntityName, entityMetadata.EntitySetName)
	return nil
}
