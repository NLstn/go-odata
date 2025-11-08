package odata_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"gorm.io/gorm"
)

func forceAssociationFailure(db *gorm.DB) {
	db.Callback().Update().Before("gorm:update").Register("force_association_failure", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "bind_test_order_items" {
			tx.AddError(errors.New("forced association failure"))
		}
	})
}

func TestPostBindingFailureRollsBackTransaction(t *testing.T) {
	service, db := setupBindTestService(t)

	if err := service.EnableChangeTracking("BindTestOrders"); err != nil {
		t.Fatalf("enable change tracking: %v", err)
	}

	forceAssociationFailure(db)

	// Seed order items to bind
	items := []BindTestOrderItem{
		{ID: 100, ProductID: 1, Quantity: 1},
		{ID: 101, ProductID: 2, Quantity: 2},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("seed order items: %v", err)
	}

	initialReq := httptest.NewRequest(http.MethodGet, "/BindTestOrders", nil)
	initialReq.Header.Set("Prefer", "odata.track-changes")
	initialRes := httptest.NewRecorder()
	service.ServeHTTP(initialRes, initialReq)
	if initialRes.Code != http.StatusOK {
		t.Fatalf("initial delta request failed: %d", initialRes.Code)
	}
	initialToken := extractDeltaToken(t, initialRes.Body.Bytes())

	body := bytes.NewBufferString(`{"OrderDate":"2024-01-01","TotalPrice":10,"Items@odata.bind":["BindTestOrderItems(100)","BindTestOrderItems(101)"]}`)
	req := httptest.NewRequest(http.MethodPost, "/BindTestOrders", body)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	service.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from binding failure, got %d", res.Code)
	}

	var orders []BindTestOrder
	if err := db.Find(&orders).Error; err != nil {
		t.Fatalf("query orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("expected no orders to be created, found %d", len(orders))
	}

	var persistedItems []BindTestOrderItem
	if err := db.Find(&persistedItems).Error; err != nil {
		t.Fatalf("query order items: %v", err)
	}
	for _, item := range persistedItems {
		if item.OrderID != nil {
			t.Fatalf("expected item %d to remain unbound", item.ID)
		}
	}

	deltaReq := httptest.NewRequest(http.MethodGet, "/BindTestOrders?$deltatoken="+url.QueryEscape(initialToken), nil)
	deltaRes := httptest.NewRecorder()
	service.ServeHTTP(deltaRes, deltaReq)
	if deltaRes.Code != http.StatusOK {
		t.Fatalf("delta request failed: %d", deltaRes.Code)
	}
	deltaBody := decodeJSON(t, deltaRes.Body.Bytes())
	changes := valueEntries(t, deltaBody)
	if len(changes) != 0 {
		t.Fatalf("expected no change tracking events, got %d", len(changes))
	}
}

func TestPatchBindingFailureRollsBackTransaction(t *testing.T) {
	service, db := setupBindTestService(t)

	if err := service.EnableChangeTracking("BindTestOrders"); err != nil {
		t.Fatalf("enable change tracking: %v", err)
	}

	forceAssociationFailure(db)

	order := BindTestOrder{ID: 200, OrderDate: "2024-01-01", TotalPrice: 5}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("seed order: %v", err)
	}

	item := BindTestOrderItem{ID: 201, ProductID: 1, Quantity: 1}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("seed order item: %v", err)
	}

	initialReq := httptest.NewRequest(http.MethodGet, "/BindTestOrders", nil)
	initialReq.Header.Set("Prefer", "odata.track-changes")
	initialRes := httptest.NewRecorder()
	service.ServeHTTP(initialRes, initialReq)
	if initialRes.Code != http.StatusOK {
		t.Fatalf("initial delta request failed: %d", initialRes.Code)
	}
	initialToken := extractDeltaToken(t, initialRes.Body.Bytes())

	patchBody := bytes.NewBufferString(`{"OrderDate":"2025-01-01","Items@odata.bind":["BindTestOrderItems(201)"]}`)
	req := httptest.NewRequest(http.MethodPatch, "/BindTestOrders(200)", patchBody)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	service.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from binding failure, got %d", res.Code)
	}

	var persistedOrder BindTestOrder
	if err := db.First(&persistedOrder, 200).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if persistedOrder.OrderDate != "2024-01-01" {
		t.Fatalf("expected order date to remain unchanged, got %s", persistedOrder.OrderDate)
	}

	var persistedItem BindTestOrderItem
	if err := db.First(&persistedItem, 201).Error; err != nil {
		t.Fatalf("reload order item: %v", err)
	}
	if persistedItem.OrderID != nil {
		t.Fatalf("expected order item to remain unbound")
	}

	deltaReq := httptest.NewRequest(http.MethodGet, "/BindTestOrders?$deltatoken="+url.QueryEscape(initialToken), nil)
	deltaRes := httptest.NewRecorder()
	service.ServeHTTP(deltaRes, deltaReq)
	if deltaRes.Code != http.StatusOK {
		t.Fatalf("delta request failed: %d", deltaRes.Code)
	}
	deltaBody := decodeJSON(t, deltaRes.Body.Bytes())
	changes := valueEntries(t, deltaBody)
	if len(changes) != 0 {
		t.Fatalf("expected no change tracking events, got %d", len(changes))
	}
}
