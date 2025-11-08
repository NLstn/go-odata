package trackchanges

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"gorm.io/gorm"
)

// ChangeType represents the type of change recorded for an entity instance.
type ChangeType string

const (
	// ChangeTypeAdded indicates that a new entity was created.
	ChangeTypeAdded ChangeType = "added"
	// ChangeTypeUpdated indicates that an existing entity was updated.
	ChangeTypeUpdated ChangeType = "updated"
	// ChangeTypeDeleted indicates that an existing entity was deleted.
	ChangeTypeDeleted ChangeType = "deleted"
)

// ChangeEvent represents a change that happened to an entity instance.
type ChangeEvent struct {
	EntitySet string
	KeyValues map[string]interface{}
	Data      map[string]interface{}
	Type      ChangeType
	Version   int64
}

type deltaToken struct {
	EntitySet string `json:"entitySet"`
	Version   int64  `json:"version"`
}

type entityHistory struct {
	Version int64
	Events  []ChangeEvent
}

// Tracker tracks entity changes and issues delta tokens that follow the OData change tracking semantics.
type Tracker struct {
	mu       sync.RWMutex
	entities map[string]*entityHistory
	db       *gorm.DB
}

// NewTracker creates a new change tracker.
func NewTracker() *Tracker {
	tracker, err := newTracker(nil)
	if err != nil {
		panic(err)
	}
	return tracker
}

// NewTrackerWithDB creates a tracker backed by persistent storage.
func NewTrackerWithDB(db *gorm.DB) (*Tracker, error) {
	if db == nil {
		return nil, fmt.Errorf("database handle is required for persistent change tracking")
	}
	return newTracker(db)
}

func newTracker(db *gorm.DB) (*Tracker, error) {
	tracker := &Tracker{entities: make(map[string]*entityHistory), db: db}
	if db == nil {
		return tracker, nil
	}

	if err := db.AutoMigrate(&changeRecord{}); err != nil {
		return nil, fmt.Errorf("failed to migrate change tracking table: %w", err)
	}

	if err := tracker.loadFromDatabase(); err != nil {
		return nil, err
	}

	return tracker, nil
}

// RegisterEntity ensures the tracker maintains history for the specified entity set.
// It is safe to call multiple times.
func (t *Tracker) RegisterEntity(entitySet string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.entities[entitySet]; !exists {
		t.entities[entitySet] = &entityHistory{}
	}
}

// RecordChange stores a change for the specified entity set and returns the new version number.
func (t *Tracker) RecordChange(entitySet string, keyValues, data map[string]interface{}, changeType ChangeType) (int64, error) {
	t.mu.Lock()

	history, exists := t.entities[entitySet]
	if !exists {
		history = &entityHistory{}
		t.entities[entitySet] = history
	}

	history.Version++

	copiedKeyValues := copyMap(keyValues)
	var copiedData map[string]interface{}
	if data != nil {
		copiedData = copyMap(data)
	}

	event := ChangeEvent{
		EntitySet: entitySet,
		KeyValues: copiedKeyValues,
		Data:      copiedData,
		Type:      changeType,
		Version:   history.Version,
	}
	history.Events = append(history.Events, event)
	version := history.Version
	t.mu.Unlock()

	if t.db != nil {
		if err := t.persistEvent(event); err != nil {
			t.mu.Lock()
			history.Version--
			if len(history.Events) > 0 {
				history.Events = history.Events[:len(history.Events)-1]
			}
			t.mu.Unlock()
			return 0, err
		}
	}

	return version, nil
}

// CurrentToken returns a delta token that represents the current state of the entity set.
func (t *Tracker) CurrentToken(entitySet string) (string, error) {
	t.mu.RLock()
	history, exists := t.entities[entitySet]
	t.mu.RUnlock()
	if !exists {
		return "", fmt.Errorf("entity set '%s' is not registered", entitySet)
	}
	return encodeToken(entitySet, history.Version)
}

// ChangesSince returns the change events that happened after the supplied delta token and a new token for subsequent requests.
func (t *Tracker) ChangesSince(token string) ([]ChangeEvent, string, error) {
	entitySet, version, err := decodeToken(token)
	if err != nil {
		return nil, "", err
	}

	t.mu.RLock()
	history, exists := t.entities[entitySet]
	if !exists {
		t.mu.RUnlock()
		return nil, "", fmt.Errorf("entity set '%s' is not registered", entitySet)
	}

	var events []ChangeEvent
	for _, event := range history.Events {
		if event.Version > version {
			// Copy to avoid exposing internal state
			events = append(events, ChangeEvent{
				EntitySet: event.EntitySet,
				KeyValues: copyMap(event.KeyValues),
				Data:      copyMap(event.Data),
				Type:      event.Type,
				Version:   event.Version,
			})
		}
	}

	newToken, encodeErr := encodeToken(entitySet, history.Version)
	t.mu.RUnlock()
	if encodeErr != nil {
		return nil, "", encodeErr
	}

	return events, newToken, nil
}

// EntitySetFromToken returns the entity set encoded in the delta token.
func (t *Tracker) EntitySetFromToken(token string) (string, error) {
	entitySet, _, err := decodeToken(token)
	return entitySet, err
}

func (t *Tracker) loadFromDatabase() error {
	var records []changeRecord
	if err := t.db.Order("entity_set asc, version asc").Find(&records).Error; err != nil {
		return fmt.Errorf("failed to load change history: %w", err)
	}

	for _, record := range records {
		keyValues, err := record.decodeKeyValues()
		if err != nil {
			return fmt.Errorf("failed to decode change record keys: %w", err)
		}

		data, err := record.decodeData()
		if err != nil {
			return fmt.Errorf("failed to decode change record data: %w", err)
		}

		history, exists := t.entities[record.EntitySet]
		if !exists {
			history = &entityHistory{}
			t.entities[record.EntitySet] = history
		}

		event := ChangeEvent{
			EntitySet: record.EntitySet,
			KeyValues: keyValues,
			Data:      data,
			Type:      record.ChangeType,
			Version:   record.Version,
		}
		history.Events = append(history.Events, event)
		if record.Version > history.Version {
			history.Version = record.Version
		}
	}

	return nil
}

func (t *Tracker) persistEvent(event ChangeEvent) error {
	record, err := newChangeRecord(event)
	if err != nil {
		return err
	}
	if err := t.db.Create(&record).Error; err != nil {
		return fmt.Errorf("failed to persist change event: %w", err)
	}
	return nil
}

func encodeToken(entitySet string, version int64) (string, error) {
	payload := deltaToken{
		EntitySet: entitySet,
		Version:   version,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode delta token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeToken(token string) (string, int64, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", 0, fmt.Errorf("invalid delta token encoding")
	}

	var payload deltaToken
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return "", 0, fmt.Errorf("invalid delta token payload")
	}

	if payload.EntitySet == "" {
		return "", 0, fmt.Errorf("delta token missing entity set")
	}

	return payload.EntitySet, payload.Version, nil
}

func copyMap(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return nil
	}

	clone := make(map[string]interface{}, len(source))
	for k, v := range source {
		clone[k] = v
	}
	return clone
}

type changeRecord struct {
	ID         uint       `gorm:"primaryKey"`
	EntitySet  string     `gorm:"size:255;not null;index:idx_entity_version,priority:1"`
	Version    int64      `gorm:"not null;index:idx_entity_version,priority:2"`
	ChangeType ChangeType `gorm:"size:16;not null"`
	KeyValues  []byte     `gorm:"not null"`
	Data       []byte
}

func (changeRecord) TableName() string {
	return "_odata_change_log"
}

func newChangeRecord(event ChangeEvent) (changeRecord, error) {
	keyJSON, err := json.Marshal(event.KeyValues)
	if err != nil {
		return changeRecord{}, fmt.Errorf("failed to encode change keys: %w", err)
	}

	var dataJSON []byte
	if event.Data != nil {
		dataJSON, err = json.Marshal(event.Data)
		if err != nil {
			return changeRecord{}, fmt.Errorf("failed to encode change data: %w", err)
		}
	}

	return changeRecord{
		EntitySet:  event.EntitySet,
		Version:    event.Version,
		ChangeType: event.Type,
		KeyValues:  keyJSON,
		Data:       dataJSON,
	}, nil
}

func (r changeRecord) decodeKeyValues() (map[string]interface{}, error) {
	var keyValues map[string]interface{}
	if err := json.Unmarshal(r.KeyValues, &keyValues); err != nil {
		return nil, err
	}
	return keyValues, nil
}

func (r changeRecord) decodeData() (map[string]interface{}, error) {
	if len(r.Data) == 0 {
		return nil, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(r.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}
