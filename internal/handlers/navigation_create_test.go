package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNavigationCreateExplicitBinding(t *testing.T) {
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"Name":"New","Parent@odata.bind":"NavTestParents(1)"}`, http.StatusCreated},
		{`{"Name":"New","Parent@odata.bind":"NavTestParents(2)"}`, http.StatusBadRequest},
		{`{"Name":"New","ParentID":"ignored"}`, http.StatusCreated},
	} {
		t.Run(tc.body, func(t *testing.T) {
			handler, db := setupNavTestHandler(t)
			if err := db.Create(&[]NavTestParent{{ID: 1}, {ID: 2}}).Error; err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodPost, "/NavTestParents(1)/Children", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.HandleNavigationProperty(w, r, "1", "Children", false)
			if w.Code != tc.status {
				t.Fatalf("status %d, want %d: %s", w.Code, tc.status, w.Body.String())
			}
			var children []NavTestChild
			if err := db.Find(&children).Error; err != nil {
				t.Fatal(err)
			}
			if tc.status == http.StatusCreated {
				if len(children) != 1 || children[0].ParentID != 1 {
					t.Fatalf("wrong relationship: %+v", children)
				}
			} else if len(children) != 0 {
				t.Fatal("contradictory binding created an entity")
			}
		})
	}
}
