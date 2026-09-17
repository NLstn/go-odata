package odata_test

import (
	"encoding/json"
	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type PipelineOrder struct {
	ID     int32               `json:"ID" gorm:"primaryKey;autoIncrement:false" odata:"key"`
	Region string              `json:"Region"`
	Lines  []PipelineOrderLine `json:"Lines" gorm:"foreignKey:OrderID;references:ID"`
}

func (PipelineOrder) TableName() string { return "PipelineOrders" }

type PipelineOrderLine struct {
	ID      int32   `json:"ID" gorm:"primaryKey;autoIncrement:false" odata:"key"`
	OrderID int32   `json:"OrderID"`
	Price   float64 `json:"Price"`
}

func (PipelineOrderLine) TableName() string { return "PipelineOrderLines" }

func newAddNestedPipelineService(t *testing.T) *odata.Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	if err = db.AutoMigrate(&PipelineOrder{}, &PipelineOrderLine{}); err != nil {
		t.Fatal(err)
	}
	for _, order := range []PipelineOrder{{ID: 1, Region: "EU"}, {ID: 2, Region: "US"}, {ID: 3, Region: "EU"}} {
		if err = db.Create(&order).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, line := range []PipelineOrderLine{{ID: 1, OrderID: 1, Price: 5}, {ID: 2, OrderID: 1, Price: 20}, {ID: 3, OrderID: 2, Price: 50}} {
		if err = db.Create(&line).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.RegisterEntity(&PipelineOrder{}); err != nil {
		t.Fatal(err)
	}
	if err = svc.RegisterEntity(&PipelineOrderLine{}); err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestAddNestedPipelineSemantics(t *testing.T) {
	svc := newAddNestedPipelineService(t)

	get := func(t *testing.T, target string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		svc.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
		return w
	}
	decode := func(t *testing.T, w *httptest.ResponseRecorder) (rows []map[string]interface{}, count float64) {
		t.Helper()
		if w.Code != 200 {
			t.Fatalf("%d: %s", w.Code, w.Body)
		}
		var body struct {
			Value []map[string]interface{} `json:"value"`
			Count float64                  `json:"@odata.count"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Value, body.Count
	}
	ids := func(rows []map[string]interface{}) []float64 {
		out := make([]float64, 0, len(rows))
		for _, row := range rows {
			out = append(out, row["ID"].(float64))
		}
		return out
	}
	nestedIDs := func(t *testing.T, row map[string]interface{}, alias string) []float64 {
		t.Helper()
		nested, ok := row[alias].([]interface{})
		if !ok {
			t.Fatalf("expected %s to be an array, got %T", alias, row[alias])
		}
		out := make([]float64, 0, len(nested))
		for _, item := range nested {
			out = append(out, item.(map[string]interface{})["ID"].(float64))
		}
		return out
	}

	t.Run("leading addnested unchanged", func(t *testing.T) {
		w := get(t, "/PipelineOrders?$apply="+url.QueryEscape("addnested(Lines,filter(Price gt 10) as FilteredLines)"))
		rows, _ := decode(t, w)
		if got := ids(rows); !reflect.DeepEqual(got, []float64{1, 2, 3}) {
			t.Fatalf("got %v, want [1 2 3]", got)
		}
		if got := nestedIDs(t, rows[0], "FilteredLines"); !reflect.DeepEqual(got, []float64{2}) {
			t.Fatalf("order 1 FilteredLines %v, want [2]", got)
		}
		if got := nestedIDs(t, rows[2], "FilteredLines"); len(got) != 0 {
			t.Fatalf("order 3 FilteredLines %v, want []", got)
		}
	})

	t.Run("addnested tail filter applied", func(t *testing.T) {
		w := get(t, "/PipelineOrders?$apply="+url.QueryEscape("addnested(Lines,filter(Price gt 10) as FilteredLines)/filter(Region eq 'EU')")+"&$count=true")
		rows, count := decode(t, w)
		if got := ids(rows); !reflect.DeepEqual(got, []float64{1, 3}) || count != 2 {
			t.Fatalf("got %v count %v, want [1 3] count 2", got, count)
		}
		if got := nestedIDs(t, rows[0], "FilteredLines"); !reflect.DeepEqual(got, []float64{2}) {
			t.Fatalf("order 1 FilteredLines %v, want [2]", got)
		}
	})

	t.Run("addnested system filter applied", func(t *testing.T) {
		w := get(t, "/PipelineOrders?$apply="+url.QueryEscape("addnested(Lines,filter(Price gt 10) as FilteredLines)")+"&$filter="+url.QueryEscape("Region eq 'US'")+"&$count=true")
		rows, count := decode(t, w)
		if got := ids(rows); !reflect.DeepEqual(got, []float64{2}) || count != 1 {
			t.Fatalf("got %v count %v, want [2] count 1", got, count)
		}
	})

	t.Run("addnested system filter with top", func(t *testing.T) {
		w := get(t, "/PipelineOrders?$apply="+url.QueryEscape("addnested(Lines,filter(Price gt 10) as FilteredLines)")+"&$filter="+url.QueryEscape("Region eq 'US'")+"&$top=1")
		rows, _ := decode(t, w)
		if got := ids(rows); !reflect.DeepEqual(got, []float64{2}) {
			t.Fatalf("got %v, want [2]", got)
		}
	})

	t.Run("leading nest unchanged", func(t *testing.T) {
		w := get(t, "/PipelineOrders?$apply="+url.QueryEscape("nest(aggregate(ID with sum as Total) as Totals)"))
		rows, _ := decode(t, w)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d: %s", len(rows), w.Body)
		}
		totals, ok := rows[0]["Totals"].([]interface{})
		if !ok || len(totals) != 1 || totals[0].(map[string]interface{})["Total"].(float64) != 6 {
			t.Fatalf("unexpected Totals: %v", rows[0]["Totals"])
		}
	})

	t.Run("concat tail nest unchanged", func(t *testing.T) {
		w := get(t, "/PipelineOrders?$apply="+url.QueryEscape("concat(filter(ID eq 1),filter(ID eq 2))/nest(aggregate(ID with sum as Total) as Totals)"))
		rows, _ := decode(t, w)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d: %s", len(rows), w.Body)
		}
		totals, ok := rows[0]["Totals"].([]interface{})
		if !ok || len(totals) != 1 || totals[0].(map[string]interface{})["Total"].(float64) != 3 {
			t.Fatalf("unexpected Totals: %v", rows[0]["Totals"])
		}
	})

	t.Run("navigation leading nest", func(t *testing.T) {
		w := get(t, "/PipelineOrders(1)/Lines?$apply="+url.QueryEscape("nest(aggregate(Price with sum as Total) as Totals)"))
		rows, _ := decode(t, w)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d: %s", len(rows), w.Body)
		}
		totals, ok := rows[0]["Totals"].([]interface{})
		if !ok || len(totals) != 1 || totals[0].(map[string]interface{})["Total"].(float64) != 25 {
			t.Fatalf("unexpected Totals: %v", rows[0]["Totals"])
		}
	})

	for _, tc := range []struct {
		name, target string
	}{
		{"non-leading nest after filter", "/PipelineOrders?$apply=" + url.QueryEscape("filter(Region eq 'EU')/nest(aggregate(ID with sum as Total) as Totals)")},
		{"non-leading nest after groupby", "/PipelineOrders?$apply=" + url.QueryEscape("groupby((Region))/nest(aggregate(Region with countdistinct as Regions) as Totals)")},
		{"non-leading addnested after filter", "/PipelineOrders?$apply=" + url.QueryEscape("filter(Region eq 'EU')/addnested(Lines,filter(Price gt 10) as FilteredLines)")},
		{"non-leading nest on navigation", "/PipelineOrders(1)/Lines?$apply=" + url.QueryEscape("filter(Price gt 1)/nest(aggregate(Price with sum as Total) as Totals)")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := get(t, tc.target)
			if w.Code != 400 {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body)
			}
			var body struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code == "" || body.Error.Message == "" || !strings.Contains(w.Body.String(), "non-leading") {
				t.Fatalf("unexpected error body: %s", w.Body)
			}
		})
	}
}
