package handlers

import (
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

// MetadataHandler handles metadata document requests
type MetadataHandler struct {
	entities map[string]*metadata.EntityMetadata
}

// NewMetadataHandler creates a new metadata handler
func NewMetadataHandler(entities map[string]*metadata.EntityMetadata) *MetadataHandler {
	return &MetadataHandler{
		entities: entities,
	}
}

// HandleMetadata handles the metadata document endpoint
func (h *MetadataHandler) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		if err := response.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed",
			fmt.Sprintf("Method %s is not supported for metadata document", r.Method)); err != nil {
			fmt.Printf("Error writing error response: %v\n", err)
		}
		return
	}

	// For now, return a simple XML response indicating metadata is not fully implemented
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)

	metadata := `<?xml version="1.0" encoding="UTF-8"?>
<edmx:Edmx xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx" Version="4.0">
  <edmx:DataServices>
    <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="ODataService">
      <EntityContainer Name="Container">
`

	// Add entity sets to metadata
	for entitySetName, entityMeta := range h.entities {
		metadata += fmt.Sprintf(`        <EntitySet Name="%s" EntityType="ODataService.%s" />
`, entitySetName, entityMeta.EntityName)
	}

	metadata += `      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	if _, err := w.Write([]byte(metadata)); err != nil {
		fmt.Printf("Error writing metadata response: %v\n", err)
	}
}
