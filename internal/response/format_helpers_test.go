package response

import "testing"

func TestBuildEntityIDSingleStringKey(t *testing.T) {
	keyValues := map[string]interface{}{"ID": "ALFKI"}
	if id := BuildEntityID("Customers", keyValues); id != "Customers('ALFKI')" {
		t.Fatalf("expected Customers('ALFKI'), got %s", id)
	}
}

func TestBuildEntityIDCompositeOrdering(t *testing.T) {
	keyValues := map[string]interface{}{"LanguageKey": "EN", "ProductID": 1}
	expected := "ProductDescriptions(LanguageKey='EN',ProductID=1)"
	if id := BuildEntityID("ProductDescriptions", keyValues); id != expected {
		t.Fatalf("expected %s, got %s", expected, id)
	}
}
