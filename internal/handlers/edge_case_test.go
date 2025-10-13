package handlers

import (
"encoding/json"
"net/http"
"net/http/httptest"
"testing"

"github.com/nlstn/go-odata/internal/metadata"
"gorm.io/driver/sqlite"
"gorm.io/gorm"
)

func TestSelectWithOnlyKey(t *testing.T) {
db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
db.AutoMigrate(&Product{})

products := []Product{
{ID: 1, Name: "Laptop", Description: "High-performance laptop", Price: 999.99, Category: "Electronics"},
}
for _, product := range products {
db.Create(&product)
}

entityMeta, _ := metadata.AnalyzeEntity(Product{})
handler := NewEntityHandler(db, entityMeta)

req := httptest.NewRequest(http.MethodGet, "/Products?$select=ID", nil)
w := httptest.NewRecorder()

handler.HandleCollection(w, req)

var response map[string]interface{}
json.NewDecoder(w.Body).Decode(&response)

value := response["value"].([]interface{})
item := value[0].(map[string]interface{})

t.Logf("Properties with select=ID: %+v", item)
t.Logf("Number of properties: %d", len(item))

// Should only have ID
if len(item) != 1 {
t.Errorf("Expected 1 property, got %d", len(item))
}
}
