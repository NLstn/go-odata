package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Pure-function unit tests
// ---------------------------------------------------------------------------

func TestResolveContentIDReference_NoMatch(t *testing.T) {
	locations := map[string]string{"1": "/Products(9)"}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain entity set URL", "/Products", "/Products"},
		{"entity URL", "/Products(1)", "/Products(1)"},
		{"no dollar sign", "Products", "Products"},
		{"dollar sign mid-path", "/foo/$1/bar", "/foo/$1/bar"},
		{"unknown content-id", "/$99/bar", "/$99/bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveContentIDReference(tt.input, locations)
			if got != tt.want {
				t.Errorf("resolveContentIDReference(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveContentIDReference_Match(t *testing.T) {
	locations := map[string]string{
		"1":  "/Products(9)",
		"10": "/Products(10)",
		"2":  "/Categories(3)",
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bare $1", "$1", "/Products(9)"},
		{"slash $1", "/$1", "/Products(9)"},
		{"$1 with navigation property", "$1/Descriptions", "/Products(9)/Descriptions"},
		{"/$1 with navigation property", "/$1/Descriptions", "/Products(9)/Descriptions"},
		{"$10 does not match $1", "$10", "/Products(10)"},
		{"$2", "/$2", "/Categories(3)"},
		{"$2 with navigation property", "/$2/Products", "/Categories(3)/Products"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveContentIDReference(tt.input, locations)
			if got != tt.want {
				t.Errorf("resolveContentIDReference(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveContentIDReference_EmptyMap(t *testing.T) {
	got := resolveContentIDReference("/$1/foo", map[string]string{})
	if got != "/$1/foo" {
		t.Errorf("expected URL unchanged with empty map, got %q", got)
	}
}

func TestExtractLocationPath_AbsoluteURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"http://localhost:8080/Products(1)", "/Products(1)"},
		{"http://example.com/odata/Customers(42)", "/odata/Customers(42)"},
		{"https://api.example.com/v1/Orders(ID=5)", "/v1/Orders(ID=5)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractLocationPath(tt.input)
			if got != tt.want {
				t.Errorf("extractLocationPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractLocationPath_RelativeAndInvalid(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/Products(1)", "/Products(1)"},
		{"not-a-url", "not-a-url"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractLocationPath(tt.input)
			if got != tt.want {
				t.Errorf("extractLocationPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration tests – full changeset execution
// ---------------------------------------------------------------------------

// ContentIDRefProduct is a test entity used for Content-ID referencing tests.
type ContentIDRefProduct struct {
	ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
	Name     string  `json:"Name"`
	Price    float64 `json:"Price"`
	Category string  `json:"Category"`
}

// setupContentIDTestHandler creates a batch handler backed by ContentIDRefProduct entities.
// The service handler routes both collection and single-entity requests (GET, POST, PATCH, DELETE).
func setupContentIDTestHandler(t *testing.T) (*BatchHandler, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&ContentIDRefProduct{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	entityMeta, err := metadata.AnalyzeEntity(ContentIDRefProduct{})
	if err != nil {
		t.Fatalf("Failed to analyze entity: %v", err)
	}

	entityHandler := NewEntityHandler(db, entityMeta, nil)
	handlers := map[string]*EntityHandler{
		"ContentIDRefProducts": entityHandler,
	}

	serviceHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if !strings.HasPrefix(path, "ContentIDRefProducts") {
			http.NotFound(w, r)
			return
		}
		if strings.Contains(path, "(") {
			keyStart := strings.Index(path, "(")
			keyEnd := strings.Index(path, ")")
			if keyStart > 0 && keyEnd > keyStart {
				key := path[keyStart+1 : keyEnd]
				entityHandler.HandleEntity(w, r, key)
			}
		} else {
			entityHandler.HandleCollection(w, r)
		}
	})

	batchHandler := NewBatchHandler(db, handlers, serviceHandler, 100)
	return batchHandler, db
}

// TestBatchChangeset_ContentIDURLReference_UpdateCreatedEntity verifies that within a
// changeset a PATCH request can reference an entity created in the same changeset by
// using "$<contentID>" as the URL prefix (OData v4 spec §11.4.9.3).
func TestBatchChangeset_ContentIDURLReference_UpdateCreatedEntity(t *testing.T) {
	handler, db := setupContentIDTestHandler(t)

	batchBoundary := "batch_cid"
	changesetBoundary := "changeset_cid"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /ContentIDRefProducts HTTP/1.1
Content-Type: application/json

{"Name":"Created","Price":10.0,"Category":"A"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /$1 HTTP/1.1
Content-Type: application/json

{"Name":"Updated via $1","Price":20.0,"Category":"A"}

--%s--

--%s--
`, batchBoundary, changesetBoundary,
		changesetBoundary, changesetBoundary, changesetBoundary,
		batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want 200. Body: %s", w.Code, w.Body.String())
	}

	respBody := w.Body.String()

	// Both sub-requests must succeed.
	if !strings.Contains(respBody, "HTTP/1.1 201") {
		t.Errorf("expected 201 Created for POST; body:\n%s", respBody)
	}
	// PATCH returns 200 (with Prefer: return=representation) or 204 (default); accept both.
	if !strings.Contains(respBody, "HTTP/1.1 200") && !strings.Contains(respBody, "HTTP/1.1 204") {
		t.Errorf("expected 200 or 204 for PATCH via $1; body:\n%s", respBody)
	}

	// Exactly one product should exist and its Name should be the patched value.
	var products []ContentIDRefProduct
	db.Find(&products)
	if len(products) != 1 {
		t.Fatalf("expected 1 product in DB, got %d", len(products))
	}
	if products[0].Name != "Updated via $1" {
		t.Errorf("product Name = %q, want %q", products[0].Name, "Updated via $1")
	}
}

// TestBatchChangeset_ContentIDURLReference_DeleteCreatedEntity verifies that a DELETE can
// target an entity created earlier in the same changeset via "$<contentID>".
func TestBatchChangeset_ContentIDURLReference_DeleteCreatedEntity(t *testing.T) {
	handler, db := setupContentIDTestHandler(t)

	batchBoundary := "batch_del"
	changesetBoundary := "changeset_del"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /ContentIDRefProducts HTTP/1.1
Content-Type: application/json

{"Name":"ToDelete","Price":5.0,"Category":"B"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

DELETE /$1 HTTP/1.1


--%s--

--%s--
`, batchBoundary, changesetBoundary,
		changesetBoundary, changesetBoundary, changesetBoundary,
		batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want 200. Body: %s", w.Code, w.Body.String())
	}

	respBody := w.Body.String()

	if !strings.Contains(respBody, "HTTP/1.1 201") {
		t.Errorf("expected 201 Created for POST; body:\n%s", respBody)
	}
	if !strings.Contains(respBody, "HTTP/1.1 204") {
		t.Errorf("expected 204 No Content for DELETE via $1; body:\n%s", respBody)
	}

	// The product must have been deleted.
	var count int64
	db.Model(&ContentIDRefProduct{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 products after DELETE via $1, got %d", count)
	}
}

// TestBatchChangeset_ContentIDURLReference_MultipleReferences verifies that multiple
// Content-IDs can be active simultaneously and are independently resolvable.
func TestBatchChangeset_ContentIDURLReference_MultipleReferences(t *testing.T) {
	handler, db := setupContentIDTestHandler(t)

	batchBoundary := "batch_multi"
	changesetBoundary := "changeset_multi"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /ContentIDRefProducts HTTP/1.1
Content-Type: application/json

{"Name":"Alpha","Price":1.0,"Category":"X"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

POST /ContentIDRefProducts HTTP/1.1
Content-Type: application/json

{"Name":"Beta","Price":2.0,"Category":"Y"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 3

PATCH /$1 HTTP/1.1
Content-Type: application/json

{"Name":"Alpha Updated","Price":1.0,"Category":"X"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 4

PATCH /$2 HTTP/1.1
Content-Type: application/json

{"Name":"Beta Updated","Price":2.0,"Category":"Y"}

--%s--

--%s--
`, batchBoundary, changesetBoundary,
		changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary,
		batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want 200. Body: %s", w.Code, w.Body.String())
	}

	respBody := w.Body.String()

	createdCount := strings.Count(respBody, "HTTP/1.1 201")
	if createdCount != 2 {
		t.Errorf("expected 2 × 201 Created, got %d; body:\n%s", createdCount, respBody)
	}
	// PATCH returns 200 (with Prefer: return=representation) or 204 (default); accept both.
	patchedCount := strings.Count(respBody, "HTTP/1.1 200") + strings.Count(respBody, "HTTP/1.1 204")
	if patchedCount != 2 {
		t.Errorf("expected 2 × 200/204 for PATCH requests, got %d; body:\n%s", patchedCount, respBody)
	}

	var products []ContentIDRefProduct
	db.Find(&products)
	if len(products) != 2 {
		t.Fatalf("expected 2 products in DB, got %d", len(products))
	}

	nameSet := map[string]bool{}
	for _, p := range products {
		nameSet[p.Name] = true
	}
	if !nameSet["Alpha Updated"] || !nameSet["Beta Updated"] {
		t.Errorf("unexpected product names in DB: %v", nameSet)
	}
}

// TestBatchChangeset_ContentIDURLReference_AmbiguousIDs verifies that Content-ID "10"
// is not accidentally resolved by a reference to "$1".
func TestBatchChangeset_ContentIDURLReference_AmbiguousIDs(t *testing.T) {
	handler, db := setupContentIDTestHandler(t)

	batchBoundary := "batch_ambig"
	changesetBoundary := "changeset_ambig"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 10

POST /ContentIDRefProducts HTTP/1.1
Content-Type: application/json

{"Name":"Product10","Price":10.0,"Category":"Z"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /$10 HTTP/1.1
Content-Type: application/json

{"Name":"Product10 Updated","Price":10.0,"Category":"Z"}

--%s--

--%s--
`, batchBoundary, changesetBoundary,
		changesetBoundary, changesetBoundary, changesetBoundary,
		batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want 200. Body: %s", w.Code, w.Body.String())
	}

	// The PATCH /$10 must succeed (Content-ID 10 is resolved, not Content-ID 1).
	// PATCH returns 200 or 204 depending on the Prefer header; accept both.
	respBody := w.Body.String()
	if !strings.Contains(respBody, "HTTP/1.1 200") && !strings.Contains(respBody, "HTTP/1.1 204") {
		t.Errorf("expected 200 or 204 for PATCH via $10; body:\n%s", respBody)
	}

	var products []ContentIDRefProduct
	db.Find(&products)
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].Name != "Product10 Updated" {
		t.Errorf("Name = %q, want %q", products[0].Name, "Product10 Updated")
	}
}

// TestBatchChangeset_ContentIDURLReference_UnresolvableReference verifies that a reference
// to an unknown Content-ID produces a 404 (entity set not found) and rolls back the
// entire changeset.
func TestBatchChangeset_ContentIDURLReference_UnresolvableReference(t *testing.T) {
	handler, db := setupContentIDTestHandler(t)

	batchBoundary := "batch_unres"
	changesetBoundary := "changeset_unres"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /ContentIDRefProducts HTTP/1.1
Content-Type: application/json

{"Name":"Survivor","Price":5.0,"Category":"C"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /$99 HTTP/1.1
Content-Type: application/json

{"Name":"Should not apply","Price":0.0,"Category":"C"}

--%s--

--%s--
`, batchBoundary, changesetBoundary,
		changesetBoundary, changesetBoundary, changesetBoundary,
		batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want 200. Body: %s", w.Code, w.Body.String())
	}

	respBody := w.Body.String()

	// The second request must fail because $99 is unknown.
	if !strings.Contains(respBody, "HTTP/1.1 404") {
		t.Errorf("expected 404 for unresolvable $99 reference; body:\n%s", respBody)
	}

	// Changeset must be rolled back – no products in the DB.
	var count int64
	db.Model(&ContentIDRefProduct{}).Count(&count)
	if count != 0 {
		t.Errorf("expected changeset rollback (0 products), got %d", count)
	}
}

// TestBatchChangeset_ContentIDURLReference_ResponseEchoesContentID verifies that
// Content-ID values are echoed back in every response MIME part, even for requests that
// use $<contentID> URL references.
func TestBatchChangeset_ContentIDURLReference_ResponseEchoesContentID(t *testing.T) {
	handler, _ := setupContentIDTestHandler(t)

	batchBoundary := "batch_echo"
	changesetBoundary := "changeset_echo"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: create-1

POST /ContentIDRefProducts HTTP/1.1
Content-Type: application/json

{"Name":"Echo Test","Price":3.0,"Category":"D"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: patch-2

PATCH /$create-1 HTTP/1.1
Content-Type: application/json

{"Name":"Echo Test Updated","Price":3.0,"Category":"D"}

--%s--

--%s--
`, batchBoundary, changesetBoundary,
		changesetBoundary, changesetBoundary, changesetBoundary,
		batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want 200. Body: %s", w.Code, w.Body.String())
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, "Content-ID: create-1") {
		t.Errorf("response missing Content-ID: create-1; body:\n%s", respBody)
	}
	if !strings.Contains(respBody, "Content-ID: patch-2") {
		t.Errorf("response missing Content-ID: patch-2; body:\n%s", respBody)
	}
}

// TestBatchChangeset_ContentIDURLReference_GetCreatedEntity verifies reading an entity
// that was created earlier in the same changeset via a $<contentID> GET request.
func TestBatchChangeset_ContentIDURLReference_GetCreatedEntity(t *testing.T) {
	handler, _ := setupContentIDTestHandler(t)

	batchBoundary := "batch_get"
	changesetBoundary := "changeset_get"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /ContentIDRefProducts HTTP/1.1
Content-Type: application/json

{"Name":"ReadBack","Price":7.0,"Category":"E"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

GET /$1 HTTP/1.1


--%s--

--%s--
`, batchBoundary, changesetBoundary,
		changesetBoundary, changesetBoundary, changesetBoundary,
		batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want 200. Body: %s", w.Code, w.Body.String())
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, "ReadBack") {
		t.Errorf("GET via $1 did not return the created entity; body:\n%s", respBody)
	}
}

// TestBatchChangeset_ContentIDURLReference_LocationHeaderDrivesResolution verifies that
// Content-ID resolution uses the Location header returned by the server (which contains
// the actual key assigned by the DB), not any key value from the request body.
func TestBatchChangeset_ContentIDURLReference_LocationHeaderDrivesResolution(t *testing.T) {
	handler, db := setupContentIDTestHandler(t)

	// Pre-populate so the auto-increment key will not be 1.
	db.Create(&ContentIDRefProduct{ID: 100, Name: "Seed", Price: 0, Category: ""})

	batchBoundary := "batch_loc"
	changesetBoundary := "changeset_loc"

	// POST without an explicit ID – the DB assigns the next auto-increment key.
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /ContentIDRefProducts HTTP/1.1
Content-Type: application/json

{"Name":"AutoKey","Price":9.0,"Category":"F"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /$1 HTTP/1.1
Content-Type: application/json

{"Name":"AutoKey Patched","Price":9.0,"Category":"F"}

--%s--

--%s--
`, batchBoundary, changesetBoundary,
		changesetBoundary, changesetBoundary, changesetBoundary,
		batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want 200. Body: %s", w.Code, w.Body.String())
	}

	respBody := w.Body.String()
	// PATCH returns 200 or 204 depending on the Prefer header; accept both.
	if !strings.Contains(respBody, "HTTP/1.1 200") && !strings.Contains(respBody, "HTTP/1.1 204") {
		t.Errorf("PATCH via $1 did not succeed; body:\n%s", respBody)
	}

	// Verify the patched name landed in the DB.
	var products []ContentIDRefProduct
	db.Where("name = ?", "AutoKey Patched").Find(&products)
	if len(products) != 1 {
		t.Errorf("expected exactly 1 product named 'AutoKey Patched', got %d", len(products))
	}
}

// TestBatchChangeset_ContentIDURLReference_ChainedOperations verifies three chained
// Content-ID references in one changeset: create → update via $1 → delete via $1.
func TestBatchChangeset_ContentIDURLReference_ChainedOperations(t *testing.T) {
	handler, db := setupContentIDTestHandler(t)

	batchBoundary := "batch_chain"
	changesetBoundary := "changeset_chain"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /ContentIDRefProducts HTTP/1.1
Content-Type: application/json

{"Name":"Original","Price":1.0,"Category":"C"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /$1 HTTP/1.1
Content-Type: application/json

{"Name":"Modified","Price":1.0,"Category":"C"}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 3

DELETE /$1 HTTP/1.1


--%s--

--%s--
`, batchBoundary, changesetBoundary,
		changesetBoundary, changesetBoundary, changesetBoundary, changesetBoundary,
		batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(body))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	handler.HandleBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %v, want 200. Body: %s", w.Code, w.Body.String())
	}

	respBody := w.Body.String()

	if !strings.Contains(respBody, "HTTP/1.1 201") {
		t.Errorf("expected 201 for POST; body:\n%s", respBody)
	}
	if !strings.Contains(respBody, "HTTP/1.1 204") {
		t.Errorf("expected 204 for PATCH and/or DELETE; body:\n%s", respBody)
	}

	// Entity must have been deleted at the end.
	var count int64
	db.Model(&ContentIDRefProduct{}).Count(&count)
	if count != 0 {
		t.Errorf("expected entity to be deleted, but %d record(s) remain", count)
	}
}
