package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	internalMetadata "github.com/nlstn/go-odata/internal/metadata"
)

func TestWriteEntityIDEquivalence(t *testing.T) {
	parts := []string{
		"", "ordinary", "https://example.com/odata", "a/b?c=d&e#f%20+;:@!$()*,-._~",
		"'O''Brien'", "\"", "\\", "\x00\x01\b\f\n\r\t\x1f", "<>&",
		"café日本語", "\u2028\u2029", "\xff\xc0", "a\u2028\"b", "\\\xff",
	}
	// The cross product checks that escaping in one component still sends the
	// entire URL through the old encoder, including unusual bytes elsewhere.
	for _, base := range parts {
		for _, set := range parts {
			for _, key := range parts {
				var got, want bytes.Buffer
				var gotEnc, wantEnc *json.Encoder
				if err := writeEntityID(&got, base, set, key, &gotEnc); err != nil {
					t.Fatal(err)
				}
				url := base + "/" + set + "(" + key + ")"
				if err := writeJSONString(&want, url, &wantEnc); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got.Bytes(), want.Bytes()) {
					t.Fatalf("URL %q: got %q, want %q", url, got.Bytes(), want.Bytes())
				}
				// Exercise reuse of an encoder after both direct and escaped IDs.
				if err := writeEntityID(&got, base, set, key, &gotEnc); err != nil {
					t.Fatal(err)
				}
				if err := writeJSONString(&want, url, &wantEnc); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got.Bytes(), want.Bytes()) {
					t.Fatalf("repeated URL %q differs", url)
				}
			}
		}
	}
}

type entityIDNamedKey int64

func TestFastEntityIDKeyRepresentations(t *testing.T) {
	type stringEntity struct {
		ID string `odata:"key"`
	}
	type namedEntity struct {
		ID entityIDNamedKey `odata:"key"`
	}
	type pointerEntity struct {
		ID *int `odata:"key"`
	}
	type compositeEntity struct {
		ID   int    `odata:"key"`
		Code string `odata:"key"`
	}
	id := 42
	entities := []interface{}{
		stringEntity{"O'Brien/%?#\\\"\u2028"}, namedEntity{42},
		pointerEntity{&id}, pointerEntity{nil}, compositeEntity{42, "a'&/\""},
	}
	for _, entity := range entities {
		md, err := internalMetadata.AnalyzeEntity(entity)
		if err != nil {
			t.Fatal(err)
		}
		provider := newEwProvider(md)
		value := reflect.ValueOf(entity)
		for _, level := range []string{MetadataMinimal, MetadataFull, MetadataNone} {
			t.Run(fmt.Sprintf("%T/%s", entity, level), func(t *testing.T) {
				ctx := &fastEntityContext{
					baseURL: "https://example.com/odata", entitySetName: "Entities",
					metadataLevel: level, metadata: provider, fullMetadata: md,
				}
				var got bytes.Buffer
				var enc *json.Encoder
				if err := writeFastEntity(&got, value, ctx, &enc); err != nil {
					t.Fatal(err)
				}
				// Compare the complete emitted ID token against the old URL
				// construction, retaining the existing key-segment representation.
				key := buildKeySegmentFromEntityCached(value, provider)
				var want bytes.Buffer
				var wantEnc *json.Encoder
				want.WriteString(`"@odata.id":`)
				if err := writeJSONString(&want, ctx.baseURL+"/Entities("+key+")", &wantEnc); err != nil {
					t.Fatal(err)
				}
				if level == MetadataNone || key == "" {
					if bytes.Contains(got.Bytes(), []byte(`"@odata.id":`)) {
						t.Fatalf("unexpected ID in %s", got.Bytes())
					}
				} else if !bytes.Contains(got.Bytes(), want.Bytes()) {
					t.Fatalf("missing ID token %s in %s", want.Bytes(), got.Bytes())
				}
			})
		}
	}
}
