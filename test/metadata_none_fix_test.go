package odata_test

import (
"encoding/json"
"net/http"
"net/http/httptest"
"testing"

odata "github.com/nlstn/go-odata"
"gorm.io/driver/sqlite"
"gorm.io/gorm"
)

// TestODataMetadataNoneOmitsContext tests that @odata.context is omitted when odata.metadata=none
func TestODataMetadataNoneOmitsContext(t *testing.T) {
// Setup test database and service
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
if err != nil {
t.Fatalf("Failed to open database: %v", err)
}

type Product struct {
ID    int     `json:"id" gorm:"primaryKey" odata:"key"`
Name  string  `json:"name"`
Price float64 `json:"price"`
}

if err := db.AutoMigrate(&Product{}); err != nil {
t.Fatalf("Failed to migrate: %v", err)
}

// Insert test data
db.Create(&Product{ID: 1, Name: "Laptop", Price: 999.99})
db.Create(&Product{ID: 2, Name: "Mouse", Price: 29.99})

service := odata.NewService(db)
service.RegisterEntity(&Product{})

tests := []struct {
name         string
endpoint     string
acceptHeader string
formatParam  string
}{
{
name:        "Collection with Accept header",
endpoint:    "/Products",
acceptHeader: "application/json;odata.metadata=none",
},
{
name:     "Collection with $format parameter",
endpoint: "/Products",
formatParam: "application/json;odata.metadata=none",
},
{
name:        "Individual entity with Accept header",
endpoint:    "/Products(1)",
acceptHeader: "application/json;odata.metadata=none",
},
{
name:     "Individual entity with $format parameter",
endpoint: "/Products(1)",
formatParam: "application/json;odata.metadata=none",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
if tt.formatParam != "" {
q := req.URL.Query()
q.Set("$format", tt.formatParam)
req.URL.RawQuery = q.Encode()
}
if tt.acceptHeader != "" {
req.Header.Set("Accept", tt.acceptHeader)
}
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected status 200, got %d", w.Code)
}

// Verify Content-Type header
contentType := w.Header().Get("Content-Type")
if contentType != "application/json;odata.metadata=none" {
t.Errorf("Content-Type = %v, want application/json;odata.metadata=none", contentType)
}

// Parse response
var response map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
t.Fatalf("Failed to parse JSON response: %v", err)
}

// Verify @odata.context is NOT present
if _, hasContext := response["@odata.context"]; hasContext {
t.Errorf("@odata.context should NOT be present with odata.metadata=none, but got: %v", response["@odata.context"])
}

// Verify data is still present
if _, hasValue := response["value"]; !hasValue {
// Check if this is an individual entity response (no "value" wrapper)
if _, hasID := response["id"]; !hasID {
t.Error("Response missing data (neither 'value' nor direct entity data found)")
}
}
})
}
}

// TestODataMetadataContextPresence tests that @odata.context is present for minimal and full but not for none
func TestODataMetadataContextPresence(t *testing.T) {
// Setup test database and service
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
if err != nil {
t.Fatalf("Failed to open database: %v", err)
}

type Product struct {
ID    int     `json:"id" gorm:"primaryKey" odata:"key"`
Name  string  `json:"name"`
Price float64 `json:"price"`
}

if err := db.AutoMigrate(&Product{}); err != nil {
t.Fatalf("Failed to migrate: %v", err)
}

db.Create(&Product{ID: 1, Name: "Laptop", Price: 999.99})

service := odata.NewService(db)
service.RegisterEntity(&Product{})

tests := []struct {
name              string
metadataLevel     string
shouldHaveContext bool
shouldHaveType    bool
}{
{
name:              "minimal metadata should have context",
metadataLevel:     "minimal",
shouldHaveContext: true,
shouldHaveType:    false,
},
{
name:              "full metadata should have context and type",
metadataLevel:     "full",
shouldHaveContext: true,
shouldHaveType:    true,
},
{
name:              "none metadata should not have context or type",
metadataLevel:     "none",
shouldHaveContext: false,
shouldHaveType:    false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
req := httptest.NewRequest(http.MethodGet, "/Products", nil)
req.Header.Set("Accept", "application/json;odata.metadata="+tt.metadataLevel)
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected status 200, got %d", w.Code)
}

var response map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
t.Fatalf("Failed to parse JSON response: %v", err)
}

// Check @odata.context presence
_, hasContext := response["@odata.context"]
if hasContext != tt.shouldHaveContext {
if tt.shouldHaveContext {
t.Errorf("@odata.context should be present for %s metadata", tt.metadataLevel)
} else {
t.Errorf("@odata.context should NOT be present for %s metadata", tt.metadataLevel)
}
}

// Check @odata.type presence in entities (if collection has items)
if value, ok := response["value"].([]interface{}); ok && len(value) > 0 {
if entity, ok := value[0].(map[string]interface{}); ok {
_, hasType := entity["@odata.type"]
if hasType != tt.shouldHaveType {
if tt.shouldHaveType {
t.Errorf("@odata.type should be present in entities for %s metadata", tt.metadataLevel)
} else {
t.Errorf("@odata.type should NOT be present in entities for %s metadata", tt.metadataLevel)
}
}
}
}
})
}
}

// TestODataMetadataNoneWithCountAndNextLink tests that @odata.count and @odata.nextLink are preserved with metadata=none
func TestODataMetadataNoneWithCountAndNextLink(t *testing.T) {
// Setup test database and service
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
if err != nil {
t.Fatalf("Failed to open database: %v", err)
}

type Product struct {
ID    int     `json:"id" gorm:"primaryKey" odata:"key"`
Name  string  `json:"name"`
Price float64 `json:"price"`
}

if err := db.AutoMigrate(&Product{}); err != nil {
t.Fatalf("Failed to migrate: %v", err)
}

// Insert more data to test pagination
for i := 1; i <= 15; i++ {
db.Create(&Product{ID: i, Name: "Product" + string(rune(i)), Price: float64(i * 10)})
}

service := odata.NewService(db)
service.RegisterEntity(&Product{})

// Test with $count and $top
req := httptest.NewRequest(http.MethodGet, "/Products?$count=true&$top=5", nil)
req.Header.Set("Accept", "application/json;odata.metadata=none")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected status 200, got %d", w.Code)
}

var response map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
t.Fatalf("Failed to parse JSON response: %v", err)
}

// Verify @odata.context is NOT present
if _, hasContext := response["@odata.context"]; hasContext {
t.Error("@odata.context should NOT be present with odata.metadata=none")
}

// Verify @odata.count IS present (per OData spec, count is allowed with none)
if _, hasCount := response["@odata.count"]; !hasCount {
t.Error("@odata.count should be present when requested, even with odata.metadata=none")
}

// Verify @odata.nextLink IS present (per OData spec, nextLink is allowed with none)
if _, hasNextLink := response["@odata.nextLink"]; !hasNextLink {
t.Error("@odata.nextLink should be present when there are more results, even with odata.metadata=none")
}

// Verify data is present
if _, hasValue := response["value"]; !hasValue {
t.Error("Response should have 'value' property")
}
}
