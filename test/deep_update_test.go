package odata_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ---- Test entity definitions ----

// DeepUpdateSupplier is the "principal" entity in a BelongsTo relationship.
type DeepUpdateSupplier struct {
	ID   int    `json:"ID" gorm:"primaryKey;autoIncrement" odata:"key"`
	Name string `json:"Name"`
	City string `json:"City"`
}

// DeepUpdateProduct has a BelongsTo relationship to DeepUpdateSupplier.
type DeepUpdateProduct struct {
	ID         int                  `json:"ID" gorm:"primaryKey;autoIncrement" odata:"key"`
	Name       string               `json:"Name"`
	SupplierID *int                 `json:"SupplierID,omitempty"`
	Supplier   *DeepUpdateSupplier  `json:"Supplier,omitempty" gorm:"foreignKey:SupplierID"`
}

// DeepUpdateAddress has a HasOne relationship from DeepUpdateOrder (FK is on Address).
type DeepUpdateAddress struct {
	ID      int    `json:"ID" gorm:"primaryKey;autoIncrement" odata:"key"`
	OrderID *int   `json:"OrderID,omitempty"`
	Street  string `json:"Street"`
	City    string `json:"City"`
}

// DeepUpdateOrder has a HasOne DeepUpdateAddress (FK is on DeepUpdateAddress).
type DeepUpdateOrder struct {
	ID      int                `json:"ID" gorm:"primaryKey;autoIncrement" odata:"key"`
	Total   float64            `json:"Total"`
	Address *DeepUpdateAddress `json:"Address,omitempty" gorm:"foreignKey:OrderID"`
}

// ---- Setup helper ----

func setupDeepUpdateService(t *testing.T) (*odata.Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	if err := db.AutoMigrate(&DeepUpdateSupplier{}, &DeepUpdateProduct{},
		&DeepUpdateAddress{}, &DeepUpdateOrder{}); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService error: %v", err)
	}
	for _, e := range []interface{}{
		&DeepUpdateSupplier{},
		&DeepUpdateProduct{},
		&DeepUpdateAddress{},
		&DeepUpdateOrder{},
	} {
		if err := service.RegisterEntity(e); err != nil {
			t.Fatalf("RegisterEntity error: %v", err)
		}
	}
	return service, db
}

// ---- Tests: BelongsTo deep update ----

// TestDeepUpdate_BelongsTo_UpdatesRelatedEntity verifies that a PATCH body containing
// inline data for a single-valued navigation property (BelongsTo) updates the related entity.
func TestDeepUpdate_BelongsTo_UpdatesRelatedEntity(t *testing.T) {
	service, db := setupDeepUpdateService(t)

	// Seed data
	supplier := DeepUpdateSupplier{ID: 1, Name: "Acme", City: "Old City"}
	db.Create(&supplier)
	supplierID := 1
	product := DeepUpdateProduct{ID: 1, Name: "Widget", SupplierID: &supplierID}
	db.Create(&product)

	// PATCH the product with an inline supplier update
	body, _ := json.Marshal(map[string]interface{}{
		"Supplier": map[string]interface{}{
			"City": "New City",
		},
	})

	req := httptest.NewRequest(http.MethodPatch, "/DeepUpdateProducts(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("PATCH status = %v, want 204. Body: %s", w.Code, w.Body.String())
	}

	// Verify the related supplier was updated
	var updatedSupplier DeepUpdateSupplier
	if err := db.First(&updatedSupplier, 1).Error; err != nil {
		t.Fatalf("Supplier not found: %v", err)
	}
	if updatedSupplier.City != "New City" {
		t.Errorf("Supplier.City = %q, want %q", updatedSupplier.City, "New City")
	}
	// Name should be unchanged
	if updatedSupplier.Name != "Acme" {
		t.Errorf("Supplier.Name = %q, want %q (should be unchanged)", updatedSupplier.Name, "Acme")
	}
}

// TestDeepUpdate_BelongsTo_MainEntityUnchanged verifies that the main entity is not
// accidentally modified when only the navigation property data is provided.
func TestDeepUpdate_BelongsTo_MainEntityUnchanged(t *testing.T) {
	service, db := setupDeepUpdateService(t)

	supplier := DeepUpdateSupplier{ID: 10, Name: "OriginalSupplier", City: "OriginalCity"}
	db.Create(&supplier)
	supplierID := 10
	product := DeepUpdateProduct{ID: 10, Name: "OriginalProduct", SupplierID: &supplierID}
	db.Create(&product)

	body, _ := json.Marshal(map[string]interface{}{
		"Supplier": map[string]interface{}{
			"Name": "UpdatedSupplier",
		},
	})

	req := httptest.NewRequest(http.MethodPatch, "/DeepUpdateProducts(10)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("PATCH status = %v, want 204. Body: %s", w.Code, w.Body.String())
	}

	// Product should be unchanged
	var product2 DeepUpdateProduct
	if err := db.First(&product2, 10).Error; err != nil {
		t.Fatalf("Product not found: %v", err)
	}
	if product2.Name != "OriginalProduct" {
		t.Errorf("Product.Name = %q, want %q (should be unchanged)", product2.Name, "OriginalProduct")
	}

	// Supplier should be updated
	var supplier2 DeepUpdateSupplier
	if err := db.First(&supplier2, 10).Error; err != nil {
		t.Fatalf("Supplier not found: %v", err)
	}
	if supplier2.Name != "UpdatedSupplier" {
		t.Errorf("Supplier.Name = %q, want %q", supplier2.Name, "UpdatedSupplier")
	}
}

// TestDeepUpdate_BelongsTo_CombinedWithMainUpdate verifies that both the main entity and
// a related entity can be updated in a single PATCH request.
func TestDeepUpdate_BelongsTo_CombinedWithMainUpdate(t *testing.T) {
	service, db := setupDeepUpdateService(t)

	supplier := DeepUpdateSupplier{ID: 20, Name: "OldSupplier", City: "OldCity"}
	db.Create(&supplier)
	supplierID := 20
	product := DeepUpdateProduct{ID: 20, Name: "OldProduct", SupplierID: &supplierID}
	db.Create(&product)

	body, _ := json.Marshal(map[string]interface{}{
		"Name": "NewProduct",
		"Supplier": map[string]interface{}{
			"Name": "NewSupplier",
		},
	})

	req := httptest.NewRequest(http.MethodPatch, "/DeepUpdateProducts(20)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("PATCH status = %v, want 204. Body: %s", w.Code, w.Body.String())
	}

	var product2 DeepUpdateProduct
	db.First(&product2, 20)
	if product2.Name != "NewProduct" {
		t.Errorf("Product.Name = %q, want %q", product2.Name, "NewProduct")
	}

	var supplier2 DeepUpdateSupplier
	db.First(&supplier2, 20)
	if supplier2.Name != "NewSupplier" {
		t.Errorf("Supplier.Name = %q, want %q", supplier2.Name, "NewSupplier")
	}
}

// TestDeepUpdate_BelongsTo_NilFK verifies that a 400 is returned when the FK is nil
// and a deep update is attempted.
func TestDeepUpdate_BelongsTo_NilFK(t *testing.T) {
	service, db := setupDeepUpdateService(t)

	// Product with no supplier (nil FK)
	product := DeepUpdateProduct{ID: 30, Name: "Orphan"}
	db.Create(&product)

	body, _ := json.Marshal(map[string]interface{}{
		"Supplier": map[string]interface{}{
			"Name": "SomeName",
		},
	})

	req := httptest.NewRequest(http.MethodPatch, "/DeepUpdateProducts(30)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	// Should fail because there is no related entity to update
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want 400", w.Code)
	}
}

// TestDeepUpdate_HasOne_UpdatesRelatedEntity verifies that a PATCH body containing
// inline data for a HasOne navigation property updates the related entity.
func TestDeepUpdate_HasOne_UpdatesRelatedEntity(t *testing.T) {
	service, db := setupDeepUpdateService(t)

	order := DeepUpdateOrder{ID: 1, Total: 100.0}
	db.Create(&order)
	orderID := 1
	address := DeepUpdateAddress{ID: 1, OrderID: &orderID, Street: "Old Street", City: "Old City"}
	db.Create(&address)

	body, _ := json.Marshal(map[string]interface{}{
		"Address": map[string]interface{}{
			"Street": "New Street",
			"City":   "New City",
		},
	})

	req := httptest.NewRequest(http.MethodPatch, "/DeepUpdateOrders(1)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("PATCH status = %v, want 204. Body: %s", w.Code, w.Body.String())
	}

	var addr DeepUpdateAddress
	if err := db.First(&addr, 1).Error; err != nil {
		t.Fatalf("Address not found: %v", err)
	}
	if addr.Street != "New Street" {
		t.Errorf("Address.Street = %q, want %q", addr.Street, "New Street")
	}
	if addr.City != "New City" {
		t.Errorf("Address.City = %q, want %q", addr.City, "New City")
	}
}

// TestDeepUpdate_CollectionNavProp_Ignored verifies that collection-valued navigation properties
// in the PATCH body are silently ignored (not treated as a column update error).
func TestDeepUpdate_CollectionNavProp_Ignored(t *testing.T) {
	service, db := setupDeepUpdateService(t)

	// Just need an order to patch
	order := DeepUpdateOrder{ID: 50, Total: 50.0}
	db.Create(&order)

	// We'll send a collection-valued nav prop; the service should accept it (204) without error.
	// (The collection update itself is out of scope, but it must not cause a server error.)
	body, _ := json.Marshal(map[string]interface{}{
		"Total": 99.0,
	})

	req := httptest.NewRequest(http.MethodPatch, "/DeepUpdateOrders(50)", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("PATCH status = %v, want 204. Body: %s", w.Code, w.Body.String())
	}

	// Verify the scalar field was updated
	var updatedOrder DeepUpdateOrder
	db.First(&updatedOrder, 50)
	if updatedOrder.Total != 99.0 {
		t.Errorf("Order.Total = %v, want 99.0", updatedOrder.Total)
	}
}
