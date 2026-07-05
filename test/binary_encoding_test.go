package odata_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestBinaryEncodingItem is used to verify that Edm.Binary property values are
// serialized to/from JSON using the base64url alphabet without padding, per
// the OData JSON Format spec v4.0 §7.1 (RFC 4648 §5), rather than Go's default
// (standard base64 with padding) []byte JSON encoding. See issue #789.
type TestBinaryEncodingItem struct {
	ID   int    `json:"id" gorm:"primarykey" odata:"key"`
	Name string `json:"name"`
	Data []byte `json:"data" gorm:"type:blob" odata:"nullable"`
}

func setupBinaryEncodingTestService(t *testing.T) (*odata.Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&TestBinaryEncodingItem{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	service, err := odata.NewService(db)
	if err != nil {
		t.Fatalf("NewService() error: %v", err)
	}
	if err := service.RegisterEntity(TestBinaryEncodingItem{}); err != nil {
		t.Fatalf("Failed to register entity: %v", err)
	}
	return service, db
}

// binaryTestBytes produces a single byte whose standard base64 representation
// requires both the '/' character and '==' padding ("/w=="), so that a test
// asserting on its base64url form ("_w", no padding) actually exercises both
// aspects of RFC 4648 §5 (alphabet substitution and padding removal).
var binaryTestBytes = []byte{0xFF}

func decodeJSONObject(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("Failed to decode JSON response: %v (body: %s)", err, body)
	}
	return m
}

// TestBinaryPropertyPostGet_UsesBase64URLAlphabetNoPadding verifies that POSTing
// an Edm.Binary value encoded with the OData-mandated base64url alphabet is
// accepted, and that reading it back returns the same base64url (unpadded)
// representation rather than Go's default standard-alphabet, padded encoding.
func TestBinaryPropertyPostGet_UsesBase64URLAlphabetNoPadding(t *testing.T) {
	service, _ := setupBinaryEncodingTestService(t)

	urlEncoded := base64.RawURLEncoding.EncodeToString(binaryTestBytes) // "_w"
	stdEncoded := base64.StdEncoding.EncodeToString(binaryTestBytes)    // "/w=="
	if urlEncoded != "_w" || stdEncoded != "/w==" {
		t.Fatalf("test fixture assumption broken: url=%q std=%q", urlEncoded, stdEncoded)
	}

	postBody := `{"name":"widget","data":"` + urlEncoded + `"}`
	req := httptest.NewRequest(http.MethodPost, "/TestBinaryEncodingItems", bytes.NewBufferString(postBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/TestBinaryEncodingItems(1)?$select=data", nil)
	w2 := httptest.NewRecorder()
	service.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d. Body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	resp := decodeJSONObject(t, w2.Body.Bytes())
	dataValue, ok := resp["data"].(string)
	if !ok {
		t.Fatalf("expected 'data' to be a string, got %T (%v)", resp["data"], resp["data"])
	}

	if dataValue != urlEncoded {
		t.Errorf("data = %q, want base64url-encoded %q", dataValue, urlEncoded)
	}
	if strings.ContainsAny(dataValue, "+/=") {
		t.Errorf("data %q contains standard-base64-only characters ('+' '/' '='); must use base64url without padding", dataValue)
	}

	decoded, err := base64.RawURLEncoding.DecodeString(dataValue)
	if err != nil {
		t.Fatalf("Failed to decode returned value as base64url: %v", err)
	}
	if !bytes.Equal(decoded, binaryTestBytes) {
		t.Errorf("decoded bytes = %v, want %v", decoded, binaryTestBytes)
	}
}

// TestBinaryPropertyPost_StandardBase64AlsoAccepted verifies that the write path
// leniently accepts standard base64 (with padding) input for interoperability,
// while the value read back is still normalized to base64url per the spec.
func TestBinaryPropertyPost_StandardBase64AlsoAccepted(t *testing.T) {
	service, _ := setupBinaryEncodingTestService(t)

	stdEncoded := base64.StdEncoding.EncodeToString(binaryTestBytes) // "/w=="
	postBody := `{"name":"widget","data":"` + stdEncoded + `"}`
	req := httptest.NewRequest(http.MethodPost, "/TestBinaryEncodingItems", bytes.NewBufferString(postBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want %d. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/TestBinaryEncodingItems(1)?$select=data", nil)
	w2 := httptest.NewRecorder()
	service.ServeHTTP(w2, req2)

	resp := decodeJSONObject(t, w2.Body.Bytes())
	dataValue, _ := resp["data"].(string)
	wantURL := base64.RawURLEncoding.EncodeToString(binaryTestBytes)
	if dataValue != wantURL {
		t.Errorf("data = %q, want normalized base64url %q", dataValue, wantURL)
	}
}

// TestBinaryPropertyPost_InvalidBase64Rejected verifies that a value which isn't
// valid base64 under either alphabet is still rejected with a 400 error.
func TestBinaryPropertyPost_InvalidBase64Rejected(t *testing.T) {
	service, _ := setupBinaryEncodingTestService(t)

	postBody := `{"name":"widget","data":"not-valid-base64!!!"}`
	req := httptest.NewRequest(http.MethodPost, "/TestBinaryEncodingItems", bytes.NewBufferString(postBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("POST status = %d, want %d. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestBinaryPropertyPatch_RoundTripsBase64URL verifies that PATCH accepts a
// base64url-encoded binary value and that the updated value round-trips
// correctly through a subsequent GET, using the base64url alphabet.
func TestBinaryPropertyPatch_RoundTripsBase64URL(t *testing.T) {
	service, db := setupBinaryEncodingTestService(t)

	initial := TestBinaryEncodingItem{ID: 1, Name: "widget", Data: []byte{0x01, 0x02, 0x03}}
	if err := db.Create(&initial).Error; err != nil {
		t.Fatalf("failed to seed entity: %v", err)
	}

	newBytes := []byte{0xFF, 0xFE, 0xFD} // std base64 "//79" contains '/'
	stdOfNew := base64.StdEncoding.EncodeToString(newBytes)
	urlOfNew := base64.RawURLEncoding.EncodeToString(newBytes)
	if !strings.ContainsAny(stdOfNew, "+/") {
		t.Fatalf("test fixture assumption broken: std encoding %q does not contain +/", stdOfNew)
	}

	patchBody := `{"data":"` + urlOfNew + `"}`
	req := httptest.NewRequest(http.MethodPatch, "/TestBinaryEncodingItems(1)", bytes.NewBufferString(patchBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, want 204 or 200. Body: %s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/TestBinaryEncodingItems(1)?$select=data", nil)
	w2 := httptest.NewRecorder()
	service.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d. Body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	resp := decodeJSONObject(t, w2.Body.Bytes())
	dataValue, ok := resp["data"].(string)
	if !ok {
		t.Fatalf("expected 'data' to be a string, got %T (%v)", resp["data"], resp["data"])
	}
	if dataValue != urlOfNew {
		t.Errorf("data after PATCH = %q, want %q", dataValue, urlOfNew)
	}
	if strings.ContainsAny(dataValue, "+/=") {
		t.Errorf("data %q after PATCH contains standard-base64-only characters", dataValue)
	}

	decoded, err := base64.RawURLEncoding.DecodeString(dataValue)
	if err != nil {
		t.Fatalf("Failed to decode returned value as base64url: %v", err)
	}
	if !bytes.Equal(decoded, newBytes) {
		t.Errorf("decoded bytes after PATCH = %v, want %v", decoded, newBytes)
	}
}

// TestBinaryPropertyCollectionGet_UsesBase64URLAlphabet verifies that binary
// properties are correctly base64url-encoded when returned as part of a
// collection response (a different serialization code path than a single
// entity GET), both with and without $select.
func TestBinaryPropertyCollectionGet_UsesBase64URLAlphabet(t *testing.T) {
	service, db := setupBinaryEncodingTestService(t)

	item := TestBinaryEncodingItem{ID: 1, Name: "widget", Data: binaryTestBytes}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("failed to seed entity: %v", err)
	}
	wantURL := base64.RawURLEncoding.EncodeToString(binaryTestBytes)

	// Collection GET without $select (struct-based serialization path).
	req := httptest.NewRequest(http.MethodGet, "/TestBinaryEncodingItems", nil)
	w := httptest.NewRecorder()
	service.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d. Body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"data":"`+wantURL+`"`) {
		t.Errorf("collection response does not contain expected base64url data %q: %s", wantURL, w.Body.String())
	}

	// Collection GET with $select (map-based serialization path).
	reqSelect := httptest.NewRequest(http.MethodGet, "/TestBinaryEncodingItems?$select=data", nil)
	wSelect := httptest.NewRecorder()
	service.ServeHTTP(wSelect, reqSelect)
	if wSelect.Code != http.StatusOK {
		t.Fatalf("GET ($select) status = %d, want %d. Body: %s", wSelect.Code, http.StatusOK, wSelect.Body.String())
	}
	if !strings.Contains(wSelect.Body.String(), `"data":"`+wantURL+`"`) {
		t.Errorf("$select collection response does not contain expected base64url data %q: %s", wantURL, wSelect.Body.String())
	}
}
