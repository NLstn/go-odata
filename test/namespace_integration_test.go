package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type NamespaceProduct struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name"`
}

type NamespaceFunctionResult struct {
	Info string `json:"Info"`
}

func setupNamespaceDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	if err := db.AutoMigrate(&NamespaceProduct{}); err != nil {
		t.Fatalf("failed to migrate namespace product: %v", err)
	}
	return db
}

func TestServiceCustomNamespace(t *testing.T) {
	db := setupNamespaceDB(t)
	service := odata.NewService(db)

	if err := service.RegisterEntity(&NamespaceProduct{}); err != nil {
		t.Fatalf("RegisterEntity failed: %v", err)
	}

	if err := service.SetNamespace("Contoso"); err != nil {
		t.Fatalf("SetNamespace failed: %v", err)
	}

	if err := service.RegisterFunction(odata.FunctionDefinition{
		Name:       "GetNamespaceInfo",
		ReturnType: reflect.TypeOf(NamespaceFunctionResult{}),
		Handler: func(http.ResponseWriter, *http.Request, interface{}, map[string]interface{}) (interface{}, error) {
			return NamespaceFunctionResult{Info: "Configured"}, nil
		},
	}); err != nil {
		t.Fatalf("RegisterFunction failed: %v", err)
	}

	// Verify XML metadata namespace
	req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status for metadata: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `Namespace="Contoso"`) {
		t.Fatalf("expected namespace Contoso in metadata, got %s", body)
	}
	if !strings.Contains(body, `Type="Contoso.NamespaceProduct"`) {
		t.Fatalf("expected fully qualified type in metadata, got %s", body)
	}

	// Verify JSON metadata namespace
	reqJSON := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
	wJSON := httptest.NewRecorder()
	service.ServeHTTP(wJSON, reqJSON)
	if wJSON.Code != http.StatusOK {
		t.Fatalf("unexpected status for JSON metadata: %d", wJSON.Code)
	}

	var jsonMeta map[string]interface{}
	if err := json.Unmarshal(wJSON.Body.Bytes(), &jsonMeta); err != nil {
		t.Fatalf("failed to parse JSON metadata: %v", err)
	}

	schema, ok := jsonMeta["Contoso"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected Contoso schema in JSON metadata, got %v", jsonMeta)
	}
	if _, ok := schema["NamespaceProduct"].(map[string]interface{}); !ok {
		t.Fatalf("expected NamespaceProduct type in Contoso schema, got %v", schema)
	}
	container, ok := schema["Container"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected Container section in JSON metadata, got %v", schema)
	}
	entitySet, ok := container["NamespaceProducts"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected NamespaceProducts entity set metadata, got %v", container)
	}
	if entityType, ok := entitySet["$Type"].(string); !ok || entityType != "Contoso.NamespaceProduct" {
		t.Fatalf("expected NamespaceProducts $Type Contoso.NamespaceProduct, got %v", entitySet["$Type"])
	}
	if entityContainer, ok := jsonMeta["$EntityContainer"].(string); !ok || entityContainer != "Contoso.Container" {
		t.Fatalf("expected $EntityContainer Contoso.Container, got %v", jsonMeta["$EntityContainer"])
	}

	// Seed data and verify @odata.type reflects namespace
	if err := db.Create(&NamespaceProduct{ID: 1, Name: "Widget"}).Error; err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	reqEntity := httptest.NewRequest(http.MethodGet, "/NamespaceProducts(1)", nil)
	reqEntity.Header.Set("Accept", "application/json;odata.metadata=full")
	wEntity := httptest.NewRecorder()
	service.ServeHTTP(wEntity, reqEntity)
	if wEntity.Code != http.StatusOK {
		t.Fatalf("unexpected status for entity fetch: %d", wEntity.Code)
	}

	var entity map[string]interface{}
	if err := json.Unmarshal(wEntity.Body.Bytes(), &entity); err != nil {
		t.Fatalf("failed to parse entity response: %v", err)
	}
	if entityType, ok := entity["@odata.type"].(string); !ok || entityType != "#Contoso.NamespaceProduct" {
		t.Fatalf("expected @odata.type=#Contoso.NamespaceProduct, got %v", entity["@odata.type"])
	}

	reqFunction := httptest.NewRequest(http.MethodGet, "/GetNamespaceInfo()", nil)
	reqFunction.Header.Set("Accept", "application/json")
	wFunction := httptest.NewRecorder()
	service.ServeHTTP(wFunction, reqFunction)
	if wFunction.Code != http.StatusOK {
		t.Fatalf("unexpected status for function: %d", wFunction.Code)
	}

	var functionPayload map[string]interface{}
	if err := json.Unmarshal(wFunction.Body.Bytes(), &functionPayload); err != nil {
		t.Fatalf("failed to parse function response: %v", err)
	}
	contextValue, ok := functionPayload["@odata.context"].(string)
	if !ok {
		t.Fatalf("expected function @odata.context string, got %T", functionPayload["@odata.context"])
	}
	if !strings.Contains(contextValue, "$metadata#Contoso.NamespaceFunctionResult") {
		t.Fatalf("expected namespace-qualified context URL, got %s", contextValue)
	}
}
