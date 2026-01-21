package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/gorm"
)

// TestHandleGetEntityOverwrite tests the GET entity overwrite handler
func TestHandleGetEntityOverwrite(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			getEntity: func(ctx *OverwriteContext) (interface{}, error) {
				return map[string]interface{}{"id": ctx.EntityKey}, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/Products(123)", nil)
		w := httptest.NewRecorder()

		handler.handleGetEntityOverwrite(w, req, "123")

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("not found error", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			getEntity: func(ctx *OverwriteContext) (interface{}, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/Products(999)", nil)
		w := httptest.NewRecorder()

		handler.handleGetEntityOverwrite(w, req, "999")

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("nil result", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			getEntity: func(ctx *OverwriteContext) (interface{}, error) {
				return nil, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/Products(999)", nil)
		w := httptest.NewRecorder()

		handler.handleGetEntityOverwrite(w, req, "999")

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			getEntity: func(ctx *OverwriteContext) (interface{}, error) {
				return nil, errors.New("database error")
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/Products(123)", nil)
		w := httptest.NewRecorder()

		handler.handleGetEntityOverwrite(w, req, "123")

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

// TestHandleDeleteEntityOverwrite tests the DELETE entity overwrite handler
func TestHandleDeleteEntityOverwrite(t *testing.T) {
	t.Run("successful deletion", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			delete: func(ctx *OverwriteContext) error {
				return nil
			},
		}

		req := httptest.NewRequest(http.MethodDelete, "/Products(123)", nil)
		w := httptest.NewRecorder()

		handler.handleDeleteEntityOverwrite(w, req, "123")

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}
	})

	t.Run("not found error", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			delete: func(ctx *OverwriteContext) error {
				return gorm.ErrRecordNotFound
			},
		}

		req := httptest.NewRequest(http.MethodDelete, "/Products(999)", nil)
		w := httptest.NewRecorder()

		handler.handleDeleteEntityOverwrite(w, req, "999")

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			delete: func(ctx *OverwriteContext) error {
				return errors.New("deletion failed")
			},
		}

		req := httptest.NewRequest(http.MethodDelete, "/Products(123)", nil)
		w := httptest.NewRecorder()

		handler.handleDeleteEntityOverwrite(w, req, "123")

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

// TestHandleUpdateEntityOverwrite tests the UPDATE entity overwrite handler
func TestHandleUpdateEntityOverwrite(t *testing.T) {
	t.Run("successful update PATCH", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			update: func(ctx *OverwriteContext, updateData map[string]interface{}, isFullReplace bool) (interface{}, error) {
				return map[string]interface{}{"id": ctx.EntityKey, "name": "Updated"}, nil
			},
		}

		body := map[string]interface{}{"name": "Updated"}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPatch, "/Products(123)", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.handleUpdateEntityOverwrite(w, req, "123", false)

		if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
			t.Errorf("Expected status 200 or 204, got %d", w.Code)
		}
	})

	t.Run("successful update PUT", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			update: func(ctx *OverwriteContext, updateData map[string]interface{}, isFullReplace bool) (interface{}, error) {
				if !isFullReplace {
					t.Error("Expected isFullReplace to be true for PUT")
				}
				return map[string]interface{}{"id": ctx.EntityKey}, nil
			},
		}

		body := map[string]interface{}{"id": "123", "name": "Replaced"}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, "/Products(123)", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.handleUpdateEntityOverwrite(w, req, "123", true)

		if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
			t.Errorf("Expected status 200 or 204, got %d", w.Code)
		}
	})

	t.Run("not found error", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			update: func(ctx *OverwriteContext, updateData map[string]interface{}, isFullReplace bool) (interface{}, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}

		body := map[string]interface{}{"name": "Updated"}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPatch, "/Products(999)", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.handleUpdateEntityOverwrite(w, req, "999", false)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("invalid content type", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.overwrite = &entityOverwriteHandlers{
			update: func(ctx *OverwriteContext, updateData map[string]interface{}, isFullReplace bool) (interface{}, error) {
				return nil, nil
			},
		}

		req := httptest.NewRequest(http.MethodPatch, "/Products(123)", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		handler.handleUpdateEntityOverwrite(w, req, "123", false)

		if w.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Expected status 415, got %d", w.Code)
		}
	})
}

// TestHandlePostEntityOverwrite tests the POST entity overwrite handler
func TestHandlePostEntityOverwrite(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.metadata.EntityType = reflect.TypeOf(struct {
			ID   int
			Name string
		}{})
		handler.overwrite = &entityOverwriteHandlers{
			create: func(ctx *OverwriteContext, entity interface{}) (interface{}, error) {
				// Return entity with ID set so location can be built
				return map[string]interface{}{"ID": 123, "Name": "New Product"}, nil
			},
		}

		body := map[string]interface{}{"Name": "New Product"}
		jsonBody, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/Products", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.handlePostEntityOverwrite(w, req)

		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Errorf("Expected status 201 or 200, got %d", w.Code)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()

		req := httptest.NewRequest(http.MethodPost, "/Products", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.handlePostEntityOverwrite(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid content type", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()

		req := httptest.NewRequest(http.MethodPost, "/Products", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		handler.handlePostEntityOverwrite(w, req)

		if w.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Expected status 415, got %d", w.Code)
		}
	})
}

// TestCopyETagProperty tests ETag property copying
func TestCopyETagProperty(t *testing.T) {
	type TestEntity struct {
		ID      int
		Version int
		Name    string
	}

	t.Run("successful copy", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.metadata.ETagProperty = &metadata.PropertyMetadata{
			FieldName: "Version",
		}

		source := &TestEntity{ID: 1, Version: 5, Name: "Source"}
		dest := &TestEntity{ID: 1, Version: 0, Name: "Dest"}

		handler.copyETagProperty(source, dest)

		if dest.Version != 5 {
			t.Errorf("Expected Version to be 5, got %d", dest.Version)
		}
	})

	t.Run("no ETag property", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.metadata.ETagProperty = nil

		source := &TestEntity{ID: 1, Version: 5}
		dest := &TestEntity{ID: 1, Version: 0}

		handler.copyETagProperty(source, dest)

		// Should not change
		if dest.Version != 0 {
			t.Errorf("Expected Version to remain 0, got %d", dest.Version)
		}
	})

	t.Run("invalid field", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.metadata.ETagProperty = &metadata.PropertyMetadata{
			FieldName: "NonExistent",
		}

		source := &TestEntity{ID: 1, Version: 5}
		dest := &TestEntity{ID: 1, Version: 0}

		// Should not panic
		handler.copyETagProperty(source, dest)
	})
}

// TestWriteInvalidQueryError tests invalid query error writing
func TestWriteInvalidQueryError(t *testing.T) {
	handler := createTestHandlerWithMetadata()

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	w := httptest.NewRecorder()

	handler.writeInvalidQueryError(w, req, errors.New("invalid filter"))

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestWriteDatabaseError tests database error writing
func TestWriteDatabaseError(t *testing.T) {
	handler := createTestHandlerWithMetadata()

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	w := httptest.NewRecorder()

	handler.writeDatabaseError(w, req, errors.New("connection failed"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

// TestRequestError tests the requestError type
func TestRequestError(t *testing.T) {
	t.Run("Error method", func(t *testing.T) {
		err := &requestError{
			StatusCode: http.StatusBadRequest,
			ErrorCode:  "BadRequest",
			Message:    "Invalid input",
		}

		if err.Error() != "Invalid input" {
			t.Errorf("Expected 'Invalid input', got %s", err.Error())
		}
	})
}

// TestGetKeyProperty tests the GetKeyProperty method
func TestGetKeyProperty(t *testing.T) {
	handler := createTestHandlerWithMetadata()
	// Set the KeyProperty field explicitly
	handler.metadata.KeyProperty = &metadata.PropertyMetadata{
		Name:      "ID",
		FieldName: "ID",
		IsKey:     true,
	}

	adapter := newMetadataAdapter(handler.metadata, handler.namespace)
	prop := adapter.GetKeyProperty()

	if prop == nil {
		t.Error("Expected to find key property")
	} else if prop.Name != "ID" {
		t.Errorf("Expected key property name 'ID', got %s", prop.Name)
	}
}

// TestGetEntitySetName tests the GetEntitySetName method
func TestGetEntitySetName(t *testing.T) {
	handler := createTestHandlerWithMetadata()
	handler.metadata.EntitySetName = "Products"

	adapter := newMetadataAdapter(handler.metadata, handler.namespace)
	name := adapter.GetEntitySetName()

	if name != "Products" {
		t.Errorf("Expected 'Products', got %s", name)
	}
}

// TestGetETagProperty tests the GetETagProperty method
func TestGetETagProperty(t *testing.T) {
	handler := createTestHandlerWithMetadata()
	handler.metadata.ETagProperty = &metadata.PropertyMetadata{
		Name:      "Version",
		FieldName: "Version",
	}

	adapter := newMetadataAdapter(handler.metadata, handler.namespace)
	prop := adapter.GetETagProperty()

	if prop == nil {
		t.Error("Expected to find ETag property")
	} else if prop.Name != "Version" {
		t.Errorf("Expected ETag property name 'Version', got %s", prop.Name)
	}
}

// TestGetNamespace tests the GetNamespace method with different scenarios
func TestGetNamespace(t *testing.T) {
	t.Run("with namespace", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.namespace = "MyNamespace"

		adapter := newMetadataAdapter(handler.metadata, handler.namespace)
		ns := adapter.GetNamespace()

		if ns != "MyNamespace" {
			t.Errorf("Expected 'MyNamespace', got %s", ns)
		}
	})

	t.Run("without namespace", func(t *testing.T) {
		handler := createTestHandlerWithMetadata()
		handler.namespace = ""

		adapter := newMetadataAdapter(handler.metadata, handler.namespace)
		ns := adapter.GetNamespace()

		// Should return empty string or default
		t.Logf("Namespace: %s", ns)
	})
}

// Helper function to create a test handler with metadata
func createTestHandlerWithMetadata() *EntityHandler {
	return &EntityHandler{
		metadata: &metadata.EntityMetadata{
			EntityName:    "Product",
			EntitySetName: "Products",
			Properties: []metadata.PropertyMetadata{
				{
					Name:      "ID",
					FieldName: "ID",
					Type:      reflect.TypeOf(0),
					IsKey:     true,
				},
				{
					Name:      "Name",
					FieldName: "Name",
					Type:      reflect.TypeOf(""),
				},
			},
		},
		logger:    createNilLogger(),
		namespace: "TestNamespace",
	}
}
