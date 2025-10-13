package handlers

import (
"encoding/json"
"fmt"
"net/http"
"net/http/httptest"
"testing"

"github.com/nlstn/go-odata/internal/metadata"
"gorm.io/driver/sqlite"
"gorm.io/gorm"
)

func TestDebugSelectQuery(t *testing.T) {
db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
db.AutoMigrate(&Product{})

products := []Product{
{ID: 1, Name: "Laptop", Description: "High-performance laptop", Price: 999.99, Category: "Electronics"},
{ID: 2, Name: "Mouse", Description: "Wireless mouse", Price: 29.99, Category: "Electronics"},
}
for _, product := range products {
db.Create(&product)
}

entityMeta, _ := metadata.AnalyzeEntity(Product{})
handler := NewEntityHandler(db, entityMeta)

req := httptest.NewRequest(http.MethodGet, "/Products?$select=Name", nil)
w := httptest.NewRecorder()

handler.HandleCollection(w, req)

var response map[string]interface{}
json.NewDecoder(w.Body).Decode(&response)

value := response["value"].([]interface{})
if len(value) > 0 {
item := value[0].(map[string]interface{})
fmt.Printf("Properties in response: %+v\n", item)
fmt.Printf("Number of properties: %d\n", len(item))
for k := range item {
fmt.Printf("  - %s\n", k)
}
}
}
