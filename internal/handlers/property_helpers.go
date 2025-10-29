package handlers

import (
	"fmt"
	"net/http"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/response"
)

// findNavigationProperty finds a navigation property by name in the entity metadata
func (h *EntityHandler) findNavigationProperty(navigationProperty string) *metadata.PropertyMetadata {
	return h.metadata.FindNavigationProperty(navigationProperty)
}

// findStructuralProperty finds a structural (non-navigation) property by name in the entity metadata
func (h *EntityHandler) findStructuralProperty(propertyName string) *metadata.PropertyMetadata {
	return h.metadata.FindStructuralProperty(propertyName)
}

// IsNavigationProperty checks if a property name is a navigation property
func (h *EntityHandler) IsNavigationProperty(propertyName string) bool {
	// Parse out any key from the property name (e.g., "RelatedProducts(2)" -> "RelatedProducts")
	navPropName, _ := h.parseNavigationPropertyWithKey(propertyName)
	return h.findNavigationProperty(navPropName) != nil
}

// IsStructuralProperty checks if a property name is a structural property
func (h *EntityHandler) IsStructuralProperty(propertyName string) bool {
	return h.findStructuralProperty(propertyName) != nil
}

// findComplexTypeProperty finds a complex type property by name in the entity metadata
func (h *EntityHandler) findComplexTypeProperty(propertyName string) *metadata.PropertyMetadata {
	return h.metadata.FindComplexTypeProperty(propertyName)
}

// IsComplexTypeProperty checks if a property name is a complex type property
func (h *EntityHandler) IsComplexTypeProperty(propertyName string) bool {
	return h.findComplexTypeProperty(propertyName) != nil
}

// writeMethodNotAllowedError writes a method not allowed error for a specific context
func (h *EntityHandler) writeMethodNotAllowedError(w http.ResponseWriter, method, context string) {
	if err := response.WriteError(w, http.StatusMethodNotAllowed, ErrMsgMethodNotAllowed,
		fmt.Sprintf("Method %s is not supported for %s", method, context)); err != nil {
		fmt.Printf(LogMsgErrorWritingErrorResponse, err)
	}
}

// writePropertyNotFoundError writes a property not found error
func (h *EntityHandler) writePropertyNotFoundError(w http.ResponseWriter, propertyName string) {
	if err := response.WriteError(w, http.StatusNotFound, "Property not found",
		fmt.Sprintf("'%s' is not a valid property for %s", propertyName, h.metadata.EntitySetName)); err != nil {
		fmt.Printf(LogMsgErrorWritingErrorResponse, err)
	}
}
