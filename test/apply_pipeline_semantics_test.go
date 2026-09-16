package odata_test

import (
	"encoding/json"
	odata "github.com/nlstn/go-odata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

type PipelineNode struct {
	ID       int32         `json:"ID" gorm:"primaryKey;autoIncrement:false" odata:"key"`
	Name     string        `json:"Name"`
	ParentID *int32        `json:"ParentID"`
	Parent   *PipelineNode `json:"Parent,omitempty" gorm:"foreignKey:ParentID;references:ID"`
}

func (PipelineNode) TableName() string { return "PipelineNodes" }
func TestApplyPipelineSemantics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	if err = db.AutoMigrate(&PipelineNode{}); err != nil {
		t.Fatal(err)
	}
	one, two := int32(1), int32(2)
	for _, node := range []PipelineNode{{ID: 1, Name: "root"}, {ID: 2, Name: "child", ParentID: &one}, {ID: 3, Name: "leaf", ParentID: &two}} {
		if err = db.Create(&node).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc, err := odata.NewService(db)
	if err != nil {
		t.Fatal(err)
	}
	if err = svc.RegisterEntity(&PipelineNode{}); err != nil {
		t.Fatal(err)
	}
	if err = svc.RegisterEntityAnnotation("PipelineNodes", "Org.OData.Aggregation.V1.RecursiveHierarchy#Tree", map[string]interface{}{"NodeProperty": map[string]interface{}{"$PropertyPath": "ID"}, "ParentNavigationProperty": map[string]interface{}{"$NavigationPropertyPath": "Parent"}}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, expr string
		ids        []float64
	}{
		{"ancestors", "ancestors($root/PipelineNodes,Tree,ID,filter(ID eq 3))", []float64{1, 2}},
		{"filtered_intermediate", "filter(ID ne 2)/ancestors($root/PipelineNodes,Tree,ID,filter(ID eq 3),keep start)", []float64{1, 3}},
		{"postorder", "traverse($root/PipelineNodes,Tree,ID,postorder,ID asc)", []float64{3, 2, 1}},
		{"nested_concat", "filter(ID eq 1)/concat(concat(identity,identity),identity)", []float64{1, 1, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			svc.ServeHTTP(w, httptest.NewRequest("GET", "/PipelineNodes?$apply="+url.QueryEscape(tc.expr)+"&$count=true", nil))
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
			ids := make([]float64, 0, len(body.Value))
			for _, row := range body.Value {
				ids = append(ids, row["ID"].(float64))
			}
			if !reflect.DeepEqual(ids, tc.ids) || body.Count != float64(len(tc.ids)) {
				t.Fatalf("got %v count %v, want %v", ids, body.Count, tc.ids)
			}
		})
	}
	if err = db.Model(&PipelineNode{}).Where("id = ?", 1).Update("parent_id", 3).Error; err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	svc.ServeHTTP(w, httptest.NewRequest("GET", "/PipelineNodes?$apply="+url.QueryEscape("traverse($root/PipelineNodes,Tree,ID,preorder)"), nil))
	if w.Code != 400 {
		t.Fatalf("cycle: %d %s", w.Code, w.Body)
	}
}
