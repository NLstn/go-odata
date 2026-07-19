package fastscan

import (
	"database/sql"
	"database/sql/driver"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// interfaceType is reflect.Type for interface{}, the fallback scan destination
// for columns that do not map to a schema field — matching GORM's
// prepareValues, which uses new(interface{}) in that case.
var interfaceType = reflect.TypeOf((*interface{})(nil)).Elem()

// FindMap runs the SELECT built on db and appends each result row to *dest as a
// map[string]interface{}. It is a faster replacement for db.Find(dest) when
// dest is a *[]map[string]interface{} — the destination that $apply/$compute
// projections scan into — and produces byte-for-byte the same maps: columns
// that map to a schema field are scanned into that field's type (so an integer
// key column yields uint, not the driver's int64), and driver.Valuer /
// sql.RawBytes values are normalized exactly the way GORM's scanIntoMap does.
//
// The speedup over GORM comes from not re-allocating a fresh scan destination
// for every column of every row: GORM's map scan calls prepareValues (a
// reflect.New per column) on each row, whereas FindMap builds the destinations
// once and reuses them. The result map and its boxed cell values are the only
// unavoidable per-row allocations, so the win grows with the row count.
//
// Downstream driver-type normalization the handlers apply on top of map results
// (normalizeComputedResultValues, aggregate coercion) is unaffected: the values
// handed back are identical to GORM's.
func FindMap(db *gorm.DB, dest *[]map[string]interface{}) error {
	if db.DryRun {
		return db.Find(dest).Error
	}

	// GORM resolves the statement schema from the model during query building and
	// types the scan destinations from its fields. Parse it up front so we can do
	// the same; without a model, columns fall back to driver-typed values just as
	// GORM's prepareValues does when Statement.Schema is nil.
	var sch *schema.Schema
	if db.Statement.Model != nil {
		if err := db.Statement.Parse(db.Statement.Model); err == nil {
			sch = db.Statement.Schema
		}
	}

	rows, err := db.Rows()
	if err != nil {
		return err
	}

	result, scanErr := scanMaps(rows, sch, *dest)
	closeErr := rows.Close()
	if scanErr != nil {
		return scanErr
	}
	if closeErr != nil {
		return closeErr
	}

	*dest = result
	db.RowsAffected = int64(len(result))
	return nil
}

// mapDest is a reusable scan destination for one column. value is the result of
// reflect.New (a pointer), scanArg caches value.Interface() so it does not have
// to be recomputed per row, and both are reused across every row of the result.
type mapDest struct {
	column  string
	value   reflect.Value
	scanArg interface{}
}

// buildMapDests mirrors GORM's prepareValues: a schema field column is scanned
// into *PointerTo(field type) (so NULLs land as a nil pointer and the value
// carries the field's Go type), and any other column into *interface{} (or the
// driver's ScanType when no schema is available).
func buildMapDests(sch *schema.Schema, columnTypes []*sql.ColumnType, columns []string) []mapDest {
	dests := make([]mapDest, len(columns))
	for idx, name := range columns {
		var t reflect.Type
		switch {
		case sch != nil:
			if field := sch.LookUpField(name); field != nil {
				t = reflect.PointerTo(field.FieldType)
			} else {
				t = interfaceType
			}
		case idx < len(columnTypes) && columnTypes[idx].ScanType() != nil:
			t = reflect.PointerTo(columnTypes[idx].ScanType())
		default:
			t = interfaceType
		}
		v := reflect.New(t)
		dests[idx] = mapDest{column: name, value: v, scanArg: v.Interface()}
	}
	return dests
}

func scanMaps(rows *sql.Rows, sch *schema.Schema, existing []map[string]interface{}) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Column types are only consulted when there is no schema to type columns by,
	// matching GORM's prepareValues. If the driver cannot report them, columns
	// fall back to interface{}, which is GORM's behavior for that case too.
	var columnTypes []*sql.ColumnType
	if sch == nil {
		if cts, err := rows.ColumnTypes(); err == nil {
			columnTypes = cts
		}
	}

	dests := buildMapDests(sch, columnTypes, columns)
	scanArgs := make([]interface{}, len(dests))
	for i := range dests {
		scanArgs[i] = dests[i].scanArg
	}

	// Reuse the caller's backing array when present, and never return nil so the
	// result serializes as [] rather than null.
	result := existing[:0]
	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}
		m := make(map[string]interface{}, len(columns))
		for i := range dests {
			// reflect.Indirect twice: once through the reflect.New pointer, once more
			// for pointer-typed fields, leaving the scanned value (or an invalid
			// Value for a NULL that landed in a nil pointer).
			rv := reflect.Indirect(reflect.Indirect(dests[i].value))
			if !rv.IsValid() {
				m[dests[i].column] = nil
				continue
			}
			val := rv.Interface()
			if valuer, ok := val.(driver.Valuer); ok {
				// Reduce a driver.Valuer to its underlying value, as GORM's
				// scanIntoMap does. GORM ignores the error; a value that
				// database/sql just scanned does not fail Value() in practice, so
				// on the off chance it does we keep the original rather than a
				// partial result.
				if v, err := valuer.Value(); err == nil {
					val = v
				}
			} else if b, ok := val.(sql.RawBytes); ok {
				val = string(b)
			}
			m[dests[i].column] = val
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
