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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
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
	// disabled is set when a scan produced an error that GORM's richer value
	// conversion might have handled; every later query for this schema then
	// uses the fallback instead of failing the same way again.
	disabled atomic.Bool
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
	if p == nil || p.disabled.Load() || !eligibleStatement(tx.Statement, sch) {
		return tx.Find(dest).Error
	}

	rows, err := tx.Rows()
	if err != nil {
		return err
	}

	result, err := scanRows(tx.Statement.Context, rows, p, sliceVal, elemType)
	closeErr := rows.Close()
	if err != nil {
		if scanErr, ok := err.(*rowScanError); ok {
			// A row failed to scan. If the context is gone this is just
			// cancellation; otherwise the plan mis-handled a driver value that
			// GORM's conversions might accept. Disable the plan for this
			// schema and re-run the query through GORM, which either succeeds
			// or reports its own (equivalent) error. Execute resets the
			// statement's SQL after Rows, so Find rebuilds it cleanly.
			if ctxErr := contextErr(tx.Statement.Context); ctxErr != nil {
				return scanErr.err
			}
			p.disabled.Store(true)
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
	return &plan{fields: fields}
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

func scanRows(ctx context.Context, rows *sql.Rows, p *plan, slice reflect.Value, elemType reflect.Type) (reflect.Value, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	columns, err := rows.Columns()
	if err != nil {
		return reflect.Value{}, err
	}

	// Bind each result column once per query. Buffered columns get a stable
	// sql.Null* destination reused across rows; direct columns are re-pointed
	// at the current element's fields each row; unknown columns (which GORM
	// would also discard for struct destinations) are scanned into a sink.
	dests := make([]interface{}, len(columns))
	var direct []directBinding
	var buffered []bufferedBinding
	var discard interface{}
	for i, column := range columns {
		fp, ok := p.fields[column]
		if !ok {
			dests[i] = &discard
			continue
		}
		switch fp.mode {
		case scanDirect:
			direct = append(direct, directBinding{colIdx: i, field: fp.field})
		case scanInt, scanUint:
			buf := new(sql.NullInt64)
			dests[i] = buf
			buffered = append(buffered, bufferedBinding{field: fp.field, mode: fp.mode, intBuf: buf})
		case scanFloat:
			buf := new(sql.NullFloat64)
			dests[i] = buf
			buffered = append(buffered, bufferedBinding{field: fp.field, mode: fp.mode, floatBuf: buf})
		case scanBool:
			buf := new(sql.NullBool)
			dests[i] = buf
			buffered = append(buffered, bufferedBinding{field: fp.field, mode: fp.mode, boolBuf: buf})
		case scanString:
			buf := new(sql.NullString)
			dests[i] = buf
			buffered = append(buffered, bufferedBinding{field: fp.field, mode: fp.mode, stringBuf: buf})
		case scanTime:
			buf := new(sql.NullTime)
			dests[i] = buf
			buffered = append(buffered, bufferedBinding{field: fp.field, mode: fp.mode, timeBuf: buf})
		}
	}

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
		for _, d := range direct {
			dests[d.colIdx] = d.field.ReflectValueOf(ctx, elem).Addr().Interface()
		}
		if err := rows.Scan(dests...); err != nil {
			return reflect.Value{}, &rowScanError{err: err}
		}
		for _, b := range buffered {
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
	}
	if err := rows.Err(); err != nil {
		return reflect.Value{}, err
	}
	return result, nil
}
