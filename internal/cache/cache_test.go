package cache

import (
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type widget struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

// widgetKey mirrors the canonical single-key formatting used by the handlers
// package so the tests exercise the same Lookup path as production.
func widgetKey(v reflect.Value) string {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return strconv.FormatUint(v.FieldByName("ID").Uint(), 10)
}

func newSourceDB(t *testing.T, widgets ...widget) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&widget{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if len(widgets) > 0 {
		if err := db.Create(&widgets).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return db
}

func newWidgetCache(t *testing.T, ttl time.Duration) *EntityCache {
	t.Helper()
	c, err := New(reflect.TypeOf(widget{}), ttl, widgetKey)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestNewValidation(t *testing.T) {
	if _, err := New(nil, time.Minute, widgetKey); err == nil {
		t.Error("expected error for nil entityType")
	}
	if _, err := New(reflect.TypeOf(widget{}), 0, widgetKey); err == nil {
		t.Error("expected error for non-positive ttl")
	}
	if _, err := New(reflect.TypeOf(widget{}), time.Minute, nil); err == nil {
		t.Error("expected error for nil keyFn")
	}
}

func TestRefreshAndLookup(t *testing.T) {
	db := newSourceDB(t,
		widget{ID: 1, Name: "a"},
		widget{ID: 2, Name: "b"},
		widget{ID: 3, Name: "c"},
	)
	c := newWidgetCache(t, time.Minute)

	if c.IsValid() {
		t.Fatal("cache should not be valid before first refresh")
	}
	if err := c.Refresh(db); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if !c.IsValid() {
		t.Fatal("cache should be valid after refresh")
	}

	snap, ok := c.Current()
	if !ok {
		t.Fatal("expected a current snapshot")
	}
	if snap.Len() != 3 {
		t.Fatalf("expected 3 entities, got %d", snap.Len())
	}

	v, found := snap.Lookup("2")
	if !found {
		t.Fatal("expected to find widget 2")
	}
	if got := v.FieldByName("Name").String(); got != "b" {
		t.Fatalf("expected Name=b, got %q", got)
	}

	if _, found := snap.Lookup("99"); found {
		t.Fatal("did not expect to find widget 99")
	}
}

func TestInvalidate(t *testing.T) {
	db := newSourceDB(t, widget{ID: 1, Name: "a"})
	c := newWidgetCache(t, time.Minute)
	if err := c.Refresh(db); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	c.Invalidate()
	if c.IsValid() {
		t.Fatal("cache should be invalid after Invalidate")
	}
	if _, ok := c.Current(); ok {
		t.Fatal("Current should report no snapshot after Invalidate")
	}
}

func TestTTLExpiry(t *testing.T) {
	db := newSourceDB(t, widget{ID: 1, Name: "a"})
	c := newWidgetCache(t, 40*time.Millisecond)
	if err := c.Refresh(db); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if !c.IsValid() {
		t.Fatal("cache should be valid immediately after refresh")
	}
	time.Sleep(80 * time.Millisecond)
	if c.IsValid() {
		t.Fatal("cache should have expired")
	}
}

// TestSnapshotImmutableAfterSwap verifies that a snapshot handed to a reader is
// unaffected by a subsequent refresh that swaps in new data.
func TestSnapshotImmutableAfterSwap(t *testing.T) {
	db := newSourceDB(t, widget{ID: 1, Name: "a"})
	c := newWidgetCache(t, time.Minute)
	if err := c.Refresh(db); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	oldSnap, _ := c.Current()

	// Mutate the source and force a fresh snapshot.
	if err := db.Create(&widget{ID: 2, Name: "b"}).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}
	c.Invalidate()
	if err := c.Refresh(db); err != nil {
		t.Fatalf("refresh 2: %v", err)
	}

	if oldSnap.Len() != 1 {
		t.Fatalf("old snapshot should still have 1 entity, got %d", oldSnap.Len())
	}
	newSnap, _ := c.Current()
	if newSnap.Len() != 2 {
		t.Fatalf("new snapshot should have 2 entities, got %d", newSnap.Len())
	}
}

// TestConcurrentReadsAndRefresh exercises the lock-free read path against
// concurrent invalidation/refresh; run with -race to catch data races.
func TestConcurrentReadsAndRefresh(t *testing.T) {
	db := newSourceDB(t,
		widget{ID: 1, Name: "a"},
		widget{ID: 2, Name: "b"},
		widget{ID: 3, Name: "c"},
	)
	c := newWidgetCache(t, time.Minute)
	if err := c.Refresh(db); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Readers.
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if snap, ok := c.Current(); ok {
					_, _ = snap.Lookup("2")
					for j := 0; j < snap.Len(); j++ {
						_ = snap.At(j).FieldByName("Name").String()
					}
				}
			}
		}()
	}

	// Refresher.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			c.Invalidate()
			if err := c.Refresh(db); err != nil {
				t.Errorf("refresh: %v", err)
				return
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)
	close(stop)
	wg.Wait()
}
