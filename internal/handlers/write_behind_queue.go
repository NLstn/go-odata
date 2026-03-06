package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nlstn/go-odata/internal/trackchanges"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	writeBehindStatusPending    = "pending"
	writeBehindStatusInProgress = "in_progress"
	writeBehindStatusFailed     = "failed"
	writeBehindStatusCompleted  = "completed"
)

const (
	defaultWriteBehindPollInterval      = 200 * time.Millisecond
	defaultWriteBehindWorkerCount       = 1
	defaultWriteBehindMaxRetries        = 5
	defaultWriteBehindBaseBackoff       = 250 * time.Millisecond
	defaultWriteBehindMaxBackoff        = 10 * time.Second
	defaultWriteBehindInProgressTimeout = 30 * time.Second
	defaultWriteBehindDrainTimeout      = 10 * time.Second
)

// WriteBehindRequest describes a change that should be persisted asynchronously.
type WriteBehindRequest struct {
	EntitySet      string
	ChangeType     trackchanges.ChangeType
	KeyValues      map[string]interface{}
	Data           map[string]interface{}
	CorrelationID  string
	IdempotencyKey string
}

// WriteBehindQueue enqueues post-commit change events for asynchronous persistence.
type WriteBehindQueue interface {
	Enqueue(ctx context.Context, req WriteBehindRequest) error
}

// WriteBehindApplyFunc applies a queued write-behind request to the system of record.
type WriteBehindApplyFunc func(ctx context.Context, req WriteBehindRequest) error

// WriteBehindQueueOptions configures queue worker behavior and retry semantics.
type WriteBehindQueueOptions struct {
	PollInterval         time.Duration
	WorkerCount          int
	MaxRetries           int
	BaseBackoff          time.Duration
	MaxBackoff           time.Duration
	InProgressTimeout    time.Duration
	ShutdownDrainTimeout time.Duration
}

// DurableWriteBehindQueue persists queued operations in the database and processes
// them asynchronously with retry/backoff semantics.
type DurableWriteBehindQueue struct {
	db     *gorm.DB
	apply  WriteBehindApplyFunc
	logger *slog.Logger
	opts   WriteBehindQueueOptions

	stopCh  chan struct{}
	wg      sync.WaitGroup
	start   sync.Once
	stop    sync.Once
	opCount uint64
}

// NewDurableWriteBehindQueue creates a DB-backed write-behind queue and migrates
// the underlying queue table.
func NewDurableWriteBehindQueue(db *gorm.DB, apply WriteBehindApplyFunc, logger *slog.Logger, opts WriteBehindQueueOptions) (*DurableWriteBehindQueue, error) {
	if db == nil {
		return nil, fmt.Errorf("write-behind queue requires database handle")
	}
	if apply == nil {
		return nil, fmt.Errorf("write-behind queue requires apply function")
	}
	if logger == nil {
		logger = slog.Default()
	}

	normalized := normalizeWriteBehindOptions(opts)

	if err := db.AutoMigrate(&writeBehindJob{}); err != nil {
		return nil, fmt.Errorf("failed to migrate write-behind queue table: %w", err)
	}

	return &DurableWriteBehindQueue{
		db:     db,
		apply:  apply,
		logger: logger,
		opts:   normalized,
		stopCh: make(chan struct{}),
	}, nil
}

// Start launches worker goroutines that process queued write-behind jobs.
func (q *DurableWriteBehindQueue) Start() {
	if q == nil {
		return
	}
	q.start.Do(func() {
		for i := 0; i < q.opts.WorkerCount; i++ {
			q.wg.Add(1)
			go q.workerLoop(i)
		}
	})
}

// Close stops workers and waits for in-flight jobs to finish up to drain timeout.
func (q *DurableWriteBehindQueue) Close() {
	if q == nil {
		return
	}
	q.stop.Do(func() {
		close(q.stopCh)

		done := make(chan struct{})
		go func() {
			defer close(done)
			q.wg.Wait()
		}()

		select {
		case <-done:
		case <-time.After(q.opts.ShutdownDrainTimeout):
			q.logger.Warn("write-behind queue shutdown timed out", "timeout", q.opts.ShutdownDrainTimeout)
		}
	})
}

// Enqueue stores a write-behind request durably for asynchronous processing.
func (q *DurableWriteBehindQueue) Enqueue(_ context.Context, req WriteBehindRequest) error {
	if q == nil {
		return nil
	}

	job, err := newWriteBehindJob(req, atomic.AddUint64(&q.opCount, 1))
	if err != nil {
		return err
	}

	return q.db.Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&writeBehindJob{}).
			Where("idempotency_key = ? AND (status = ? OR status = ? OR (status = ? AND retry_at IS NOT NULL))",
				job.IdempotencyKey,
				writeBehindStatusPending,
				writeBehindStatusInProgress,
				writeBehindStatusFailed,
			).
			Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return nil
		}

		return tx.Create(job).Error
	})
}

func (q *DurableWriteBehindQueue) workerLoop(workerID int) {
	defer q.wg.Done()

	ticker := time.NewTicker(q.opts.PollInterval)
	defer ticker.Stop()

	for {
		if q.processOnce(context.Background(), workerID) {
			continue
		}

		select {
		case <-q.stopCh:
			return
		case <-ticker.C:
		}
	}
}

// processOnce processes one job if available and returns true when work was done.
func (q *DurableWriteBehindQueue) processOnce(ctx context.Context, workerID int) bool {
	job, err := q.claimDueJob()
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			q.logger.Error("failed to claim write-behind job", "error", err)
		}
		return false
	}
	if job == nil {
		return false
	}

	req, err := job.decodeRequest()
	if err != nil {
		q.failJob(*job, fmt.Errorf("decode job payload: %w", err))
		return true
	}

	if err := q.apply(ctx, req); err != nil {
		q.failJob(*job, err)
		return true
	}

	if err := q.completeJob(job.ID); err != nil {
		q.logger.Error("failed to mark write-behind job completed", "jobID", job.ID, "error", err)
		return true
	}

	q.logger.Debug("write-behind job completed",
		"jobID", job.ID,
		"worker", workerID,
		"entitySet", job.EntitySet,
		"changeType", job.ChangeType,
	)

	return true
}

func (q *DurableWriteBehindQueue) claimDueJob() (*writeBehindJob, error) {
	now := time.Now().UTC()
	leaseUntil := now.Add(q.opts.InProgressTimeout)

	var claimed *writeBehindJob
	err := q.db.Transaction(func(tx *gorm.DB) error {
		var job writeBehindJob
		if err := tx.Where(
			"(status = ?) OR (status = ? AND retry_at IS NOT NULL AND retry_at <= ?) OR (status = ? AND lease_until IS NOT NULL AND lease_until <= ?)",
			writeBehindStatusPending,
			writeBehindStatusFailed, now,
			writeBehindStatusInProgress, now,
		).Order("id asc").First(&job).Error; err != nil {
			return err
		}

		job.Attempts++
		job.Status = writeBehindStatusInProgress
		job.LeaseUntil = &leaseUntil
		job.LastAttemptAt = &now
		job.UpdatedAt = now

		updates := map[string]interface{}{
			"attempts":        job.Attempts,
			"status":          job.Status,
			"lease_until":     job.LeaseUntil,
			"last_attempt_at": job.LastAttemptAt,
			"updated_at":      job.UpdatedAt,
		}

		result := tx.Model(&writeBehindJob{}).
			Where("id = ?", job.ID).
			Where("status <> ?", writeBehindStatusCompleted).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		claimed = &job
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

func (q *DurableWriteBehindQueue) completeJob(id uint) error {
	now := time.Now().UTC()
	return q.db.Model(&writeBehindJob{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       writeBehindStatusCompleted,
			"processed_at": &now,
			"retry_at":     nil,
			"lease_until":  nil,
			"last_error":   "",
			"updated_at":   now,
		}).Error
}

func (q *DurableWriteBehindQueue) failJob(job writeBehindJob, processErr error) {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":       writeBehindStatusFailed,
		"last_error":   processErr.Error(),
		"lease_until":  nil,
		"updated_at":   now,
		"processed_at": nil,
	}

	if job.Attempts >= q.opts.MaxRetries {
		updates["retry_at"] = nil
		q.logger.Error("write-behind job moved to poison state",
			"jobID", job.ID,
			"entitySet", job.EntitySet,
			"attempts", job.Attempts,
			"error", processErr,
		)
	} else {
		retryAt := now.Add(backoffDuration(job.Attempts, q.opts.BaseBackoff, q.opts.MaxBackoff))
		updates["retry_at"] = &retryAt
		q.logger.Warn("write-behind job failed; scheduled retry",
			"jobID", job.ID,
			"entitySet", job.EntitySet,
			"attempts", job.Attempts,
			"retryAt", retryAt,
			"error", processErr,
		)
	}

	if err := q.db.Model(&writeBehindJob{}).
		Where("id = ?", job.ID).
		Updates(updates).Error; err != nil {
		q.logger.Error("failed to update write-behind job after failure", "jobID", job.ID, "error", err)
	}
}

func normalizeWriteBehindOptions(opts WriteBehindQueueOptions) WriteBehindQueueOptions {
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultWriteBehindPollInterval
	}
	if opts.WorkerCount <= 0 {
		opts.WorkerCount = defaultWriteBehindWorkerCount
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = defaultWriteBehindMaxRetries
	}
	if opts.BaseBackoff <= 0 {
		opts.BaseBackoff = defaultWriteBehindBaseBackoff
	}
	if opts.MaxBackoff <= 0 {
		opts.MaxBackoff = defaultWriteBehindMaxBackoff
	}
	if opts.InProgressTimeout <= 0 {
		opts.InProgressTimeout = defaultWriteBehindInProgressTimeout
	}
	if opts.ShutdownDrainTimeout <= 0 {
		opts.ShutdownDrainTimeout = defaultWriteBehindDrainTimeout
	}
	if opts.MaxBackoff < opts.BaseBackoff {
		opts.MaxBackoff = opts.BaseBackoff
	}
	return opts
}

func backoffDuration(attempt int, base, max time.Duration) time.Duration {
	if attempt <= 1 {
		return base
	}
	backoff := base
	for i := 1; i < attempt; i++ {
		backoff *= 2
		if backoff >= max {
			return max
		}
	}
	if backoff > max {
		return max
	}
	return backoff
}

func newWriteBehindJob(req WriteBehindRequest, sequence uint64) (*writeBehindJob, error) {
	if req.EntitySet == "" {
		return nil, fmt.Errorf("write-behind request missing entity set")
	}
	if req.ChangeType == "" {
		return nil, fmt.Errorf("write-behind request missing change type")
	}
	if len(req.KeyValues) == 0 {
		return nil, fmt.Errorf("write-behind request missing key values")
	}

	keyValuesJSON, err := json.Marshal(req.KeyValues)
	if err != nil {
		return nil, fmt.Errorf("encode key values: %w", err)
	}

	var dataJSON []byte
	if req.Data != nil {
		dataJSON, err = json.Marshal(req.Data)
		if err != nil {
			return nil, fmt.Errorf("encode entity data: %w", err)
		}
	}

	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = buildIdempotencyKey(req)
	}

	now := time.Now().UTC()
	retryAt := now
	operationID := fmt.Sprintf("wb_%d_%d", now.UnixNano(), sequence)

	return &writeBehindJob{
		OperationID:    operationID,
		EntitySet:      req.EntitySet,
		ChangeType:     req.ChangeType,
		KeyValues:      keyValuesJSON,
		Data:           dataJSON,
		Status:         writeBehindStatusPending,
		Attempts:       0,
		RetryAt:        &retryAt,
		LastError:      "",
		CorrelationID:  req.CorrelationID,
		IdempotencyKey: idempotencyKey,
	}, nil
}

func buildIdempotencyKey(req WriteBehindRequest) string {
	hasher := sha256.New()
	hasher.Write([]byte(req.EntitySet))
	hasher.Write([]byte("|"))
	hasher.Write([]byte(req.ChangeType))
	hasher.Write([]byte("|"))
	keys, err := json.Marshal(req.KeyValues)
	if err != nil {
		keys = []byte(fmt.Sprintf("%v", req.KeyValues))
	}
	hasher.Write(keys)
	hasher.Write([]byte("|"))
	if req.Data != nil {
		data, err := json.Marshal(req.Data)
		if err != nil {
			data = []byte(fmt.Sprintf("%v", req.Data))
		}
		hasher.Write(data)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// NewEntityWriteBehindApplier creates an applier that persists queued entity changes
// using handler metadata and GORM operations.
func NewEntityWriteBehindApplier(db *gorm.DB, resolver func(entitySet string) (*EntityHandler, bool)) WriteBehindApplyFunc {
	return func(ctx context.Context, req WriteBehindRequest) error {
		if db == nil {
			return fmt.Errorf("write-behind apply requires db")
		}
		if resolver == nil {
			return fmt.Errorf("write-behind apply requires handler resolver")
		}
		h, ok := resolver(req.EntitySet)
		if !ok || h == nil || h.metadata == nil {
			return fmt.Errorf("unknown entity set '%s'", req.EntitySet)
		}

		switch req.ChangeType {
		case trackchanges.ChangeTypeDeleted:
			return applyDeleteByKeys(db.WithContext(ctx), h, req.KeyValues)
		case trackchanges.ChangeTypeAdded, trackchanges.ChangeTypeUpdated:
			return applyUpsertByData(db.WithContext(ctx), h, req.Data)
		default:
			return fmt.Errorf("unsupported change type '%s'", req.ChangeType)
		}
	}
}

func applyUpsertByData(db *gorm.DB, h *EntityHandler, data map[string]interface{}) error {
	if data == nil {
		return fmt.Errorf("upsert requires entity data")
	}

	entity := reflect.New(h.metadata.EntityType).Interface()
	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data for upsert: %w", err)
	}
	if err := json.Unmarshal(encoded, entity); err != nil {
		return fmt.Errorf("unmarshal data for upsert: %w", err)
	}

	conflictColumns := make([]clause.Column, 0, len(h.metadata.KeyProperties))
	for _, keyProp := range h.metadata.KeyProperties {
		name := keyProp.ColumnName
		if name == "" {
			name = keyProp.JsonName
		}
		if name == "" {
			name = keyProp.Name
		}
		if name == "" {
			continue
		}
		conflictColumns = append(conflictColumns, clause.Column{Name: name})
	}

	if len(conflictColumns) == 0 {
		return db.Create(entity).Error
	}

	return db.Clauses(clause.OnConflict{
		Columns:   conflictColumns,
		UpdateAll: true,
	}).Create(entity).Error
}

func applyDeleteByKeys(db *gorm.DB, h *EntityHandler, keyValues map[string]interface{}) error {
	if len(keyValues) == 0 {
		return fmt.Errorf("delete requires key values")
	}

	model := reflect.New(h.metadata.EntityType).Interface()
	query := db.Model(model)
	for _, keyProp := range h.metadata.KeyProperties {
		value, ok := keyValues[keyProp.JsonName]
		if !ok {
			value, ok = keyValues[keyProp.Name]
		}
		if !ok {
			return fmt.Errorf("missing key value for '%s'", keyProp.JsonName)
		}
		column := keyProp.ColumnName
		if column == "" {
			column = keyProp.JsonName
		}
		query = query.Where(fmt.Sprintf("%s = ?", column), value)
	}

	return query.Delete(model).Error
}

type writeBehindJob struct {
	ID             uint                    `gorm:"primaryKey"`
	OperationID    string                  `gorm:"size:96;not null;uniqueIndex:idx_wb_operation_id"`
	EntitySet      string                  `gorm:"size:255;not null;index:idx_wb_due,priority:4"`
	ChangeType     trackchanges.ChangeType `gorm:"size:16;not null"`
	KeyValues      []byte                  `gorm:"not null"`
	Data           []byte
	Status         string     `gorm:"size:32;not null;index:idx_wb_due,priority:1"`
	Attempts       int        `gorm:"not null;default:0"`
	RetryAt        *time.Time `gorm:"index:idx_wb_due,priority:2"`
	LeaseUntil     *time.Time `gorm:"index:idx_wb_due,priority:3"`
	LastError      string
	CorrelationID  string `gorm:"size:128;index"`
	IdempotencyKey string `gorm:"size:64;not null;index:idx_wb_idempotency"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastAttemptAt  *time.Time
	ProcessedAt    *time.Time
}

func (writeBehindJob) TableName() string {
	return "_odata_write_behind_queue"
}

func (j *writeBehindJob) decodeRequest() (WriteBehindRequest, error) {
	var keyValues map[string]interface{}
	if err := json.Unmarshal(j.KeyValues, &keyValues); err != nil {
		return WriteBehindRequest{}, err
	}

	var data map[string]interface{}
	if len(j.Data) > 0 {
		if err := json.Unmarshal(j.Data, &data); err != nil {
			return WriteBehindRequest{}, err
		}
	}

	return WriteBehindRequest{
		EntitySet:      j.EntitySet,
		ChangeType:     j.ChangeType,
		KeyValues:      keyValues,
		Data:           data,
		CorrelationID:  j.CorrelationID,
		IdempotencyKey: j.IdempotencyKey,
	}, nil
}
