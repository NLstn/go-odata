package response

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	internalMetadata "github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
)

// This test locks the direct struct writer (entity_writer.go) to produce
// byte-for-byte identical output to the legacy OrderedMap path
// (addNavigationLinks + envelope.marshalTo) across metadata levels and $select
// shapes. The direct writer is the whole point of issue #861, and its correctness
// hinges on reproducing the slow path exactly; this test is the regression guard.

// ewStatus is a flags enum used to exercise the enum serialization branch.
type ewStatus int32

func (ewStatus) EnumMembers() map[string]int {
	return map[string]int{"None": 0, "A": 1, "B": 2, "C": 4}
}

// ewAddress is a complex (embedded) type.
type ewAddress struct {
	Street string `json:"Street"`
	City   string `json:"City"`
}

// ewEntity is a representative entity: key+etag, scalars, nullable pointer, enum,
// time, binary, a complex type, and both single and collection navigation props.
type ewEntity struct {
	ID          uint       `json:"ID" odata:"key"`
	Name        string     `json:"Name"`
	Description *string    `json:"Description" odata:"nullable"`
	Price       float64    `json:"Price"`
	Status      ewStatus   `json:"Status" odata:"enum=ewStatus,flags"`
	Version     int        `json:"Version" odata:"etag"`
	CreatedAt   time.Time  `json:"CreatedAt"`
	Payload     []byte     `json:"Payload" odata:"nullable"`
	Address     *ewAddress `json:"Address,omitempty" gorm:"embedded"`
	Category    *ewCat     `json:"Category,omitempty" gorm:"foreignKey:CatID;references:ID"`
	CatID       *uint      `json:"CatID" odata:"nullable"`
	Related     []ewCat    `json:"Related,omitempty" gorm:"many2many:ew_rel;"`
}

type ewCat struct {
	ID   uint   `json:"ID" odata:"key"`
	Name string `json:"Name"`
}

// ewProvider adapts an *internalMetadata.EntityMetadata to EntityMetadataProvider,
// mirroring handlers.metadataAdapter (which the response package can't import).
type ewProvider struct {
	md        *internalMetadata.EntityMetadata
	namespace string
	props     []PropertyMetadata
	propMap   map[string]*PropertyMetadata
}

func newEwProvider(md *internalMetadata.EntityMetadata) *ewProvider {
	p := &ewProvider{md: md, namespace: "Test"}
	p.propMap = make(map[string]*PropertyMetadata, len(md.Properties))
	for i := range md.Properties {
		src := &md.Properties[i]
		rp := PropertyMetadata{
			Name:              src.Name,
			JsonName:          src.JsonName,
			IsNavigationProp:  src.IsNavigationProp,
			NavigationTarget:  src.NavigationTarget,
			NavigationIsArray: src.NavigationIsArray,
		}
		p.props = append(p.props, rp)
	}
	for i := range p.props {
		p.propMap[p.props[i].Name] = &p.props[i]
	}
	return p
}

func (p *ewProvider) GetProperties() []PropertyMetadata            { return p.props }
func (p *ewProvider) GetPropertyMap() map[string]*PropertyMetadata { return p.propMap }
func (p *ewProvider) GetEntitySetName() string                     { return p.md.EntitySetName }
func (p *ewProvider) GetNamespace() string                         { return p.namespace }

func (p *ewProvider) GetKeyProperties() []PropertyMetadata {
	var keys []PropertyMetadata
	for i := range p.props {
		if src := p.md.FindProperty(p.props[i].Name); src != nil && src.IsKey {
			keys = append(keys, p.props[i])
		}
	}
	return keys
}

func (p *ewProvider) GetKeyProperty() *PropertyMetadata {
	keys := p.GetKeyProperties()
	if len(keys) == 0 {
		return nil
	}
	return &keys[0]
}

func (p *ewProvider) GetETagProperty() *PropertyMetadata {
	if p.md.ETagProperty == nil {
		return nil
	}
	if rp, ok := p.propMap[p.md.ETagProperty.Name]; ok {
		return rp
	}
	return nil
}

// legacyCollectionBody reproduces the pre-#861 slow path: build each entity as an
// OrderedMap via addNavigationLinks, then serialize the envelope with marshalTo.
func legacyCollectionBody(t *testing.T, data interface{}, provider EntityMetadataProvider, fullMD *internalMetadata.EntityMetadata, metadataLevel string, selectedNavProps []string, contextURL string) string {
	t.Helper()
	r := httptest.NewRequest("GET", "http://example.com/EwEntities", nil)
	transformed := addNavigationLinks(data, provider, nil, selectedNavProps, r, "EwEntities", metadataLevel, fullMD)

	envelope := AcquireOrderedMapWithCapacity(2)
	if contextURL != "" {
		envelope.Set("@odata.context", contextURL)
	}
	envelope.Set("value", transformed)

	var buf bytes.Buffer
	if err := envelope.marshalTo(&buf); err != nil {
		t.Fatalf("legacy marshalTo: %v", err)
	}
	envelope.Release()
	releaseOrderedMaps(transformed)
	return buf.String()
}

func TestFastWriterMatchesLegacyPath(t *testing.T) {
	fullMD, err := internalMetadata.AnalyzeEntity(&ewEntity{})
	if err != nil {
		t.Fatalf("AnalyzeEntity: %v", err)
	}
	fullMD.EntitySetName = "EwEntities"
	provider := newEwProvider(fullMD)

	desc := "a description"
	cat := ewCat{ID: 7, Name: "Cat7"}
	data := []ewEntity{
		{
			ID: 1, Name: "First", Description: &desc, Price: 9.99, Status: 1 | 4,
			Version: 3, CreatedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			Payload: []byte("hello"), Address: &ewAddress{Street: "1 St", City: "Town"},
			Category: &cat, CatID: uintPtr(7), Related: []ewCat{cat},
		},
		{
			ID: 2, Name: "Second", Description: nil, Price: 0, Status: 0,
			Version: 1, CreatedAt: time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
			Payload: nil, Address: nil, Category: nil, CatID: nil,
		},
	}

	cases := []struct {
		name             string
		metadataLevel    string
		selectedProps    []string
		selectedNavProps []string
	}{
		{"minimal_full_shape", MetadataMinimal, nil, nil},
		{"none_full_shape", MetadataNone, nil, nil},
		{"full_full_shape", MetadataFull, nil, nil},
		{"minimal_select_scalars", MetadataMinimal, []string{"Name", "Price"}, nil},
		{"full_select_scalars", MetadataFull, []string{"Name", "Price"}, nil},
		{"none_select_scalars", MetadataNone, []string{"Name", "Price"}, nil},
		{"minimal_select_with_complex", MetadataMinimal, []string{"Name", "Address"}, nil},
		{"minimal_select_etag_field", MetadataMinimal, []string{"Name", "Version"}, nil},
		{"full_select_enum_binary", MetadataFull, []string{"Status", "Payload"}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// FAST path: through the public collection writer.
			rec := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "http://example.com/EwEntities", nil)
			if tc.metadataLevel != MetadataMinimal {
				r.Header.Set("Accept", "application/json;odata.metadata="+tc.metadataLevel)
			}
			if err := WriteODataCollectionWithNavigationAndSelect(rec, r, "EwEntities", data, nil, nil, nil, provider, nil, tc.selectedNavProps, fullMD, tc.selectedProps, 0); err != nil {
				t.Fatalf("fast write: %v", err)
			}
			fastBody := rec.Body.String()

			// Sanity: the fast path must actually have been taken (struct slice, no expand).
			if _, ok := canFastWriteCollection(data, fullMD, nil); !ok {
				t.Fatalf("expected fast path to be eligible")
			}

			// LEGACY path: project to maps first (as the old handler did for $select),
			// then run the OrderedMap serializer.
			var legacyData interface{} = data
			if len(tc.selectedProps) > 0 {
				legacyData = query.ApplySelect(data, tc.selectedProps, fullMD, nil)
			}
			contextURL := ""
			if tc.metadataLevel != MetadataNone {
				contextURL = buildContextURLWithSelect(r, "EwEntities", tc.selectedProps)
			}
			legacyBody := legacyCollectionBody(t, legacyData, provider, fullMD, tc.metadataLevel, tc.selectedNavProps, contextURL)

			if len(tc.selectedProps) == 0 {
				// Plain GET already used the struct→OrderedMap path, so the direct
				// writer must be byte-for-byte identical.
				if fastBody != legacyBody {
					t.Errorf("byte mismatch for %s\n fast:   %s\n legacy: %s", tc.name, fastBody, legacyBody)
				}
				return
			}
			// $select is intentionally moved onto the struct path, so keys now appear
			// in declaration order instead of the legacy map path's alphabetical order.
			// Assert the same set of keys/values (order-insensitive) — same information,
			// consistent ordering with plain GET.
			if !jsonSemanticEqual(t, fastBody, legacyBody) {
				t.Errorf("semantic mismatch for %s\n fast:   %s\n legacy: %s", tc.name, fastBody, legacyBody)
			}
		})
	}
}

// jsonSemanticEqual reports whether two JSON documents are equal ignoring object
// key order (arrays remain order-sensitive).
func jsonSemanticEqual(t *testing.T, a, b string) bool {
	t.Helper()
	var av, bv interface{}
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		t.Fatalf("unmarshal fast body: %v", err)
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		t.Fatalf("unmarshal legacy body: %v", err)
	}
	return reflect.DeepEqual(av, bv)
}

func uintPtr(v uint) *uint { return &v }
