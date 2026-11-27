package odata_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GroupedPageItem struct {
	ID    int `json:"Id" gorm:"primaryKey;autoIncrement:false" odata:"key"`
	Group int `json:"Grp"`
	Kind  int `json:"Kind"`
}

// Groups with equal aggregate values must not be skipped or repeated when the
// server pages through a grouped result sorted on that aggregate.
func TestGroupedCollectionContinuationWithTies(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&GroupedPageItem{}); err != nil {
		t.Fatal(err)
	}
	// Six groups; groups 1,2,3 have two rows and groups 4,5,6 have one row, so the
	// COUNT aggregate ties inside both halves.
	id := 0
	var rows []GroupedPageItem
	for g := 1; g <= 6; g++ {
		n := 1
		if g <= 3 {
			n = 2
		}
		for i := 0; i < n; i++ {
			id++
			rows = append(rows, GroupedPageItem{ID: id, Group: g, Kind: g % 2})
		}
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RegisterEntity(&GroupedPageItem{}); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, order string }{
		{"aggregate-ties", "N"},
		{"aggregate-desc-ties", "N desc"},
		{"group-then-aggregate", "Grp,N"},
		{"no-orderby", ""},
	}
	for _, tc := range cases {
		for size := 1; size <= 4; size++ {
			t.Run(fmt.Sprintf("%s/size-%d", tc.name, size), func(t *testing.T) {
				q := url.Values{"$apply": {"groupby((Grp),aggregate($count as N))"}}
				if tc.order != "" {
					q.Set("$orderby", tc.order)
				}
				next := "/GroupedPageItems?" + q.Encode()
				var got []int
				for pages := 0; next != "" && pages < 20; pages++ {
					r := httptest.NewRequest(http.MethodGet, next, nil)
					r.Header.Set("Prefer", fmt.Sprintf("odata.maxpagesize=%d", size))
					r.Header.Set("OData-MaxVersion", "4.0")
					w := httptest.NewRecorder()
					svc.ServeHTTP(w, r)
					if w.Code != http.StatusOK {
						t.Fatalf("status %d: %s", w.Code, w.Body.String())
					}
					var page struct {
						Next  string `json:"@odata.nextLink"`
						Value []struct {
							Grp int `json:"Grp"`
						} `json:"value"`
					}
					if err := json.NewDecoder(w.Body).Decode(&page); err != nil {
						t.Fatal(err)
					}
					for _, v := range page.Value {
						got = append(got, v.Grp)
					}
					next = ""
					if page.Next != "" {
						u, err := url.Parse(page.Next)
						if err != nil {
							t.Fatal(err)
						}
						next = u.RequestURI()
					}
				}
				sort.Ints(got)
				want := []int{1, 2, 3, 4, 5, 6}
				if fmt.Sprint(got) != fmt.Sprint(want) {
					t.Fatalf("groups seen %v; want each of %v exactly once", got, want)
				}
			})
		}
	}
}

// SQLite happens to return ties in a stable order, so the traversal above cannot fail
// there. The defect is that the generated ORDER BY is not a total order, which is
// engine-independent and observable in the SQL, so assert on the statement itself.
func TestGroupedCollectionOrderIsTotal(t *testing.T) {
	var statements []string
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.New(sqlWriter(func(s string) { statements = append(statements, s) }), logger.Config{LogLevel: logger.Info}),
	})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&GroupedPageItem{}); err != nil {
		t.Fatal(err)
	}
	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RegisterEntity(&GroupedPageItem{}); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, order, wantOrderBy string }{
		{"aggregate", "N", "ORDER BY `N`,`group`"},
		{"aggregate-desc", "N desc", "ORDER BY `N` DESC,`group`"},
		{"group-key-first", "Grp,N", "ORDER BY `group`,`N`"},
		{"no-orderby", "", "ORDER BY `group`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			statements = nil
			target := "/GroupedPageItems?$apply=" + url.QueryEscape("groupby((Grp),aggregate($count as N))")
			if tc.order != "" {
				target += "&$orderby=" + url.QueryEscape(tc.order)
			}
			r := httptest.NewRequest(http.MethodGet, target, nil)
			r.Header.Set("Prefer", "odata.maxpagesize=1")
			w := httptest.NewRecorder()
			svc.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			for _, sql := range statements {
				if strings.Contains(sql, "GROUP BY") {
					if !strings.Contains(sql, tc.wantOrderBy) {
						t.Fatalf("grouped query is not totally ordered:\n%s\nwant it to contain %q", sql, tc.wantOrderBy)
					}
					return
				}
			}
			t.Fatal("no grouped statement captured")
		})
	}
}

type sqlWriter func(string)

func (f sqlWriter) Printf(format string, args ...interface{}) { f(fmt.Sprintf(format, args...)) }
