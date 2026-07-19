package response

import (
	"reflect"
	"testing"

	metadata "github.com/nlstn/go-odata/internal/metadata"
)

func TestGetEntityFieldPlan(t *testing.T) {
	type planEntity struct {
		ID       int     `json:"ID"`
		Name     string  `json:"Name"`
		Secret   string  `json:"-"`
		hidden   string  //nolint:unused // present to verify unexported fields are skipped
		Category *string `json:"Category"`
	}

	md := &metadata.EntityMetadata{
		Properties: []metadata.PropertyMetadata{
			{Name: "ID", JsonName: "ID"},
			{Name: "Name", JsonName: "Name"},
			{Name: "Category", JsonName: "Category", IsNavigationProp: true, NavigationTarget: "Categories", NavigationIsArray: true},
		},
	}

	plan := getEntityFieldPlan(md, reflect.TypeOf(planEntity{}))
	if len(plan.entries) != reflect.TypeOf(planEntity{}).NumField() {
		t.Fatalf("plan has %d entries, want %d", len(plan.entries), reflect.TypeOf(planEntity{}).NumField())
	}

	// Field 0: ID — plain, fullProp resolved, not nav.
	if e := plan.entries[0]; e.skip || e.navProp != nil || e.fullProp == nil || e.jsonName != "ID" {
		t.Errorf("ID entry = %+v, want plain resolved field", e)
	}
	// Field 2: Secret (json:"-") — skipped.
	if e := plan.entries[2]; !e.skip {
		t.Errorf("Secret entry = %+v, want skip", e)
	}
	// Field 3: hidden (unexported) — skipped.
	if e := plan.entries[3]; !e.skip {
		t.Errorf("hidden entry = %+v, want skip", e)
	}
	// Field 4: Category — navigation, navProp synthesized with matching values.
	e := plan.entries[4]
	if e.skip || e.navProp == nil {
		t.Fatalf("Category entry = %+v, want navigation with navProp", e)
	}
	if !e.navProp.IsNavigationProp || e.navProp.Name != "Category" || e.navProp.JsonName != "Category" ||
		e.navProp.NavigationTarget != "Categories" || !e.navProp.NavigationIsArray {
		t.Errorf("Category navProp = %+v, want values copied from metadata", e.navProp)
	}
}

func TestGetEntityFieldPlan_Cached(t *testing.T) {
	type cachedEntity struct {
		ID int `json:"ID"`
	}
	md := &metadata.EntityMetadata{Properties: []metadata.PropertyMetadata{{Name: "ID", JsonName: "ID"}}}
	p1 := getEntityFieldPlan(md, reflect.TypeOf(cachedEntity{}))
	p2 := getEntityFieldPlan(md, reflect.TypeOf(cachedEntity{}))
	if p1 != p2 {
		t.Error("expected the plan to be cached and reused (same pointer)")
	}
}
