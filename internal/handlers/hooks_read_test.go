package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type readHookEntity struct {
	ID       int             `json:"ID" odata:"key"`
	Name     string          `json:"Name"`
	Tenant   string          `json:"Tenant"`
	Children []readHookChild `json:"Children" gorm:"foreignKey:ReadHookEntityID;references:ID"`
}

type readHookChild struct {
	ID               int    `json:"ID" odata:"key"`
	ReadHookEntityID int    `json:"ReadHookEntityID"`
	Tenant           string `json:"Tenant"`
	Value            string `json:"Value"`
}

func tenantScopes(r *http.Request) ([]func(*gorm.DB) *gorm.DB, error) {
	tenant := r.Header.Get("X-Tenant")
	if tenant == "" {
		return nil, fmt.Errorf("missing tenant")
	}
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("tenant = ?", tenant)
	}
	return []func(*gorm.DB) *gorm.DB{scope}, nil
}

func (readHookEntity) BeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	if r.Header.Get("X-Deny") == "collection" {
		return nil, fmt.Errorf("collection denied")
	}
	return tenantScopes(r)
}

func (readHookEntity) AfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error) {
	if r.Header.Get("X-Override") == "collection" {
		return []map[string]interface{}{{"Custom": "collection"}}, nil
	}
	return nil, nil
}

func (readHookEntity) BeforeReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	if r.Header.Get("X-Deny") == "entity" {
		return nil, fmt.Errorf("entity denied")
	}
	return tenantScopes(r)
}

func (readHookEntity) AfterReadEntity(ctx context.Context, r *http.Request, opts *query.QueryOptions, entity interface{}) (interface{}, error) {
	if r.Header.Get("X-Override") == "entity" {
		return map[string]interface{}{"Message": "entity"}, nil
	}
	if actual, ok := entity.(*readHookEntity); ok {
		actual.Name = actual.Name + "-mutated"
	}
	return nil, nil
}

func (readHookChild) BeforeReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions) ([]func(*gorm.DB) *gorm.DB, error) {
	return tenantScopes(r)
}

func (readHookChild) AfterReadCollection(ctx context.Context, r *http.Request, opts *query.QueryOptions, results interface{}) (interface{}, error) {
	if r.Header.Get("X-Override") == "nav" {
		return []map[string]interface{}{{"Custom": "nav"}}, nil
	}
	return nil, nil
}

func setupReadHookHandler(t *testing.T) *EntityHandler {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&readHookEntity{}, &readHookChild{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}

	parents := []readHookEntity{
		{ID: 1, Name: "Alpha", Tenant: "tenantA"},
		{ID: 2, Name: "Beta", Tenant: "tenantB"},
	}
	for _, parent := range parents {
		if err := db.Create(&parent).Error; err != nil {
			t.Fatalf("failed to seed parent: %v", err)
		}
	}

	children := []readHookChild{
		{ID: 10, ReadHookEntityID: 1, Tenant: "tenantA", Value: "childA"},
		{ID: 11, ReadHookEntityID: 1, Tenant: "tenantB", Value: "childB other"},
		{ID: 12, ReadHookEntityID: 2, Tenant: "tenantB", Value: "childB"},
	}
	for _, child := range children {
		if err := db.Create(&child).Error; err != nil {
			t.Fatalf("failed to seed child: %v", err)
		}
	}

	entityMeta, err := metadata.AnalyzeEntity(readHookEntity{})
	if err != nil {
		t.Fatalf("failed to analyze entity: %v", err)
	}
	entityMeta.EntitySetName = "ReadHookEntities"

	childMeta, err := metadata.AnalyzeEntity(readHookChild{})
	if err != nil {
		t.Fatalf("failed to analyze child: %v", err)
	}
	childMeta.EntitySetName = "ReadHookChildren"

	handler := NewEntityHandler(db, entityMeta, nil)
	handler.SetEntitiesMetadata(map[string]*metadata.EntityMetadata{
		childMeta.EntityName:    childMeta,
		childMeta.EntitySetName: childMeta,
	})

	return handler
}

func decodeBody(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	return payload
}

func TestHandleGetCollectionReadHooks(t *testing.T) {
	t.Run("scopes apply to data and count", func(t *testing.T) {
		handler := setupReadHookHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/ReadHookEntities?$count=true", nil)
		req.Header.Set("X-Tenant", "tenantA")
		rr := httptest.NewRecorder()

		handler.handleGetCollection(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		payload := decodeBody(t, rr.Body.Bytes())
		count, ok := payload["@odata.count"].(float64)
		if !ok || int(count) != 1 {
			t.Fatalf("expected count 1, got %v", payload["@odata.count"])
		}

		values, ok := payload["value"].([]interface{})
		if !ok || len(values) != 1 {
			t.Fatalf("expected single value, got %v", payload["value"])
		}
		item, ok := values[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map result, got %T", values[0])
		}
		if item["Name"] != "Alpha" {
			t.Fatalf("expected Name Alpha, got %v", item["Name"])
		}
	})

	t.Run("before hook error returns forbidden", func(t *testing.T) {
		handler := setupReadHookHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/ReadHookEntities", nil)
		rr := httptest.NewRecorder()

		handler.handleGetCollection(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", rr.Code)
		}
	})

	t.Run("after hook override honored", func(t *testing.T) {
		handler := setupReadHookHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/ReadHookEntities", nil)
		req.Header.Set("X-Tenant", "tenantA")
		req.Header.Set("X-Override", "collection")
		rr := httptest.NewRecorder()

		handler.handleGetCollection(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		payload := decodeBody(t, rr.Body.Bytes())
		values, ok := payload["value"].([]interface{})
		if !ok || len(values) != 1 {
			t.Fatalf("expected override value, got %v", payload["value"])
		}
		item, ok := values[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected override map, got %T", values[0])
		}
		if item["Custom"] != "collection" {
			t.Fatalf("expected override content, got %v", item["Custom"])
		}
	})
}

func TestHandleCollectionRefScopesApplied(t *testing.T) {
	handler := setupReadHookHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/ReadHookEntities/$ref?$count=true", nil)
	req.Header.Set("X-Tenant", "tenantA")
	rr := httptest.NewRecorder()

	handler.HandleCollectionRef(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	payload := decodeBody(t, rr.Body.Bytes())
	values, ok := payload["value"].([]interface{})
	if !ok || len(values) != 1 {
		t.Fatalf("expected single reference, got %v", payload["value"])
	}
	ref, ok := values[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map ref, got %T", values[0])
	}
	if ref["@odata.id"] != "http://example.com/ReadHookEntities(1)" {
		t.Fatalf("unexpected reference id %v", ref["@odata.id"])
	}
}

func TestHandleGetEntityReadHooks(t *testing.T) {
	t.Run("mutation and scopes", func(t *testing.T) {
		handler := setupReadHookHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/ReadHookEntities(1)", nil)
		req.Header.Set("X-Tenant", "tenantA")
		rr := httptest.NewRecorder()

		handler.handleGetEntity(rr, req, "1")
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		payload := decodeBody(t, rr.Body.Bytes())
		if payload["Name"] != "Alpha-mutated" {
			t.Fatalf("expected mutated name, got %v", payload["Name"])
		}
	})

	t.Run("before hook error", func(t *testing.T) {
		handler := setupReadHookHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/ReadHookEntities(1)", nil)
		rr := httptest.NewRecorder()

		handler.handleGetEntity(rr, req, "1")
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", rr.Code)
		}
	})

	t.Run("after hook override honored", func(t *testing.T) {
		handler := setupReadHookHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/ReadHookEntities(1)", nil)
		req.Header.Set("X-Tenant", "tenantA")
		req.Header.Set("X-Override", "entity")
		rr := httptest.NewRecorder()

		handler.handleGetEntity(rr, req, "1")
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		payload := decodeBody(t, rr.Body.Bytes())
		if _, ok := payload["Message"]; !ok {
			t.Fatalf("expected override payload, got %v", payload)
		}
	})
}

func TestNavigationCollectionReadHooks(t *testing.T) {
	navPropName := "Children"

	t.Run("scopes apply to navigation reads", func(t *testing.T) {
		handler := setupReadHookHandler(t)
		navProp := handler.findNavigationProperty(navPropName)
		if navProp == nil {
			t.Fatalf("navigation property %s not found", navPropName)
		}

		req := httptest.NewRequest(http.MethodGet, "/ReadHookEntities(1)/Children?$count=true", nil)
		req.Header.Set("X-Tenant", "tenantA")
		rr := httptest.NewRecorder()

		handler.handleNavigationCollectionWithQueryOptions(rr, req, "1", navProp, false)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		payload := decodeBody(t, rr.Body.Bytes())
		count, ok := payload["@odata.count"].(float64)
		if !ok || int(count) != 1 {
			t.Fatalf("expected count 1, got %v", payload["@odata.count"])
		}
		values, ok := payload["value"].([]interface{})
		if !ok || len(values) != 1 {
			t.Fatalf("expected single child, got %v", payload["value"])
		}
		child, ok := values[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected child map, got %T", values[0])
		}
		if child["Value"] != "childA" {
			t.Fatalf("unexpected child value %v", child["Value"])
		}
	})

	t.Run("navigation before hook error", func(t *testing.T) {
		handler := setupReadHookHandler(t)
		navProp := handler.findNavigationProperty(navPropName)
		if navProp == nil {
			t.Fatalf("navigation property %s not found", navPropName)
		}

		req := httptest.NewRequest(http.MethodGet, "/ReadHookEntities(1)/Children", nil)
		rr := httptest.NewRecorder()

		handler.handleNavigationCollectionWithQueryOptions(rr, req, "1", navProp, false)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", rr.Code)
		}
	})

	t.Run("navigation after hook override honored", func(t *testing.T) {
		handler := setupReadHookHandler(t)
		navProp := handler.findNavigationProperty(navPropName)
		if navProp == nil {
			t.Fatalf("navigation property %s not found", navPropName)
		}

		req := httptest.NewRequest(http.MethodGet, "/ReadHookEntities(1)/Children", nil)
		req.Header.Set("X-Tenant", "tenantA")
		req.Header.Set("X-Override", "nav")
		rr := httptest.NewRecorder()

		handler.handleNavigationCollectionWithQueryOptions(rr, req, "1", navProp, false)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		payload := decodeBody(t, rr.Body.Bytes())
		values, ok := payload["value"].([]interface{})
		if !ok || len(values) != 1 {
			t.Fatalf("expected override value, got %v", payload["value"])
		}
		item, ok := values[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected override map, got %T", values[0])
		}
		if item["Custom"] != "nav" {
			t.Fatalf("expected navigation override content, got %v", item["Custom"])
		}
	})
}
