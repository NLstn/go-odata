package handlers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandleGetCollectionHonorsContextCancellation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.AutoMigrate(&Product{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	db.Callback().Query().Before("gorm:query").Register("test_wait_for_cancel", func(db *gorm.DB) {
		if db.Statement == nil || db.Statement.Context == nil {
			return
		}
		<-db.Statement.Context.Done()
		db.Error = db.Statement.Context.Err()
	})

	entityMeta, err := metadata.AnalyzeEntity(Product{})
	if err != nil {
		t.Fatalf("failed to analyze entity metadata: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewEntityHandler(db, entityMeta, logger)

	req := httptest.NewRequest(http.MethodGet, "/Products", nil)
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		handler.handleGetCollection(w, req)
		close(done)
	}()

	time.AfterFunc(10*time.Millisecond, cancel)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not return after context cancellation")
	}

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
