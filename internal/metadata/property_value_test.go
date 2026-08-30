package metadata

import "testing"

func TestReadPropertyValue(t *testing.T) {
	t.Parallel()

	type entity struct {
		ID int
	}
	type row map[string]interface{}

	property := PropertyMetadata{Name: "ID", FieldName: "ID", JsonName: "id"}
	tests := []struct {
		name   string
		entity interface{}
		want   interface{}
		ok     bool
	}{
		{name: "struct", entity: entity{ID: 1}, want: 1, ok: true},
		{name: "struct pointer", entity: &entity{ID: 2}, want: 2, ok: true},
		{name: "map JSON name", entity: map[string]interface{}{"id": 3}, want: 3, ok: true},
		{name: "named map", entity: row{"id": 4}, want: 4, ok: true},
		{name: "map field name fallback", entity: map[string]interface{}{"ID": 5}, want: 5, ok: true},
		{name: "nil", entity: nil, ok: false},
		{name: "nil pointer", entity: (*entity)(nil), ok: false},
		{name: "unsupported", entity: []int{1}, ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ReadPropertyValue(test.entity, property)
			if ok != test.ok {
				t.Fatalf("ReadPropertyValue() ok = %v, want %v", ok, test.ok)
			}
			if got != test.want {
				t.Fatalf("ReadPropertyValue() = %v, want %v", got, test.want)
			}
		})
	}
}
