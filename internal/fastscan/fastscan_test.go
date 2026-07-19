package fastscan

import (
	"database/sql"
	"database/sql/driver"
	"errors"
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

// TestFindGrowsPastInitialCapacity seeds more rows than the destination
// slice's initial capacity, forcing scanRows's manual grow path (doubling
// the backing array) to run more than once, and checks every row still
// lands in the right place afterward.
func TestFindGrowsPastInitialCapacity(t *testing.T) {
	db := openDB(t)
	if err := db.AutoMigrate(&Widget{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	const n = 50
	widgets := make([]Widget, 0, n)
	for i := 0; i < n; i++ {
		widgets = append(widgets, Widget{Name: fmt.Sprintf("widget-%d", i)})
	}
	if err := db.Create(&widgets).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	// A capacity of 1 forces several doublings (1 -> 2 -> 4 -> ... -> 64)
	// before all n rows fit.
	results := make([]Widget, 0, 1)
	if err := Find(db.Order("id"), &results); err != nil {
		t.Fatalf("fastscan.Find: %v", err)
	}
	if len(results) != n {
		t.Fatalf("expected %d widgets, got %d", n, len(results))
	}
	for i, w := range results {
		want := fmt.Sprintf("widget-%d", i)
		if w.Name != want {
			t.Fatalf("result[%d].Name = %q, want %q", i, w.Name, want)
		}
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

// firstBoth runs the same query through fastscan.First and GORM's First and
// requires identical results.
func firstBoth(t *testing.T, build func() *gorm.DB, makeDest func() interface{}) interface{} {
	t.Helper()
	fast := makeDest()
	if err := First(build(), fast); err != nil {
		t.Fatalf("fastscan.First: %v", err)
	}
	slow := makeDest()
	if err := build().First(slow).Error; err != nil {
		t.Fatalf("gorm First: %v", err)
	}
	fastVal := reflect.ValueOf(fast).Elem().Interface()
	slowVal := reflect.ValueOf(slow).Elem().Interface()
	if !reflect.DeepEqual(fastVal, slowVal) {
		t.Fatalf("fastscan.First result differs from GORM:\nfast: %#v\ngorm: %#v", fastVal, slowVal)
	}
	return fast
}

func TestFirstMatchesGorm(t *testing.T) {
	db := openDB(t)
	widgets := seedWidgets(t, db)

	full := firstBoth(t,
		func() *gorm.DB { return db.Where("id = ?", widgets[0].ID) },
		func() interface{} { return &Widget{} },
	).(*Widget)
	if full.Name != "full" || full.Description == nil || *full.Description != "a widget" {
		t.Errorf("full widget scanned incorrectly: %+v", full)
	}
	if full.Payload.Raw != "payload" || full.Status != Status(3) || !full.Active {
		t.Errorf("scanner/named/bool fields scanned incorrectly: %+v", full)
	}

	nulls := firstBoth(t,
		func() *gorm.DB { return db.Where("id = ?", widgets[1].ID) },
		func() interface{} { return &Widget{} },
	).(*Widget)
	if nulls.Description != nil || nulls.Quantity != nil || nulls.Data != nil {
		t.Errorf("NULL columns must stay nil: %+v", nulls)
	}
}

func TestFirstOrdersByPrimaryKey(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	// With no WHERE, First must return the lowest primary key, exactly like
	// gorm.First's implicit ORDER BY primary key.
	got := firstBoth(t,
		func() *gorm.DB { return db.Model(&Widget{}) },
		func() interface{} { return &Widget{} },
	).(*Widget)
	if got.Name != "full" {
		t.Fatalf("First did not order by primary key: got %+v", got)
	}
}

func TestFirstReturnsRecordNotFound(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	var w Widget
	if err := First(db.Where("id = ?", 99999), &w); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
	// GORM agrees on the no-row error.
	var g Widget
	if err := db.Where("id = ?", 99999).First(&g).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("gorm baseline: expected ErrRecordNotFound, got %v", err)
	}
}

func TestFirstFallsBackForAfterFindHook(t *testing.T) {
	db := openDB(t)
	if err := db.AutoMigrate(&Hooked{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&Hooked{Name: "h"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	var h Hooked
	if err := First(db.Model(&Hooked{}), &h); err != nil {
		t.Fatalf("fastscan.First: %v", err)
	}
	if h.Count != 42 {
		t.Fatalf("AfterFind hook did not run, so fallback was skipped: %+v", h)
	}
}

func TestFirstFallsBackForPreload(t *testing.T) {
	db := openDB(t)
	if err := db.AutoMigrate(&Parent{}, &Child{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	parent := Parent{Name: "p", Children: []Child{{Name: "c1"}, {Name: "c2"}}}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	var got Parent
	if err := First(db.Preload("Children").Where("id = ?", parent.ID), &got); err != nil {
		t.Fatalf("fastscan.First: %v", err)
	}
	if len(got.Children) != 2 {
		t.Fatalf("preload did not populate children, so fallback was skipped: %+v", got)
	}
}

func TestFirstNonStructDestFallsBack(t *testing.T) {
	db := openDB(t)
	seedWidgets(t, db)

	m := map[string]interface{}{}
	if err := First(db.Model(&Widget{}), &m); err != nil {
		t.Fatalf("fastscan.First with map dest: %v", err)
	}
	if m["name"] == nil {
		t.Fatalf("map fallback did not populate: %+v", m)
	}
}

// gadgetPlan runs a fresh Find against the gadgets table (created by the
// caller) so the schema gets parsed and its plan built, then returns the plan.
func gadgetPlan(t *testing.T, db *gorm.DB) *plan {
	t.Helper()
	stmt := db.Session(&gorm.Session{}).Model(&Gadget{}).Statement
	if parseErr := stmt.Parse(&Gadget{}); parseErr != nil {
		t.Fatalf("parse: %v", parseErr)
	}
	p := planFor(stmt.Schema)
	if p == nil {
		t.Fatal("expected an eligible plan for Gadget")
	}
	return p
}

type Gadget struct {
	ID     uint `gorm:"primaryKey"`
	Amount int
	Name   string
}

func TestScanErrorPoisonsColumnAndFallsBack(t *testing.T) {
	db := openDB(t)
	if err := db.Exec("CREATE TABLE gadgets (id integer primary key, amount integer, name text)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	// SQLite's flexible typing lets TEXT live in an INTEGER column. The fast
	// path's sql.NullInt64 rejects it; GORM Find reports its own conversion
	// error. Either way the caller sees an error, and only the offending
	// column ("amount") is poisoned.
	if err := db.Exec("INSERT INTO gadgets (id, amount, name) VALUES (1, 'not-a-number', 'widget')").Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	var results []Gadget
	err := Find(db.Table("gadgets").Model(&Gadget{}), &results)
	if err == nil {
		t.Fatal("expected a conversion error from both scan paths")
	}
	if !strings.Contains(err.Error(), "not-a-number") && !strings.Contains(strings.ToLower(err.Error()), "convert") {
		t.Logf("conversion error: %v", err)
	}

	p := gadgetPlan(t, db)
	if !p.isPoisoned("amount") {
		t.Error("amount column was not poisoned after a scan error")
	}
	if p.isPoisoned("id") || p.isPoisoned("name") {
		t.Error("poisoning a scan error should not affect unrelated columns")
	}

	// A query that doesn't touch the poisoned column should still take the
	// fast path and succeed.
	var partial []Gadget
	if err := Find(db.Table("gadgets").Model(&Gadget{}).Select("id", "name"), &partial); err != nil {
		t.Fatalf("Find selecting only healthy columns: %v", err)
	}
	if len(partial) != 1 || partial[0].Name != "widget" {
		t.Fatalf("unexpected result for healthy-column select: %+v", partial)
	}

	// A query that doesn't restrict columns still includes the poisoned one.
	// It should skip straight to the GORM fallback (not attempt — and fail —
	// the fast scan again) and surface GORM's own equivalent conversion error.
	var full []Gadget
	if err := Find(db.Table("gadgets").Model(&Gadget{}), &full); err == nil {
		t.Fatal("expected the fallback to surface GORM's own conversion error for the still-poisoned column")
	}
}

func TestPoisonExpiresAfterRetryWindow(t *testing.T) {
	original := poisonRetryWindow
	poisonRetryWindow = 10 * time.Millisecond
	defer func() { poisonRetryWindow = original }()

	db := openDB(t)
	if err := db.Exec("CREATE TABLE gadgets (id integer primary key, amount integer, name text)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	p := gadgetPlan(t, db)
	p.poisonColumn("amount")
	if !p.isPoisoned("amount") {
		t.Fatal("expected amount to be poisoned immediately after poisoning")
	}
	if p.poisonCount.Load() != 1 {
		t.Fatalf("poisonCount = %d, want 1", p.poisonCount.Load())
	}

	time.Sleep(2 * poisonRetryWindow)
	if p.isPoisoned("amount") {
		t.Error("expected amount to no longer be poisoned after the retry window elapsed")
	}
	if p.poisonCount.Load() != 0 {
		t.Errorf("poisonCount = %d, want 0 after expiry", p.poisonCount.Load())
	}
}

func TestRecordScanFailureFallsBackToPoisonAllForUnrecognizedError(t *testing.T) {
	db := openDB(t)
	if err := db.Exec("CREATE TABLE gadgets (id integer primary key, amount integer, name text)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	p := gadgetPlan(t, db)

	p.recordScanFailure(errors.New("some unrelated driver failure"), []string{"id", "amount", "name"})

	for _, col := range []string{"id", "amount", "name"} {
		if !p.isPoisoned(col) {
			t.Errorf("expected column %q to be poisoned when the error can't be attributed to one column", col)
		}
	}
}

func TestLikelyTouchesPoisoned(t *testing.T) {
	db := openDB(t)
	if err := db.Exec("CREATE TABLE gadgets (id integer primary key, amount integer, name text)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	p := gadgetPlan(t, db)
	p.poisonColumn("amount")

	stmt := db.Session(&gorm.Session{}).Model(&Gadget{}).Statement
	if parseErr := stmt.Parse(&Gadget{}); parseErr != nil {
		t.Fatalf("parse: %v", parseErr)
	}

	if !p.likelyTouchesPoisoned(stmt, stmt.Schema) {
		t.Error("a query with no explicit Select should be treated as touching the poisoned column")
	}

	stmt.Selects = []string{"ID", "Name"}
	if p.likelyTouchesPoisoned(stmt, stmt.Schema) {
		t.Error("an explicit select excluding the poisoned column should not be flagged")
	}

	stmt.Selects = []string{"ID", "Amount"}
	if !p.likelyTouchesPoisoned(stmt, stmt.Schema) {
		t.Error("an explicit select including the poisoned column should be flagged")
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
