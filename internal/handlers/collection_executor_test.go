package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/hookerrors"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/gorm"
)

type odataErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Message string `json:"message"`
		} `json:"details"`
	} `json:"error"`
}

func decodeODataError(t *testing.T, recorder *httptest.ResponseRecorder) odataErrorResponse {
	t.Helper()

	var resp odataErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	return resp
}

func TestExecuteCollectionQueryErrors(t *testing.T) {
	handler := NewEntityHandler(nil, &metadata.EntityMetadata{EntityName: "Widget", EntitySetName: "Widgets"}, nil)

	t.Run("errRequestHandled", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

		handler.executeCollectionQuery(recorder, request, &collectionExecutionContext{
			ParseQueryOptions: func() (*query.QueryOptions, error) {
				return nil, errRequestHandled
			},
			FetchFunc: func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error) {
				t.Fatal("FetchFunc should not be called")
				return nil, nil
			},
			WriteResponse: func(*query.QueryOptions, interface{}, *int64, *string) error {
				t.Fatal("WriteResponse should not be called")
				return nil
			},
		})

		if recorder.Body.Len() != 0 {
			t.Fatalf("expected empty response body, got %q", recorder.Body.String())
		}

		if len(recorder.Header()) != 0 {
			t.Fatalf("expected no headers, got %v", recorder.Header())
		}
	})

	t.Run("GeospatialNotEnabledError", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

		handler.executeCollectionQuery(recorder, request, &collectionExecutionContext{
			ParseQueryOptions: func() (*query.QueryOptions, error) {
				return nil, &GeospatialNotEnabledError{}
			},
			FetchFunc: func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error) {
				t.Fatal("FetchFunc should not be called")
				return nil, nil
			},
			WriteResponse: func(*query.QueryOptions, interface{}, *int64, *string) error {
				t.Fatal("WriteResponse should not be called")
				return nil
			},
		})

		if recorder.Code != http.StatusNotImplemented {
			t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, recorder.Code)
		}

		resp := decodeODataError(t, recorder)
		if resp.Error.Code != "501" {
			t.Fatalf("expected code 501, got %q", resp.Error.Code)
		}
		if resp.Error.Message != "Geospatial features not enabled" {
			t.Fatalf("expected message %q, got %q", "Geospatial features not enabled", resp.Error.Message)
		}
		if len(resp.Error.Details) != 1 || resp.Error.Details[0].Message != "geospatial features are not enabled for this service" {
			t.Fatalf("expected geospatial detail, got %#v", resp.Error.Details)
		}
	})

	t.Run("HookError", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

		handler.executeCollectionQuery(recorder, request, &collectionExecutionContext{
			ParseQueryOptions: func() (*query.QueryOptions, error) {
				return nil, &hookerrors.HookError{
					StatusCode: http.StatusConflict,
					Message:    "Hook failed",
					Err:        errors.New("hook detail"),
				}
			},
			FetchFunc: func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error) {
				t.Fatal("FetchFunc should not be called")
				return nil, nil
			},
			WriteResponse: func(*query.QueryOptions, interface{}, *int64, *string) error {
				t.Fatal("WriteResponse should not be called")
				return nil
			},
		})

		if recorder.Code != http.StatusConflict {
			t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
		}

		resp := decodeODataError(t, recorder)
		if resp.Error.Code != "409" {
			t.Fatalf("expected code 409, got %q", resp.Error.Code)
		}
		if resp.Error.Message != "Hook failed" {
			t.Fatalf("expected message %q, got %q", "Hook failed", resp.Error.Message)
		}
		if len(resp.Error.Details) != 1 || resp.Error.Details[0].Message != "hook detail" {
			t.Fatalf("expected hook detail, got %#v", resp.Error.Details)
		}
	})

	t.Run("collectionRequestError defaults", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

		handler.executeCollectionQuery(recorder, request, &collectionExecutionContext{
			ParseQueryOptions: func() (*query.QueryOptions, error) {
				return nil, &collectionRequestError{Message: "request failed"}
			},
			FetchFunc: func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error) {
				t.Fatal("FetchFunc should not be called")
				return nil, nil
			},
			WriteResponse: func(*query.QueryOptions, interface{}, *int64, *string) error {
				t.Fatal("WriteResponse should not be called")
				return nil
			},
		})

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
		}

		resp := decodeODataError(t, recorder)
		if resp.Error.Code != "400" {
			t.Fatalf("expected code 400, got %q", resp.Error.Code)
		}
		if resp.Error.Message != ErrMsgInvalidQueryOptions {
			t.Fatalf("expected message %q, got %q", ErrMsgInvalidQueryOptions, resp.Error.Message)
		}
		if len(resp.Error.Details) != 1 || resp.Error.Details[0].Message != "request failed" {
			t.Fatalf("expected detail %q, got %#v", "request failed", resp.Error.Details)
		}
	})

	t.Run("collectionRequestError custom", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

		handler.executeCollectionQuery(recorder, request, &collectionExecutionContext{
			ParseQueryOptions: func() (*query.QueryOptions, error) {
				return nil, &collectionRequestError{
					StatusCode: http.StatusConflict,
					ErrorCode:  "CustomCode",
					Message:    "custom detail",
				}
			},
			FetchFunc: func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error) {
				t.Fatal("FetchFunc should not be called")
				return nil, nil
			},
			WriteResponse: func(*query.QueryOptions, interface{}, *int64, *string) error {
				t.Fatal("WriteResponse should not be called")
				return nil
			},
		})

		if recorder.Code != http.StatusConflict {
			t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
		}

		resp := decodeODataError(t, recorder)
		if resp.Error.Code != "409" {
			t.Fatalf("expected code 409, got %q", resp.Error.Code)
		}
		if resp.Error.Message != "CustomCode" {
			t.Fatalf("expected message %q, got %q", "CustomCode", resp.Error.Message)
		}
		if len(resp.Error.Details) != 1 || resp.Error.Details[0].Message != "custom detail" {
			t.Fatalf("expected detail %q, got %#v", "custom detail", resp.Error.Details)
		}
	})

	t.Run("generic error", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/widgets", nil)
		queryOptions := &query.QueryOptions{}

		handler.executeCollectionQuery(recorder, request, &collectionExecutionContext{
			ParseQueryOptions: func() (*query.QueryOptions, error) {
				return queryOptions, nil
			},
			FetchFunc: func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error) {
				return nil, errors.New("fetch failed")
			},
			WriteResponse: func(*query.QueryOptions, interface{}, *int64, *string) error {
				t.Fatal("WriteResponse should not be called")
				return nil
			},
		})

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
		}

		resp := decodeODataError(t, recorder)
		if resp.Error.Code != "500" {
			t.Fatalf("expected code 500, got %q", resp.Error.Code)
		}
		if resp.Error.Message != ErrMsgDatabaseError {
			t.Fatalf("expected message %q, got %q", ErrMsgDatabaseError, resp.Error.Message)
		}
		if len(resp.Error.Details) != 1 || resp.Error.Details[0].Message != "fetch failed" {
			t.Fatalf("expected detail %q, got %#v", "fetch failed", resp.Error.Details)
		}
	})
}

func TestExecuteCollectionQueryHappyPath(t *testing.T) {
	handler := NewEntityHandler(nil, &metadata.EntityMetadata{EntityName: "Widget", EntitySetName: "Widgets"}, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

	queryOptions := &query.QueryOptions{}
	var callOrder []string

	count := int64(3)
	nextLinkValue := "next-link"
	fetchResults := []string{"a", "b"}

	handler.executeCollectionQuery(recorder, request, &collectionExecutionContext{
		ParseQueryOptions: func() (*query.QueryOptions, error) {
			callOrder = append(callOrder, "ParseQueryOptions")
			return queryOptions, nil
		},
		BeforeRead: func(opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
			callOrder = append(callOrder, "BeforeRead")
			if opts != queryOptions {
				t.Fatalf("expected query options to match")
			}
			return []func(*gorm.DB) *gorm.DB{func(db *gorm.DB) *gorm.DB { return db }}, nil
		},
		CountFunc: func(opts *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (*int64, error) {
			callOrder = append(callOrder, "CountFunc")
			if opts != queryOptions {
				t.Fatalf("expected query options to match")
			}
			if len(scopes) != 1 {
				t.Fatalf("expected one scope, got %d", len(scopes))
			}
			return &count, nil
		},
		FetchFunc: func(opts *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, error) {
			callOrder = append(callOrder, "FetchFunc")
			if opts != queryOptions {
				t.Fatalf("expected query options to match")
			}
			if len(scopes) != 1 {
				t.Fatalf("expected one scope, got %d", len(scopes))
			}
			return fetchResults, nil
		},
		NextLinkFunc: func(opts *query.QueryOptions, results interface{}) (*string, interface{}, error) {
			callOrder = append(callOrder, "NextLinkFunc")
			if opts != queryOptions {
				t.Fatalf("expected query options to match")
			}
			if !reflect.DeepEqual(results, fetchResults) {
				t.Fatalf("expected fetch results")
			}
			return &nextLinkValue, "next-results", nil
		},
		AfterRead: func(opts *query.QueryOptions, results interface{}) (interface{}, bool, error) {
			callOrder = append(callOrder, "AfterRead")
			if opts != queryOptions {
				t.Fatalf("expected query options to match")
			}
			if results != "next-results" {
				t.Fatalf("expected next-results")
			}
			return "override-results", true, nil
		},
		WriteResponse: func(opts *query.QueryOptions, results interface{}, totalCount *int64, nextLink *string) error {
			callOrder = append(callOrder, "WriteResponse")
			if opts != queryOptions {
				t.Fatalf("expected query options to match")
			}
			if results != "override-results" {
				t.Fatalf("expected override results, got %v", results)
			}
			if totalCount == nil || *totalCount != count {
				t.Fatalf("expected count %d, got %v", count, totalCount)
			}
			if nextLink == nil || *nextLink != nextLinkValue {
				t.Fatalf("expected next link %q, got %v", nextLinkValue, nextLink)
			}
			recorder.WriteHeader(http.StatusOK)
			return nil
		},
	})

	expectedOrder := []string{
		"ParseQueryOptions",
		"BeforeRead",
		"CountFunc",
		"FetchFunc",
		"NextLinkFunc",
		"AfterRead",
		"WriteResponse",
	}

	if len(callOrder) != len(expectedOrder) {
		t.Fatalf("expected %d calls, got %d: %v", len(expectedOrder), len(callOrder), callOrder)
	}
	for i, call := range expectedOrder {
		if callOrder[i] != call {
			t.Fatalf("expected call %q at position %d, got %q", call, i, callOrder[i])
		}
	}
}

func TestExecuteCollectionQuery_MissingCallbacks(t *testing.T) {
	handler := NewEntityHandler(nil, &metadata.EntityMetadata{EntityName: "Widget", EntitySetName: "Widgets"}, nil)

	t.Run("nil context", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

		handler.executeCollectionQuery(recorder, request, nil)

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
		}

		resp := decodeODataError(t, recorder)
		if resp.Error.Code != "500" {
			t.Fatalf("expected error code '500', got %q", resp.Error.Code)
		}
		if resp.Error.Message != "Internal error" {
			t.Fatalf("expected message 'Internal error', got %q", resp.Error.Message)
		}
	})

	t.Run("missing ParseQueryOptions", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

		handler.executeCollectionQuery(recorder, request, &collectionExecutionContext{
			ParseQueryOptions: nil,
			FetchFunc: func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error) {
				return nil, nil
			},
			WriteResponse: func(*query.QueryOptions, interface{}, *int64, *string) error {
				return nil
			},
		})

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
		}

		resp := decodeODataError(t, recorder)
		if resp.Error.Code != "500" {
			t.Fatalf("expected error code '500', got %q", resp.Error.Code)
		}
		if resp.Error.Message != "Internal error" {
			t.Fatalf("expected message 'Internal error', got %q", resp.Error.Message)
		}
	})

	t.Run("missing FetchFunc", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

		handler.executeCollectionQuery(recorder, request, &collectionExecutionContext{
			ParseQueryOptions: func() (*query.QueryOptions, error) {
				return &query.QueryOptions{}, nil
			},
			FetchFunc: nil,
			WriteResponse: func(*query.QueryOptions, interface{}, *int64, *string) error {
				return nil
			},
		})

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
		}

		resp := decodeODataError(t, recorder)
		if resp.Error.Code != "500" {
			t.Fatalf("expected error code '500', got %q", resp.Error.Code)
		}
		if resp.Error.Message != "Internal error" {
			t.Fatalf("expected message 'Internal error', got %q", resp.Error.Message)
		}
	})

	t.Run("missing WriteResponse", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/widgets", nil)

		handler.executeCollectionQuery(recorder, request, &collectionExecutionContext{
			ParseQueryOptions: func() (*query.QueryOptions, error) {
				return &query.QueryOptions{}, nil
			},
			FetchFunc: func(*query.QueryOptions, []func(*gorm.DB) *gorm.DB) (interface{}, error) {
				return nil, nil
			},
			WriteResponse: nil,
		})

		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
		}

		resp := decodeODataError(t, recorder)
		if resp.Error.Code != "500" {
			t.Fatalf("expected error code '500', got %q", resp.Error.Code)
		}
		if resp.Error.Message != "Internal error" {
			t.Fatalf("expected message 'Internal error', got %q", resp.Error.Message)
		}
	})
}
