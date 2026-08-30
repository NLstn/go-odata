package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/auth"
	"github.com/nlstn/go-odata/internal/metadata"
)

type denyMutationPolicy struct{}

func (denyMutationPolicy) Authorize(_ auth.AuthContext, _ auth.ResourceDescriptor, _ auth.Operation) auth.Decision {
	return auth.Deny("mutation denied")
}

func TestOverwriteWriteHandlersValidatePayloadBeforeCallback(t *testing.T) {
	t.Parallel()

	type entity struct {
		ID   int
		Name string
	}

	newHandler := func() (*EntityHandler, *bool) {
		called := false
		handler := createTestHandlerWithMetadata()
		handler.metadata.EntityType = reflect.TypeOf(entity{})
		handler.metadata.Properties = []metadata.PropertyMetadata{
			{Name: "ID", FieldName: "ID", JsonName: "ID", Type: reflect.TypeOf(0), IsKey: true},
			{Name: "Name", FieldName: "Name", JsonName: "Name", Type: reflect.TypeOf(""), IsRequired: true},
		}
		handler.metadata.KeyProperties = []metadata.PropertyMetadata{handler.metadata.Properties[0]}
		handler.overwrite = &entityOverwriteHandlers{
			create: func(*OverwriteContext, interface{}) (interface{}, error) {
				called = true
				return nil, nil
			},
			update: func(*OverwriteContext, map[string]interface{}, bool) (interface{}, error) {
				called = true
				return nil, nil
			},
		}
		return handler, &called
	}

	tests := []struct {
		name   string
		method string
		body   string
		run    func(*EntityHandler, http.ResponseWriter, *http.Request)
	}{
		{
			name:   "create missing required property",
			method: http.MethodPost,
			body:   `{}`,
			run: func(handler *EntityHandler, writer http.ResponseWriter, request *http.Request) {
				handler.handlePostEntity(writer, request)
			},
		},
		{
			name:   "patch modifies key",
			method: http.MethodPatch,
			body:   `{"ID":2}`,
			run: func(handler *EntityHandler, writer http.ResponseWriter, request *http.Request) {
				handler.handlePatchEntity(writer, request, "1")
			},
		},
		{
			name:   "put missing required property",
			method: http.MethodPut,
			body:   `{"ID":1}`,
			run: func(handler *EntityHandler, writer http.ResponseWriter, request *http.Request) {
				handler.handlePutEntity(writer, request, "1")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler, called := newHandler()
			request := httptest.NewRequest(test.method, "/Products", bytes.NewBufferString(test.body))
			request.Header.Set("Content-Type", "application/json")
			writer := httptest.NewRecorder()

			test.run(handler, writer, request)

			if writer.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", writer.Code, http.StatusBadRequest)
			}
			if *called {
				t.Fatal("overwrite handler was called with an invalid payload")
			}
		})
	}
}

func TestOverwriteWriteHandlersRequireAuthorization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		run    func(*EntityHandler, http.ResponseWriter, *http.Request)
	}{
		{
			name:   "create",
			method: http.MethodPost,
			path:   "/Products",
			run: func(handler *EntityHandler, writer http.ResponseWriter, request *http.Request) {
				handler.handlePostEntity(writer, request)
			},
		},
		{
			name:   "patch",
			method: http.MethodPatch,
			path:   "/Products(1)",
			run: func(handler *EntityHandler, writer http.ResponseWriter, request *http.Request) {
				handler.handlePatchEntity(writer, request, "1")
			},
		},
		{
			name:   "put",
			method: http.MethodPut,
			path:   "/Products(1)",
			run: func(handler *EntityHandler, writer http.ResponseWriter, request *http.Request) {
				handler.handlePutEntity(writer, request, "1")
			},
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			path:   "/Products(1)",
			run: func(handler *EntityHandler, writer http.ResponseWriter, request *http.Request) {
				handler.handleDeleteEntity(writer, request, "1")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			called := false
			handler := createTestHandlerWithMetadata()
			handler.policy = denyMutationPolicy{}
			handler.overwrite = &entityOverwriteHandlers{
				create: func(*OverwriteContext, interface{}) (interface{}, error) {
					called = true
					return nil, nil
				},
				update: func(*OverwriteContext, map[string]interface{}, bool) (interface{}, error) {
					called = true
					return nil, nil
				},
				delete: func(*OverwriteContext) error {
					called = true
					return nil
				},
			}

			request := httptest.NewRequest(test.method, test.path, bytes.NewBufferString(`{"Name":"updated"}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer test")
			writer := httptest.NewRecorder()

			test.run(handler, writer, request)

			if writer.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", writer.Code, http.StatusForbidden)
			}
			if called {
				t.Fatal("overwrite handler was called after authorization denial")
			}
		})
	}
}
