package trackchanges

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
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
}

// NewTracker creates a new change tracker.
func NewTracker() *Tracker {
	return &Tracker{entities: make(map[string]*entityHistory)}
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
func (t *Tracker) RecordChange(entitySet string, keyValues, data map[string]interface{}, changeType ChangeType) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

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

	history.Events = append(history.Events, ChangeEvent{
		EntitySet: entitySet,
		KeyValues: copiedKeyValues,
		Data:      copiedData,
		Type:      changeType,
		Version:   history.Version,
	})

	return history.Version
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
