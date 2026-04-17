package odata_test

import (
"encoding/json"
"net/http"
"net/http/httptest"
"net/url"
"testing"
)

// TestOptionalDollarPrefix_V401_FilterWithoutDollar tests that $filter can be specified without $
// in OData v4.01.
func TestOptionalDollarPrefix_V401_FilterWithoutDollar(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})
db.Create(&TestProduct{ID: 2, Name: "Mouse", Price: 29.99})

req := httptest.NewRequest(http.MethodGet, "/TestProducts?filter="+url.QueryEscape("Price gt 100"), nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value, ok := result["value"].([]interface{})
if !ok {
t.Fatal("Expected 'value' array in response")
}

// Only Laptop (Price=999.99) should match filter Price gt 100
if len(value) != 1 {
t.Errorf("Expected 1 product (Laptop), got %d", len(value))
}
}

// TestOptionalDollarPrefix_V401_SelectWithoutDollar tests that $select can be specified without $
// in OData v4.01.
func TestOptionalDollarPrefix_V401_SelectWithoutDollar(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})

req := httptest.NewRequest(http.MethodGet, "/TestProducts?select=name", nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value, ok := result["value"].([]interface{})
if !ok {
t.Fatal("Expected 'value' array in response")
}
if len(value) != 1 {
t.Fatalf("Expected 1 product, got %d", len(value))
}

product, ok := value[0].(map[string]interface{})
if !ok {
t.Fatal("Expected product to be a JSON object")
}
// price should not be in the response (was not selected)
if _, hasPrice := product["price"]; hasPrice {
t.Error("Expected 'price' to be excluded by $select, but it was present")
}
// name should be present
if _, hasName := product["name"]; !hasName {
t.Error("Expected 'name' to be present after $select=name")
}
}

// TestOptionalDollarPrefix_V401_TopWithoutDollar tests that $top can be specified without $
// in OData v4.01.
func TestOptionalDollarPrefix_V401_TopWithoutDollar(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "A", Price: 10})
db.Create(&TestProduct{ID: 2, Name: "B", Price: 20})
db.Create(&TestProduct{ID: 3, Name: "C", Price: 30})

req := httptest.NewRequest(http.MethodGet, "/TestProducts?top=2", nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value, ok := result["value"].([]interface{})
if !ok {
t.Fatal("Expected 'value' array in response")
}
if len(value) != 2 {
t.Errorf("Expected 2 products (top=2), got %d", len(value))
}
}

// TestOptionalDollarPrefix_V401_SkipWithoutDollar tests that $skip can be specified without $
// in OData v4.01.
func TestOptionalDollarPrefix_V401_SkipWithoutDollar(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "A", Price: 10})
db.Create(&TestProduct{ID: 2, Name: "B", Price: 20})
db.Create(&TestProduct{ID: 3, Name: "C", Price: 30})

req := httptest.NewRequest(http.MethodGet, "/TestProducts?skip=2&$orderby="+url.QueryEscape("id asc"), nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value, ok := result["value"].([]interface{})
if !ok {
t.Fatal("Expected 'value' array in response")
}
if len(value) != 1 {
t.Errorf("Expected 1 product (skip=2 of 3), got %d", len(value))
}
}

// TestOptionalDollarPrefix_V401_OrderByWithoutDollar tests that $orderby can be specified without $
// in OData v4.01.
func TestOptionalDollarPrefix_V401_OrderByWithoutDollar(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Alpha", Price: 30})
db.Create(&TestProduct{ID: 2, Name: "Beta", Price: 10})
db.Create(&TestProduct{ID: 3, Name: "Gamma", Price: 20})

req := httptest.NewRequest(http.MethodGet, "/TestProducts?orderby="+url.QueryEscape("price asc"), nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value, ok := result["value"].([]interface{})
if !ok {
t.Fatal("Expected 'value' array in response")
}
if len(value) != 3 {
t.Fatalf("Expected 3 products, got %d", len(value))
}

// First item should be Beta (Price=10, lowest)
first, ok := value[0].(map[string]interface{})
if !ok {
t.Fatal("Expected product to be a JSON object")
}
if first["name"] != "Beta" {
t.Errorf("Expected first product to be 'Beta' (lowest price), got %v", first["name"])
}
}

// TestOptionalDollarPrefix_V401_CountWithoutDollar tests that $count can be specified without $
// in OData v4.01.
func TestOptionalDollarPrefix_V401_CountWithoutDollar(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "A", Price: 10})
db.Create(&TestProduct{ID: 2, Name: "B", Price: 20})

req := httptest.NewRequest(http.MethodGet, "/TestProducts?count=true", nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

if _, hasCount := result["@odata.count"]; !hasCount {
t.Error("Expected '@odata.count' in response when count=true")
}
}

// TestOptionalDollarPrefix_V401_MixedCaseDollarPrefix tests that mixed-case $-prefixed options
// work in OData v4.01.
func TestOptionalDollarPrefix_V401_MixedCaseDollarPrefix(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})
db.Create(&TestProduct{ID: 2, Name: "Mouse", Price: 29.99})

req := httptest.NewRequest(http.MethodGet, "/TestProducts?$TOP=1", nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value, ok := result["value"].([]interface{})
if !ok {
t.Fatal("Expected 'value' array in response")
}
if len(value) != 1 {
t.Errorf("Expected 1 product ($TOP=1), got %d", len(value))
}
}

// TestOptionalDollarPrefix_V401_CombinedNoDollarOptions tests combining multiple options
// without $ prefix in OData v4.01.
func TestOptionalDollarPrefix_V401_CombinedNoDollarOptions(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})
db.Create(&TestProduct{ID: 2, Name: "Mouse", Price: 29.99})
db.Create(&TestProduct{ID: 3, Name: "Keyboard", Price: 79.99})

// Combine filter + select + top without $ prefixes
req := httptest.NewRequest(http.MethodGet,
"/TestProducts?filter="+url.QueryEscape("Price gt 50")+"&select=name&top=5", nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value, ok := result["value"].([]interface{})
if !ok {
t.Fatal("Expected 'value' array in response")
}
// Laptop (999.99) and Keyboard (79.99) have Price > 50
if len(value) != 2 {
t.Errorf("Expected 2 products (filter Price gt 50), got %d", len(value))
}
// Each product should only have 'name' field (from select=name)
for _, item := range value {
product, ok := item.(map[string]interface{})
if !ok {
t.Fatal("Expected product to be a JSON object")
}
if _, hasPrice := product["price"]; hasPrice {
t.Error("Expected 'price' to be excluded by select=name")
}
}
}

// TestOptionalDollarPrefix_V401_DuplicateAcrossDollarForms tests that providing the same option
// with and without $ is rejected as a duplicate in OData v4.01.
func TestOptionalDollarPrefix_V401_DuplicateAcrossDollarForms(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})

// Provide both $filter and filter (same option in both forms)
req := httptest.NewRequest(http.MethodGet,
"/TestProducts?$filter="+url.QueryEscape("Price gt 100")+"&filter="+url.QueryEscape("Price gt 50"), nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusBadRequest {
t.Errorf("Expected 400 Bad Request for duplicate filter option across $ and non-$ forms, got %d: %s", w.Code, w.Body.String())
}
}

// TestOptionalDollarPrefix_V40_NoDollarTreatedAsCustomParam tests that query options without $
// are silently ignored (treated as custom query parameters) in OData v4.0.
func TestOptionalDollarPrefix_V40_NoDollarTreatedAsCustomParam(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})
db.Create(&TestProduct{ID: 2, Name: "Mouse", Price: 29.99})

// filter=Price gt 100 without $ in v4.0 should be ignored (treated as custom param)
// so ALL products are returned, not just those matching the filter
req := httptest.NewRequest(http.MethodGet, "/TestProducts?filter="+url.QueryEscape("Price gt 100"), nil)
req.Header.Set("OData-MaxVersion", "4.0")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value, ok := result["value"].([]interface{})
if !ok {
t.Fatal("Expected 'value' array in response")
}
// Both products should be returned (filter is ignored in v4.0 without $)
if len(value) != 2 {
t.Errorf("Expected 2 products (filter without $ should be ignored in v4.0), got %d", len(value))
}
}

// TestOptionalDollarPrefix_V40_DollarFilterStillWorks tests that $filter with $ prefix
// still works in OData v4.0 (backward compatibility).
func TestOptionalDollarPrefix_V40_DollarFilterStillWorks(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})
db.Create(&TestProduct{ID: 2, Name: "Mouse", Price: 29.99})

req := httptest.NewRequest(http.MethodGet, "/TestProducts?$filter="+url.QueryEscape("Price gt 100"), nil)
req.Header.Set("OData-MaxVersion", "4.0")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value, ok := result["value"].([]interface{})
if !ok {
t.Fatal("Expected 'value' array in response")
}
// Only Laptop (Price=999.99) should match
if len(value) != 1 {
t.Errorf("Expected 1 product ($filter with $ should work in v4.0), got %d", len(value))
}
}

// TestOptionalDollarPrefix_V40_MixedCaseDollarRejected tests that mixed-case $-prefixed options
// are rejected in OData v4.0.
func TestOptionalDollarPrefix_V40_MixedCaseDollarRejected(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})

// $TOP=1 (mixed case) should be rejected in v4.0 mode
req := httptest.NewRequest(http.MethodGet, "/TestProducts?$TOP=1", nil)
req.Header.Set("OData-MaxVersion", "4.0")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusBadRequest {
t.Errorf("Expected 400 Bad Request for mixed-case $TOP in v4.0, got %d: %s", w.Code, w.Body.String())
}
}

// TestOptionalDollarPrefix_DefaultVersion_NoDollarOptionsWork tests that query options without $
// work when no OData-MaxVersion is specified (defaults to 4.01).
func TestOptionalDollarPrefix_DefaultVersion_NoDollarOptionsWork(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})
db.Create(&TestProduct{ID: 2, Name: "Mouse", Price: 29.99})

// No OData-MaxVersion header → defaults to v4.01 → non-$ options should work
req := httptest.NewRequest(http.MethodGet, "/TestProducts?top=1", nil)
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
}

var result map[string]interface{}
if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
t.Fatalf("Failed to parse response: %v", err)
}

value, ok := result["value"].([]interface{})
if !ok {
t.Fatal("Expected 'value' array in response")
}
if len(value) != 1 {
t.Errorf("Expected 1 product (top=1), got %d", len(value))
}
}

// TestOptionalDollarPrefix_V401_UnknownNoDollarParamIgnored tests that unknown parameters
// without $ are silently ignored in OData v4.01 (treated as custom query options).
func TestOptionalDollarPrefix_V401_UnknownNoDollarParamIgnored(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})

// "filtre" is not a known OData option → should be silently ignored
req := httptest.NewRequest(http.MethodGet, "/TestProducts?filtre="+url.QueryEscape("Price gt 100"), nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusOK {
t.Errorf("Expected 200 OK for unknown non-$ parameter, got %d: %s", w.Code, w.Body.String())
}
}

// TestOptionalDollarPrefix_V401_UnknownDollarParamRejected tests that unknown parameters
// with $ are rejected in OData v4.01.
func TestOptionalDollarPrefix_V401_UnknownDollarParamRejected(t *testing.T) {
service, db := setupTestService(t)
db.Create(&TestProduct{ID: 1, Name: "Laptop", Price: 999.99})

// "$filtre" has $ prefix but is not a known OData option → should be rejected
req := httptest.NewRequest(http.MethodGet, "/TestProducts?$filtre="+url.QueryEscape("Price gt 100"), nil)
req.Header.Set("OData-MaxVersion", "4.01")
w := httptest.NewRecorder()

service.ServeHTTP(w, req)

if w.Code != http.StatusBadRequest {
t.Errorf("Expected 400 Bad Request for unknown $-prefixed parameter, got %d: %s", w.Code, w.Body.String())
}
}
