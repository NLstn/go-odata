package odata_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestIEEE754CompatibleExpandedCount(t *testing.T) {
	service, _ := setupNavKeyPredicateService(t)
	for _, selectValue := range []string{"", "ID,Products"} {
		query := url.Values{"$expand": {"Products($count=true)"}}
		if selectValue != "" {
			query.Set("$select", selectValue)
		}
		r := httptest.NewRequest(http.MethodGet, "/NavKeyPredicateCategories?"+query.Encode(), nil)
		r.Header.Set("Accept", "application/json;IEEE754Compatible=true")
		w := httptest.NewRecorder()
		service.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
		var body struct {
			Value []map[string]interface{} `json:"value"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Value) != 1 || body.Value[0]["Products@odata.count"] != "2" {
			t.Fatalf("expected a string expanded count: %s", w.Body.String())
		}
	}
}

func TestConstantBooleanFilterCollectionAndCount(t *testing.T) {
	service, _ := setupNavKeyPredicateService(t)
	for _, version := range []string{"4.0", "4.01"} {
		for _, tc := range []struct {
			filter string
			count  int
		}{
			{"false", 0}, {"true", 3}, {"not false", 3}, {"not true", 0},
			{"false and ID eq 1", 0}, {"false or ID eq 1", 1},
		} {
			t.Run(version+"/"+tc.filter, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/NavKeyPredicateProducts?"+url.Values{"$filter": {tc.filter}, "$count": {"true"}}.Encode(), nil)
				req.Header.Set("OData-MaxVersion", version)
				w := httptest.NewRecorder()
				service.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Fatalf("status %d: %s", w.Code, w.Body.String())
				}
				var body struct {
					Count int                      `json:"@odata.count"`
					Value []NavKeyPredicateProduct `json:"value"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Count != tc.count || len(body.Value) != tc.count {
					t.Fatalf("count=%d rows=%d, want %d", body.Count, len(body.Value), tc.count)
				}
			})
		}
	}
}

func TestInvalidPagingAndCountValues(t *testing.T) {
	service, _ := setupNavKeyPredicateService(t)
	for _, version := range []string{"4.0", "4.01"} {
		for _, option := range []string{"$top", "$skip", "$count"} {
			for _, value := range []string{"", "1.5", "1junk", "1e2", "+1", "-1", " 1", "1 ", "9999999999999999999999999999"} {
				t.Run(version+"/"+option+"="+value, func(t *testing.T) {
					req := httptest.NewRequest(http.MethodGet, "/NavKeyPredicateProducts?"+url.Values{option: {value}}.Encode(), nil)
					req.Header.Set("OData-MaxVersion", version)
					w := httptest.NewRecorder()
					service.ServeHTTP(w, req)
					if w.Code != http.StatusBadRequest {
						t.Fatalf("status %d: %s", w.Code, w.Body.String())
					}
					var body map[string]interface{}
					if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body["error"] == nil {
						t.Fatalf("expected OData error: %s", w.Body.String())
					}
				})
			}
		}
	}
}

func TestCreateThroughCollectionNavigation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		path   string
		body   string
		status int
	}{
		{"implicit foreign key", "/NavKeyPredicateCategories(1)/Products", `{"Name":"New product"}`, http.StatusCreated},
		{"ignore supplied foreign key", "/NavKeyPredicateCategories(1)/Products", `{"Name":"New product","CategoryID":999}`, http.StatusCreated},
		{"missing parent", "/NavKeyPredicateCategories(999)/Products", `{"Name":"New product"}`, http.StatusNotFound},
		{"invalid payload", "/NavKeyPredicateCategories(1)/Products", `{"Unknown":1}`, http.StatusBadRequest},
		{"individual navigation entity", "/NavKeyPredicateCategories(1)/Products(1)", `{"Name":"New product"}`, http.StatusMethodNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, db := setupNavKeyPredicateService(t)
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			service.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status %d, want %d: %s", w.Code, tc.status, w.Body.String())
			}
			var products []NavKeyPredicateProduct
			if err := db.Where("name = ?", "New product").Find(&products).Error; err != nil {
				t.Fatal(err)
			}
			if tc.status != http.StatusCreated {
				if len(products) != 0 {
					t.Fatal("failed create persisted an entity")
				}
				return
			}
			if len(products) != 1 || products[0].CategoryID == nil || *products[0].CategoryID != 1 {
				t.Fatalf("entity was not linked to parent: %+v", products)
			}
			if w.Header().Get("Location") == "" {
				t.Fatal("missing Location")
			}
			read := httptest.NewRecorder()
			service.ServeHTTP(read, httptest.NewRequest(http.MethodGet, w.Header().Get("Location"), nil))
			if read.Code != http.StatusOK {
				t.Fatalf("created entity cannot be read: %s", read.Body.String())
			}
		})
	}
}
