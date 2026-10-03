package handlers

import (
	"bufio"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func submitMultipartBatch(t *testing.T, handler *BatchHandler, body, prefer string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/$batch", strings.NewReader(strings.ReplaceAll(body, "\n", "\r\n")))
	r.Header.Set("Content-Type", "multipart/mixed;boundary=batch")
	r.Header.Set("Prefer", prefer)
	w := httptest.NewRecorder()
	handler.HandleBatch(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("batch status %d: %s", w.Code, w.Body.String())
	}
	return w
}

func multipartResponseReader(t *testing.T, contentType string, body io.Reader) *multipart.Reader {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/mixed" || params["boundary"] == "" {
		t.Fatalf("invalid multipart Content-Type %q: %v", contentType, err)
	}
	return multipart.NewReader(body, params["boundary"])
}

func assertBatchHTTPPart(t *testing.T, reader *multipart.Reader, status int, contentID string) *http.Response {
	t.Helper()
	part, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	if part.Header.Get("Content-Type") != "application/http" || part.Header.Get("Content-ID") != contentID {
		t.Fatalf("unexpected MIME headers: %v", part.Header)
	}
	resp, err := http.ReadResponse(bufio.NewReader(part), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != status {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d, want %d: %s", resp.StatusCode, status, body)
	}
	return resp
}

func assertMultipartEnd(t *testing.T, reader *multipart.Reader) {
	t.Helper()
	if _, err := reader.NextPart(); err != io.EOF {
		t.Fatalf("expected MIME end, got %v", err)
	}
}

func TestMultipartBatchURLForms(t *testing.T) {
	handler, db, _ := setupBatchTestHandler(t)
	if err := db.Create(&BatchTestProduct{ID: 1, Name: "Original"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"BatchTestProducts(1)", "/BatchTestProducts(1)", "http://localhost:19091/BatchTestProducts(1)"} {
		t.Run(target, func(t *testing.T) {
			body := fmt.Sprintf("--batch\nContent-Type: application/http\n\nGET %s HTTP/1.1\nHost: localhost:19091\n\n\n--batch--\n", target)
			w := submitMultipartBatch(t, handler, body, "")
			reader := multipartResponseReader(t, w.Header().Get("Content-Type"), w.Body)
			resp := assertBatchHTTPPart(t, reader, http.StatusOK, "")
			data, err := io.ReadAll(resp.Body)
			if err != nil || !strings.Contains(string(data), "Original") {
				t.Fatalf("wrong entity response: %s, err=%v", data, err)
			}
			assertMultipartEnd(t, reader)
		})
	}
}

func TestMultipartChangesetResponseEnvelope(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint("failure=", fail), func(t *testing.T) {
			handler, db, service := setupBatchTestHandler(t)
			if err := db.Create(&BatchTestProduct{ID: 1, Name: "Original"}).Error; err != nil {
				t.Fatal(err)
			}
			second := "PATCH /BatchTestProducts(1) HTTP/1.1\nContent-Type: application/json\n\n{\"Price\":10}"
			if fail {
				second = "DELETE /BatchTestProducts(999) HTTP/1.1\n\n"
			}
			body := "--batch\nContent-Type: multipart/mixed;boundary=changeset\n\n" +
				"--changeset\nContent-Type: application/http\nContent-ID: 1\n\nPATCH http://localhost:19091/BatchTestProducts(1) HTTP/1.1\nContent-Type: application/json\n\n{\"Name\":\"Updated\"}\n" +
				"--changeset\nContent-Type: application/http\nContent-ID: 2\n\n" + second + "\n--changeset--\n--batch--\n"
			w := submitMultipartBatch(t, handler, body, "")
			outer := multipartResponseReader(t, w.Header().Get("Content-Type"), w.Body)
			if fail {
				assertBatchHTTPPart(t, outer, http.StatusNotFound, "2")
			} else {
				part, err := outer.NextPart()
				if err != nil {
					t.Fatal(err)
				}
				inner := multipartResponseReader(t, part.Header.Get("Content-Type"), part)
				assertBatchHTTPPart(t, inner, http.StatusNoContent, "1")
				assertBatchHTTPPart(t, inner, http.StatusNoContent, "2")
				assertMultipartEnd(t, inner)
			}
			assertMultipartEnd(t, outer)
			read := httptest.NewRecorder()
			service.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/BatchTestProducts(1)", nil))
			wantName := "Updated"
			if fail {
				wantName = "Original"
			}
			if read.Code != http.StatusOK || !strings.Contains(read.Body.String(), wantName) {
				t.Fatalf("wrong persisted state: %s", read.Body.String())
			}
		})
	}
}

func TestMultipartBatchStopOnError(t *testing.T) {
	for _, prefer := range []string{"", "odata.continue-on-error"} {
		t.Run(prefer, func(t *testing.T) {
			handler, _, _ := setupBatchTestHandler(t)
			body := "--batch\nContent-Type: application/http\n\nGET /BatchTestProducts(999) HTTP/1.1\n\n\n" +
				"--batch\nContent-Type: application/http\n\nGET /BatchTestProducts HTTP/1.1\n\n\n--batch--\n"
			w := submitMultipartBatch(t, handler, body, prefer)
			reader := multipartResponseReader(t, w.Header().Get("Content-Type"), w.Body)
			assertBatchHTTPPart(t, reader, http.StatusNotFound, "")
			if prefer != "" {
				assertBatchHTTPPart(t, reader, http.StatusOK, "")
			}
			assertMultipartEnd(t, reader)
		})
	}
}
