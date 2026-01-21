package trackchanges

import (
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTrackerRecordAndRetrieve(t *testing.T) {
	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}
	tracker.RegisterEntity("Products")

	token, err := tracker.CurrentToken("Products")
	if err != nil {
		t.Fatalf("CurrentToken failed: %v", err)
	}

	entitySet, err := tracker.EntitySetFromToken(token)
	if err != nil {
		t.Fatalf("EntitySetFromToken failed: %v", err)
	}
	if entitySet != "Products" {
		t.Fatalf("expected entity set Products, got %s", entitySet)
	}

	if _, err := tracker.RecordChange("Products", map[string]interface{}{"ID": 1}, map[string]interface{}{"ID": 1, "Name": "Laptop"}, ChangeTypeAdded); err != nil {
		t.Fatalf("RecordChange failed: %v", err)
	}
	if _, err := tracker.RecordChange("Products", map[string]interface{}{"ID": 1}, map[string]interface{}{"ID": 1, "Name": "Laptop Updated"}, ChangeTypeUpdated); err != nil {
		t.Fatalf("RecordChange failed: %v", err)
	}

	events, newToken, err := tracker.ChangesSince(token)
	if err != nil {
		t.Fatalf("ChangesSince failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if newToken == token {
		t.Fatal("expected new token to differ from original token")
	}
	if events[0].Type != ChangeTypeAdded {
		t.Fatalf("expected first event to be added, got %s", events[0].Type)
	}
	if events[1].Type != ChangeTypeUpdated {
		t.Fatalf("expected second event to be updated, got %s", events[1].Type)
	}
}

func TestTrackerInvalidToken(t *testing.T) {
	tracker, err := NewTracker()
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}
	tracker.RegisterEntity("Products")

	if _, _, err := tracker.ChangesSince("not-a-token"); err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestPersistentTrackerLoadsHistory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "tracker.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	tracker, err := NewTrackerWithDB(db)
	if err != nil {
		t.Fatalf("create persistent tracker: %v", err)
	}

	tracker.RegisterEntity("Products")
	token, err := tracker.CurrentToken("Products")
	if err != nil {
		t.Fatalf("current token: %v", err)
	}

	if _, err := tracker.RecordChange("Products", map[string]interface{}{"ID": 1}, map[string]interface{}{"ID": 1}, ChangeTypeAdded); err != nil {
		t.Fatalf("record change: %v", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}

	dbReloaded, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}

	trackerReloaded, err := NewTrackerWithDB(dbReloaded)
	if err != nil {
		t.Fatalf("reload tracker: %v", err)
	}

	trackerReloaded.RegisterEntity("Products")
	events, newToken, err := trackerReloaded.ChangesSince(token)
	if err != nil {
		t.Fatalf("changes since: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event after reload, got %d", len(events))
	}
	if newToken == token {
		t.Fatal("expected new token to differ after reload")
	}
	if events[0].Version != 1 {
		t.Fatalf("expected version 1, got %d", events[0].Version)
	}
}
