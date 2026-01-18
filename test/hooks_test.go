package odata_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestEntity is a test entity with lifecycle hooks
type TestEntity struct {
	ID                 uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name               string `json:"Name"`
	BeforeCreateCalled bool   `json:"-" gorm:"-"`
	AfterCreateCalled  bool   `json:"-" gorm:"-"`
	BeforeUpdateCalled bool   `json:"-" gorm:"-"`
	AfterUpdateCalled  bool   `json:"-" gorm:"-"`
	BeforeDeleteCalled bool   `json:"-" gorm:"-"`
	AfterDeleteCalled  bool   `json:"-" gorm:"-"`
}

// Global variables to track hook calls (since GORM creates new instances)
var (
	beforeCreateCalled bool
	afterCreateCalled  bool
	beforeUpdateCalled bool
	afterUpdateCalled  bool
	beforeDeleteCalled bool
	afterDeleteCalled  bool
	shouldFailHook     bool
)

// ODataBeforeCreate hook
func (e TestEntity) ODataBeforeCreate(ctx context.Context, r *http.Request) error {
	beforeCreateCalled = true
	if shouldFailHook {
		return fmt.Errorf("before create hook failed")
	}
	return nil
}

// ODataAfterCreate hook
func (e TestEntity) ODataAfterCreate(ctx context.Context, r *http.Request) error {
	afterCreateCalled = true
	return nil
}

// ODataBeforeUpdate hook
func (e TestEntity) ODataBeforeUpdate(ctx context.Context, r *http.Request) error {
	beforeUpdateCalled = true
	if shouldFailHook {
		return fmt.Errorf("before update hook failed")
	}
	return nil
}

// ODataAfterUpdate hook
func (e TestEntity) ODataAfterUpdate(ctx context.Context, r *http.Request) error {
	afterUpdateCalled = true
	return nil
}

// ODataBeforeDelete hook
func (e TestEntity) ODataBeforeDelete(ctx context.Context, r *http.Request) error {
	beforeDeleteCalled = true
	if shouldFailHook {
		return fmt.Errorf("before delete hook failed")
	}
	return nil
}

// ODataAfterDelete hook
func (e TestEntity) ODataAfterDelete(ctx context.Context, r *http.Request) error {
	afterDeleteCalled = true
	return nil
}

// resetHookTracking resets all hook tracking variables
func resetHookTracking() {
	beforeCreateCalled = false
	afterCreateCalled = false
	beforeUpdateCalled = false
	afterUpdateCalled = false
	beforeDeleteCalled = false
	afterDeleteCalled = false
	shouldFailHook = false
}

func TestEntityHooks_Create(t *testing.T) {
	resetHookTracking()

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(TestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Create test server
	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Create a new entity
	entity := map[string]interface{}{
		"Name": "Test Entity",
	}
	body, _ := json.Marshal(entity)

	req, _ := http.NewRequest("POST", server.URL+"/TestEntities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Verify response
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Verify hooks were called
	if !beforeCreateCalled {
		t.Error("BeforeCreate hook was not called")
	}
	if !afterCreateCalled {
		t.Error("AfterCreate hook was not called")
	}
}

func TestEntityHooks_CreateFailure(t *testing.T) {
	resetHookTracking()
	shouldFailHook = true

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(TestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Create test server
	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Try to create a new entity (should fail)
	entity := map[string]interface{}{
		"Name": "Test Entity",
	}
	body, _ := json.Marshal(entity)

	req, _ := http.NewRequest("POST", server.URL+"/TestEntities", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Verify response - should be forbidden
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}

	// Verify BeforeCreate was called
	if !beforeCreateCalled {
		t.Error("BeforeCreate hook was not called")
	}

	// Verify AfterCreate was NOT called (because creation failed)
	if afterCreateCalled {
		t.Error("AfterCreate hook should not have been called after failed creation")
	}

	// Verify entity was not created
	var count int64
	db.Model(&TestEntity{}).Count(&count)
	if count != 0 {
		t.Errorf("Entity should not have been created, found %d entities", count)
	}
}

func TestEntityHooks_Update(t *testing.T) {
	resetHookTracking()

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create a test entity directly in DB
	testEntity := TestEntity{ID: 1, Name: "Original Name"}
	db.Create(&testEntity)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(TestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Create test server
	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Update the entity
	update := map[string]interface{}{
		"Name": "Updated Name",
	}
	body, _ := json.Marshal(update)

	req, _ := http.NewRequest("PATCH", server.URL+"/TestEntities(1)", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Verify response
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	// Verify hooks were called
	if !beforeUpdateCalled {
		t.Error("BeforeUpdate hook was not called")
	}
	if !afterUpdateCalled {
		t.Error("AfterUpdate hook was not called")
	}
}

func TestEntityHooks_UpdateFailure(t *testing.T) {
	resetHookTracking()
	shouldFailHook = true

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create a test entity directly in DB
	testEntity := TestEntity{ID: 1, Name: "Original Name"}
	db.Create(&testEntity)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(TestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Create test server
	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Try to update the entity (should fail)
	update := map[string]interface{}{
		"Name": "Updated Name",
	}
	body, _ := json.Marshal(update)

	req, _ := http.NewRequest("PATCH", server.URL+"/TestEntities(1)", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Verify response - should be forbidden
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}

	// Verify BeforeUpdate was called
	if !beforeUpdateCalled {
		t.Error("BeforeUpdate hook was not called")
	}

	// Verify AfterUpdate was NOT called (because update failed)
	if afterUpdateCalled {
		t.Error("AfterUpdate hook should not have been called after failed update")
	}

	// Verify entity was not updated
	var entity TestEntity
	db.First(&entity, 1)
	if entity.Name != "Original Name" {
		t.Errorf("Entity should not have been updated, got name: %s", entity.Name)
	}
}

func TestEntityHooks_Delete(t *testing.T) {
	resetHookTracking()

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create a test entity directly in DB
	testEntity := TestEntity{ID: 1, Name: "Test Entity"}
	db.Create(&testEntity)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(TestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Create test server
	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Delete the entity
	req, _ := http.NewRequest("DELETE", server.URL+"/TestEntities(1)", nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Verify response
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	// Verify hooks were called
	if !beforeDeleteCalled {
		t.Error("BeforeDelete hook was not called")
	}
	if !afterDeleteCalled {
		t.Error("AfterDelete hook was not called")
	}
}

func TestEntityHooks_DeleteFailure(t *testing.T) {
	resetHookTracking()
	shouldFailHook = true

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create a test entity directly in DB
	testEntity := TestEntity{ID: 1, Name: "Test Entity"}
	db.Create(&testEntity)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(TestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Create test server
	server := httptest.NewServer(service.Handler())
	defer server.Close()

	// Try to delete the entity (should fail)
	req, _ := http.NewRequest("DELETE", server.URL+"/TestEntities(1)", nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Verify response - should be forbidden
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}

	// Verify BeforeDelete was called
	if !beforeDeleteCalled {
		t.Error("BeforeDelete hook was not called")
	}

	// Verify AfterDelete was NOT called (because deletion failed)
	if afterDeleteCalled {
		t.Error("AfterDelete hook should not have been called after failed deletion")
	}

	// Verify entity was not deleted
	var count int64
	db.Model(&TestEntity{}).Count(&count)
	if count != 1 {
		t.Errorf("Entity should not have been deleted, found %d entities", count)
	}
}

// TestEntityHooks_BatchCreate verifies that OData hooks are called for batch sub-requests
func TestEntityHooks_BatchCreate(t *testing.T) {
	resetHookTracking()

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(TestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Create batch request with POST (create) operation
	batchBoundary := "batch_hooks_test"
	changesetBoundary := "changeset_hooks_test"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

POST /TestEntities HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Batch Created Entity"}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Batch request failed with status %d: %s", w.Code, w.Body.String())
	}

	// Verify hooks were called
	if !beforeCreateCalled {
		t.Error("BeforeCreate hook was not called for batch sub-request")
	}
	if !afterCreateCalled {
		t.Error("AfterCreate hook was not called for batch sub-request")
	}

	// Verify entity was created
	var count int64
	db.Model(&TestEntity{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 entity in database, got %d", count)
	}
}

// TestEntityHooks_BatchUpdate verifies that update hooks are called for batch sub-requests
func TestEntityHooks_BatchUpdate(t *testing.T) {
	resetHookTracking()

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create a test entity
	testEntity := TestEntity{ID: 1, Name: "Original Name"}
	db.Create(&testEntity)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(TestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Create batch request with PATCH (update) operation
	batchBoundary := "batch_update_hooks_test"
	changesetBoundary := "changeset_update_hooks_test"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

PATCH /TestEntities(1) HTTP/1.1
Host: localhost
Content-Type: application/json

{"Name":"Updated via Batch"}

--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Batch request failed with status %d: %s", w.Code, w.Body.String())
	}

	// Verify hooks were called
	if !beforeUpdateCalled {
		t.Error("BeforeUpdate hook was not called for batch sub-request")
	}
	if !afterUpdateCalled {
		t.Error("AfterUpdate hook was not called for batch sub-request")
	}

	// Verify entity was updated
	var entity TestEntity
	db.First(&entity, 1)
	if entity.Name != "Updated via Batch" {
		t.Errorf("Entity was not updated, got name: %s", entity.Name)
	}
}

// TestEntityHooks_BatchDelete verifies that delete hooks are called for batch sub-requests
func TestEntityHooks_BatchDelete(t *testing.T) {
	resetHookTracking()

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&TestEntity{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Create a test entity
	testEntity := TestEntity{ID: 1, Name: "To Be Deleted"}
	db.Create(&testEntity)

	// Create OData service
	service, err := odata.NewService(db)
	if err != nil { t.Fatalf("NewService() error: %v", err) }
	if err := service.RegisterEntity(TestEntity{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}

	// Create batch request with DELETE operation
	batchBoundary := "batch_delete_hooks_test"
	changesetBoundary := "changeset_delete_hooks_test"
	body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary

DELETE /TestEntities(1) HTTP/1.1
Host: localhost


--%s--

--%s--
`, batchBoundary, changesetBoundary, changesetBoundary, changesetBoundary, batchBoundary)

	req := httptest.NewRequest(http.MethodPost, "/$batch", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
	w := httptest.NewRecorder()

	service.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Batch request failed with status %d: %s", w.Code, w.Body.String())
	}

	// Verify hooks were called
	if !beforeDeleteCalled {
		t.Error("BeforeDelete hook was not called for batch sub-request")
	}
	if !afterDeleteCalled {
		t.Error("AfterDelete hook was not called for batch sub-request")
	}

	// Verify entity was deleted
	var count int64
	db.Model(&TestEntity{}).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 entities in database, got %d", count)
	}
}
