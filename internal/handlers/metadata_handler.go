package handlers

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

// MetadataHandler handles metadata document requests
type MetadataHandler struct {
	entities map[string]*metadata.EntityMetadata
	// Cached metadata documents
	cachedXML  string
	cachedJSON []byte
	onceXML    sync.Once
	onceJSON   sync.Once
	namespace  string
}

const defaultNamespace = "ODataService"

// NewMetadataHandler creates a new metadata handler
func NewMetadataHandler(entities map[string]*metadata.EntityMetadata) *MetadataHandler {
	return &MetadataHandler{
		entities:  entities,
		namespace: defaultNamespace,
	}
}

// SetNamespace updates the namespace used for metadata generation and clears cached documents.
func (h *MetadataHandler) SetNamespace(namespace string) {
	trimmed := strings.TrimSpace(namespace)
	if trimmed == "" {
		trimmed = defaultNamespace
	}
	if trimmed == h.namespace {
		return
	}
	h.namespace = trimmed
	h.cachedXML = ""
	h.cachedJSON = nil
	h.onceXML = sync.Once{}
	h.onceJSON = sync.Once{}
}

func (h *MetadataHandler) namespaceOrDefault() string {
	if strings.TrimSpace(h.namespace) == "" {
		return defaultNamespace
	}
	return h.namespace
}

// HandleMetadata handles the metadata document endpoint
func (h *MetadataHandler) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.handleGetMetadata(w, r)
	case http.MethodOptions:
		h.handleOptionsMetadata(w)
	default:
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for metadata document", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
	}
}

// handleGetMetadata handles GET requests for metadata document
func (h *MetadataHandler) handleGetMetadata(w http.ResponseWriter, r *http.Request) {
	useJSON := shouldReturnJSON(r)

	if useJSON {
		h.handleMetadataJSON(w, r)
	} else {
		h.handleMetadataXML(w, r)
	}
}

// handleOptionsMetadata handles OPTIONS requests for metadata document
func (h *MetadataHandler) handleOptionsMetadata(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD, OPTIONS")
	w.WriteHeader(http.StatusOK)
}

func (h *MetadataHandler) newMetadataModel() metadataModel {
	return metadataModel{
		namespace: h.namespaceOrDefault(),
		entities:  h.entities,
	}
}

type metadataModel struct {
	namespace string
	entities  map[string]*metadata.EntityMetadata
}

type enumTypeInfo struct {
	Members        []metadata.EnumMember
	IsFlags        bool
	UnderlyingType string
}

func (m metadataModel) qualifiedTypeName(typeName string) string {
	return fmt.Sprintf("%s.%s", m.namespace, typeName)
}

func (m metadataModel) collectEnumDefinitions() map[string]*enumTypeInfo {
	enumDefinitions := make(map[string]*enumTypeInfo)

	for _, entityMeta := range m.entities {
		for _, prop := range entityMeta.Properties {
			if !prop.IsEnum || prop.EnumTypeName == "" {
				continue
			}

			info, exists := enumDefinitions[prop.EnumTypeName]
			if !exists {
				info = &enumTypeInfo{}
				enumDefinitions[prop.EnumTypeName] = info
			}

			if len(info.Members) == 0 && len(prop.EnumMembers) > 0 {
				info.Members = append([]metadata.EnumMember(nil), prop.EnumMembers...)
			}
			if info.UnderlyingType == "" && prop.EnumUnderlyingType != "" {
				info.UnderlyingType = prop.EnumUnderlyingType
			}
			if prop.IsFlags {
				info.IsFlags = true
			}
		}
	}

	return enumDefinitions
}

// getEdmType converts a Go type to an EDM (Entity Data Model) type
func getEdmType(goType reflect.Type) string {
	// Handle pointer types
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	// Check for specific types by name
	typeName := goType.String()
	switch typeName {
	case "time.Time":
		return "Edm.DateTimeOffset"
	case "uuid.UUID", "github.com/google/uuid.UUID":
		return "Edm.Guid"
	}

	switch goType.Kind() {
	case reflect.String:
		return "Edm.String"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "Edm.Int32"
	case reflect.Int64:
		return "Edm.Int64"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "Edm.Int32"
	case reflect.Uint64:
		return "Edm.Int64"
	case reflect.Float32:
		return "Edm.Single"
	case reflect.Float64:
		return "Edm.Double"
	case reflect.Bool:
		return "Edm.Boolean"
	default:
		if goType.Kind() == reflect.Slice && goType.Elem().Kind() == reflect.Uint8 {
			return "Edm.Binary"
		}
		return "Edm.String"
	}
}

// pluralize creates a simple pluralized form of the entity name
func pluralize(word string) string {
	if word == "" {
		return word
	}

	switch {
	case strings.HasSuffix(word, "y"):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") || strings.HasSuffix(word, "z") ||
		strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh"):
		return word + "es"
	default:
		return word + "s"
	}
}
