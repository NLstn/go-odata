package odata_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDynamicEntityService(t *testing.T) *odata.Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func dynamicProductDefinition(disabledMethods ...string) odata.EntityDefinition {
	nullable := false
	return odata.EntityDefinition{
		Name:          "DynamicProduct",
		EntitySetName: "DynamicProducts",
		Properties: []odata.PropertyDefinition{
			{Name: "ID", Type: odata.EdmInt64},
			{Name: "Name", Type: odata.EdmString, Nullable: &nullable},
			{Name: "Price", Type: odata.EdmDecimal},
		},
		Keys:            []string{"ID"},
		DisabledMethods: disabledMethods,
	}
}

func performDynamicRequest(service *odata.Service, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	service.ServeHTTP(response, request)
	return response
}

func TestRegisterDynamicEntityPublishesReadOnlyEntityAfterCacheWarmup(t *testing.T) {
	service := newDynamicEntityService(t)
	performDynamicRequest(service, http.MethodGet, "/", "")
	performDynamicRequest(service, http.MethodGet, "/$metadata", "")

	row := map[string]interface{}{"ID": int64(1), "Name": "Keyboard", "Price": 99.5}
	var collectionCalled, entityCalled, countCalled bool
	err := service.RegisterDynamicEntity(
		dynamicProductDefinition(http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete),
		&odata.EntityOverwrite{
			GetCollection: func(ctx *odata.OverwriteContext) (*odata.CollectionResult, error) {
				collectionCalled = true
				if ctx.QueryOptions.Top == nil || *ctx.QueryOptions.Top != 1 || !ctx.QueryOptions.Count {
					t.Errorf("query options = %+v, want $top=1 and $count=true", ctx.QueryOptions)
				}
				count := int64(1)
				return &odata.CollectionResult{Items: []map[string]interface{}{row}, Count: &count}, nil
			},
			GetEntity: func(ctx *odata.OverwriteContext) (interface{}, error) {
				entityCalled = true
				if got, ok := ctx.EntityKeyValues["ID"].(int64); !ok || got != 1 {
					t.Errorf("ID key = %#v, want int64(1)", ctx.EntityKeyValues["ID"])
				}
				return row, nil
			},
			GetCount: func(*odata.OverwriteContext) (int64, error) {
				countCalled = true
				return 1, nil
			},
		},
	)
	if err != nil {
		t.Fatalf("RegisterDynamicEntity() error = %v", err)
	}

	serviceDocument := performDynamicRequest(service, http.MethodGet, "/", "")
	if serviceDocument.Code != http.StatusOK || !strings.Contains(serviceDocument.Body.String(), `"name":"DynamicProducts"`) {
		t.Fatalf("service document status/body = %d %s", serviceDocument.Code, serviceDocument.Body.String())
	}
	metadataDocument := performDynamicRequest(service, http.MethodGet, "/$metadata", "")
	metadataBody := metadataDocument.Body.String()
	for _, expected := range []string{
		`EntityType Name="DynamicProduct"`,
		`PropertyRef Name="ID"`,
		`Property Name="ID" Type="Edm.Int64" Nullable="false"`,
		`Property Name="Name" Type="Edm.String" Nullable="false"`,
		`EntitySet Name="DynamicProducts"`,
	} {
		if !strings.Contains(metadataBody, expected) {
			t.Errorf("metadata does not contain %q: %s", expected, metadataBody)
		}
	}

	collection := performDynamicRequest(service, http.MethodGet, "/DynamicProducts?$top=1&$count=true", "")
	if collection.Code != http.StatusOK || !collectionCalled {
		t.Fatalf("collection status = %d, called = %t, body = %s", collection.Code, collectionCalled, collection.Body.String())
	}
	if !strings.Contains(collection.Body.String(), `"@odata.id":"http://example.com/DynamicProducts(1)"`) {
		t.Fatalf("collection body lacks entity ID: %s", collection.Body.String())
	}

	entity := performDynamicRequest(service, http.MethodGet, "/DynamicProducts(1)", "")
	if entity.Code != http.StatusOK || !entityCalled {
		t.Fatalf("entity status = %d, called = %t, body = %s", entity.Code, entityCalled, entity.Body.String())
	}
	count := performDynamicRequest(service, http.MethodGet, "/DynamicProducts/$count", "")
	if count.Code != http.StatusOK || strings.TrimSpace(count.Body.String()) != "1" || !countCalled {
		t.Fatalf("count status/body/called = %d %q %t", count.Code, count.Body.String(), countCalled)
	}
	entityCalled = false
	invalidKey := performDynamicRequest(service, http.MethodGet, "/DynamicProducts(not-an-integer)", "")
	if invalidKey.Code != http.StatusBadRequest || entityCalled {
		t.Fatalf("invalid key status/called = %d %t, body = %s", invalidKey.Code, entityCalled, invalidKey.Body.String())
	}
}

func TestRegisterDynamicEntitySupportsMapBackedMutations(t *testing.T) {
	service := newDynamicEntityService(t)
	created := false
	updates := make([]bool, 0, 2)
	selectSeen := false
	deleted := false

	err := service.RegisterDynamicEntity(dynamicProductDefinition(http.MethodGet), &odata.EntityOverwrite{
		Create: func(_ *odata.OverwriteContext, entity interface{}) (interface{}, error) {
			payload, ok := entity.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("create payload type %T", entity)
			}
			created = true
			return payload, nil
		},
		Update: func(ctx *odata.OverwriteContext, updateData map[string]interface{}, fullReplace bool) (interface{}, error) {
			updates = append(updates, fullReplace)
			selectSeen = len(ctx.QueryOptions.Select) == 1 && ctx.QueryOptions.Select[0] == "Name"
			return map[string]interface{}{"ID": int64(1), "Name": updateData["Name"], "Price": 10.5}, nil
		},
		Delete: func(*odata.OverwriteContext) error {
			deleted = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterDynamicEntity() error = %v", err)
	}

	invalidCreate := performDynamicRequest(service, http.MethodPost, "/DynamicProducts", `{"ID":"invalid","Name":"Keyboard"}`)
	if invalidCreate.Code != http.StatusBadRequest || created {
		t.Fatalf("invalid create status/called = %d %t, body = %s", invalidCreate.Code, created, invalidCreate.Body.String())
	}

	create := performDynamicRequest(service, http.MethodPost, "/DynamicProducts", `{"ID":1,"Name":"Keyboard"}`)
	if create.Code != http.StatusCreated || !created {
		t.Fatalf("create status/called = %d %t, body = %s", create.Code, created, create.Body.String())
	}
	invalidUpdate := performDynamicRequest(service, http.MethodPatch, "/DynamicProducts(1)", `{"Price":"invalid"}`)
	if invalidUpdate.Code != http.StatusBadRequest || len(updates) != 0 {
		t.Fatalf("invalid update status/calls = %d %v, body = %s", invalidUpdate.Code, updates, invalidUpdate.Body.String())
	}
	patchRequest := httptest.NewRequest(http.MethodPatch, "/DynamicProducts(1)?$select=Name", strings.NewReader(`{"Name":"Mouse"}`))
	patchRequest.Header.Set("Content-Type", "application/json")
	patchRequest.Header.Set("Prefer", "return=representation")
	patchResponse := httptest.NewRecorder()
	service.ServeHTTP(patchResponse, patchRequest)
	if patchResponse.Code != http.StatusOK || !selectSeen || strings.Contains(patchResponse.Body.String(), `"Price"`) {
		t.Fatalf("projected PATCH status/select/body = %d %t %s", patchResponse.Code, selectSeen, patchResponse.Body.String())
	}

	invalidOption := performDynamicRequest(service, http.MethodPatch, "/DynamicProducts(1)?$filter=Name%20eq%20'Mouse'", `{"Name":"Mouse"}`)
	if invalidOption.Code != http.StatusBadRequest || len(updates) != 1 {
		t.Fatalf("invalid option status/calls = %d %v", invalidOption.Code, updates)
	}

	putResponse := performDynamicRequest(service, http.MethodPut, "/DynamicProducts(1)", `{"ID":1,"Name":"Mouse"}`)
	if putResponse.Code != http.StatusOK && putResponse.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d, body = %s", putResponse.Code, putResponse.Body.String())
	}
	if len(updates) != 2 || updates[0] || !updates[1] {
		t.Fatalf("update full-replace flags = %v, want [false true]", updates)
	}
	deleteResponse := performDynamicRequest(service, http.MethodDelete, "/DynamicProducts(1)", "")
	if deleteResponse.Code != http.StatusNoContent || !deleted {
		t.Fatalf("delete status/called = %d %t", deleteResponse.Code, deleted)
	}
}

func TestRegisterDynamicEntityPreservesNumericPayloads(t *testing.T) {
	service := newDynamicEntityService(t)
	var captured map[string]interface{}
	err := service.RegisterDynamicEntity(dynamicProductDefinition(http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodDelete), &odata.EntityOverwrite{
		Create: func(_ *odata.OverwriteContext, entity interface{}) (interface{}, error) {
			captured = entity.(map[string]interface{})
			return captured, nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterDynamicEntity() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/DynamicProducts", strings.NewReader(`{"ID":"9223372036854775807","Name":"Keyboard","Price":"1234567890.123456789"}`))
	request.Header.Set("Content-Type", "application/json;IEEE754Compatible=true")
	response := httptest.NewRecorder()
	service.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if id, ok := captured["ID"].(int64); !ok || id != int64(9223372036854775807) {
		t.Fatalf("ID = %#v, want exact max int64", captured["ID"])
	}
	if got := fmt.Sprint(captured["Price"]); got != "1234567890.123456789" {
		t.Fatalf("Price = %q, want exact decimal", got)
	}
	if got := response.Header().Get("Location"); got != "http://example.com/DynamicProducts(9223372036854775807)" {
		t.Fatalf("Location = %q", got)
	}

	invalid := performDynamicRequest(service, http.MethodPost, "/DynamicProducts", `{"ID":1.5,"Name":"Keyboard"}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("fractional Int64 status = %d, want %d", invalid.Code, http.StatusBadRequest)
	}
}

func TestRegisterDynamicEntityRejectsIncompleteCallbacksAtomically(t *testing.T) {
	service := newDynamicEntityService(t)
	err := service.RegisterDynamicEntity(
		dynamicProductDefinition(http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete),
		&odata.EntityOverwrite{
			GetCollection: func(*odata.OverwriteContext) (*odata.CollectionResult, error) {
				return &odata.CollectionResult{Items: []map[string]interface{}{}}, nil
			},
			GetEntity: func(*odata.OverwriteContext) (interface{}, error) { return nil, nil },
		},
	)
	if err == nil || !strings.Contains(err.Error(), "requires GetCount") {
		t.Fatalf("RegisterDynamicEntity() error = %v, want missing GetCount", err)
	}

	response := performDynamicRequest(service, http.MethodGet, "/DynamicProducts", "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("failed registration route status = %d, want %d", response.Code, http.StatusNotFound)
	}
	serviceDocument := performDynamicRequest(service, http.MethodGet, "/", "")
	body, readErr := io.ReadAll(serviceDocument.Body)
	if readErr != nil {
		t.Fatalf("ReadAll() error = %v", readErr)
	}
	if bytes.Contains(body, []byte("DynamicProducts")) {
		t.Fatalf("failed registration leaked into service document: %s", body)
	}
}
