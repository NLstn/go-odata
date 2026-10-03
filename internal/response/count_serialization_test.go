package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
)

func TestIEEE754CompatibleCollectionCount(t *testing.T) {
	md, err := metadata.AnalyzeEntity(ewCat{})
	if err != nil {
		t.Fatal(err)
	}
	provider := newEwProvider(md)
	count := int64(9007199254740993)
	writers := map[string]func(http.ResponseWriter, *http.Request) error{
		"plain collection": func(w http.ResponseWriter, r *http.Request) error {
			return WriteODataCollection(w, r, "Cats", []ewCat{{ID: 1}}, &count, nil)
		},
		"fast collection": func(w http.ResponseWriter, r *http.Request) error {
			return WriteODataCollectionWithNavigation(w, r, "Cats", []ewCat{{ID: 1}}, &count, nil, provider, nil, nil, md)
		},
		"map collection": func(w http.ResponseWriter, r *http.Request) error {
			return WriteODataCollectionWithNavigation(w, r, "Cats", []map[string]interface{}{{"ID": 1}}, &count, nil, provider, nil, nil, md)
		},
		"reference collection": func(w http.ResponseWriter, r *http.Request) error {
			return WriteEntityReferenceCollection(w, r, []string{"Cats(1)"}, &count, nil)
		},
	}
	for name, write := range writers {
		for _, compatible := range []bool{true, false} {
			for _, viaFormat := range []bool{false, true} {
				t.Run(name+"/compatible="+strconv.FormatBool(compatible)+"/format="+strconv.FormatBool(viaFormat), func(t *testing.T) {
					mediaType := "application/json;IEEE754Compatible=" + strconv.FormatBool(compatible)
					r := httptest.NewRequest(http.MethodGet, "/Cats", nil)
					if viaFormat {
						q := r.URL.Query()
						q.Set("$format", mediaType)
						r.URL.RawQuery = q.Encode()
					} else {
						r.Header.Set("Accept", mediaType)
					}
					w := httptest.NewRecorder()
					if err := write(w, r); err != nil {
						t.Fatal(err)
					}
					var body map[string]json.RawMessage
					if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
						t.Fatal(err)
					}
					want := strconv.FormatInt(count, 10)
					if compatible {
						want = `"` + want + `"`
					}
					if string(body["@odata.count"]) != want {
						t.Fatalf("count %s, want %s", body["@odata.count"], want)
					}
					if strings.Contains(w.Header().Get("Content-Type"), "IEEE754Compatible=true") != compatible {
						t.Fatalf("incorrect Content-Type: %s", w.Header().Get("Content-Type"))
					}
					head := httptest.NewRecorder()
					r.Method = http.MethodHead
					if err := write(head, r); err != nil {
						t.Fatal(err)
					}
					if head.Body.Len() != 0 || head.Header().Get("Content-Length") != strconv.Itoa(w.Body.Len()) {
						// Streaming collection and reference writers append a newline for GET.
						if head.Body.Len() != 0 || head.Header().Get("Content-Length") != strconv.Itoa(len(strings.TrimSuffix(w.Body.String(), "\n"))) {
							t.Fatalf("HEAD length %s, GET length %d", head.Header().Get("Content-Length"), w.Body.Len())
						}
					}
				})
			}
		}
	}
}
