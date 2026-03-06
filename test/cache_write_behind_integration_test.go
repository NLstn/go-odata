package odata_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	odata "github.com/nlstn/go-odata"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Phase6Product struct {
	ID   int    `json:"id" gorm:"primaryKey" odata:"key"`
	Name string `json:"name"`
}

func TestCacheWriteBehind_QueueProgressVisibleInDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "phase6-write-behind.db")
	dsn := fmt.Sprintf("%s?_busy_timeout=5000&_journal_mode=WAL", dbPath)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&Phase6Product{}))

	service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{
		Cache: odata.CacheConfig{
			Enabled: true,
			WriteBehind: odata.WriteBehindConfig{
				Enabled:           true,
				PollInterval:      10 * time.Millisecond,
				WorkerCount:       1,
				MaxRetries:        3,
				BaseBackoff:       5 * time.Millisecond,
				MaxBackoff:        20 * time.Millisecond,
				MaxQueueSize:      32,
				InProgressTimeout: 200 * time.Millisecond,
			},
			Consistency: odata.ConsistencyConfig{
				Enabled:      true,
				InstanceID:   "phase6-wb-instance",
				PollInterval: 10 * time.Millisecond,
				BatchSize:    20,
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = service.Close() })

	require.NoError(t, service.RegisterEntity(&Phase6Product{}))

	createBody := `{"id":1,"name":"queued"}`
	createReq := httptest.NewRequest(http.MethodPost, "/Phase6Products", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	service.ServeHTTP(createRes, createReq)
	require.Equal(t, http.StatusCreated, createRes.Code, createRes.Body.String())

	require.Eventually(t, func() bool {
		var pending int64
		if err := db.Table("_odata_write_behind_queue").
			Where("status IN ?", []string{"pending", "in_progress", "failed"}).
			Count(&pending).Error; err != nil {
			return false
		}

		var completed int64
		if err := db.Table("_odata_write_behind_queue").Where("status = ?", "completed").Count(&completed).Error; err != nil {
			return false
		}

		return pending == 0 && completed >= 1
	}, 3*time.Second, 20*time.Millisecond)

	// Cache invalidation events provide operational visibility for cross-instance convergence.
	require.Eventually(t, func() bool {
		var invalidations int64
		err := db.Table("_odata_cache_invalidation_events").Count(&invalidations).Error
		return err == nil && invalidations >= 1
	}, 3*time.Second, 20*time.Millisecond)
}

func TestCacheWriteBehind_ConcurrentReadWriteAndShutdown(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "phase6-concurrency.db")
	dsn := fmt.Sprintf("%s?_busy_timeout=5000&_journal_mode=WAL", dbPath)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Phase6Product{}))

	service, err := odata.NewServiceWithConfig(db, odata.ServiceConfig{
		Cache: odata.CacheConfig{
			Enabled: true,
			WriteBehind: odata.WriteBehindConfig{
				Enabled:           true,
				PollInterval:      10 * time.Millisecond,
				WorkerCount:       1,
				MaxRetries:        3,
				BaseBackoff:       5 * time.Millisecond,
				MaxBackoff:        20 * time.Millisecond,
				InProgressTimeout: 200 * time.Millisecond,
			},
			Consistency: odata.ConsistencyConfig{
				Enabled:      true,
				InstanceID:   "phase6-concurrency-instance",
				PollInterval: 10 * time.Millisecond,
				BatchSize:    20,
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = service.Close() })

	require.NoError(t, service.RegisterEntity(&Phase6Product{}))

	createReq := httptest.NewRequest(http.MethodPost, "/Phase6Products", strings.NewReader(`{"id":7,"name":"seed"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	service.ServeHTTP(createRes, createReq)
	require.Equal(t, http.StatusCreated, createRes.Code, createRes.Body.String())

	var wg sync.WaitGroup
	errCh := make(chan error, 32)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 40; j++ {
			body := fmt.Sprintf(`{"name":"w-%d"}`, j)
			req := httptest.NewRequest(http.MethodPatch, "/Phase6Products(7)", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			service.ServeHTTP(res, req)
			if res.Code != http.StatusNoContent {
				errCh <- fmt.Errorf("patch returned %d", res.Code)
				return
			}
		}
	}()

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				req := httptest.NewRequest(http.MethodGet, "/Phase6Products(7)", nil)
				res := httptest.NewRecorder()
				service.ServeHTTP(res, req)
				if res.Code != http.StatusOK {
					errCh <- fmt.Errorf("get returned %d", res.Code)
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for callErr := range errCh {
		require.NoError(t, callErr)
	}

	// Ensure queue workers can drain quickly during service shutdown under concurrent load.
	closed := make(chan error, 1)
	go func() {
		closed <- service.Close()
	}()

	select {
	case closeErr := <-closed:
		require.NoError(t, closeErr)
	case <-time.After(2 * time.Second):
		t.Fatal("service close timed out under concurrent cache/write-behind load")
	}
}

func TestCacheConsistency_MultiInstanceConvergesAfterUpdate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "phase6-multi-instance.db")
	dsn := fmt.Sprintf("%s?_busy_timeout=5000&_journal_mode=WAL", dbPath)

	dbA, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	dbB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, dbA.AutoMigrate(&Phase6Product{}))

	serviceA, err := odata.NewServiceWithConfig(dbA, odata.ServiceConfig{
		Cache: odata.CacheConfig{
			Enabled: true,
			Consistency: odata.ConsistencyConfig{
				Enabled:             true,
				InstanceID:          "phase6-instance-a",
				PollInterval:        10 * time.Millisecond,
				BatchSize:           20,
				SkipOwnEvents:       true,
				ReconcileInterval:   50 * time.Millisecond,
				ReconcileEntitySets: []string{"Phase6Products"},
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = serviceA.Close() })

	serviceB, err := odata.NewServiceWithConfig(dbB, odata.ServiceConfig{
		Cache: odata.CacheConfig{
			Enabled: true,
			Consistency: odata.ConsistencyConfig{
				Enabled:             true,
				InstanceID:          "phase6-instance-b",
				PollInterval:        10 * time.Millisecond,
				BatchSize:           20,
				SkipOwnEvents:       true,
				ReconcileInterval:   50 * time.Millisecond,
				ReconcileEntitySets: []string{"Phase6Products"},
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = serviceB.Close() })

	require.NoError(t, serviceA.RegisterEntity(&Phase6Product{}))
	require.NoError(t, serviceB.RegisterEntity(&Phase6Product{}))

	createReq := httptest.NewRequest(http.MethodPost, "/Phase6Products", strings.NewReader(`{"id":11,"name":"v1"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	serviceA.ServeHTTP(createRes, createReq)
	require.Equal(t, http.StatusCreated, createRes.Code, createRes.Body.String())

	readRes := getEntityByID(t, serviceB, 11)
	require.Equal(t, "v1", readRes["name"])

	patchReq := httptest.NewRequest(http.MethodPatch, "/Phase6Products(11)", strings.NewReader(`{"name":"v2"}`))
	patchReq.Header.Set("Content-Type", "application/json")
	patchRes := httptest.NewRecorder()
	serviceA.ServeHTTP(patchRes, patchReq)
	require.Equal(t, http.StatusNoContent, patchRes.Code, patchRes.Body.String())

	require.Eventually(t, func() bool {
		entity := getEntityByID(t, serviceB, 11)
		name, _ := entity["name"].(string)
		return name == "v2"
	}, 3*time.Second, 20*time.Millisecond)
}

func getEntityByID(t *testing.T, service *odata.Service, id int) map[string]interface{} {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/Phase6Products(%d)", id), nil)
	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code, res.Body.String())

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &payload))
	return payload
}
