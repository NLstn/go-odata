package response

import "testing"

func TestOmitNullValuesRemovesNullKeysFromMap(t *testing.T) {
	m := map[string]interface{}{
		"ID":          1,
		"Name":        "Widget",
		"Description": nil,
	}

	OmitNullValues(m)

	if _, ok := m["Description"]; ok {
		t.Errorf("expected Description to be removed, got %v", m)
	}
	if m["Name"] != "Widget" {
		t.Errorf("expected Name to be preserved, got %v", m["Name"])
	}
}

func TestOmitNullValuesRemovesTypedNilPointer(t *testing.T) {
	var nilStr *string
	m := map[string]interface{}{
		"Description": nilStr,
	}

	OmitNullValues(m)

	if _, ok := m["Description"]; ok {
		t.Errorf("expected typed-nil Description to be removed, got %v", m)
	}
}

func TestOmitNullValuesRemovesNullKeysFromOrderedMap(t *testing.T) {
	om := NewOrderedMap()
	om.Set("ID", 1)
	om.Set("Description", nil)

	OmitNullValues(om)

	if _, exists := om.values["Description"]; exists {
		t.Errorf("expected Description to be removed, got %v", om.values)
	}
	if om.values["ID"] != 1 {
		t.Errorf("expected ID to be preserved, got %v", om.values["ID"])
	}
}

func TestOmitNullValuesRecursesIntoNestedOrderedMaps(t *testing.T) {
	nested := NewOrderedMap()
	nested.Set("City", "Berlin")
	nested.Set("Zip", nil)

	om := NewOrderedMap()
	om.Set("Name", "Widget")
	om.Set("Address", nested)

	OmitNullValues(om)

	if _, exists := nested.values["Zip"]; exists {
		t.Errorf("expected nested Zip to be removed, got %v", nested.values)
	}
	if nested.values["City"] != "Berlin" {
		t.Errorf("expected nested City to be preserved, got %v", nested.values["City"])
	}
}
