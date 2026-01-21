package odata_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type TransactionContextEntity struct {
	ID   uint   `json:"ID" gorm:"primaryKey" odata:"key"`
	Name string `json:"Name"`
}

type TransactionAudit struct {
	ID      uint `gorm:"primaryKey"`
	Counter int
}

var (
	hookTransactionObserved  bool
	hookTransactionAttempted bool
)

func (e *TransactionContextEntity) ODataBeforeUpdate(ctx context.Context, _ *http.Request) error {
	hookTransactionAttempted = true
	tx, ok := odata.TransactionFromContext(ctx)
	if !ok {
		return fmt.Errorf("transaction not available in context")
	}
	hookTransactionObserved = true
	if err := tx.Model(&TransactionAudit{}).
		Where("id = ?", e.ID).
		Update("counter", gorm.Expr("counter + 1")).Error; err != nil {
		return err
	}
	return fmt.Errorf("abort update for test")
}

func resetTransactionHookFlags() {
	hookTransactionObserved = false
	hookTransactionAttempted = false
}

func TestHookTransactionRollsBackOnAbort(t *testing.T) {
	resetTransactionHookFlags()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	if err := db.AutoMigrate(&TransactionContextEntity{}, &TransactionAudit{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	entity := TransactionContextEntity{ID: 1, Name: "original"}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("seed entity: %v", err)
	}

	audit := TransactionAudit{ID: 1, Counter: 0}
	if err := db.Create(&audit).Error; err != nil {
		t.Fatalf("seed audit: %v", err)
	}

	var seeded TransactionContextEntity
	if err := db.First(&seeded, entity.ID).Error; err != nil {
		t.Fatalf("entity not persisted: %v", err)
	}

	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(TransactionContextEntity{}); err != nil {
		t.Fatalf("register entity: %v", err)
	}

	body := bytes.NewBufferString(`{"Name":"updated"}`)
	req := httptest.NewRequest(http.MethodPatch, "/TransactionContextEntities(1)", body)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)
	resp := res.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 from aborted hook, got %d", resp.StatusCode)
	}

	if !hookTransactionAttempted {
		t.Fatalf("hook did not execute")
	}
	if !hookTransactionObserved {
		t.Fatalf("hook did not observe transaction")
	}

	var persisted TransactionContextEntity
	if err := db.First(&persisted, entity.ID).Error; err != nil {
		t.Fatalf("reload entity: %v", err)
	}
	if persisted.Name != "original" {
		t.Fatalf("entity update was committed: got %q", persisted.Name)
	}

	var persistedAudit TransactionAudit
	if err := db.First(&persistedAudit, audit.ID).Error; err != nil {
		t.Fatalf("reload audit: %v", err)
	}
	if persistedAudit.Counter != 0 {
		t.Fatalf("audit update was committed despite rollback: counter=%d", persistedAudit.Counter)
	}
}
