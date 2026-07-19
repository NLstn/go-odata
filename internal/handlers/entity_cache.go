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

// EntityCacheNormalizeFunc returns a cache.NormalizeFunc that precomputes,
// once per entity when a snapshot is built (or rebuilt on refresh), the
// comparison-ready value of every scalar property the in-memory filter/sort
// evaluator can compare (see isCacheComparableType). Each entity's result
// slice is indexed by that property's position in entityMeta.Properties,
// matching resolveScalarPropertyIndex, so a filter/sort comparison reads a
// precomputed value directly out of the snapshot instead of re-deriving and
// re-boxing it via reflection on every comparison of every request.
func EntityCacheNormalizeFunc(entityMeta *metadata.EntityMetadata) cache.NormalizeFunc {
	props := entityMeta.Properties
	return func(entity reflect.Value) []interface{} {
		for entity.Kind() == reflect.Ptr {
			if entity.IsNil() {
				return nil
			}
			entity = entity.Elem()
		}
		norm := make([]interface{}, len(props))
		for i := range props {
			p := &props[i]
			if p.IsNavigationProp || p.IsComplexType || p.IsEnum || !isCacheComparableType(p.Type) {
				continue
			}
			norm[i] = normalizeCacheScalar(entityFieldValue(entity, p))
		}
		return norm
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
	_, prop, ok := h.resolveScalarPropertyIndex(name)
	return prop, ok
}

// resolveScalarPropertyIndex is resolveScalarProperty's index-returning form.
// The index is the property's position in h.metadata.Properties, which is
// also how EntityCacheNormalizeFunc indexes its precomputed values — so
// filter/sort preparation (done once per query) can resolve a property name
// once and every subsequent per-entity comparison is a plain slice index.
func (h *EntityHandler) resolveScalarPropertyIndex(name string) (int, *metadata.PropertyMetadata, bool) {
	if name == "" || strings.Contains(name, "/") {
		return 0, nil, false
	}
	for i := range h.metadata.Properties {
		p := &h.metadata.Properties[i]
		if p.IsNavigationProp || p.IsComplexType || p.IsEnum {
			continue
		}
		if strings.EqualFold(p.JsonName, name) || strings.EqualFold(p.Name, name) {
			if isCacheComparableType(p.Type) {
				return i, p, true
			}
			return 0, nil, false
		}
	}
	return 0, nil, false
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
func entityFieldValue(entity reflect.Value, prop *metadata.PropertyMetadata) interface{} {
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

// preparedFilterNode is a query.FilterExpression with its property lookup and
// literal value(s) already resolved: the property name search
// (resolveScalarPropertyIndex) and normalizeCacheScalar(expr.Value) both run
// once here, in prepareFilter, rather than being repeated for every entity
// evalPreparedFilter is asked to test. Evaluating an entity against a
// prepared tree is then plain array indexing and comparisons — no
// reflection or re-boxing.
type preparedFilterNode struct {
	left, right *preparedFilterNode
	logical     query.LogicalOperator
	isNot       bool

	// Leaf fields (left == nil && right == nil):
	propIndex int
	operator  query.FilterOperator
	value     interface{}   // normalizeCacheScalar(expr.Value), precomputed once
	values    []interface{} // OpIn only: each element pre-normalized once
}

// prepareFilter resolves expr into a preparedFilterNode once per query. It
// assumes filterSupported(expr) already returned true, so every property and
// operator here is one evalPreparedFilter can evaluate.
func (h *EntityHandler) prepareFilter(expr *query.FilterExpression) *preparedFilterNode {
	if expr == nil {
		return nil
	}

	if expr.Left != nil && expr.Right != nil {
		return &preparedFilterNode{
			left:    h.prepareFilter(expr.Left),
			right:   h.prepareFilter(expr.Right),
			logical: expr.Logical,
			isNot:   expr.IsNot,
		}
	}

	idx, _, ok := h.resolveScalarPropertyIndex(expr.Property)
	if !ok {
		// filterSupported already guarantees this is resolvable; treat an
		// unexpected miss as "matches nothing" rather than panicking.
		return &preparedFilterNode{propIndex: -1, operator: expr.Operator, isNot: expr.IsNot}
	}
	node := &preparedFilterNode{propIndex: idx, operator: expr.Operator, isNot: expr.IsNot}
	if expr.Operator == query.OpIn {
		if values, ok := expr.Value.([]interface{}); ok {
			node.values = make([]interface{}, len(values))
			for i, v := range values {
				node.values[i] = normalizeCacheScalar(v)
			}
		}
		return node
	}
	node.value = normalizeCacheScalar(expr.Value)
	return node
}

// evalPreparedFilter evaluates a prepared filter tree against one entity's
// precomputed comparison values (see EntityCacheNormalizeFunc), with no
// reflection or normalization on the hot path.
func evalPreparedFilter(norm []interface{}, node *preparedFilterNode) bool {
	if node == nil {
		return true
	}

	if node.left != nil && node.right != nil {
		left := evalPreparedFilter(norm, node.left)
		right := evalPreparedFilter(norm, node.right)
		var result bool
		switch node.logical {
		case query.LogicalAnd:
			result = left && right
		case query.LogicalOr:
			result = left || right
		}
		if node.isNot {
			return !result
		}
		return result
	}

	if node.propIndex < 0 {
		return false
	}
	left := norm[node.propIndex]

	var result bool
	if node.operator == query.OpIn {
		for _, v := range node.values {
			if evaluateFilterComparison(left, query.OpEqual, v) {
				result = true
				break
			}
		}
	} else {
		result = evaluateFilterComparison(left, node.operator, node.value)
	}
	if node.isNot {
		return !result
	}
	return result
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

// preparedOrderByItem is a query.OrderByItem with its property lookup already
// resolved, once per query, instead of on every sort comparison.
type preparedOrderByItem struct {
	propIndex  int
	descending bool
}

// prepareOrderBy resolves orderBy into preparedOrderByItems once per query.
// Items whose property can't be resolved are skipped, matching
// resolveScalarProperty's contract (snapshotSupportsCollection already
// guarantees every item here is resolvable).
func (h *EntityHandler) prepareOrderBy(orderBy []query.OrderByItem) []preparedOrderByItem {
	prepared := make([]preparedOrderByItem, 0, len(orderBy))
	for _, item := range orderBy {
		idx, _, ok := h.resolveScalarPropertyIndex(item.Property)
		if !ok {
			continue
		}
		prepared = append(prepared, preparedOrderByItem{propIndex: idx, descending: item.Descending})
	}
	return prepared
}

// snapshotMatch pairs a mutable copy of one matching entity (mutable because
// downstream $expand populates navigation fields on it) with its precomputed
// comparison values, shared directly from the snapshot: sorting never needs
// to re-derive or re-normalize anything from the entity itself.
type snapshotMatch struct {
	entity reflect.Value
	norm   []interface{}
}

// sortMatches sorts matches in place by the given prepared $orderby items,
// comparing precomputed values instead of re-deriving them from the entity.
func sortMatches(matches []snapshotMatch, orderBy []preparedOrderByItem) {
	if len(orderBy) == 0 || len(matches) < 2 {
		return
	}
	sort.SliceStable(matches, func(i, j int) bool {
		for _, item := range orderBy {
			lv := matches[i].norm[item.propIndex]
			rv := matches[j].norm[item.propIndex]
			cmp := compareMapValues(lv, lv != nil, rv, rv != nil)
			if cmp == 0 {
				continue
			}
			if item.descending {
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
	prepared := h.prepareFilter(queryOptions.Filter)

	matches := make([]snapshotMatch, 0)
	for i := 0; i < snap.Len(); i++ {
		norm := snap.Normalized(i)
		if prepared == nil || evalPreparedFilter(norm, prepared) {
			entity := snap.At(i)
			// Copy the struct out of the snapshot's backing array so that
			// downstream $expand (which populates navigation fields) never
			// mutates the shared, immutable snapshot. norm is read-only and
			// safe to share as-is.
			cp := reflect.New(h.metadata.EntityType).Elem()
			cp.Set(entity)
			matches = append(matches, snapshotMatch{entity: cp, norm: norm})
		}
	}

	orderBy := queryOptions.OrderBy
	if len(orderBy) == 0 {
		orderBy = h.defaultKeyOrderBy()
	}
	sortMatches(matches, h.prepareOrderBy(orderBy))

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
	for i, m := range matches {
		out.Index(i).Set(m.entity)
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
	if len(queryOptions.Select) > 0 && !query.CanDeferSelectProjection(queryOptions.Select, queryOptions.Expand, h.metadata) {
		// See collection_read.go: flat $select projection is deferred to the response
		// serializer, which emits only the selected fields directly from the structs.
		sliceValue = query.ApplySelect(sliceValue, queryOptions.Select, h.metadata, queryOptions.Expand)
	}

	return sliceValue, nil
}

// countSnapshot counts entities in the snapshot matching filter.
func (h *EntityHandler) countSnapshot(snap *cache.Snapshot, filter *query.FilterExpression) int64 {
	if filter == nil {
		return int64(snap.Len())
	}
	prepared := h.prepareFilter(filter)
	var count int64
	for i := 0; i < snap.Len(); i++ {
		if evalPreparedFilter(snap.Normalized(i), prepared) {
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
