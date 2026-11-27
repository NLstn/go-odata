package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type CursorRegressionItem struct {
	ID     int  `json:"Id" gorm:"primaryKey" odata:"key"`
	Bucket int  `json:"Bucket"`
	Rank   *int `json:"Rank"`
}

func TestCollectionCursorRegressions(t *testing.T) {
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
	if err := db.AutoMigrate(&CursorRegressionItem{}); err != nil {
		t.Fatal(err)
	}
	rank := 7
	rows := []CursorRegressionItem{{ID: 1, Bucket: 3}, {ID: 2, Bucket: 1, Rank: &rank}, {ID: 3, Bucket: 1}, {ID: 4, Bucket: 2, Rank: &rank}}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RegisterEntity(&CursorRegressionItem{}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name       string
		query      url.Values
		ids        []int
		linkOption string
	}{
		{"hidden-secondary-sort", url.Values{"$select": {"Id"}, "$orderby": {"Id,Bucket"}}, []int{1, 2, 3, 4}, "$skiptoken"},
		{"hidden-leading-sort", url.Values{"$select": {"Id"}, "$orderby": {"Bucket,Id"}}, []int{2, 3, 4, 1}, "$skiptoken"},
		{"null-secondary-sort", url.Values{"$orderby": {"Id,Rank"}}, []int{1, 2, 3, 4}, "$skiptoken"},
		{"nulls-first", url.Values{"$orderby": {"Rank asc"}}, []int{1, 3, 2, 4}, "$skiptoken"},
		{"nulls-last", url.Values{"$orderby": {"Rank desc"}}, []int{2, 4, 1, 3}, "$skiptoken"},
		{"grouped-entity-key", url.Values{"$apply": {"groupby((Bucket,Id),aggregate($count as N))"}}, []int{2, 3, 4, 1}, "$skip"},
		{"grouped-partial-order", url.Values{"$apply": {"groupby((Bucket,Id),aggregate($count as N))"}, "$orderby": {"Bucket"}}, []int{2, 3, 4, 1}, "$skip"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			next := "/CursorRegressionItems?" + tc.query.Encode()
			got := make([]int, 0, len(tc.ids))
			for i := range tc.ids {
				if next == "" {
					t.Fatalf("continuation ended early: %v", got)
				}
				r := httptest.NewRequest(http.MethodGet, next, nil)
				r.Header.Set("Prefer", "odata.maxpagesize=1")
				r.Header.Set("OData-MaxVersion", "4.0")
				w := httptest.NewRecorder()
				svc.ServeHTTP(w, r)
				if w.Code != http.StatusOK {
					t.Fatalf("page %d: status %d: %s", i+1, w.Code, w.Body.String())
				}
				var page struct {
					Next  string                       `json:"@odata.nextLink"`
					Value []map[string]json.RawMessage `json:"value"`
				}
				if err := json.NewDecoder(w.Body).Decode(&page); err != nil {
					t.Fatal(err)
				}
				if len(page.Value) != 1 {
					t.Fatalf("page %d: want one result; got %s", i+1, w.Body.String())
				}
				row := page.Value[0]
				if tc.query.Has("$select") && (row["Bucket"] != nil || row["Rank"] != nil) {
					t.Fatalf("response included unselected fields: %s", w.Body.String())
				}
				var id int
				if err := json.Unmarshal(row["Id"], &id); err != nil {
					t.Fatal(err)
				}
				got = append(got, id)
				if page.Next != "" {
					link, err := url.Parse(page.Next)
					if err != nil {
						t.Fatal(err)
					}
					if !link.Query().Has(tc.linkOption) {
						t.Errorf("page %d: want %s in next link, got %s", i+1, tc.linkOption, page.Next)
					}
				}
				next = page.Next
			}
			if next != "" || !slices.Equal(got, tc.ids) {
				t.Fatalf("got %v (remaining continuation: %t); want %v", got, next != "", tc.ids)
			}
		})
	}
}
