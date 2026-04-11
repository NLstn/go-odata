package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ContextTestProduct is the entity used by context URL tests.
type ContextTestProduct struct {
	ID       int     `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string  `json:"Name"`
	Price    float64 `json:"Price"`
	Category string  `json:"Category"`
}

func setupContextTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&ContextTestProduct{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	products := []ContextTestProduct{
		{ID: 1, Name: "Laptop", Price: 999.99, Category: "Electronics"},
		{ID: 2, Name: "Mouse", Price: 29.99, Category: "Electronics"},
		{ID: 3, Name: "Chair", Price: 249.99, Category: "Furniture"},
	}
	for _, p := range products {
		if err := db.Create(&p).Error; err != nil {
			t.Fatalf("Failed to create product: %v", err)
		}
	}

	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := svc.RegisterEntity(&ContextTestProduct{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	return svc, db
}

// getContextURL is a helper that parses @odata.context from a JSON response body.
func getContextURL(t *testing.T, body []byte) string {
	t.Helper()

	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	ctx, ok := m["@odata.context"]
	if !ok {
		return ""
	}
	s, ok := ctx.(string)
	if !ok {
		t.Fatalf("@odata.context is not a string: %T", ctx)
	}
	return s
}

// TestContextURL_PlainCollection verifies that a plain collection request produces
// #ContextTestProducts as the context URL fragment.
func TestContextURL_PlainCollection(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts", nil)
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ContextTestProducts"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

// TestContextURL_CollectionWithSelect verifies that $select causes the context URL to include
// the selected property list: #ContextTestProducts(Name,Price).
func TestContextURL_CollectionWithSelect(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts?$select=Name,Price", nil)
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ContextTestProducts(Name,Price)"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

// TestContextURL_SingleEntityNoSelect verifies that a single-entity GET produces
// #ContextTestProducts/$entity when no $select is present.
func TestContextURL_SingleEntityNoSelect(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts(1)", nil)
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ContextTestProducts/$entity"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

// TestContextURL_SingleEntityWithSelect verifies that a single-entity GET with $select produces
// #ContextTestProducts(Name,Price)/$entity.
func TestContextURL_SingleEntityWithSelect(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts(1)?$select=Name,Price", nil)
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ContextTestProducts(Name,Price)/$entity"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

// TestContextURL_MetadataNoneOmitsContext verifies that odata.metadata=none suppresses @odata.context.
func TestContextURL_MetadataNoneOmitsContext(t *testing.T) {
	svc, _ := setupContextTestService(t)

	for _, url := range []string{
		"/ContextTestProducts",
		"/ContextTestProducts?$select=Name",
		"/ContextTestProducts(1)",
		"/ContextTestProducts(1)?$select=Name",
	} {
		t.Run(url, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.Header.Set("Accept", "application/json;odata.metadata=none")
			w := httptest.NewRecorder()
			svc.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected 200, got %d", w.Code)
			}

			ctx := getContextURL(t, w.Body.Bytes())
			if ctx != "" {
				t.Errorf("Expected @odata.context to be absent with metadata=none, got %q", ctx)
			}
		})
	}
}

// TestContextURL_ApplyAggregate verifies that $apply=aggregate(...) produces a context URL
// containing the aggregate alias properties: #ContextTestProducts(TotalPrice).
func TestContextURL_ApplyAggregate(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts", nil)
	q := req.URL.Query()
	q.Set("$apply", "aggregate(Price with sum as TotalPrice)")
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ContextTestProducts(TotalPrice)"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

// TestContextURL_ApplyGroupByWithAggregate verifies that $apply=groupby(...,aggregate(...)) produces
// a context URL that lists both the groupby properties and the aggregate alias:
// #ContextTestProducts(Category,Count).
func TestContextURL_ApplyGroupByWithAggregate(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts", nil)
	q := req.URL.Query()
	q.Set("$apply", "groupby((Category),aggregate($count as Count))")
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ContextTestProducts(Category,Count)"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

// TestContextURL_ApplyGroupByNoAggregate verifies that $apply=groupby((Category)) produces
// #ContextTestProducts(Category) as the context URL fragment.
func TestContextURL_ApplyGroupByNoAggregate(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet,
		"/ContextTestProducts?$apply=groupby((Category))", nil)
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ContextTestProducts(Category)"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

func TestContextURL_ApplyJoin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&ApplyJoinProduct{}, &ApplyJoinSale{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	product := ApplyJoinProduct{ID: 1, Name: "Laptop", Category: "Electronics"}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("Failed to create product: %v", err)
	}
	if err := db.Create(&ApplyJoinSale{ID: 1, ApplyJoinProductID: 1, Amount: 999}).Error; err != nil {
		t.Fatalf("Failed to create sale: %v", err)
	}

	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := svc.RegisterEntity(&ApplyJoinProduct{}); err != nil {
		t.Fatalf("Failed to register product entity: %v", err)
	}
	if err := svc.RegisterEntity(&ApplyJoinSale{}); err != nil {
		t.Fatalf("Failed to register sale entity: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ApplyJoinProducts?$apply=join(Sales%20as%20Sale)", nil)
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ApplyJoinProducts(Sale())"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

// TestContextURL_SelectSingleProp verifies that $select with a single property produces
// #ContextTestProducts(Name) as the context URL fragment.
func TestContextURL_SelectSingleProp(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts?$select=Name", nil)
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ContextTestProducts(Name)"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

// TestContextURL_FullMetadataWithSelect verifies that full metadata still includes
// the select-shaped context URL.
func TestContextURL_FullMetadataWithSelect(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts?$select=Name,Price", nil)
	req.Header.Set("Accept", "application/json;odata.metadata=full")
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ContextTestProducts(Name,Price)"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

// TestContextURL_EntityFullMetadataWithSelect verifies entity context URL with full metadata + $select.
func TestContextURL_EntityFullMetadataWithSelect(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts(1)?$select=Name", nil)
	req.Header.Set("Accept", "application/json;odata.metadata=full")
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	ctx := getContextURL(t, w.Body.Bytes())
	wantSuffix := "$metadata#ContextTestProducts(Name)/$entity"
	if ctx == "" {
		t.Fatal("@odata.context is missing from response")
	}
	if len(ctx) < len(wantSuffix) || ctx[len(ctx)-len(wantSuffix):] != wantSuffix {
		t.Errorf("@odata.context = %q, want suffix %q", ctx, wantSuffix)
	}
}

// TestContextURL_ApplyNoneMetadata verifies $apply + metadata=none omits context URL.
func TestContextURL_ApplyNoneMetadata(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts", nil)
	q := req.URL.Query()
	q.Set("$apply", "aggregate(Price with sum as TotalPrice)")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json;odata.metadata=none")

	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ctx := getContextURL(t, w.Body.Bytes())
	if ctx != "" {
		t.Errorf("Expected @odata.context to be absent with metadata=none, got %q", ctx)
	}
}

// TestContextURL_ExactFormat verifies the complete, exact @odata.context URL for a collection
// with $select using a known host.
func TestContextURL_ExactFormat(t *testing.T) {
	svc, _ := setupContextTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/ContextTestProducts?$select=Name,Price", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	ctx := getContextURL(t, w.Body.Bytes())
	want := "http://example.com/$metadata#ContextTestProducts(Name,Price)"
	if ctx != want {
		t.Errorf("@odata.context = %q, want %q", ctx, want)
	}
}
