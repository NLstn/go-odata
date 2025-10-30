package async

import (
	"net/http"
	"reflect"
	"testing"
)

func TestSerializeHeadersRoundTrip(t *testing.T) {
	header := http.Header{
		"Content-Type":       []string{"application/json"},
		"Preference-Applied": []string{"respond-async", "return=minimal"},
	}

	data, err := serializeHeaders(header)
	if err != nil {
		t.Fatalf("serializeHeaders error: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected serialized data, got empty slice")
	}

	decoded, err := deserializeHeaders(data)
	if err != nil {
		t.Fatalf("deserializeHeaders error: %v", err)
	}

	if !reflect.DeepEqual(header, decoded) {
		t.Fatalf("headers mismatch: want %v got %v", header, decoded)
	}
}

func TestStoredResponseSerializationRoundTrip(t *testing.T) {
	resp := &StoredResponse{
		StatusCode: http.StatusCreated,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: []byte(`{"status":"ok"}`),
	}

	status, hdr, body, err := storedResponseToRecord(resp)
	if err != nil {
		t.Fatalf("storedResponseToRecord error: %v", err)
	}
	if status == nil || *status != http.StatusCreated {
		t.Fatalf("expected status %d, got %v", http.StatusCreated, status)
	}

	restored, err := storedResponseFromRecord(status, hdr, body)
	if err != nil {
		t.Fatalf("storedResponseFromRecord error: %v", err)
	}

	if !reflect.DeepEqual(resp, restored) {
		t.Fatalf("restored response mismatch: want %+v got %+v", resp, restored)
	}
}

func TestStoredResponseSerializationNil(t *testing.T) {
	status, hdr, body, err := storedResponseToRecord(nil)
	if err != nil {
		t.Fatalf("storedResponseToRecord error: %v", err)
	}
	if status != nil || hdr != nil || body != nil {
		t.Fatalf("expected nil outputs for nil response")
	}

	restored, err := storedResponseFromRecord(status, hdr, body)
	if err != nil {
		t.Fatalf("storedResponseFromRecord error: %v", err)
	}
	if restored != nil {
		t.Fatalf("expected nil restored response, got %+v", restored)
	}
}
