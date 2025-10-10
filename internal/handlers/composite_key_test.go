package handlers

import (
"net/http"
"net/http/httptest"
"testing"
"github.com/nlstn/go-odata/internal/metadata"
)

type CompositeKeyEntity struct {
ID1  int    `json:"id1" odata:"key"`
ID2  int    `json:"id2" odata:"key"`
Name string `json:"name"`
}

func TestMetadataHandlerJSONWithCompositeKey(t *testing.T) {
entities := make(map[string]*metadata.EntityMetadata)

entityMeta, err := metadata.AnalyzeEntity(CompositeKeyEntity{})
if err != nil {
t.Fatalf("Error analyzing entity: %v", err)
}

t.Logf("KeyProperty: %v", entityMeta.KeyProperty)
t.Logf("KeyProperties: %v", len(entityMeta.KeyProperties))

entities["CompositeKeyEntities"] = entityMeta

handler := NewMetadataHandler(entities)

req := httptest.NewRequest(http.MethodGet, "/$metadata?$format=json", nil)
w := httptest.NewRecorder()

handler.HandleMetadata(w, req)

if w.Code != http.StatusOK {
t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
}
}
