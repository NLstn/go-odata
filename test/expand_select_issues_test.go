package odata_test

import (
"encoding/json"
odata "github.com/nlstn/go-odata"
"net/http"
"net/http/httptest"
"testing"

"gorm.io/driver/sqlite"
"gorm.io/gorm"
)

// Product entity for testing
type Product struct {
ID    uint   `json:"ID" gorm:"primaryKey" odata:"key"`
Name  string `json:"Name"`
Price float64 `json:"Price"`
}

// ProductDescription entity for testing
type ProductDescription struct {
ProductID   uint    `json:"ProductID" gorm:"primaryKey" odata:"key"`
LanguageKey string  `json:"LanguageKey" gorm:"primaryKey;size:2" odata:"key"`
Description string  `json:"Description"`
Product     Product `json:"Product" gorm:"foreignKey:ProductID"`
}

func setupTestDB(t *testing.T) *gorm.DB {
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
if err != nil {
t.Fatalf("Failed to connect to database: %v", err)
}

if err := db.AutoMigrate(&Product{}, &ProductDescription{}); err != nil {
t.Fatalf("Failed to migrate database: %v", err)
}

products := []Product{
{ID: 1, Name: "Laptop", Price: 999.99},
{ID: 2, Name: "Mouse", Price: 29.99},
}

descriptions := []ProductDescription{
{ProductID: 1, LanguageKey: "EN", Description: "High-performance laptop"},
{ProductID: 1, LanguageKey: "DE", Description: "Hochleistungs-Laptop"},
{ProductID: 2, LanguageKey: "EN", Description: "Wireless mouse"},
}

if err := db.Create(&products).Error; err != nil {
t.Fatalf("Failed to seed products: %v", err)
}

if err := db.Create(&descriptions).Error; err != nil {
t.Fatalf("Failed to seed descriptions: %v", err)
}

return db
}

func TestIssue1_ExpandWithSelect(t *testing.T) {
db := setupTestDB(t)
service := odata.NewService(db)
if err := service.RegisterEntity(&Product{}); err != nil {
t.Fatalf("Failed to register Product entity: %v", err)
}
if err := service.RegisterEntity(&ProductDescription{}); err != nil {
t.Fatalf("Failed to register ProductDescription entity: %v", err)
}

// Issue 1: /ProductDescriptions?$expand=Product&$select=LanguageKey
// Should return LanguageKey and full Product entity
t.Run("Expand Product with Select LanguageKey", func(t *testing.T) {
req := httptest.NewRequest("GET", "/ProductDescriptions?$expand=Product&$select=LanguageKey", nil)
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
}

var response map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value := response["value"].([]interface{})
if len(value) == 0 {
t.Fatal("Expected at least one result")
}

item := value[0].(map[string]interface{})

// Should have LanguageKey
if _, ok := item["LanguageKey"]; !ok {
t.Error("Expected LanguageKey property")
}

// Should have full Product entity (not just Name)
product, ok := item["Product"].(map[string]interface{})
if !ok {
t.Fatalf("Expected Product to be expanded as a full entity, got: %T %v", item["Product"], item["Product"])
}

// Product should have ID, Name, and Price (all properties)
if _, ok := product["ID"]; !ok {
t.Error("Expected Product.ID property")
}
if _, ok := product["Name"]; !ok {
t.Error("Expected Product.Name property")
}
if _, ok := product["Price"]; !ok {
t.Error("Expected Product.Price property")
}
})

// Issue 1 extended: /ProductDescriptions?$expand=Product($select=Name)&$select=LanguageKey
// Should return LanguageKey and Product with only Name
t.Run("Expand Product with nested Select Name and Select LanguageKey", func(t *testing.T) {
req := httptest.NewRequest("GET", "/ProductDescriptions?$expand=Product($select=Name)&$select=LanguageKey", nil)
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
}

var response map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value := response["value"].([]interface{})
if len(value) == 0 {
t.Fatal("Expected at least one result")
}

item := value[0].(map[string]interface{})

// Should have LanguageKey
if _, ok := item["LanguageKey"]; !ok {
t.Error("Expected LanguageKey property")
}

// Should have Product with only Name (and key ID)
product, ok := item["Product"].(map[string]interface{})
if !ok {
t.Fatalf("Expected Product to be expanded, got: %T", item["Product"])
}

// Product should have Name
if _, ok := product["Name"]; !ok {
t.Error("Expected Product.Name property")
}

// Product should NOT have Price (not selected)
if _, ok := product["Price"]; ok {
t.Error("Did not expect Product.Price property (not selected)")
}
})
}

func TestIssue2_SelectExpandedEntityProperty(t *testing.T) {
db := setupTestDB(t)
service := odata.NewService(db)
if err := service.RegisterEntity(&Product{}); err != nil {
t.Fatalf("Failed to register Product entity: %v", err)
}
if err := service.RegisterEntity(&ProductDescription{}); err != nil {
t.Fatalf("Failed to register ProductDescription entity: %v", err)
}

// Issue 2: /ProductDescriptions?$expand=Product&$select=LanguageKey,Product/Name
// Should expand Product and select only LanguageKey from ProductDescription and Name from Product
t.Run("Select property from expanded entity", func(t *testing.T) {
req := httptest.NewRequest("GET", "/ProductDescriptions?$expand=Product&$select=LanguageKey,Product/Name", nil)
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
}

var response map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value := response["value"].([]interface{})
if len(value) == 0 {
t.Fatal("Expected at least one result")
}

item := value[0].(map[string]interface{})

// Should have LanguageKey
if _, ok := item["LanguageKey"]; !ok {
t.Error("Expected LanguageKey property")
}

// Should NOT have Description (not selected)
if _, ok := item["Description"]; ok {
t.Error("Did not expect Description property (not selected)")
}

// Should have Product with only Name (and key ID)
product, ok := item["Product"].(map[string]interface{})
if !ok {
t.Fatalf("Expected Product to be expanded, got: %T", item["Product"])
}

// Product should have Name
if _, ok := product["Name"]; !ok {
t.Error("Expected Product.Name property")
}

// Product should NOT have Price (not selected)
if _, ok := product["Price"]; ok {
t.Error("Did not expect Product.Price property (not selected)")
}
})
}
