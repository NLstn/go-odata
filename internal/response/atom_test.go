package response

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestIsAtomFormat tests the IsAtomFormat function
func TestIsAtomFormat(t *testing.T) {
	tests := []struct {
		name         string
		acceptHeader string
		formatParam  string
		want         bool
	}{
		{
			name:        "format=atom returns true",
			formatParam: "atom",
			want:        true,
		},
		{
			name:        "format=application/atom+xml returns true",
			formatParam: "application/atom+xml",
			want:        true,
		},
		{
			name:        "format=json returns false",
			formatParam: "json",
			want:        false,
		},
		{
			name:         "Accept: application/atom+xml returns true",
			acceptHeader: "application/atom+xml",
			want:         true,
		},
		{
			name:         "Accept: application/json returns false",
			acceptHeader: "application/json",
			want:         false,
		},
		{
			name: "no format or accept returns false",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawURL := "/test"
			if tt.formatParam != "" {
				rawURL += "?$format=" + url.QueryEscape(tt.formatParam)
			}
			req := httptest.NewRequest(http.MethodGet, rawURL, nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			got := IsAtomFormat(req)
			if got != tt.want {
				t.Errorf("IsAtomFormat() = %v, want %v (format=%q, accept=%q)", got, tt.want, tt.formatParam, tt.acceptHeader)
			}
		})
	}
}

// TestWriteAtomCollection tests the WriteAtomCollection function
func TestWriteAtomCollection(t *testing.T) {
	type Product struct {
		ProductID   int    `json:"ProductID"`
		ProductName string `json:"ProductName"`
	}

	products := []Product{
		{ProductID: 1, ProductName: "Widget"},
		{ProductID: 2, ProductName: "Gadget"},
	}

	count := int64(2)

	req := httptest.NewRequest(http.MethodGet, "/Products?$format=atom", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()

	keyProps := []PropertyMetadata{
		{Name: "ProductID", JsonName: "ProductID"},
	}

	err := WriteAtomCollection(w, req, "Products", products, &count, nil, nil, keyProps)
	if err != nil {
		t.Fatalf("WriteAtomCollection returned error: %v", err)
	}

	resp := w.Result()

	// Check Content-Type
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/atom+xml") {
		t.Errorf("expected Content-Type to contain application/atom+xml, got %q", ct)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()

	// Validate it is valid XML
	var feed interface{}
	if err := xml.Unmarshal([]byte(body), &feed); err != nil {
		t.Errorf("response is not valid XML: %v\nBody: %s", err, body)
	}

	// Check key structural elements
	if !strings.Contains(body, "<feed") {
		t.Errorf("expected <feed> element in response")
	}
	if !strings.Contains(body, "<entry>") {
		t.Errorf("expected <entry> elements in response")
	}
	if !strings.Contains(body, "Products") {
		t.Errorf("expected entity set name in response")
	}
	// Check properties are present
	if !strings.Contains(body, "Widget") {
		t.Errorf("expected property value 'Widget' in response")
	}
	if !strings.Contains(body, "Gadget") {
		t.Errorf("expected property value 'Gadget' in response")
	}
}

// TestWriteAtomEntity tests the WriteAtomEntity function
func TestWriteAtomEntity(t *testing.T) {
	type Product struct {
		ProductID   int    `json:"ProductID"`
		ProductName string `json:"ProductName"`
	}

	product := &Product{ProductID: 1, ProductName: "Widget"}

	req := httptest.NewRequest(http.MethodGet, "/Products(1)?$format=atom", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()

	err := WriteAtomEntity(w, req, "Products", "http://localhost:8080/Products(1)", product, "", 0)
	if err != nil {
		t.Fatalf("WriteAtomEntity returned error: %v", err)
	}

	resp := w.Result()

	// Check Content-Type
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/atom+xml") {
		t.Errorf("expected Content-Type to contain application/atom+xml, got %q", ct)
	}

	body := w.Body.String()

	// Validate it is valid XML
	var entry interface{}
	if err := xml.Unmarshal([]byte(body), &entry); err != nil {
		t.Errorf("response is not valid XML: %v\nBody: %s", err, body)
	}

	// Check key structural elements
	if !strings.Contains(body, "<entry") {
		t.Errorf("expected <entry> element in response")
	}
	if !strings.Contains(body, "Widget") {
		t.Errorf("expected property value 'Widget' in response")
	}
}

// TestExtractAtomDataProperties tests that control keys are filtered out
func TestExtractAtomDataProperties(t *testing.T) {
	om := NewOrderedMap()
	om.Set("@odata.context", "http://host/$metadata#Products")
	om.Set("@odata.id", "http://host/Products(1)")
	om.Set("__temp_entity_id", "http://host/Products(1)")
	om.Set("ProductID", 1)
	om.Set("ProductName", "Widget")

	props := extractAtomDataProperties(om)

	for _, p := range props {
		if strings.HasPrefix(p.name, "@") || strings.HasPrefix(p.name, "__temp_") {
			t.Errorf("extractAtomDataProperties included control key: %q", p.name)
		}
	}

	found := map[string]bool{}
	for _, p := range props {
		found[p.name] = true
	}
	if !found["ProductID"] {
		t.Error("expected ProductID in extracted properties")
	}
	if !found["ProductName"] {
		t.Error("expected ProductName in extracted properties")
	}
}

// TestWriteAtomCollectionWithOrderedMapData tests atom serialization with *OrderedMap data
func TestWriteAtomCollectionWithOrderedMapData(t *testing.T) {
	om := NewOrderedMap()
	om.Set("@odata.id", "http://localhost:8080/Products(42)")
	om.Set("ProductID", 42)
	om.Set("ProductName", "Gizmo")

	data := []interface{}{om}

	req := httptest.NewRequest(http.MethodGet, "/Products?$format=atom", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()

	err := WriteAtomCollection(w, req, "Products", data, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("WriteAtomCollection returned error: %v", err)
	}

	body := w.Body.String()

	if !strings.Contains(body, "http://localhost:8080/Products(42)") {
		t.Errorf("expected entity ID in response body, got: %s", body)
	}
	if !strings.Contains(body, "Gizmo") {
		t.Errorf("expected property value 'Gizmo' in response")
	}
}
