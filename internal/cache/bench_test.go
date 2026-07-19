package cache

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// These benchmarks compare the read path of the OLD entity cache backend (an
// in-memory SQLite store opened with mode=memory&cache=shared, exactly as the
// previous implementation did) against the NEW lock-free snapshot cache, under
// concurrent load. They isolate the cache layer from HTTP/serialisation cost so
// the difference reflects the storage backend alone.
//
// Run:  go test -run '^$' -bench . -benchmem ./internal/cache/
//
// The key result is throughput scaling with b.RunParallel: shared-cache SQLite
// serialises reads behind a process-wide lock, so it does not scale with cores,
// while the snapshot's atomic-pointer reads do.

const benchN = 2000

// --- OLD backend: in-memory SQLite with shared cache + connection pool --------

func newSharedCacheSQLite(b *testing.B) *gorm.DB {
	b.Helper()
	// Same DSN shape as the previous cache implementation.
	dsn := fmt.Sprintf("file:bench_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		b.Fatalf("sqlDB: %v", err)
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(25) // matches the previous implementation's pool size
	if err := db.AutoMigrate(&widget{}); err != nil {
		b.Fatalf("migrate: %v", err)
	}
	rows := make([]widget, benchN)
	for i := range rows {
		rows[i] = widget{ID: uint(i + 1), Name: "name-" + strconv.Itoa(i%64)}
	}
	if err := db.CreateInBatches(rows, 100).Error; err != nil {
		b.Fatalf("seed: %v", err)
	}
	return db
}

func newLoadedSnapshot(b *testing.B) *Snapshot {
	b.Helper()
	rows := make([]widget, benchN)
	for i := range rows {
		rows[i] = widget{ID: uint(i + 1), Name: "name-" + strconv.Itoa(i%64)}
	}
	entities := reflect.ValueOf(rows)
	byKey := make(map[string]int, entities.Len())
	for i := 0; i < entities.Len(); i++ {
		byKey[widgetKey(entities.Index(i))] = i
	}
	return &Snapshot{entities: entities, byKey: byKey, expiresAt: time.Now().Add(time.Hour)}
}

// --- Key lookup: SELECT ... WHERE id = ?  vs  O(1) map lookup -----------------

func BenchmarkKeyLookup_OldSQLiteSharedCache(b *testing.B) {
	db := newSharedCacheSQLite(b)
	var ctr uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := uint(atomic.AddUint64(&ctr, 1)%benchN) + 1
			var w widget
			if err := db.Where("id = ?", id).Take(&w).Error; err != nil {
				b.Error(err)
				return
			}
		}
	})
}

func BenchmarkKeyLookup_NewSnapshot(b *testing.B) {
	snap := newLoadedSnapshot(b)
	var ctr uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := atomic.AddUint64(&ctr, 1) % benchN
			key := strconv.FormatUint(id+1, 10)
			v, ok := snap.Lookup(key)
			if !ok {
				b.Errorf("missing key %s", key)
				return
			}
			_ = v.FieldByName("Name").String()
		}
	})
}

// --- Filtered collection read: SELECT ... WHERE name = ?  vs  in-memory scan --

func BenchmarkFilterScan_OldSQLiteSharedCache(b *testing.B) {
	db := newSharedCacheSQLite(b)
	var ctr uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			name := "name-" + strconv.FormatUint(atomic.AddUint64(&ctr, 1)%64, 10)
			var out []widget
			if err := db.Where("name = ?", name).Find(&out).Error; err != nil {
				b.Error(err)
				return
			}
		}
	})
}

func BenchmarkFilterScan_NewSnapshot(b *testing.B) {
	snap := newLoadedSnapshot(b)
	var ctr uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			name := "name-" + strconv.FormatUint(atomic.AddUint64(&ctr, 1)%64, 10)
			out := make([]widget, 0, benchN/64)
			for i := 0; i < snap.Len(); i++ {
				e := snap.At(i)
				if e.FieldByName("Name").String() == name {
					var w widget
					w.ID = uint(e.FieldByName("ID").Uint())
					w.Name = e.FieldByName("Name").String()
					out = append(out, w)
				}
			}
			// Consume the result so the compiler cannot elide the scan.
			runtime.KeepAlive(out)
		}
	})
}
