package handlers

import (
	"context"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/nlstn/go-odata/internal/cache"
	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
	"gorm.io/gorm"
)

// keyComponentSeparator joins the individual key-property components of a
// composite key into a single canonical string. It is a byte that cannot appear
// in a rendered scalar value, so distinct key tuples never collide.
const keyComponentSeparator = "\x00"

// EntityCacheKeyFunc returns a cache.KeyFunc that derives the canonical key of an
// entity from its key properties. The result must match canonicalKeyFromValues so
// that request-time key lookups resolve against the snapshot's index.
func EntityCacheKeyFunc(entityMeta *metadata.EntityMetadata) cache.KeyFunc {
	keyProps := entityMeta.KeyProperties
	return func(entity reflect.Value) string {
		for entity.Kind() == reflect.Ptr {
			if entity.IsNil() {
				return ""
			}
			entity = entity.Elem()
		}
		parts := make([]string, len(keyProps))
		for i, kp := range keyProps {
			field := entity.FieldByName(kp.FieldName)
			if field.IsValid() {
				parts[i] = canonicalKeyComponent(field.Interface())
			}
		}
		return strings.Join(parts, keyComponentSeparator)
	}
}

// canonicalKeyFromValues builds the canonical key string from a parsed key-value
// map (keyed by JSON/OData property name, as produced by parseEntityKeyValues).
// It returns false when a key property is missing from the map.
func (h *EntityHandler) canonicalKeyFromValues(vals map[string]interface{}) (string, bool) {
	if len(vals) == 0 {
		return "", false
	}
	parts := make([]string, len(h.metadata.KeyProperties))
	for i, kp := range h.metadata.KeyProperties {
		v, ok := vals[kp.JsonName]
		if !ok {
			v, ok = vals[kp.Name]
		}
		if !ok {
			return "", false
		}
		parts[i] = canonicalKeyComponent(v)
	}
	return strings.Join(parts, keyComponentSeparator), true
}

// canonicalKeyComponent renders a single scalar key value to a stable string.
// It dereferences pointers and normalises numeric types so that the entity-side
// and request-side representations of the same key are byte-identical regardless
// of the concrete Go type each side happens to hold.
func canonicalKeyComponent(v interface{}) string {
	if v == nil {
		return "\x01nil"
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return "\x01nil"
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	case reflect.String:
		return rv.String()
	default:
		return "\x02" + rv.String()
	}
}

// cacheSnapshot returns the current entity snapshot, refreshing it from the
// primary database when it is missing or expired. It returns false when caching
// is disabled or a refresh fails, so callers transparently fall back to the
// primary database.
func (h *EntityHandler) cacheSnapshot(ctx context.Context) (*cache.Snapshot, bool) {
	if h.entityCache == nil {
		return nil, false
	}
	if snap, ok := h.entityCache.Current(); ok {
		return snap, true
	}
	if err := h.entityCache.Refresh(h.db.WithContext(ctx)); err != nil {
		h.logger.Warn("Failed to refresh entity cache, falling back to primary database",
			"entitySet", h.metadata.EntitySetName,
			"error", err)
		return nil, false
	}
	return h.entityCache.Current()
}

// resolveScalarProperty maps an OData property name to its metadata, but only
// when the property is a plain scalar the in-memory evaluator can compare
// faithfully. Navigation properties, complex types, enums, and non-scalar Go
// kinds (time.Time, []byte, …) return false so the query falls back to SQL.
func (h *EntityHandler) resolveScalarProperty(name string) (*metadata.PropertyMetadata, bool) {
	if name == "" || strings.Contains(name, "/") {
		return nil, false
	}
	for i := range h.metadata.Properties {
		p := &h.metadata.Properties[i]
		if p.IsNavigationProp || p.IsComplexType || p.IsEnum {
			continue
		}
		if strings.EqualFold(p.JsonName, name) || strings.EqualFold(p.Name, name) {
			if isCacheComparableType(p.Type) {
				return p, true
			}
			return nil, false
		}
	}
	return nil, false
}

// isCacheComparableType reports whether values of t can be compared by the
// in-memory evaluator. Pointers are unwrapped to their element type.
func isCacheComparableType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.String:
		return true
	default:
		return false
	}
}

// entityFieldValue returns the value of prop on entity, dereferencing pointers.
// A nil pointer field yields a nil interface.
func (h *EntityHandler) entityFieldValue(entity reflect.Value, prop *metadata.PropertyMetadata) interface{} {
	for entity.Kind() == reflect.Ptr {
		if entity.IsNil() {
			return nil
		}
		entity = entity.Elem()
	}
	field := entity.FieldByName(prop.FieldName)
	if !field.IsValid() {
		return nil
	}
	for field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return nil
		}
		field = field.Elem()
	}
	if !field.CanInterface() {
		return nil
	}
	return field.Interface()
}

// normalizeCacheScalar reduces a value to one of the concrete types the shared
// comparison helpers (evaluateFilterComparison / compareMapValues) understand,
// so named numeric types compare consistently on both sides of an operator.
func normalizeCacheScalar(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	case reflect.Bool:
		return rv.Bool()
	case reflect.String:
		return rv.String()
	default:
		return rv.Interface()
	}
}

// filterSupported reports whether the entire filter tree is within the subset the
// in-memory evaluator handles correctly. It is deliberately conservative: any
// operator, function, or property it does not explicitly support forces a
// fallback to the SQL path rather than risk returning wrong results.
func (h *EntityHandler) filterSupported(expr *query.FilterExpression) bool {
	if expr == nil {
		return true
	}

	if expr.Left != nil || expr.Right != nil {
		// A logical combination requires both branches and a known connective.
		if expr.Left == nil || expr.Right == nil {
			return false
		}
		if expr.Logical != query.LogicalAnd && expr.Logical != query.LogicalOr {
			return false
		}
		return h.filterSupported(expr.Left) && h.filterSupported(expr.Right)
	}

	prop, ok := h.resolveScalarProperty(expr.Property)
	if !ok {
		return false
	}

	switch expr.Operator {
	case query.OpEqual, query.OpNotEqual,
		query.OpGreaterThan, query.OpGreaterThanOrEqual,
		query.OpLessThan, query.OpLessThanOrEqual:
		return true
	case query.OpContains, query.OpStartsWith, query.OpEndsWith:
		// Ordinal string functions only make sense for string properties.
		return prop.Type != nil && underlyingKind(prop.Type) == reflect.String
	case query.OpIn:
		_, isSlice := expr.Value.([]interface{})
		return isSlice
	default:
		return false
	}
}

func underlyingKind(t reflect.Type) reflect.Kind {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind()
}

// evalEntityFilter evaluates a supported filter expression against a single
// entity. It assumes filterSupported(expr) already returned true.
func (h *EntityHandler) evalEntityFilter(entity reflect.Value, expr *query.FilterExpression) bool {
	if expr == nil {
		return true
	}

	if expr.Left != nil && expr.Right != nil {
		left := h.evalEntityFilter(entity, expr.Left)
		right := h.evalEntityFilter(entity, expr.Right)
		var result bool
		switch expr.Logical {
		case query.LogicalAnd:
			result = left && right
		case query.LogicalOr:
			result = left || right
		}
		if expr.IsNot {
			return !result
		}
		return result
	}

	prop, ok := h.resolveScalarProperty(expr.Property)
	if !ok {
		return false
	}
	left := h.entityFieldValue(entity, prop)

	result := h.evalLeafComparison(left, expr.Operator, expr.Value)
	if expr.IsNot {
		return !result
	}
	return result
}

func (h *EntityHandler) evalLeafComparison(left interface{}, op query.FilterOperator, right interface{}) bool {
	if op == query.OpIn {
		values, ok := right.([]interface{})
		if !ok {
			return false
		}
		normLeft := normalizeCacheScalar(left)
		for _, item := range values {
			if evaluateFilterComparison(normLeft, query.OpEqual, normalizeCacheScalar(item)) {
				return true
			}
		}
		return false
	}
	return evaluateFilterComparison(normalizeCacheScalar(left), op, normalizeCacheScalar(right))
}

// snapshotSupportsCollection reports whether a collection query can be served
// entirely from the in-memory snapshot. $select and $expand are permitted
// because they are resolved downstream exactly as on the SQL path ($expand still
// hits the primary database for related rows).
func (h *EntityHandler) snapshotSupportsCollection(queryOptions *query.QueryOptions) bool {
	if queryOptions == nil {
		return false
	}
	if len(queryOptions.Apply) > 0 || queryOptions.Compute != nil {
		return false
	}
	if queryOptions.Search != "" || queryOptions.SkipToken != nil || queryOptions.DeltaToken != nil {
		return false
	}
	if query.ShouldUseMapResults(queryOptions) {
		return false
	}
	if !h.filterSupported(queryOptions.Filter) {
		return false
	}
	for _, ob := range queryOptions.OrderBy {
		if _, ok := h.resolveScalarProperty(ob.Property); !ok {
			return false
		}
	}
	return true
}

// snapshotSupportsCount reports whether a $count can be computed from the
// snapshot. It is looser than the collection check because ordering and paging
// do not affect a count.
func (h *EntityHandler) snapshotSupportsCount(queryOptions *query.QueryOptions) bool {
	if queryOptions == nil {
		return true
	}
	if len(queryOptions.Apply) > 0 || queryOptions.Search != "" {
		return false
	}
	return h.filterSupported(queryOptions.Filter)
}

// defaultKeyOrderBy returns an $orderby over the key properties, used to give a
// stable, deterministic result order when the request specifies none — mirroring
// the implicit primary-key ordering the SQL path applies.
func (h *EntityHandler) defaultKeyOrderBy() []query.OrderByItem {
	items := make([]query.OrderByItem, 0, len(h.metadata.KeyProperties))
	for _, kp := range h.metadata.KeyProperties {
		items = append(items, query.OrderByItem{Property: kp.JsonName})
	}
	return items
}

// sortEntityValues sorts copies in place by the given $orderby items.
func (h *EntityHandler) sortEntityValues(values []reflect.Value, orderBy []query.OrderByItem) {
	if len(orderBy) == 0 || len(values) < 2 {
		return
	}
	sort.SliceStable(values, func(i, j int) bool {
		for _, item := range orderBy {
			prop, ok := h.resolveScalarProperty(item.Property)
			if !ok {
				continue
			}
			lv := h.entityFieldValue(values[i], prop)
			rv := h.entityFieldValue(values[j], prop)
			cmp := compareMapValues(normalizeCacheScalar(lv), lv != nil, normalizeCacheScalar(rv), rv != nil)
			if cmp == 0 {
				continue
			}
			if item.Descending {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

// queryCollectionSnapshot filters, orders, and pages the snapshot in memory,
// returning a *[]T (matching the pointer-to-slice shape the SQL path produces)
// so downstream $expand/$select post-processing is identical.
func (h *EntityHandler) queryCollectionSnapshot(snap *cache.Snapshot, queryOptions, modifiedOptions *query.QueryOptions) interface{} {
	filter := queryOptions.Filter

	matches := make([]reflect.Value, 0)
	for i := 0; i < snap.Len(); i++ {
		entity := snap.At(i)
		if filter == nil || h.evalEntityFilter(entity, filter) {
			// Copy the struct out of the snapshot's backing array so that
			// downstream $expand (which populates navigation fields) never
			// mutates the shared, immutable snapshot.
			cp := reflect.New(h.metadata.EntityType).Elem()
			cp.Set(entity)
			matches = append(matches, cp)
		}
	}

	orderBy := queryOptions.OrderBy
	if len(orderBy) == 0 {
		orderBy = h.defaultKeyOrderBy()
	}
	h.sortEntityValues(matches, orderBy)

	// Apply $skip then $top (modifiedOptions.Top is already top+1 so the caller
	// can detect whether a next page exists).
	if queryOptions.Skip != nil {
		skip := *queryOptions.Skip
		if skip < 0 {
			skip = 0
		}
		if skip >= len(matches) {
			matches = matches[:0]
		} else {
			matches = matches[skip:]
		}
	}
	if modifiedOptions.Top != nil {
		top := *modifiedOptions.Top
		if top < 0 {
			top = 0
		}
		if top < len(matches) {
			matches = matches[:top]
		}
	}

	sliceType := reflect.SliceOf(h.metadata.EntityType)
	out := reflect.MakeSlice(sliceType, len(matches), len(matches))
	for i, e := range matches {
		out.Index(i).Set(e)
	}
	resultsPtr := reflect.New(sliceType)
	resultsPtr.Elem().Set(out)
	return resultsPtr.Interface()
}

// postProcessCachedCollection applies the shared downstream steps ($expand and
// $select) to a snapshot result set. $expand is resolved against the primary
// database, exactly as on the SQL path.
func (h *EntityHandler) postProcessCachedCollection(ctx context.Context, resultsPtr interface{}, queryOptions *query.QueryOptions) (interface{}, error) {
	if len(queryOptions.Expand) > 0 {
		baseDB := h.db.WithContext(ctx)
		if err := query.ApplyPerParentExpand(baseDB, resultsPtr, queryOptions.Expand, h.metadata); err != nil {
			return nil, err
		}
	}

	sliceValue := reflect.ValueOf(resultsPtr).Elem().Interface()

	if len(queryOptions.Expand) > 0 {
		sliceValue = query.ApplyExpandComputeToResults(sliceValue, queryOptions.Expand)
	}
	if len(queryOptions.Select) > 0 {
		sliceValue = query.ApplySelect(sliceValue, queryOptions.Select, h.metadata, queryOptions.Expand)
	}

	return sliceValue, nil
}

// countSnapshot counts entities in the snapshot matching filter.
func (h *EntityHandler) countSnapshot(snap *cache.Snapshot, filter *query.FilterExpression) int64 {
	if filter == nil {
		return int64(snap.Len())
	}
	var count int64
	for i := 0; i < snap.Len(); i++ {
		if h.evalEntityFilter(snap.At(i), filter) {
			count++
		}
	}
	return count
}

// fetchEntityByKeyFromSnapshot serves a key read from the snapshot. The returned
// served flag is true when the snapshot is authoritative for this key (found or
// not); callers only fall back to the primary database when served is false.
func (h *EntityHandler) fetchEntityByKeyFromSnapshot(ctx context.Context, entityKey string, queryOptions *query.QueryOptions, scopes []func(*gorm.DB) *gorm.DB) (interface{}, bool, bool, error) {
	if len(scopes) > 0 {
		return nil, false, false, nil
	}
	snap, ok := h.cacheSnapshot(ctx)
	if !ok {
		return nil, false, false, nil
	}

	keyVals := parseEntityKeyValues(entityKey, h.metadata.KeyProperties)
	canonical, kok := h.canonicalKeyFromValues(keyVals)
	if !kok {
		return nil, false, false, nil
	}

	entity, found := snap.Lookup(canonical)
	if !found {
		// The snapshot is authoritative (no scopes): the entity does not exist.
		return nil, false, true, nil
	}

	result := reflect.New(h.metadata.EntityType)
	result.Elem().Set(entity)
	resultIface := result.Interface()

	if len(queryOptions.Expand) > 0 {
		baseDB := h.db.WithContext(ctx)
		if err := query.ApplyPerParentExpand(baseDB, resultIface, queryOptions.Expand, h.metadata); err != nil {
			return nil, false, true, err
		}
	}

	return resultIface, true, true, nil
}
