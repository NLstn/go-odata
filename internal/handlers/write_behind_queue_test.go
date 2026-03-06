package handlers

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/trackchanges"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type writeBehindTestEntity struct {
	ID   int    `json:"id" gorm:"primaryKey" odata:"key"`
	Name string `json:"name"`
}

type captureWriteBehindQueue struct {
	requests []WriteBehindRequest
}

func (c *captureWriteBehindQueue) Enqueue(_ context.Context, req WriteBehindRequest) error {
	c.requests = append(c.requests, req)
	return nil
}

func TestFinalizeChangeEventsEnqueuesWriteBehind(t *testing.T) {
	h, _ := newWriteBehindTestHandler(t)
	capture := &captureWriteBehindQueue{}
	h.SetWriteBehindQueue(capture)

	h.finalizeChangeEvents(context.Background(), []changeEvent{{
		entity:     &writeBehindTestEntity{ID: 1, Name: "queued"},
		changeType: trackchanges.ChangeTypeAdded,
	}})

	require.Len(t, capture.requests, 1)
	req := capture.requests[0]
	require.Equal(t, h.metadata.EntitySetName, req.EntitySet)
	require.Equal(t, trackchanges.ChangeTypeAdded, req.ChangeType)
	require.Equal(t, 1, req.KeyValues["id"])
	require.Equal(t, "queued", req.Data["name"])
}

func TestDurableWriteBehindQueueProcessesEntityChanges(t *testing.T) {
	h, db := newWriteBehindTestHandler(t)

	queue, err := NewDurableWriteBehindQueue(
		db,
		NewEntityWriteBehindApplier(db, func(entitySet string) (*EntityHandler, bool) {
			if entitySet == h.metadata.EntitySetName {
				return h, true
			}
			return nil, false
		}),
		slog.Default(),
		WriteBehindQueueOptions{
			PollInterval:      10 * time.Millisecond,
			WorkerCount:       1,
			MaxRetries:        3,
			BaseBackoff:       10 * time.Millisecond,
			MaxBackoff:        20 * time.Millisecond,
			InProgressTimeout: 100 * time.Millisecond,
		},
	)
	require.NoError(t, err)
	queue.Start()
	defer queue.Close()

	err = queue.Enqueue(context.Background(), WriteBehindRequest{
		EntitySet:  h.metadata.EntitySetName,
		ChangeType: trackchanges.ChangeTypeAdded,
		KeyValues:  map[string]interface{}{"id": 1},
		Data:       map[string]interface{}{"id": 1, "name": "first"},
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		var found writeBehindTestEntity
		err := db.First(&found, "id = ?", 1).Error
		return err == nil && found.Name == "first"
	}, 2*time.Second, 20*time.Millisecond)

	err = queue.Enqueue(context.Background(), WriteBehindRequest{
		EntitySet:  h.metadata.EntitySetName,
		ChangeType: trackchanges.ChangeTypeUpdated,
		KeyValues:  map[string]interface{}{"id": 1},
		Data:       map[string]interface{}{"id": 1, "name": "updated"},
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		var found writeBehindTestEntity
		err := db.First(&found, "id = ?", 1).Error
		return err == nil && found.Name == "updated"
	}, 2*time.Second, 20*time.Millisecond)

	err = queue.Enqueue(context.Background(), WriteBehindRequest{
		EntitySet:  h.metadata.EntitySetName,
		ChangeType: trackchanges.ChangeTypeDeleted,
		KeyValues:  map[string]interface{}{"id": 1},
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		var count int64
		err := db.Model(&writeBehindTestEntity{}).Where("id = ?", 1).Count(&count).Error
		return err == nil && count == 0
	}, 2*time.Second, 20*time.Millisecond)
}

func TestDurableWriteBehindQueueRetriesToPoisonState(t *testing.T) {
	h, db := newWriteBehindTestHandler(t)

	queue, err := NewDurableWriteBehindQueue(
		db,
		func(context.Context, WriteBehindRequest) error {
			return errors.New("boom")
		},
		slog.Default(),
		WriteBehindQueueOptions{
			PollInterval:      10 * time.Millisecond,
			WorkerCount:       1,
			MaxRetries:        2,
			BaseBackoff:       10 * time.Millisecond,
			MaxBackoff:        20 * time.Millisecond,
			InProgressTimeout: 100 * time.Millisecond,
		},
	)
	require.NoError(t, err)
	queue.Start()
	defer queue.Close()

	err = queue.Enqueue(context.Background(), WriteBehindRequest{
		EntitySet:  h.metadata.EntitySetName,
		ChangeType: trackchanges.ChangeTypeAdded,
		KeyValues:  map[string]interface{}{"id": 9},
		Data:       map[string]interface{}{"id": 9, "name": "never"},
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		var job writeBehindJob
		err := db.Where("entity_set = ?", h.metadata.EntitySetName).First(&job).Error
		if err != nil {
			return false
		}
		return job.Status == writeBehindStatusFailed && job.RetryAt == nil && job.Attempts == 2
	}, 3*time.Second, 20*time.Millisecond)
}

func newWriteBehindTestHandler(t *testing.T) (*EntityHandler, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&writeBehindTestEntity{}))

	meta, err := metadata.AnalyzeEntity(&writeBehindTestEntity{})
	require.NoError(t, err)

	h := NewEntityHandler(db, meta, slog.Default())
	return h, db
}
