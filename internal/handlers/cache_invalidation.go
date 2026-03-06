package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultCacheInvalidationPollInterval = 500 * time.Millisecond
	defaultCacheInvalidationBatchSize    = 100
)

// CacheInvalidationEvent describes an entity-set change propagated for cache convergence.
type CacheInvalidationEvent struct {
	EntitySet      string
	ChangeType     trackchanges.ChangeType
	KeyValues      map[string]interface{}
	Data           map[string]interface{}
	CorrelationID  string
	SourceInstance string
}

// CacheInvalidationAppender persists cache convergence events.
type CacheInvalidationAppender interface {
	Append(ctx context.Context, event CacheInvalidationEvent) error
}

// StorageChangeReplayer is implemented by storage backends that can apply
// external change events to local cache state deterministically.
type StorageChangeReplayer interface {
	ReplayEntityChange(h *EntityHandler, keyValues map[string]interface{}, data map[string]interface{}, changeType trackchanges.ChangeType)
}

// DBCacheInvalidationLog persists invalidation events and checkpoints in the DB.
type DBCacheInvalidationLog struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewDBCacheInvalidationLog creates a DB-backed invalidation log store.
func NewDBCacheInvalidationLog(db *gorm.DB, logger *slog.Logger) (*DBCacheInvalidationLog, error) {
	if db == nil {
		return nil, fmt.Errorf("cache invalidation log requires database handle")
	}
	if logger == nil {
		logger = slog.Default()
	}

	if err := db.AutoMigrate(&cacheInvalidationEvent{}, &cacheInvalidationCheckpoint{}); err != nil {
		return nil, fmt.Errorf("failed to migrate cache invalidation tables: %w", err)
	}

	return &DBCacheInvalidationLog{db: db, logger: logger}, nil
}

// Append records a cache invalidation event durably.
func (l *DBCacheInvalidationLog) Append(_ context.Context, event CacheInvalidationEvent) error {
	if l == nil {
		return nil
	}
	if event.EntitySet == "" {
		return fmt.Errorf("cache invalidation event missing entity set")
	}
	if event.ChangeType == "" {
		return fmt.Errorf("cache invalidation event missing change type")
	}
	if len(event.KeyValues) == 0 {
		return fmt.Errorf("cache invalidation event missing key values")
	}

	keyValues, err := json.Marshal(event.KeyValues)
	if err != nil {
		return fmt.Errorf("encode invalidation key values: %w", err)
	}

	var data []byte
	if event.Data != nil {
		data, err = json.Marshal(event.Data)
		if err != nil {
			return fmt.Errorf("encode invalidation entity data: %w", err)
		}
	}

	row := &cacheInvalidationEvent{
		EntitySet:      event.EntitySet,
		ChangeType:     event.ChangeType,
		KeyValues:      keyValues,
		Data:           data,
		CorrelationID:  event.CorrelationID,
		SourceInstance: event.SourceInstance,
	}

	return l.db.Create(row).Error
}

type CacheInvalidationPollerOptions struct {
	InstanceID   string
	PollInterval time.Duration
	BatchSize    int
	SkipOwnEvent bool
}

// CacheInvalidationPoller tails DB invalidation events and applies them to local storage.
type CacheInvalidationPoller struct {
	db      *gorm.DB
	logger  *slog.Logger
	resolve func(entitySet string) (*EntityHandler, bool)
	opts    CacheInvalidationPollerOptions

	stopCh chan struct{}
	wg     sync.WaitGroup
	start  sync.Once
	stop   sync.Once
}

// NewCacheInvalidationPoller creates a poller over the DB invalidation event log.
func NewCacheInvalidationPoller(db *gorm.DB, logger *slog.Logger, resolve func(entitySet string) (*EntityHandler, bool), opts CacheInvalidationPollerOptions) (*CacheInvalidationPoller, error) {
	if db == nil {
		return nil, fmt.Errorf("cache invalidation poller requires database handle")
	}
	if resolve == nil {
		return nil, fmt.Errorf("cache invalidation poller requires entity handler resolver")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if opts.InstanceID == "" {
		return nil, fmt.Errorf("cache invalidation poller requires instance ID")
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultCacheInvalidationPollInterval
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaultCacheInvalidationBatchSize
	}

	if err := db.AutoMigrate(&cacheInvalidationEvent{}, &cacheInvalidationCheckpoint{}); err != nil {
		return nil, fmt.Errorf("failed to migrate cache invalidation tables: %w", err)
	}

	return &CacheInvalidationPoller{
		db:      db,
		logger:  logger,
		resolve: resolve,
		opts:    opts,
		stopCh:  make(chan struct{}),
	}, nil
}

// Start launches background polling loop.
func (p *CacheInvalidationPoller) Start() {
	if p == nil {
		return
	}
	p.start.Do(func() {
		p.wg.Add(1)
		go p.loop()
	})
}

// Close stops the poller and waits for worker shutdown.
func (p *CacheInvalidationPoller) Close() {
	if p == nil {
		return
	}
	p.stop.Do(func() {
		close(p.stopCh)
		p.wg.Wait()
	})
}

func (p *CacheInvalidationPoller) loop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.opts.PollInterval)
	defer ticker.Stop()

	for {
		if err := p.processBatch(); err != nil {
			p.logger.Error("cache invalidation poll failed", "error", err)
		}

		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
		}
	}
}

func (p *CacheInvalidationPoller) processBatch() error {
	checkpoint, err := p.loadCheckpoint()
	if err != nil {
		return err
	}

	var events []cacheInvalidationEvent
	if err := p.db.Where("id > ?", checkpoint).
		Order("id asc").
		Limit(p.opts.BatchSize).
		Find(&events).Error; err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	lastApplied := checkpoint
	for _, event := range events {
		if p.opts.SkipOwnEvent && event.SourceInstance != "" && event.SourceInstance == p.opts.InstanceID {
			lastApplied = event.ID
			continue
		}

		decoded, err := event.decode()
		if err != nil {
			return fmt.Errorf("decode invalidation event %d: %w", event.ID, err)
		}

		h, ok := p.resolve(decoded.EntitySet)
		if !ok || h == nil {
			lastApplied = event.ID
			continue
		}

		replayer, ok := h.storage.(StorageChangeReplayer)
		if !ok {
			lastApplied = event.ID
			continue
		}

		replayer.ReplayEntityChange(h, decoded.KeyValues, decoded.Data, decoded.ChangeType)
		lastApplied = event.ID
	}

	return p.saveCheckpoint(lastApplied)
}

func (p *CacheInvalidationPoller) loadCheckpoint() (uint, error) {
	var row cacheInvalidationCheckpoint
	err := p.db.First(&row, "instance_id = ?", p.opts.InstanceID).Error
	if err == nil {
		return row.LastEventID, nil
	}
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	return 0, err
}

func (p *CacheInvalidationPoller) saveCheckpoint(lastEventID uint) error {
	now := time.Now().UTC()
	return p.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "instance_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_event_id", "updated_at"}),
	}).Create(&cacheInvalidationCheckpoint{
		InstanceID:  p.opts.InstanceID,
		LastEventID: lastEventID,
		UpdatedAt:   now,
	}).Error
}

type cacheInvalidationEvent struct {
	ID             uint                    `gorm:"primaryKey"`
	EntitySet      string                  `gorm:"size:255;not null;index:idx_cache_inv_events_entity"`
	ChangeType     trackchanges.ChangeType `gorm:"size:16;not null"`
	KeyValues      []byte                  `gorm:"not null"`
	Data           []byte
	CorrelationID  string `gorm:"size:128;index"`
	SourceInstance string `gorm:"size:96;index"`
	CreatedAt      time.Time
}

func (cacheInvalidationEvent) TableName() string {
	return "_odata_cache_invalidation_events"
}

func (e *cacheInvalidationEvent) decode() (CacheInvalidationEvent, error) {
	var keyValues map[string]interface{}
	if err := json.Unmarshal(e.KeyValues, &keyValues); err != nil {
		return CacheInvalidationEvent{}, err
	}

	var data map[string]interface{}
	if len(e.Data) > 0 {
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return CacheInvalidationEvent{}, err
		}
	}

	return CacheInvalidationEvent{
		EntitySet:      e.EntitySet,
		ChangeType:     e.ChangeType,
		KeyValues:      keyValues,
		Data:           data,
		CorrelationID:  e.CorrelationID,
		SourceInstance: e.SourceInstance,
	}, nil
}

type cacheInvalidationCheckpoint struct {
	InstanceID  string `gorm:"primaryKey;size:96"`
	LastEventID uint   `gorm:"not null;default:0"`
	UpdatedAt   time.Time
}

func (cacheInvalidationCheckpoint) TableName() string {
	return "_odata_cache_invalidation_checkpoints"
}

func materializeEntityFromData(metaType reflect.Type, data map[string]interface{}) (interface{}, error) {
	if metaType == nil || data == nil {
		return nil, fmt.Errorf("entity materialization requires type and data")
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	entity := reflect.New(metaType).Interface()
	if err := json.Unmarshal(encoded, entity); err != nil {
		return nil, err
	}

	return entity, nil
}
