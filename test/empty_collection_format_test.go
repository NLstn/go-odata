package odata

import (
"encoding/json"
"io"
"net/http"
"net/http/httptest"
"testing"

"github.com/nlstn/go-odata"
"gorm.io/driver/sqlite"
"gorm.io/gorm"
)

type EmptyTestProduct struct {
ID       uint    `json:"ID" gorm:"primaryKey" odata:"key"`
Name     string  `json:"Name"`
Price    float64 `json:"Price"`
Category string  `json:"Category"`
}

func TestODataV4EmptyCollectionFormat(t *testing.T) {
// Setup database
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
if err != nil {
t.Fatalf("Failed to open database: %v", err)
}

// Auto-migrate
if err := db.AutoMigrate(&EmptyTestProduct{}); err != nil {
t.Fatalf("Failed to migrate: %v", err)
}

// Create OData service
service := odata.NewService(db)
service.RegisterEntity(&EmptyTestProduct{})

tests := []struct {
name        string
path        string
setupData   func(*gorm.DB)
description string
}{
{
name: "Empty collection - no data",
path: "/EmptyTestProducts",
setupData: func(db *gorm.DB) {
// No data added
},
description: "Collection with no entities should return empty array []",
},
{
name: "Empty collection - filter returns no results",
path: "/EmptyTestProducts?$filter=Category%20eq%20%27NonExistent%27",
setupData: func(db *gorm.DB) {
db.Create(&EmptyTestProduct{ID: 1, Name: "Test", Price: 10.0, Category: "Electronics"})
},
description: "Filter that matches no entities should return empty array []",
},
{
name: "Empty collection - with $top",
path: "/EmptyTestProducts?$top=10",
setupData: func(db *gorm.DB) {
// No data
},
description: "Empty collection with $top should return empty array []",
},
{
name: "Empty collection - with $skip",
path: "/EmptyTestProducts?$skip=5",
setupData: func(db *gorm.DB) {
// No data
},
description: "Empty collection with $skip should return empty array []",
},
{
name: "Empty collection - with $count=true",
path: "/EmptyTestProducts?$count=true",
setupData: func(db *gorm.DB) {
// No data
},
description: "Empty collection with $count should return empty array [] and count of 0",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
// Clear the database
db.Exec("DELETE FROM empty_test_products")

// Setup test data
tt.setupData(db)

// Make request
req := httptest.NewRequest(http.MethodGet, tt.path, nil)
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

// Check status
if w.Code != http.StatusOK {
t.Errorf("Status = %v, want %v. Body: %s", w.Code, http.StatusOK, w.Body.String())
return
}

// Parse response
body, err := io.ReadAll(w.Body)
if err != nil {
t.Fatalf("Failed to read body: %v", err)
}

var response map[string]interface{}
if err := json.Unmarshal(body, &response); err != nil {
t.Fatalf("Failed to parse JSON: %v. Body: %s", err, string(body))
}

// Check that value field exists
value, exists := response["value"]
if !exists {
t.Error("Response missing 'value' field")
return
}

// CRITICAL: value must be an empty array [], not null
if value == nil {
t.Errorf("FAIL: %s - 'value' field is null, but OData v4 spec requires empty array []", tt.description)
t.Logf("Full response: %s", string(body))
return
}

// Check that it's an array
arr, ok := value.([]interface{})
if !ok {
t.Errorf("FAIL: 'value' field is not an array (type: %T)", value)
return
}

// Check that it's empty
if len(arr) != 0 {
t.Errorf("FAIL: expected empty array, got length %d", len(arr))
return
}

// If $count=true was specified, check the count field
if response["@odata.count"] != nil {
count, ok := response["@odata.count"].(float64)
if !ok {
t.Errorf("@odata.count is not a number")
} else if count != 0 {
t.Errorf("@odata.count = %v, want 0", count)
}
}

t.Logf("PASS: %s - Returns [] as expected", tt.description)
})
}
}
