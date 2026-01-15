package response

import (
	"reflect"
	"testing"

	"github.com/nlstn/go-odata/internal/metadata"
	"github.com/nlstn/go-odata/internal/query"
)

type expandToy struct {
	ID      int    `json:"id" odata:"key"`
	ChildID int    `json:"childId"`
	Name    string `json:"name"`
}

type expandChild struct {
	ID       int         `json:"id" odata:"key"`
	ParentID int         `json:"parentId"`
	Toys     []expandToy `json:"toys" gorm:"foreignKey:ChildID;references:ID"`
}

type expandParent struct {
	ID       int           `json:"id" odata:"key"`
	Children []expandChild `json:"children" gorm:"foreignKey:ParentID;references:ID"`
}

func TestApplyExpandOptionToValueNilExpandOption(t *testing.T) {
	value := map[string]interface{}{"name": "sample"}

	updated, count := ApplyExpandOptionToValue(value, nil, nil)

	if !reflect.DeepEqual(updated, value) {
		t.Fatalf("expected value to be unchanged, got %#v", updated)
	}
	if count != nil {
		t.Fatalf("expected nil count, got %v", *count)
	}
}

func TestExpandedCollectionCount(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		if count := expandedCollectionCount(nil); count != nil {
			t.Fatalf("expected nil count, got %v", *count)
		}
	})

	t.Run("pointer to slice", func(t *testing.T) {
		values := []string{"a", "b"}
		count := expandedCollectionCount(&values)
		if count == nil || *count != 2 {
			t.Fatalf("expected count 2, got %v", count)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		values := []int{}
		count := expandedCollectionCount(values)
		if count == nil || *count != 0 {
			t.Fatalf("expected count 0, got %v", count)
		}
	})

	t.Run("non-collection", func(t *testing.T) {
		if count := expandedCollectionCount("value"); count != nil {
			t.Fatalf("expected nil count, got %v", *count)
		}
	})
}

func TestApplyExpandOptionToValueNestedExpand(t *testing.T) {
	parentMeta := mustAnalyzeEntity(t, expandParent{})
	childMeta := mustAnalyzeEntity(t, expandChild{})
	toyMeta := mustAnalyzeEntity(t, expandToy{})

	registry := map[string]*metadata.EntityMetadata{
		parentMeta.EntityName: parentMeta,
		childMeta.EntityName:  childMeta,
		toyMeta.EntityName:    toyMeta,
	}
	parentMeta.SetEntitiesRegistry(registry)
	childMeta.SetEntitiesRegistry(registry)

	expandOpt := &query.ExpandOption{
		Expand: []query.ExpandOption{
			{
				NavigationProperty: "Children",
				Count:              true,
				Expand: []query.ExpandOption{
					{
						NavigationProperty: "Toys",
						Count:              true,
					},
				},
			},
		},
	}

	parent := expandParent{
		ID: 1,
		Children: []expandChild{
			{
				ID:       10,
				ParentID: 1,
				Toys: []expandToy{
					{ID: 100, ChildID: 10, Name: "Rocket"},
					{ID: 101, ChildID: 10, Name: "Puzzle"},
				},
			},
		},
	}

	updated, count := ApplyExpandOptionToValue(parent, expandOpt, parentMeta)
	if count != nil {
		t.Fatalf("expected nil count on parent value, got %v", *count)
	}

	assertExpandedChildren(t, updated)

	entityMap := map[string]interface{}{
		"id":       2,
		"children": parent.Children,
	}

	updatedMap, mapCount := ApplyExpandOptionToValue(entityMap, expandOpt, parentMeta)
	if mapCount != nil {
		t.Fatalf("expected nil count on map value, got %v", *mapCount)
	}

	assertExpandedChildren(t, updatedMap)
}

func mustAnalyzeEntity(t *testing.T, entity interface{}) *metadata.EntityMetadata {
	t.Helper()

	meta, err := metadata.AnalyzeEntity(entity)
	if err != nil {
		t.Fatalf("AnalyzeEntity() error = %v", err)
	}

	return meta
}

func assertExpandedChildren(t *testing.T, value interface{}) {
	t.Helper()

	entityMap, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", value)
	}

	childrenCount, ok := entityMap["children@odata.count"].(int)
	if !ok {
		t.Fatalf("expected children@odata.count to be int, got %T", entityMap["children@odata.count"])
	}
	if childrenCount != 1 {
		t.Fatalf("expected children@odata.count to be 1, got %d", childrenCount)
	}

	childrenRaw, ok := entityMap["children"]
	if !ok {
		t.Fatal("expected children key")
	}

	children, ok := childrenRaw.([]interface{})
	if !ok {
		t.Fatalf("expected children slice, got %T", childrenRaw)
	}
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}

	childMap, ok := children[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected child to be map, got %T", children[0])
	}

	toysCount, ok := childMap["toys@odata.count"].(int)
	if !ok {
		t.Fatalf("expected toys@odata.count to be int, got %T", childMap["toys@odata.count"])
	}
	if toysCount != 2 {
		t.Fatalf("expected toys@odata.count to be 2, got %d", toysCount)
	}

	toysRaw, ok := childMap["toys"]
	if !ok {
		t.Fatal("expected toys key")
	}

	toys, ok := toysRaw.([]expandToy)
	if !ok {
		t.Fatalf("expected toys to be []Toy, got %T", toysRaw)
	}
	if len(toys) != 2 {
		t.Fatalf("expected 2 toys, got %d", len(toys))
	}
}
