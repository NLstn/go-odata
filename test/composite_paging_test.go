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

type CompositePageItem struct {
	GroupID int `json:"GroupId" gorm:"primaryKey;autoIncrement:false" odata:"key"`
	ItemID  int `json:"ItemId" gorm:"primaryKey;autoIncrement:false" odata:"key"`
}

func TestCompositeCollectionContinuation(t *testing.T) {
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
	rows := []CompositePageItem{{1, 1}, {1, 2}, {2, 1}, {2, 2}, {3, 1}}
	if err := db.AutoMigrate(&CompositePageItem{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RegisterEntity(&CompositePageItem{}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, order, top string
		want             []CompositePageItem
	}{
		{"ascending", "GroupId asc,ItemId asc", "", rows},
		{"descending", "GroupId desc,ItemId desc", "", []CompositePageItem{{3, 1}, {2, 2}, {2, 1}, {1, 2}, {1, 1}}},
		{"mixed-with-top", "GroupId asc,ItemId desc", "3", []CompositePageItem{{1, 2}, {1, 1}, {2, 2}}},
		{"implicit", "", "", rows},
		{"partial-order", "GroupId asc", "", rows},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := url.Values{}
			if tc.order != "" {
				q.Set("$orderby", tc.order)
			}
			if tc.top != "" {
				q.Set("$top", tc.top)
			}
			next := "/CompositePageItems?" + q.Encode()
			var got []CompositePageItem
			for range tc.want {
				if next == "" {
					t.Fatalf("continuation ended early: %v", got)
				}
				r := httptest.NewRequest(http.MethodGet, next, nil)
				r.Header.Set("Prefer", "odata.maxpagesize=1")
				r.Header.Set("OData-MaxVersion", "4.0")
				w := httptest.NewRecorder()
				svc.ServeHTTP(w, r)
				if w.Code != http.StatusOK {
					t.Fatalf("status = %d: %s", w.Code, w.Body.String())
				}
				var page struct {
					Next  string              `json:"@odata.nextLink"`
					Value []CompositePageItem `json:"value"`
				}
				if err := json.NewDecoder(w.Body).Decode(&page); err != nil {
					t.Fatal(err)
				}
				if len(page.Value) != 1 {
					t.Fatalf("wanted one row per page; got %v after %v", page.Value, got)
				}
				got = append(got, page.Value[0])
				next = page.Next
			}
			if next != "" || !slices.Equal(got, tc.want) {
				t.Fatalf("got %v (remaining continuation: %t); want %v", got, next != "", tc.want)
			}
		})
	}
}
