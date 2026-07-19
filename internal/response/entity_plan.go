package response

import (
	"reflect"
	"sync"

	internalMetadata "github.com/nlstn/go-odata/internal/metadata"
)

// entityFieldPlan is a cached, per-(entity type, *EntityMetadata) resolution of
// the per-field metadata that processStructEntityOrderedInto would otherwise look
// up from two hash maps on every field of every entity in every response.
//
// The plan pre-resolves, for each struct field (indexed to match
// getFieldInfos(t) / entity.Field(j)):
//   - whether the field is skipped (unexported or json:"-"),
//   - its navigation-property descriptor (synthesized from the stable
//     *EntityMetadata; only set for navigation fields), and
//   - its full property metadata (for enum/stream/annotation/complex handling).
//
// Every value it carries is identical to what the per-field map lookups
// (propMetaMap[name], fullPropMetaByName[name]) returned before — the plan just
// does those lookups once per type instead of once per field per entity.
type entityFieldPlan struct {
	entries []entityFieldEntry
}

type entityFieldEntry struct {
	skip     bool
	jsonName string
	// navProp is non-nil only for navigation-property fields. It is synthesized
	// from the *EntityMetadata property so it matches the response.PropertyMetadata
	// the metadata provider would have returned (which is itself a field-for-field
	// copy of the same *EntityMetadata property — see metadataAdapter).
	navProp  *PropertyMetadata
	fullProp *internalMetadata.PropertyMetadata
	// goName is the Go struct field name (used by the direct struct writer to test
	// $select membership against either the Go name or the JSON name).
	goName string
	// isEnum/isBinary pre-classify the field so the direct struct writer avoids a
	// per-field metadata check. isEnum mirrors enumOrRaw's condition; isBinary is
	// true only for []byte fields (Edm.Binary), which serialize as base64url.
	isEnum   bool
	isBinary bool
	// quotedKey is the pre-rendered JSON object key including quotes and the colon,
	// e.g. `"Name":`. The direct writer emits it with a single buf.Write instead of
	// re-quoting and re-scanning the (static) key for escapes on every row. It is
	// nil when the JSON name would need escaping, in which case the writer falls
	// back to writeJSONKey; escaping is checked once here, at plan-build time.
	quotedKey []byte
}

type entityPlanKey struct {
	t  reflect.Type
	md *internalMetadata.EntityMetadata
}

var entityFieldPlanCache sync.Map // entityPlanKey -> *entityFieldPlan

// getEntityFieldPlan returns the cached field plan for entities of type t
// described by md. The result is safe to reuse concurrently and is never mutated
// after construction.
func getEntityFieldPlan(md *internalMetadata.EntityMetadata, t reflect.Type) *entityFieldPlan {
	key := entityPlanKey{t: t, md: md}
	if cached, ok := entityFieldPlanCache.Load(key); ok {
		return cached.(*entityFieldPlan) //nolint:errcheck // value type guaranteed by our Store calls
	}

	byName := getFullPropMetaByName(md)
	infos := getFieldInfos(t)
	entries := make([]entityFieldEntry, len(infos))
	for j := range infos {
		info := infos[j]
		e := entityFieldEntry{jsonName: info.JsonName, goName: info.Name}
		if !info.IsExported || info.JsonName == "" {
			e.skip = true
			entries[j] = e
			continue
		}
		fp := byName[info.Name]
		e.fullProp = fp
		if fp != nil && fp.IsNavigationProp {
			e.navProp = &PropertyMetadata{
				Name:              fp.Name,
				JsonName:          fp.JsonName,
				IsNavigationProp:  true,
				NavigationTarget:  fp.NavigationTarget,
				NavigationIsArray: fp.NavigationIsArray,
			}
		}
		if fp != nil && fp.IsEnum && len(fp.EnumMembers) > 0 {
			e.isEnum = true
		}
		// A []byte field is Edm.Binary and serializes as a base64url string via
		// EncodeEdmBinary, exactly as the OrderedMap path does.
		if ft := t.Field(j).Type; ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
			e.isBinary = true
		}
		// Pre-render the JSON key (`"name":`) once, skipping the per-row re-quote and
		// escape scan. Left nil when the name would need escaping (rare for struct
		// tags), so the writer falls back to writeJSONKey for exact output.
		if !needsEscaping(info.JsonName) {
			e.quotedKey = append(append(append(make([]byte, 0, len(info.JsonName)+3), '"'), info.JsonName...), '"', ':')
		}
		entries[j] = e
	}

	plan := &entityFieldPlan{entries: entries}
	actual, _ := entityFieldPlanCache.LoadOrStore(key, plan)
	return actual.(*entityFieldPlan) //nolint:errcheck // value type guaranteed by our Store calls
}
