package skiptoken

import (
	"encoding/base64"
	"testing"
)

type TestEntity struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func TestEncode(t *testing.T) {
	token := &SkipToken{
		KeyValues: map[string]interface{}{
			"id": 123,
		},
	}

	encoded, err := Encode(token)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if encoded == "" {
		t.Error("Expected non-empty encoded token")
	}

	// Verify it's valid base64
	if _, err := base64.URLEncoding.DecodeString(encoded); err != nil {
		t.Errorf("Encoded token is not valid base64: %v", err)
	}
}

func TestEncodeNil(t *testing.T) {
	_, err := Encode(nil)
	if err == nil {
		t.Error("Expected error when encoding nil token")
	}
}

func TestDecode(t *testing.T) {
	original := &SkipToken{
		KeyValues: map[string]interface{}{
			"id": float64(456), // JSON numbers are float64
		},
	}

	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.KeyValues["id"] != float64(456) {
		t.Errorf("Expected id=456, got %v", decoded.KeyValues["id"])
	}
}

func TestDecodeEmpty(t *testing.T) {
	_, err := Decode("")
	if err == nil {
		t.Error("Expected error when decoding empty string")
	}
}

func TestDecodeInvalid(t *testing.T) {
	_, err := Decode("invalid-base64!")
	if err == nil {
		t.Error("Expected error when decoding invalid base64")
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := &SkipToken{
		KeyValues: map[string]interface{}{
			"id":   float64(789),
			"code": "ABC123",
		},
		OrderByValues: map[string]interface{}{
			"price": float64(99.99),
			"name":  "Product X",
		},
	}

	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify key values
	if decoded.KeyValues["id"] != float64(789) {
		t.Errorf("Expected id=789, got %v", decoded.KeyValues["id"])
	}
	if decoded.KeyValues["code"] != "ABC123" {
		t.Errorf("Expected code=ABC123, got %v", decoded.KeyValues["code"])
	}

	// Verify orderby values
	if decoded.OrderByValues["price"] != float64(99.99) {
		t.Errorf("Expected price=99.99, got %v", decoded.OrderByValues["price"])
	}
	if decoded.OrderByValues["name"] != "Product X" {
		t.Errorf("Expected name=Product X, got %v", decoded.OrderByValues["name"])
	}
}

func TestExtractFromEntity_Struct(t *testing.T) {
	entity := TestEntity{
		ID:    100,
		Name:  "Test Product",
		Price: 49.99,
	}

	token, err := ExtractFromEntity(entity, []string{"id"}, []string{"name"})
	if err != nil {
		t.Fatalf("ExtractFromEntity failed: %v", err)
	}

	if token.KeyValues["id"] != 100 {
		t.Errorf("Expected id=100, got %v", token.KeyValues["id"])
	}

	if token.OrderByValues["name"] != "Test Product" {
		t.Errorf("Expected name=Test Product, got %v", token.OrderByValues["name"])
	}
}

func TestExtractFromEntity_StructPointer(t *testing.T) {
	entity := &TestEntity{
		ID:    200,
		Name:  "Another Product",
		Price: 29.99,
	}

	token, err := ExtractFromEntity(entity, []string{"id"}, []string{"price"})
	if err != nil {
		t.Fatalf("ExtractFromEntity failed: %v", err)
	}

	if token.KeyValues["id"] != 200 {
		t.Errorf("Expected id=200, got %v", token.KeyValues["id"])
	}

	if token.OrderByValues["price"] != 29.99 {
		t.Errorf("Expected price=29.99, got %v", token.OrderByValues["price"])
	}
}

func TestExtractFromEntity_Map(t *testing.T) {
	entity := map[string]interface{}{
		"id":    300,
		"name":  "Map Product",
		"price": 19.99,
	}

	token, err := ExtractFromEntity(entity, []string{"id"}, []string{"name"})
	if err != nil {
		t.Fatalf("ExtractFromEntity failed: %v", err)
	}

	if token.KeyValues["id"] != 300 {
		t.Errorf("Expected id=300, got %v", token.KeyValues["id"])
	}

	if token.OrderByValues["name"] != "Map Product" {
		t.Errorf("Expected name=Map Product, got %v", token.OrderByValues["name"])
	}
}

func TestExtractFromEntity_CompositeKey(t *testing.T) {
	type CompositeEntity struct {
		ID1  int    `json:"id1"`
		ID2  string `json:"id2"`
		Name string `json:"name"`
	}

	entity := CompositeEntity{
		ID1:  1,
		ID2:  "A",
		Name: "Composite",
	}

	token, err := ExtractFromEntity(entity, []string{"id1", "id2"}, nil)
	if err != nil {
		t.Fatalf("ExtractFromEntity failed: %v", err)
	}

	if token.KeyValues["id1"] != 1 {
		t.Errorf("Expected id1=1, got %v", token.KeyValues["id1"])
	}
	if token.KeyValues["id2"] != "A" {
		t.Errorf("Expected id2=A, got %v", token.KeyValues["id2"])
	}
}

func TestExtractFromEntity_NoOrderBy(t *testing.T) {
	entity := TestEntity{
		ID:    400,
		Name:  "No OrderBy",
		Price: 9.99,
	}

	token, err := ExtractFromEntity(entity, []string{"id"}, nil)
	if err != nil {
		t.Fatalf("ExtractFromEntity failed: %v", err)
	}

	if token.KeyValues["id"] != 400 {
		t.Errorf("Expected id=400, got %v", token.KeyValues["id"])
	}

	if token.OrderByValues != nil {
		t.Error("Expected OrderByValues to be nil when no orderby properties provided")
	}
}

func TestExtractFromEntity_OrderByWithSuffix(t *testing.T) {
	entity := TestEntity{
		ID:    500,
		Name:  "OrderBy Suffix Test",
		Price: 99.99,
	}

	// Test with " desc" suffix
	token, err := ExtractFromEntity(entity, []string{"id"}, []string{"price desc"})
	if err != nil {
		t.Fatalf("ExtractFromEntity failed: %v", err)
	}

	if token.OrderByValues["price"] != 99.99 {
		t.Errorf("Expected price=99.99, got %v", token.OrderByValues["price"])
	}

	// Test with " asc" suffix
	token, err = ExtractFromEntity(entity, []string{"id"}, []string{"name asc"})
	if err != nil {
		t.Fatalf("ExtractFromEntity failed: %v", err)
	}

	if token.OrderByValues["name"] != "OrderBy Suffix Test" {
		t.Errorf("Expected name=OrderBy Suffix Test, got %v", token.OrderByValues["name"])
	}
}

func TestExtractFromEntity_InvalidKey(t *testing.T) {
	entity := TestEntity{
		ID:    600,
		Name:  "Invalid Test",
		Price: 5.99,
	}

	_, err := ExtractFromEntity(entity, []string{"nonexistent"}, nil)
	if err == nil {
		t.Error("Expected error when extracting non-existent key property")
	}
}

func TestExtractFromEntity_InvalidOrderBy(t *testing.T) {
	entity := TestEntity{
		ID:    700,
		Name:  "Invalid OrderBy",
		Price: 3.99,
	}

	_, err := ExtractFromEntity(entity, []string{"id"}, []string{"nonexistent"})
	if err == nil {
		t.Error("Expected error when extracting non-existent orderby property")
	}
}
