// Package fastscan provides an optimized replacement for GORM's Find when
// reading pages of flat entity structs. GORM still builds the SQL (so dialect
// quoting, soft deletes, clauses, and registered callbacks behave exactly as
// with Find), but the rows are scanned with a compiled per-schema plan that
// passes field addresses straight to database/sql instead of going through
// GORM's per-row boxing and Set conversion machinery.
//
// Queries or destination types the plan cannot faithfully reproduce — model
// hooks, preloads, association joins, serializer fields, exotic field types —
// fall back to db.Find, so behavior is always at least as correct as GORM's.
package fastscan

import (
	"context"
	"database/sql"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// scanMode describes how a column value reaches its struct field.
type scanMode uint8

const (
	// scanDirect passes the field's address to rows.Scan. database/sql handles
	// NULL natively for pointer fields, sql.Scanner implementations, and
	// []byte, so no intermediate buffer is needed.
	scanDirect scanMode = iota
	// The buffered modes scan into a reusable sql.Null* buffer and copy the
	// value into the field afterwards, so a NULL column leaves the field at
	// its zero value — matching GORM's behavior for non-pointer fields.
	scanInt
	scanUint
	scanFloat
	scanBool
	scanString
	scanTime
)

type fieldPlan struct {
	field *schema.Field
	mode  scanMode
}

// plan is the compiled scan strategy for one GORM schema. A nil plan means
// the schema is ineligible and callers must use the GORM fallback.
type plan struct {
	fields map[string]fieldPlan // keyed by DB column name
	// bindings owns complete per-query scan binding sets. A set contains the
	// destinations slice and every sql.Null* buffer, so concurrent queries
	// never share mutable scan state.
	bindings sync.Pool
	// poisoned tracks, per DB column name, the time a scan of that column
	// last produced an error that GORM's richer value conversion might have
	// handled. A query that touches a currently-poisoned column falls back
	// to GORM instead of risking the same failure; other columns on the same
	// schema keep using the fast path. Entries expire after poisonRetryWindow
	// so a transient bad value (or a value that no longer occurs) doesn't
	// poison the column forever.
	poisoned sync.Map // db column name -> time.Time
	// poisonCount is a best-effort (may lag a just-expired entry) count of
	// entries in poisoned, used to skip the poison check entirely in the
	// common case where nothing is poisoned.
	poisonCount atomic.Int32
}

// poisonRetryWindow bounds how long a column stays poisoned before it is
// given another chance. Keeping this short means a truly incompatible column
// (e.g. legacy TEXT in an INTEGER column, or a driver without parseTime) is
// quickly re-poisoned at negligible cost, while a transient bad value doesn't
// forfeit the fast path for the life of the process. Variable (not const) so
// tests can shrink it instead of sleeping tens of seconds.
var poisonRetryWindow = 30 * time.Second

// isPoisoned reports whether column is currently poisoned, lazily expiring
// (and uncounting) the entry once poisonRetryWindow has elapsed.
func (p *plan) isPoisoned(column string) bool {
	v, ok := p.poisoned.Load(column)
	if !ok {
		return false
	}
	at, ok := v.(time.Time)
	if !ok || time.Since(at) > poisonRetryWindow {
		if p.poisoned.CompareAndDelete(column, v) {
			p.poisonCount.Add(-1)
		}
		return false
	}
	return true
}

// anyPoisoned reports whether any of columns is currently poisoned.
func (p *plan) anyPoisoned(columns []string) bool {
	if p.poisonCount.Load() <= 0 {
		return false
	}
	for _, c := range columns {
		if p.isPoisoned(c) {
			return true
		}
	}
	return false
}

// poisonColumn marks column as poisoned as of now, extending its poisoned
// window if it was already poisoned.
func (p *plan) poisonColumn(column string) {
	if _, loaded := p.poisoned.Swap(column, time.Now()); !loaded {
		p.poisonCount.Add(1)
	}
}

// poisonAll marks every column in the schema as poisoned. It is the
// conservative fallback used when a scan error can't be attributed to a
// specific column, matching the previous whole-plan-disable behavior.
func (p *plan) poisonAll() {
	for name := range p.fields {
		p.poisonColumn(name)
	}
}

// scanErrorColumnIndexRe matches the column index that database/sql's
// (*Rows).Scan embeds in conversion errors, e.g. `sql: Scan error on column
// index 1, name "amount": converting driver.Value type string ...`. This
// format has been stable in the standard library for many Go releases; if it
// ever changes, recordScanFailure falls back to poisoning every column.
var scanErrorColumnIndexRe = regexp.MustCompile(`column index (\d+)`)

// recordScanFailure poisons the column responsible for a scan error, or
// every column in the schema if the offending one can't be identified.
func (p *plan) recordScanFailure(err error, columns []string) {
	if m := scanErrorColumnIndexRe.FindStringSubmatch(err.Error()); m != nil {
		if idx, convErr := strconv.Atoi(m[1]); convErr == nil && idx >= 0 && idx < len(columns) {
			p.poisonColumn(columns[idx])
			return
		}
	}
	p.poisonAll()
}

var (
	// planCache maps *schema.Schema to *plan. GORM caches parsed schemas per
	// connection config, so the schema pointer is a stable identity that also
	// keys correctly when multiple databases (e.g. the entity cache DB) hold
	// schemas for the same Go type.
	planCache sync.Map

	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
	timeType    = reflect.TypeOf(time.Time{})
)

// Find executes the SELECT described by db and scans the result into dest,
// which must be a pointer to a slice of structs. It has db.Find semantics:
// the destination slice is reset, reused when it has capacity, and left
// non-nil even for an empty result. Like Find, it consumes db's statement.
func Find(db *gorm.DB, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Pointer || destVal.IsNil() || destVal.Elem().Kind() != reflect.Slice {
		return db.Find(dest).Error
	}
	sliceVal := destVal.Elem()
	elemType := sliceVal.Type().Elem()
	if elemType.Kind() != reflect.Struct {
		return db.Find(dest).Error
	}
	if db.DryRun {
		return db.Find(dest).Error
	}

	// Mirror what Find's Execute does: default the model to the destination,
	// then parse it so the schema (and its cached scan plan) is available.
	tx := db
	if tx.Statement.Model == nil {
		tx = tx.Model(dest)
	}
	if err := tx.Statement.Parse(tx.Statement.Model); err != nil {
		return tx.Find(dest).Error
	}
	sch := tx.Statement.Schema
	if sch == nil || sch.ModelType != elemType {
		return tx.Find(dest).Error
	}
	p := planFor(sch)
	if p == nil || !eligibleStatement(tx.Statement, sch) || p.likelyTouchesPoisoned(tx.Statement, sch) {
		return tx.Find(dest).Error
	}

	rows, err := tx.Rows()
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		if closeErr := rows.Close(); closeErr != nil {
			return closeErr
		}
		return err
	}
	if p.anyPoisoned(columns) {
		// A column this query touches failed to scan recently enough that it
		// is still poisoned; don't waste a scan attempt repeating a failure
		// we already know about. Execute resets the statement's SQL after
		// Rows, so Find rebuilds it cleanly.
		if closeErr := rows.Close(); closeErr != nil {
			return closeErr
		}
		return tx.Find(dest).Error
	}

	result, err := scanRows(tx.Statement.Context, rows, p, sliceVal, elemType, columns)
	closeErr := rows.Close()
	if err != nil {
		if scanErr, ok := err.(*rowScanError); ok {
			// A row failed to scan. If the context is gone this is just
			// cancellation; otherwise the plan mis-handled a driver value that
			// GORM's conversions might accept. Poison the offending column (or
			// every column, if it can't be identified) and re-run the query
			// through GORM, which either succeeds or reports its own
			// (equivalent) error. Execute resets the statement's SQL after
			// Rows, so Find rebuilds it cleanly.
			if ctxErr := contextErr(tx.Statement.Context); ctxErr != nil {
				return scanErr.err
			}
			p.recordScanFailure(scanErr.err, columns)
			return tx.Find(dest).Error
		}
		return err
	}
	if closeErr != nil {
		return closeErr
	}

	sliceVal.Set(result)
	tx.RowsAffected = int64(result.Len())
	return nil
}

// First executes db as a single-row query with db.First semantics: it adds
// LIMIT 1 and an ORDER BY primary key (matching gorm.First's clauses), scans
// the row into dest — a pointer to a struct — with the compiled plan, and
// returns gorm.ErrRecordNotFound when no row matches. Queries or destination
// types the plan cannot faithfully reproduce fall back to db.First, so behavior
// is always at least as correct as GORM's.
func First(db *gorm.DB, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Pointer || destVal.IsNil() || destVal.Elem().Kind() != reflect.Struct {
		return db.First(dest).Error
	}
	elemType := destVal.Elem().Type()
	// time.Time is a struct but a leaf scan target, not an entity to populate.
	if elemType == timeType {
		return db.First(dest).Error
	}
	if db.DryRun {
		return db.First(dest).Error
	}

	// Mirror what First's Execute does: default the model to the destination,
	// then parse it so the schema (and its cached scan plan) is available.
	tx := db
	if tx.Statement.Model == nil {
		tx = tx.Model(dest)
	}
	if err := tx.Statement.Parse(tx.Statement.Model); err != nil {
		return tx.First(dest).Error
	}
	sch := tx.Statement.Schema
	if sch == nil || sch.ModelType != elemType {
		return tx.First(dest).Error
	}
	p := planFor(sch)
	if p == nil || !eligibleStatement(tx.Statement, sch) || p.likelyTouchesPoisoned(tx.Statement, sch) {
		return tx.First(dest).Error
	}

	// Reproduce gorm.First's clauses so the SQL is identical: LIMIT 1 and an
	// ORDER BY primary key (the clause builder expands clause.PrimaryKey from
	// the parsed schema). A ByKey query already has a unique WHERE, but keeping
	// the same SQL preserves dialect and plan-cache behavior.
	tx = tx.Limit(1)
	if len(sch.PrimaryFields) > 0 {
		tx = tx.Order(clause.OrderByColumn{
			Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
		})
	}

	rows, err := tx.Rows()
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		if closeErr := rows.Close(); closeErr != nil {
			return closeErr
		}
		return err
	}
	if p.anyPoisoned(columns) {
		// A column this query touches failed to scan recently enough that it
		// is still poisoned; don't waste a scan attempt repeating a failure
		// we already know about.
		if closeErr := rows.Close(); closeErr != nil {
			return closeErr
		}
		return tx.First(dest).Error
	}

	found, err := scanFirstRow(tx.Statement.Context, rows, p, destVal.Elem(), columns)
	closeErr := rows.Close()
	if err != nil {
		if scanErr, ok := err.(*rowScanError); ok {
			// A row failed to scan. If the context is gone this is just
			// cancellation; otherwise the plan mis-handled a driver value that
			// GORM's conversions might accept. Poison the offending column (or
			// every column, if it can't be identified) and re-run through
			// GORM, which either succeeds or reports its own (equivalent)
			// error.
			if ctxErr := contextErr(tx.Statement.Context); ctxErr != nil {
				return scanErr.err
			}
			p.recordScanFailure(scanErr.err, columns)
			return tx.First(dest).Error
		}
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if !found {
		return gorm.ErrRecordNotFound
	}
	tx.RowsAffected = 1
	return nil
}

// planFor returns the cached scan plan for sch, or nil when the schema is
// ineligible for fast scanning.
func planFor(sch *schema.Schema) *plan {
	if cached, ok := planCache.Load(sch); ok {
		if p, ok := cached.(*plan); ok {
			return p
		}
		return nil
	}
	built := buildPlan(sch)
	// Concurrent builds produce equivalent plans; whichever lands first wins.
	actual, _ := planCache.LoadOrStore(sch, built)
	if p, ok := actual.(*plan); ok {
		return p
	}
	return nil
}

// likelyTouchesPoisoned reports whether executing stmt against sch could
// touch a currently-poisoned column, without running the query. This lets
// Find/First skip straight to the GORM fallback — as the old whole-plan
// disable did — when nothing would survive the fast path anyway, instead of
// paying for a query that anyPoisoned would just reject afterward.
func (p *plan) likelyTouchesPoisoned(stmt *gorm.Statement, sch *schema.Schema) bool {
	if p.poisonCount.Load() <= 0 {
		return false
	}
	if len(stmt.Selects) == 0 {
		// No explicit selection: GORM selects every schema column, so a
		// poisoned column (there is at least one) is definitely included.
		return true
	}
	for _, sel := range stmt.Selects {
		dbName := sel
		if field, ok := sch.FieldsByName[sel]; ok {
			dbName = field.DBName
		}
		if p.isPoisoned(dbName) {
			return true
		}
	}
	return false
}

func buildPlan(sch *schema.Schema) *plan {
	// AfterFind hooks only run through GORM's scan path.
	if sch == nil || sch.AfterFind || len(sch.FieldsByDBName) == 0 {
		return nil
	}
	fields := make(map[string]fieldPlan, len(sch.FieldsByDBName))
	for name, field := range sch.FieldsByDBName {
		// Serializer fields (gorm:"serializer:...") need GORM's decode logic.
		if field.Serializer != nil || field.ReflectValueOf == nil {
			return nil
		}
		mode, ok := fieldScanMode(field.FieldType)
		if !ok {
			return nil
		}
		fields[name] = fieldPlan{field: field, mode: mode}
	}
	p := &plan{fields: fields}
	p.bindings.New = func() interface{} { return &bindingSet{} }
	return p
}

// fieldScanMode decides how a field of the given type is scanned, or reports
// that the whole schema must fall back to GORM.
func fieldScanMode(ft reflect.Type) (scanMode, bool) {
	// sql.Scanner implementations (sql.NullString, gorm.DeletedAt, custom
	// types) handle NULL and driver value conversion themselves.
	if ft.Implements(scannerType) || reflect.PointerTo(ft).Implements(scannerType) {
		return scanDirect, true
	}
	switch ft.Kind() {
	case reflect.Pointer:
		elem := ft.Elem()
		if elem == timeType {
			return scanDirect, true
		}
		switch elem.Kind() {
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.String:
			return scanDirect, true
		case reflect.Slice:
			if elem.Elem().Kind() == reflect.Uint8 {
				return scanDirect, true
			}
		}
	case reflect.Bool:
		return scanBool, true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return scanInt, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return scanUint, true
	case reflect.Float32, reflect.Float64:
		return scanFloat, true
	case reflect.String:
		return scanString, true
	case reflect.Slice:
		if ft.Elem().Kind() == reflect.Uint8 {
			return scanDirect, true
		}
	case reflect.Struct:
		if ft == timeType {
			return scanTime, true
		}
	}
	return 0, false
}

// eligibleStatement rejects statements whose results GORM would post-process:
// preloads populate associations after scanning, and joins that reference a
// relationship make GORM scan joined columns into nested structs.
func eligibleStatement(stmt *gorm.Statement, sch *schema.Schema) bool {
	if len(stmt.Preloads) > 0 {
		return false
	}
	for _, join := range stmt.Joins {
		name := join.Name
		if idx := strings.IndexByte(name, '.'); idx > 0 {
			name = name[:idx]
		}
		if _, ok := sch.Relationships.Relations[name]; ok {
			return false
		}
	}
	return true
}

// rowScanError marks errors from rows.Scan, which trigger the GORM fallback;
// errors from the connection or result set are returned as-is.
type rowScanError struct{ err error }

func (e *rowScanError) Error() string { return e.err.Error() }
func (e *rowScanError) Unwrap() error { return e.err }

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

type directBinding struct {
	colIdx int
	field  *schema.Field
}

// bufferedBinding scans a column into a reusable sql.Null* buffer and copies
// the value into the struct field afterwards. Exactly one buffer, matching
// the mode, is non-nil.
type bufferedBinding struct {
	field     *schema.Field
	mode      scanMode
	intBuf    *sql.NullInt64
	floatBuf  *sql.NullFloat64
	boolBuf   *sql.NullBool
	stringBuf *sql.NullString
	timeBuf   *sql.NullTime
}

// bindingSet is all mutable state needed to scan one query result. It is
// acquired from a plan-local pool and returned only after rows are fully
// consumed, which makes the plan safe for concurrent readers.
type bindingSet struct {
	dests    []interface{}
	direct   []directBinding
	buffered []bufferedBinding
	discard  interface{}
}

func (p *plan) acquireBindingSet(columns []string) *bindingSet {
	b, ok := p.bindings.Get().(*bindingSet)
	if !ok || b == nil {
		b = &bindingSet{}
	}
	if cap(b.dests) < len(columns) {
		b.dests = make([]interface{}, len(columns))
	} else {
		b.dests = b.dests[:len(columns)]
		clear(b.dests)
	}
	b.direct = b.direct[:0]
	b.buffered = b.buffered[:0]

	for i, column := range columns {
		fp, ok := p.fields[column]
		if !ok {
			b.dests[i] = &b.discard
			continue
		}
		switch fp.mode {
		case scanDirect:
			b.direct = append(b.direct, directBinding{colIdx: i, field: fp.field})
		case scanInt, scanUint, scanFloat, scanBool, scanString, scanTime:
			index := len(b.buffered)
			if index == cap(b.buffered) {
				b.buffered = append(b.buffered, bufferedBinding{})
			} else {
				b.buffered = b.buffered[:index+1]
			}
			binding := &b.buffered[index]
			binding.field = fp.field
			binding.mode = fp.mode
			// Restoring the slice length above preserves every buffer allocation
			// associated with this binding slot across equivalent queries.
			switch fp.mode {
			case scanInt, scanUint:
				if binding.intBuf == nil {
					binding.intBuf = new(sql.NullInt64)
				}
				b.dests[i] = binding.intBuf
			case scanFloat:
				if binding.floatBuf == nil {
					binding.floatBuf = new(sql.NullFloat64)
				}
				b.dests[i] = binding.floatBuf
			case scanBool:
				if binding.boolBuf == nil {
					binding.boolBuf = new(sql.NullBool)
				}
				b.dests[i] = binding.boolBuf
			case scanString:
				if binding.stringBuf == nil {
					binding.stringBuf = new(sql.NullString)
				}
				b.dests[i] = binding.stringBuf
			case scanTime:
				if binding.timeBuf == nil {
					binding.timeBuf = new(sql.NullTime)
				}
				b.dests[i] = binding.timeBuf
			}
		}
	}
	return b
}

func (p *plan) releaseBindingSet(b *bindingSet) {
	// Do not retain pointers to the result slice's fields or discarded driver
	// values after a request has completed. Keep only the reusable buffers.
	clear(b.dests)
	b.discard = nil
	for i := range b.buffered {
		binding := &b.buffered[i]
		binding.field = nil
		if binding.intBuf != nil {
			*binding.intBuf = sql.NullInt64{}
		}
		if binding.floatBuf != nil {
			*binding.floatBuf = sql.NullFloat64{}
		}
		if binding.boolBuf != nil {
			*binding.boolBuf = sql.NullBool{}
		}
		if binding.stringBuf != nil {
			*binding.stringBuf = sql.NullString{}
		}
		if binding.timeBuf != nil {
			*binding.timeBuf = sql.NullTime{}
		}
	}
	b.direct = b.direct[:0]
	b.buffered = b.buffered[:0]
	p.bindings.Put(b)
}

func scanRows(ctx context.Context, rows *sql.Rows, p *plan, slice reflect.Value, elemType reflect.Type, columns []string) (reflect.Value, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	bindings := p.acquireBindingSet(columns)
	defer p.releaseBindingSet(bindings)

	// Reuse the caller's backing array when it has capacity (GORM does the
	// same), and hand back a non-nil slice even for an empty result so it
	// serializes as [] rather than null.
	result := slice
	if result.IsNil() {
		result = reflect.MakeSlice(slice.Type(), 0, 8)
	} else if result.Len() > 0 {
		result = result.Slice(0, 0)
	}

	zero := reflect.Zero(elemType)
	for rows.Next() {
		result = reflect.Append(result, zero)
		elem := result.Index(result.Len() - 1)
		if err := scanRowInto(ctx, rows, bindings, elem); err != nil {
			return reflect.Value{}, err
		}
	}
	if err := rows.Err(); err != nil {
		return reflect.Value{}, err
	}
	return result, nil
}

// scanFirstRow scans at most one row into elem (an addressable struct value),
// reporting whether a row was present. It mirrors gorm.First: the query already
// carries LIMIT 1, so only the leading row is consumed.
func scanFirstRow(ctx context.Context, rows *sql.Rows, p *plan, elem reflect.Value, columns []string) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	bindings := p.acquireBindingSet(columns)
	defer p.releaseBindingSet(bindings)

	if !rows.Next() {
		return false, rows.Err()
	}
	if err := scanRowInto(ctx, rows, bindings, elem); err != nil {
		return false, err
	}
	return true, rows.Err()
}

// scanRowInto scans the current row into elem (an addressable struct value)
// using the prepared binding set: direct fields receive their addresses, then
// buffered scalars are copied out with NULL leaving the field at its zero value,
// exactly as GORM does.
func scanRowInto(ctx context.Context, rows *sql.Rows, bindings *bindingSet, elem reflect.Value) error {
	for _, d := range bindings.direct {
		bindings.dests[d.colIdx] = d.field.ReflectValueOf(ctx, elem).Addr().Interface()
	}
	if err := rows.Scan(bindings.dests...); err != nil {
		return &rowScanError{err: err}
	}
	for _, b := range bindings.buffered {
		// A NULL column leaves the field at its zero value, like GORM.
		switch b.mode {
		case scanInt:
			if b.intBuf.Valid {
				b.field.ReflectValueOf(ctx, elem).SetInt(b.intBuf.Int64)
			}
		case scanUint:
			if b.intBuf.Valid {
				b.field.ReflectValueOf(ctx, elem).SetUint(uint64(b.intBuf.Int64))
			}
		case scanFloat:
			if b.floatBuf.Valid {
				b.field.ReflectValueOf(ctx, elem).SetFloat(b.floatBuf.Float64)
			}
		case scanBool:
			if b.boolBuf.Valid {
				b.field.ReflectValueOf(ctx, elem).SetBool(b.boolBuf.Bool)
			}
		case scanString:
			if b.stringBuf.Valid {
				b.field.ReflectValueOf(ctx, elem).SetString(b.stringBuf.String)
			}
		case scanTime:
			if b.timeBuf.Valid {
				b.field.ReflectValueOf(ctx, elem).Set(reflect.ValueOf(b.timeBuf.Time))
			}
		}
	}
	return nil
}
