package response

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/etag"
	internalMetadata "github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/preference"
	"github.com/nlstn/go-odata/internal/query"
)

// The direct struct writer is the write-side twin of internal/fastscan: it emits
// a collection of entity structs straight into the response buffer from a cached
// per-(type, metadata-level) field plan, without materializing a per-entity
// OrderedMap or map[string]interface{} first. This eliminates the per-field
// interface{} boxing and the hashed map insert + key-slice append that
// processStructEntityOrderedInto pays for every field of every row.
//
// It is deliberately scoped to the shapes it can reproduce byte-for-byte:
//   - the collection is a slice of structs (or pointers to structs), i.e. not the
//     map results produced by $apply / $compute,
//   - no $expand is requested (expanded navigation values keep the OrderedMap
//     path, which handles truncation/count/nextLink and nested annotations),
//   - no per-item post-processing that rewrites the OrderedMap slice is active
//     (Prefer: omit-values, and $index annotations).
//
// Anything outside that set falls back to the existing addNavigationLinks /
// OrderedMap path, so output and behavior are unchanged for those requests.

// fastEntityContext carries the request-static parameters the direct writer needs.
// Everything in it is constant across the rows of a single response.
type fastEntityContext struct {
	baseURL       string
	entitySetName string
	metadataLevel string
	metadata      EntityMetadataProvider
	fullMetadata  *internalMetadata.EntityMetadata
	// selectedNavProps drives which navigation links appear under minimal metadata.
	selectedNavProps []string
	// expandOptions holds the $expand tree. Expanded navigation properties emit
	// their (possibly nested-transformed) value plus @odata.count/@odata.nextLink;
	// unexpanded ones fall back to navigation-link behavior.
	expandOptions    []query.ExpandOption
	annotationFilter *string
	// selectedSet, when non-nil, restricts emitted structural properties to the
	// named set (matching either the Go field name or the JSON name) plus key
	// properties. nil means "emit all structural properties" (no $select).
	selectedSet map[string]struct{}
	// keySet holds the Go names and JSON names of key properties, which are always
	// emitted even under $select (mirroring query.ApplySelect).
	keySet map[string]struct{}
}

// canFastWriteCollection reports whether data is a slice of structs the direct
// writer can serialize, and returns the slice reflect.Value. The second result is
// false when the caller must use the OrderedMap fallback path. $expand is handled
// by the fast path (expanded values are emitted via the shared JSON path), so
// $expand requests whose results remain structs stay on the fast path; only when
// an upstream stage converted the results to maps (e.g. $expand combined with a
// projecting $select) does the element check below route them to the fallback.
func canFastWriteCollection(data interface{}, fullMetadata *internalMetadata.EntityMetadata) (reflect.Value, bool) {
	if fullMetadata == nil {
		return reflect.Value{}, false
	}
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return reflect.Value{}, false
	}
	elem := v.Type().Elem()
	for elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	return v, true
}

// buildSelectedSet converts a $select list into the membership set used by the
// direct writer, or returns nil when every structural property should be emitted
// (no $select, or a wildcard '*' select). Tokens are matched later against both
// the Go field name and the JSON name, so both forms are stored verbatim.
func buildSelectedSet(selectedProps []string) map[string]struct{} {
	if len(selectedProps) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(selectedProps))
	for _, p := range selectedProps {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "*" {
			return nil // wildcard: emit everything, same as no $select
		}
		set[p] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

// buildKeySet returns the set of key property Go names and JSON names.
func buildKeySet(md EntityMetadataProvider) map[string]struct{} {
	keyProps := md.GetKeyProperties()
	set := make(map[string]struct{}, len(keyProps)*2)
	for i := range keyProps {
		set[keyProps[i].Name] = struct{}{}
		if keyProps[i].JsonName != "" {
			set[keyProps[i].JsonName] = struct{}{}
		}
	}
	return set
}

// writeFastCollection serializes an entire collection response (envelope plus
// entity rows) directly into buf. It assumes canFastWriteCollection returned true
// for data. The envelope keys are written in the OData-mandated order
// (@odata.context, @odata.count, @odata.nextLink, @odata.deltaLink, value),
// matching writeODataCollectionWithNavigationResponse's OrderedMap envelope.
func writeFastCollection(buf *bytes.Buffer, slice reflect.Value, ctx *fastEntityContext, contextURL string, count *int64, nextLink, deltaLink *string) error {
	var enc *json.Encoder

	buf.WriteByte('{')
	first := true
	writeEnvelopeKey := func(key string) {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		writeJSONKey(buf, key)
		buf.WriteByte(':')
	}

	if contextURL != "" {
		writeEnvelopeKey("@odata.context")
		if err := writeJSONString(buf, contextURL, &enc); err != nil {
			return err
		}
	}
	if count != nil {
		writeEnvelopeKey("@odata.count")
		writeInt(buf, *count)
	}
	if nextLink != nil && *nextLink != "" {
		writeEnvelopeKey("@odata.nextLink")
		if err := writeJSONString(buf, *nextLink, &enc); err != nil {
			return err
		}
	}
	if deltaLink != nil && *deltaLink != "" {
		writeEnvelopeKey("@odata.deltaLink")
		if err := writeJSONString(buf, *deltaLink, &enc); err != nil {
			return err
		}
	}

	writeEnvelopeKey("value")
	buf.WriteByte('[')
	for i := 0; i < slice.Len(); i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		entity := slice.Index(i)
		for entity.Kind() == reflect.Ptr {
			if entity.IsNil() {
				break
			}
			entity = entity.Elem()
		}
		if entity.Kind() != reflect.Struct {
			// Defensive: canFastWriteCollection guarantees struct elements, but a
			// nil pointer element serializes as JSON null just like encoding/json.
			buf.WriteString("null")
			continue
		}
		if err := writeFastEntity(buf, entity, ctx, &enc); err != nil {
			return err
		}
	}
	buf.WriteByte(']')

	buf.WriteByte('}')
	return nil
}

// writeFastEntity writes a single entity struct as a JSON object into buf,
// reproducing the key set and order that processStructEntityOrderedInto would
// have produced via an OrderedMap. When ctx.selectedSet is non-nil the structural
// properties are projected to the selected set (plus keys), matching the map that
// query.ApplySelect would otherwise have built — see writeFastEntity's handling
// of ETag and stream properties for the select-specific gating.
func writeFastEntity(buf *bytes.Buffer, entity reflect.Value, ctx *fastEntityContext, enc **json.Encoder) error {
	t := entity.Type()
	plan := getEntityFieldPlan(ctx.fullMetadata, t)
	infos := getFieldInfos(t)
	if len(plan.entries) != len(infos) {
		// Defensive: same type always yields matching lengths.
		return writeFastEntityFallback(buf, entity, ctx)
	}

	metadataLevel := ctx.metadataLevel
	selecting := ctx.selectedSet != nil

	var keySegment string
	needsKeySegment := metadataLevel == MetadataFull || metadataLevel == MetadataMinimal
	if needsKeySegment {
		keySegment = buildKeySegmentFromEntityCached(entity, ctx.metadata)
	}

	buf.WriteByte('{')
	first := true
	writeKey := func(key string) {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		writeJSONKey(buf, key)
		buf.WriteByte(':')
	}

	// @odata.etag — present when the entity has an ETag property and metadata is
	// not "none". Under $select the ETag mirrors the map path: it is emitted only
	// when the ETag property itself survives projection (generateFromMap returns
	// "" when the field is absent from the projected map).
	if ctx.fullMetadata.ETagProperty != nil && metadataLevel != MetadataNone {
		if !selecting || ctx.isSelected(ctx.fullMetadata.ETagProperty.FieldName, ctx.fullMetadata.ETagProperty.JsonName) {
			entityInterface := addressableInterface(entity)
			if etagValue := etag.Generate(entityInterface, ctx.fullMetadata); etagValue != "" {
				writeKey("@odata.etag")
				if err := writeJSONString(buf, etagValue, enc); err != nil {
					return err
				}
			}
		}
	}

	// @odata.id — full and minimal metadata.
	if needsKeySegment && keySegment != "" {
		writeKey("@odata.id")
		var sb strings.Builder
		sb.Grow(len(ctx.baseURL) + len(ctx.entitySetName) + len(keySegment) + 3)
		sb.WriteString(ctx.baseURL)
		sb.WriteByte('/')
		sb.WriteString(ctx.entitySetName)
		sb.WriteByte('(')
		sb.WriteString(keySegment)
		sb.WriteByte(')')
		if err := writeJSONString(buf, sb.String(), enc); err != nil {
			return err
		}
	}

	// @odata.type and entity-level annotations — full metadata only.
	if metadataLevel == MetadataFull {
		writeKey("@odata.type")
		entityTypeName := getEntityTypeFromSetName(ctx.entitySetName)
		namespace := ctx.metadata.GetNamespace()
		var sb strings.Builder
		sb.Grow(1 + len(namespace) + 1 + len(entityTypeName))
		sb.WriteByte('#')
		sb.WriteString(namespace)
		sb.WriteByte('.')
		sb.WriteString(entityTypeName)
		if err := writeJSONString(buf, sb.String(), enc); err != nil {
			return err
		}

		if ctx.fullMetadata.Annotations != nil && ctx.fullMetadata.Annotations.Len() > 0 {
			for _, annotation := range ctx.fullMetadata.Annotations.Get() {
				if ctx.annotationFilter != nil && !preference.MatchesAnnotationFilter(annotation.QualifiedTerm(), *ctx.annotationFilter) {
					continue
				}
				writeKey("@" + annotation.QualifiedTerm())
				if err := encodeFallback(buf, enc, annotation.Value); err != nil {
					return err
				}
			}
		}
	}

	// Structural and navigation properties in declaration order.
	for j := range infos {
		e := &plan.entries[j]
		if e.skip {
			continue
		}
		info := &infos[j]

		if e.navProp != nil && e.navProp.IsNavigationProp {
			// Expanded navigation property: emit its value (with nested
			// $select/$expand/$ref/$count already applied by ApplyExpandOptionToValue)
			// plus @odata.count and @odata.nextLink, mirroring the expand branch of
			// processNavigationPropertyOrderedWithMetadata.
			if expandOpt := query.FindExpandOption(ctx.expandOptions, e.navProp.Name, e.navProp.JsonName); expandOpt != nil {
				if err := ctx.writeExpandedNavigation(buf, entity.Field(j), e.navProp, info.JsonName, expandOpt, keySegment, writeKey, enc); err != nil {
					return err
				}
				continue
			}
			// Unexpanded navigation property: a navigation link only (full metadata,
			// or minimal metadata when the property is selected for links) — mirroring
			// the non-expand branch of processNavigationPropertyOrderedWithMetadata.
			emitLink := metadataLevel == MetadataFull ||
				(metadataLevel == MetadataMinimal && isPropertySelectedForLinks(*e.navProp, ctx.selectedNavProps))
			if emitLink && keySegment != "" {
				writeKey(info.JsonName + "@odata.navigationLink")
				navLink := ctx.baseURL + "/" + ctx.entitySetName + "(" + keySegment + ")/" + e.navProp.JsonName
				if err := writeJSONString(buf, navLink, enc); err != nil {
					return err
				}
			}
			continue
		}

		if selecting && !ctx.isSelected(e.goName, info.JsonName) {
			continue
		}

		// Stream properties are emitted as annotations below, never as inline values.
		if e.fullProp != nil && e.fullProp.IsStream {
			continue
		}

		// Property-level annotations precede the value under full metadata.
		if metadataLevel == MetadataFull && e.fullProp != nil && e.fullProp.Annotations != nil && e.fullProp.Annotations.Len() > 0 {
			for _, annotation := range e.fullProp.Annotations.Get() {
				if ctx.annotationFilter != nil && !preference.MatchesAnnotationFilter(annotation.QualifiedTerm(), *ctx.annotationFilter) {
					continue
				}
				writeKey(info.JsonName + "@" + annotation.QualifiedTerm())
				if err := encodeFallback(buf, enc, annotation.Value); err != nil {
					return err
				}
			}
		}

		writeKey(info.JsonName)
		if err := writeFastFieldValue(buf, entity.Field(j), e, enc); err != nil {
			return err
		}
	}

	// Named stream property annotations (OData §8.8). The map path ($select) never
	// emits these, so they are gated to the non-select case to preserve output.
	if !selecting && metadataLevel != MetadataNone && keySegment != "" {
		for i := range ctx.fullMetadata.StreamProperties {
			streamProp := &ctx.fullMetadata.StreamProperties[i]
			entityURL := ctx.baseURL + "/" + ctx.entitySetName + "(" + keySegment + ")"
			writeKey(streamProp.JsonName + "@odata.mediaReadLink")
			if err := writeJSONString(buf, entityURL+"/"+streamProp.JsonName+"/$value", enc); err != nil {
				return err
			}
			if streamProp.StreamContentTypeField != "" {
				ctField := entity.FieldByName(streamProp.StreamContentTypeField)
				if ctField.IsValid() && ctField.Kind() == reflect.String {
					if ct := ctField.String(); ct != "" {
						writeKey(streamProp.JsonName + "@odata.mediaContentType")
						if err := writeJSONString(buf, ct, enc); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	buf.WriteByte('}')
	return nil
}

// writeExpandedNavigation writes an expanded navigation property: its @odata.count
// (when $count was requested), its @odata.nextLink (when a $top-limited collection
// was truncated), and its value. It reproduces the expand branch of
// processNavigationPropertyOrderedWithMetadata exactly, including the key order
// (count, nextLink, value). The value — with nested $select/$expand/$ref/$count
// already applied by ApplyExpandOptionToValue — is serialized through encodeFallback,
// which yields the same bytes the OrderedMap path's marshalTo produces for it.
func (ctx *fastEntityContext) writeExpandedNavigation(buf *bytes.Buffer, fieldValue reflect.Value, navProp *PropertyMetadata, jsonName string, expandOpt *query.ExpandOption, keySegment string, writeKey func(string), enc **json.Encoder) error {
	truncated := false
	if expandOpt.Top != nil && navProp.NavigationIsArray {
		fieldValue, truncated = TruncateExpandedCollectionToTop(fieldValue, *expandOpt.Top)
	}

	updatedValue := fieldValue.Interface()
	var count *int
	if targetMetadata, err := ctx.fullMetadata.ResolveNavigationTarget(navProp.Name); err == nil {
		updatedValue, count = ApplyExpandOptionToValue(updatedValue, expandOpt, targetMetadata)
	}

	if count != nil {
		writeKey(jsonName + "@odata.count")
		writeInt(buf, int64(*count))
	}

	if truncated && keySegment != "" {
		writeKey(jsonName + "@odata.nextLink")
		nextLink := BuildExpandedCollectionNextLink(ctx.baseURL, ctx.entitySetName, keySegment, jsonName, expandOpt)
		if err := writeJSONString(buf, nextLink, enc); err != nil {
			return err
		}
	}

	writeKey(jsonName)
	return encodeFallback(buf, enc, updatedValue)
}

// isSelected reports whether a property (by Go name or JSON name) survives $select
// projection: it is either in the selected set or a key property.
func (ctx *fastEntityContext) isSelected(goName, jsonName string) bool {
	if _, ok := ctx.selectedSet[goName]; ok {
		return true
	}
	if _, ok := ctx.selectedSet[jsonName]; ok {
		return true
	}
	if _, ok := ctx.keySet[goName]; ok {
		return true
	}
	if _, ok := ctx.keySet[jsonName]; ok {
		return true
	}
	return false
}

// writeFastFieldValue writes a single structural field value into buf. Enum and
// binary fields are routed through the same boxed conversion the OrderedMap path
// used (enumOrRaw / EncodeEdmBinary); every other kind is written directly from
// the reflect.Value via writeComplexField, avoiding an interface{} allocation.
func writeFastFieldValue(buf *bytes.Buffer, fv reflect.Value, e *entityFieldEntry, enc **json.Encoder) error {
	if e.isEnum {
		return writeBoxedValue(buf, enumOrRaw(fv, e.fullProp), enc)
	}
	if e.isBinary {
		return writeBoxedValue(buf, EncodeEdmBinary(fv.Interface()), enc)
	}
	return writeComplexField(buf, fv, enc)
}

// writeBoxedValue writes an already-boxed scalar value (from enum/binary
// conversion) using the same primitives the OrderedMap path uses, so the bytes
// are identical. Enum conversion yields a string; binary yields a base64url
// string or nil.
func writeBoxedValue(buf *bytes.Buffer, value interface{}, enc **json.Encoder) error {
	switch v := value.(type) {
	case string:
		return writeJSONString(buf, v, enc)
	case nil:
		buf.WriteString("null")
		return nil
	default:
		return encodeFallback(buf, enc, v)
	}
}

// addressableInterface returns an interface{} for the entity suitable for
// etag.Generate, preferring an addressable pointer so pointer-receiver reflection
// works, exactly as processStructEntityOrderedInto did.
func addressableInterface(entity reflect.Value) interface{} {
	if entity.Kind() == reflect.Ptr {
		return entity.Interface()
	}
	if entity.CanAddr() {
		return entity.Addr().Interface()
	}
	return entity.Interface()
}

// writeFastEntityFallback serializes one entity through the existing OrderedMap
// path when the direct writer cannot proceed (a defensive path that should not
// normally be reached). It preserves output by delegating to the same helper the
// slow path uses.
func writeFastEntityFallback(buf *bytes.Buffer, entity reflect.Value, ctx *fastEntityContext) error {
	om := AcquireOrderedMapWithCapacity(entity.NumField() + 3)
	processStructEntityOrderedInto(om, entity, ctx.metadata, nil, ctx.selectedNavProps, ctx.baseURL, ctx.entitySetName, ctx.metadataLevel, ctx.fullMetadata, ctx.annotationFilter)
	err := om.marshalTo(buf)
	om.Release()
	return err
}

// writeFastCollectionToResponse is the entry point used by the collection writer.
// It builds the context, serializes into a pooled buffer, and writes the result
// (or Content-Length for HEAD). It returns the buffer's byte length and any error.
func writeFastCollectionToResponse(w http.ResponseWriter, r *http.Request, slice reflect.Value, ctx *fastEntityContext, contextURL string, count *int64, nextLink, deltaLink *string) error {
	buf := bufferPool.Get().(*bytes.Buffer) //nolint:errcheck // sync.Pool.Get() doesn't return error
	buf.Reset()
	defer releasePooledBuffer(buf)

	if err := writeFastCollection(buf, slice, ctx, contextURL, count, nextLink, deltaLink); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json;odata.metadata="+ctx.metadataLevel)
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return nil
	}
	_, err := w.Write(buf.Bytes())
	return err
}
