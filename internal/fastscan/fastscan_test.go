package fastscan

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Status is a named integer type, like an OData enum property.
type Status int32

// Payload implements sql.Scanner/driver.Valuer like custom column types do.
type Payload struct {
	Raw string
}

func (p *Payload) Scan(value interface{}) error {
	switch v := value.(type) {
	case nil:
		p.Raw = ""
	case string:
		p.Raw = v
	case []byte:
		p.Raw = string(v)
	default:
		return fmt.Errorf("unsupported payload type %T", value)
	}
	return nil
}

func (p Payload) Value() (driver.Value, error) {
	if p.Raw == "" {
		return nil, nil
	}
	return p.Raw, nil
}

type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Widget exercises every scan mode: buffered scalars, NULLs into non-pointer
// fields, pointer fields, named types, []byte, time.Time, sql.Scanner
// implementations, soft delete, and embedded structs.
type Widget struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Description *string
	Price       float64
	Quantity    *int
	Active      bool
	Status      Status
	Data        []byte
	Payload     Payload `gorm:"type:text"`
	Deleted     gorm.DeletedAt
	Timestamps
}

// Hooked has an AfterFind hook, which must force the GORM fallback.
type Hooked struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Count int `gorm:"-"`
}

func (h *Hooked) AfterFind(tx *gorm.DB) error {
	h.Count = 42
	return nil
}

// Parent/Child exercise the preload fallback.
type Parent struct {
	ID       uint `gorm:"primaryKey"`
	Name     string
	Children []Child `gorm:"foreignKey:ParentID"`
}

type Child struct {
	ID       uint `gorm:"primaryKey"`
	ParentID uint
	Name     string
	Parent   *Parent `gorm:"foreignKey:ParentID"`
}

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return db
}

func seedWidgets(t *testing.T, db *gorm.DB) []Widget {
	t.Helper()
	if err := db.AutoMigrate(&Widget{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	desc := "a widget"
	qty := 7
	now := time.Now().UTC().Truncate(time.Second)
	widgets := []Widget{
		{
			Name:        "full",
			Description: &desc,
			Price:       12.5,
			Quantity:    &qty,
			Active:      true,
			Status:      Status(3),
			Data:        []byte{0x1, 0x2},
			Payload:     Payload{Raw: "payload"},
			Timestamps:  Timestamps{CreatedAt: now, UpdatedAt: now},
		},
		{
			// Nullable columns stay NULL: Description, Quantity, Data, Payload.
			Name:       "nulls",
			Price:      0,
			Active:     false,
			Status:     Status(0),
			Timestamps: Timestamps{CreatedAt: now, UpdatedAt: now},
		},
	}
	if err := db.Create(&widgets).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return widgets
}

// findBoth runs the same query through fastscan.Find and GORM's Find and
// requires identical results.
func findBoth(t *testing.T, build func() *gorm.DB, makeDest func() interface{}) interface{} {
	t.Helper()
	fast := makeDest()
	if err := Find(build(), fast); err != nil {
		t.Fatalf("fastscan.Find: %v", err)
	}
	slow := makeDest()
	if err := build().Find(slow).Error; err != nil {
		t.Fatalf("gorm Find: %v", err)
	}
	fastVal := reflect.ValueOf(fast).Elem().Interface()
	slowVal := reflect.ValueOf(slow).Elem().Interface()
	if !reflect.DeepEqual(fastVal, slowVal) {
		t.Fatalf("fastscan result differs from GORM:\nfast: %#v\ngorm: %#v", fastVal, slowVal)
	}
	return fast
}

func TestFindMatchesGorm(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	results := findBoth(t,
		func() *gorm.DB { return db.Order("id") },
		func() interface{} { return &[]Widget{} },
	).(*[]Widget)

	if len(*results) != 2 {
		t.Fatalf("expected 2 widgets, got %d", len(*results))
	}
	full, nulls := (*results)[0], (*results)[1]
	if full.Description == nil || *full.Description != "a widget" {
		t.Errorf("pointer field not scanned: %+v", full.Description)
	}
	if full.Payload.Raw != "payload" {
		t.Errorf("scanner field not scanned: %+v", full.Payload)
	}
	if full.Status != Status(3) {
		t.Errorf("named int field not scanned: %v", full.Status)
	}
	if !full.Active {
		t.Error("bool field not scanned")
	}
	if full.CreatedAt.IsZero() {
		t.Error("embedded time field not scanned")
	}
	if nulls.Description != nil || nulls.Quantity != nil || nulls.Data != nil {
		t.Errorf("NULL columns must stay nil: %+v", nulls)
	}
}

func TestFindWithSelectSubset(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	results := findBoth(t,
		func() *gorm.DB { return db.Select("id", "name").Order("id") },
		func() interface{} { return &[]Widget{} },
	).(*[]Widget)

	if (*results)[0].Name != "full" || (*results)[0].Price != 0 {
		t.Errorf("select subset scanned incorrectly: %+v", (*results)[0])
	}
}

func TestFindWithFilterAndLimit(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	results := findBoth(t,
		func() *gorm.DB { return db.Where("price > ?", 1.0).Order("id").Limit(5) },
		func() interface{} { return &[]Widget{} },
	).(*[]Widget)

	if len(*results) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(*results))
	}
}

func TestFindEmptyResultIsNonNil(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	var results []Widget
	if err := Find(db.Where("1 = 0"), &results); err != nil {
		t.Fatalf("fastscan.Find: %v", err)
	}
	if results == nil {
		t.Fatal("empty result must be a non-nil slice so it serializes as []")
	}
	if len(results) != 0 {
		t.Fatalf("expected empty result, got %d rows", len(results))
	}
}

func TestFindReusesPresizedSlice(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	results := make([]Widget, 0, 16)
	if err := Find(db.Order("id"), &results); err != nil {
		t.Fatalf("fastscan.Find: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 widgets, got %d", len(results))
	}
	if cap(results) != 16 {
		t.Errorf("pre-sized backing array not reused: cap = %d", cap(results))
	}
}

func TestFindConcurrentUsesIndependentBindingSets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:fastscan_concurrent?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("database handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	seedWidgets(t, db)

	const (
		goroutines = 16
		queries    = 25
	)
	start := make(chan struct{})
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for range queries {
				var results []Widget
				if err := Find(db.Order("id"), &results); err != nil {
					errs <- err
					return
				}
				if len(results) != 2 || results[0].Name != "full" || results[1].Name != "nulls" {
					errs <- fmt.Errorf("unexpected concurrent result: %+v", results)
					return
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestFindRespectsSoftDelete(t *testing.T) {
	db := openDB(t)
	widgets := seedWidgets(t, db)

	if err := db.Delete(&Widget{}, widgets[1].ID).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	results := findBoth(t,
		func() *gorm.DB { return db.Order("id") },
		func() interface{} { return &[]Widget{} },
	).(*[]Widget)

	if len(*results) != 1 || (*results)[0].Name != "full" {
		t.Fatalf("soft-deleted row not filtered: %+v", *results)
	}
}

func TestFindFallsBackForAfterFindHook(t *testing.T) {
	db := openDB(t)
	if err := db.AutoMigrate(&Hooked{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&Hooked{Name: "h"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	var results []Hooked
	if err := Find(db.Order("id"), &results); err != nil {
		t.Fatalf("fastscan.Find: %v", err)
	}
	if len(results) != 1 || results[0].Count != 42 {
		t.Fatalf("AfterFind hook did not run, so fallback was skipped: %+v", results)
	}
}

func TestFindFallsBackForPreload(t *testing.T) {
	db := openDB(t)
	if err := db.AutoMigrate(&Parent{}, &Child{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	parent := Parent{Name: "p", Children: []Child{{Name: "c1"}, {Name: "c2"}}}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	var results []Parent
	if err := Find(db.Preload("Children").Order("id"), &results); err != nil {
		t.Fatalf("fastscan.Find: %v", err)
	}
	if len(results) != 1 || len(results[0].Children) != 2 {
		t.Fatalf("preload did not populate children, so fallback was skipped: %+v", results)
	}
}

func TestFindFallsBackForRelationJoin(t *testing.T) {
	db := openDB(t)
	if err := db.AutoMigrate(&Parent{}, &Child{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&Parent{Name: "p", Children: []Child{{Name: "c1"}}}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	// A raw SQL join stays on the fast path; the results must match GORM.
	results := findBoth(t,
		func() *gorm.DB {
			return db.Model(&Parent{}).
				Joins("JOIN children ON children.parent_id = parents.id").
				Order("parents.id")
		},
		func() interface{} { return &[]Parent{} },
	).(*[]Parent)
	if len(*results) != 1 {
		t.Fatalf("raw join returned %d rows", len(*results))
	}

	// An association (belongs-to) join must fall back so GORM scans the
	// joined parent columns into the nested struct.
	var withJoin []Child
	if err := Find(db.Joins("Parent").Order("children.id"), &withJoin); err != nil {
		t.Fatalf("fastscan.Find with association join: %v", err)
	}
	var gormJoin []Child
	if err := db.Joins("Parent").Order("children.id").Find(&gormJoin).Error; err != nil {
		t.Fatalf("gorm Find with association join: %v", err)
	}
	if !reflect.DeepEqual(withJoin, gormJoin) {
		t.Fatalf("association join result differs:\nfast: %+v\ngorm: %+v", withJoin, gormJoin)
	}
	if len(withJoin) != 1 || withJoin[0].Parent == nil || withJoin[0].Parent.Name != "p" {
		t.Fatalf("joined parent not populated, so fallback was skipped: %+v", withJoin)
	}
}

func TestFindNonSliceDestFallsBack(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	var maps []map[string]interface{}
	if err := Find(db.Model(&Widget{}).Order("id"), &maps); err != nil {
		t.Fatalf("fastscan.Find with map dest: %v", err)
	}
	if len(maps) != 2 {
		t.Fatalf("map fallback returned %d rows", len(maps))
	}
}

func TestScanErrorPoisonsPlanAndFallsBack(t *testing.T) {
	db := openDB(t)
	if err := db.Exec("CREATE TABLE gadgets (id integer primary key, amount integer)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	// SQLite's flexible typing lets TEXT live in an INTEGER column. The fast
	// path's sql.NullInt64 rejects it; GORM Find reports its own conversion
	// error. Either way the caller sees an error, and the plan is disabled.
	if err := db.Exec("INSERT INTO gadgets (id, amount) VALUES (1, 'not-a-number')").Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	type Gadget struct {
		ID     uint `gorm:"primaryKey"`
		Amount int
	}

	var results []Gadget
	err := Find(db.Table("gadgets").Model(&Gadget{}), &results)
	if err == nil {
		t.Fatal("expected a conversion error from both scan paths")
	}
	if !strings.Contains(err.Error(), "not-a-number") && !strings.Contains(strings.ToLower(err.Error()), "convert") {
		t.Logf("conversion error: %v", err)
	}

	stmt := db.Session(&gorm.Session{}).Model(&Gadget{}).Statement
	if parseErr := stmt.Parse(&Gadget{}); parseErr != nil {
		t.Fatalf("parse: %v", parseErr)
	}
	p := planFor(stmt.Schema)
	if p == nil {
		t.Fatal("expected an eligible plan for Gadget")
	}
	if !p.disabled.Load() {
		t.Error("plan was not disabled after a scan error")
	}
}

func TestFieldScanModes(t *testing.T) {
	cases := []struct {
		typ  reflect.Type
		mode scanMode
		ok   bool
	}{
		{reflect.TypeOf(""), scanString, true},
		{reflect.TypeOf(0), scanInt, true},
		{reflect.TypeOf(uint8(0)), scanUint, true},
		{reflect.TypeOf(float32(0)), scanFloat, true},
		{reflect.TypeOf(false), scanBool, true},
		{reflect.TypeOf(Status(0)), scanInt, true},
		{reflect.TypeOf(time.Time{}), scanTime, true},
		{reflect.TypeOf([]byte(nil)), scanDirect, true},
		{reflect.TypeOf((*string)(nil)), scanDirect, true},
		{reflect.TypeOf((*time.Time)(nil)), scanDirect, true},
		{reflect.TypeOf(sql.NullString{}), scanDirect, true},
		{reflect.TypeOf(gorm.DeletedAt{}), scanDirect, true},
		{reflect.TypeOf(Payload{}), scanDirect, true},
		{reflect.TypeOf(struct{ X int }{}), 0, false},
		{reflect.TypeOf(map[string]string(nil)), 0, false},
		{reflect.TypeOf((**string)(nil)), 0, false},
	}
	for _, tc := range cases {
		mode, ok := fieldScanMode(tc.typ)
		if ok != tc.ok || (ok && mode != tc.mode) {
			t.Errorf("fieldScanMode(%v) = (%v, %v), want (%v, %v)", tc.typ, mode, ok, tc.mode, tc.ok)
		}
	}
}
