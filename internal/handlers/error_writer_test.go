package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type failingWriter struct {
	header http.Header
}

func (f *failingWriter) Header() http.Header {
	return f.header
}

func (f *failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

func (f *failingWriter) WriteHeader(statusCode int) {}

func TestWriteErrorSuccess(t *testing.T) {
	recorder := httptest.NewRecorder()

	WriteError(recorder, http.StatusBadRequest, "TestCode", "test detail")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	errorSection, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected error object in response: %v", body)
	}

	if errorSection["code"] != "400" {
		t.Fatalf("expected code '400', got %v", errorSection["code"])
	}

	if errorSection["message"] != "TestCode" {
		t.Fatalf("expected message 'TestCode', got %v", errorSection["message"])
	}

	details, ok := errorSection["details"].([]interface{})
	if !ok || len(details) != 1 {
		t.Fatalf("expected single detail entry, got %v", errorSection["details"])
	}

	detail, ok := details[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected detail object, got %T", details[0])
	}

	if detail["message"] != "test detail" {
		t.Fatalf("expected detail 'test detail', got %v", detail["message"])
	}
}

func TestWriteErrorLogsOnFailure(t *testing.T) {
	fw := &failingWriter{header: make(http.Header)}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	WriteError(fw, http.StatusInternalServerError, "TestCode", "test detail")

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stdout = oldStdout

	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read log output: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("failed to close reader: %v", err)
	}

	if !strings.Contains(string(output), "Error writing error response: write error") {
		t.Fatalf("expected fallback log message, got %q", string(output))
	}
}
