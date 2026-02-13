package metadata

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type enumMethodInt int

type enumMethodUint16 uint16

type enumRegistered int16

type enumBadSignature int32

type enumNilMap int

type enumUnsupportedValue int

type enumInt8 int8

type enumUint8 uint8

type enumInt16 int16

type enumUint16 uint16

type enumInt32 int32

type enumUint32 uint32

type enumInt64 int64

type enumUint64 uint64

type enumInt int

type enumUint uint

func (enumMethodInt) EnumMembers() map[string]int {
	return map[string]int{
		"One":  1,
		"Zero": 0,
	}
}

func (enumMethodUint16) EnumMembers() map[string]uint16 {
	return map[string]uint16{
		"First":  1,
		"Second": 2,
	}
}

func (enumBadSignature) EnumMembers(_ int) map[string]int {
	return map[string]int{
		"Bad": 1,
	}
}

func (enumNilMap) EnumMembers() map[string]int {
	return nil
}

func (enumUnsupportedValue) EnumMembers() map[string]float64 {
	return map[string]float64{
		"Bad": 1.1,
	}
}

func withCleanEnumRegistry(t *testing.T) func() {
	t.Helper()
	enumRegistry.Lock()
	saved := make(map[reflect.Type][]EnumMember, len(enumRegistry.data))
	for key, members := range enumRegistry.data {
		copied := make([]EnumMember, len(members))
		copy(copied, members)
		saved[key] = copied
	}
	enumRegistry.data = make(map[reflect.Type][]EnumMember)
	enumRegistry.Unlock()

	return func() {
		enumRegistry.Lock()
		enumRegistry.data = saved
		enumRegistry.Unlock()
	}
}

func TestRegisterEnumMembersErrors(t *testing.T) {
	restore := withCleanEnumRegistry(t)
	defer restore()

	baseType := reflect.TypeOf(enumMethodInt(0))

	tests := []struct {
		name string
		err  string
		fn   func() error
	}{
		{
			name: "nil type",
			err:  "cannot be nil",
			fn: func() error {
				return RegisterEnumMembers(nil, []EnumMember{{Name: "A", Value: 1}})
			},
		},
		{
			name: "empty members",
			err:  "must have at least one member",
			fn: func() error {
				return RegisterEnumMembers(baseType, nil)
			},
		},
		{
			name: "duplicate names",
			err:  "duplicate member name",
			fn: func() error {
				return RegisterEnumMembers(baseType, []EnumMember{{Name: "A", Value: 1}, {Name: "A", Value: 2}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.err) {
				t.Fatalf("expected error to contain %q, got %q", tt.err, err.Error())
			}
		})
	}
}

func TestRegisterEnumMembersAllowsDuplicateValuesAndSortsByValueThenName(t *testing.T) {
	restore := withCleanEnumRegistry(t)
	defer restore()

	baseType := reflect.TypeOf(enumRegistered(0))
	err := RegisterEnumMembers(baseType, []EnumMember{
		{Name: "Beta", Value: 1},
		{Name: "Alpha", Value: 1},
		{Name: "Gamma", Value: 2},
	})
	if err != nil {
		t.Fatalf("RegisterEnumMembers() error = %v", err)
	}

	resolved, _, err := ResolveEnumMembers(baseType)
	if err != nil {
		t.Fatalf("ResolveEnumMembers() error = %v", err)
	}

	if len(resolved) != 3 {
		t.Fatalf("expected 3 enum members, got %d", len(resolved))
	}

	if resolved[0].Name != "Alpha" || resolved[0].Value != 1 {
		t.Fatalf("unexpected first member: %+v", resolved[0])
	}
	if resolved[1].Name != "Beta" || resolved[1].Value != 1 {
		t.Fatalf("unexpected second member: %+v", resolved[1])
	}
	if resolved[2].Name != "Gamma" || resolved[2].Value != 2 {
		t.Fatalf("unexpected third member: %+v", resolved[2])
	}
}

func TestResolveEnumMembersFromRegistry(t *testing.T) {
	restore := withCleanEnumRegistry(t)
	defer restore()

	baseType := reflect.TypeOf(enumRegistered(0))

	members := []EnumMember{
		{Name: "First", Value: 1},
		{Name: "Second", Value: 2},
	}
	if err := RegisterEnumMembers(baseType, members); err != nil {
		t.Fatalf("RegisterEnumMembers() error = %v", err)
	}

	resolved, resolvedType, err := ResolveEnumMembers(baseType)
	if err != nil {
		t.Fatalf("ResolveEnumMembers() error = %v", err)
	}
	if resolvedType != baseType {
		t.Fatalf("resolved type = %v, want %v", resolvedType, baseType)
	}
	if len(resolved) != len(members) {
		t.Fatalf("resolved members length = %d, want %d", len(resolved), len(members))
	}
	for i, member := range resolved {
		if member != members[i] {
			t.Fatalf("member[%d] = %+v, want %+v", i, member, members[i])
		}
	}
}

func TestResolveEnumMembersFromMethod(t *testing.T) {
	restore := withCleanEnumRegistry(t)
	defer restore()

	members, resolvedType, err := ResolveEnumMembers(reflect.TypeOf(enumMethodInt(0)))
	if err != nil {
		t.Fatalf("ResolveEnumMembers() error = %v", err)
	}
	if resolvedType.Kind() != reflect.Int {
		t.Fatalf("resolved type kind = %s, want %s", resolvedType.Kind(), reflect.Int)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
	if members[0].Name != "Zero" || members[1].Name != "One" {
		t.Fatalf("unexpected members: %+v", members)
	}

	additional, resolvedType, err := ResolveEnumMembers(reflect.TypeOf(enumMethodUint16(0)))
	if err != nil {
		t.Fatalf("ResolveEnumMembers() error = %v", err)
	}
	if resolvedType.Kind() != reflect.Uint16 {
		t.Fatalf("resolved type kind = %s, want %s", resolvedType.Kind(), reflect.Uint16)
	}
	if len(additional) != 2 {
		t.Fatalf("expected 2 members, got %d", len(additional))
	}
}

func TestExtractEnumMembersViaMethodErrors(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		err  string
	}{
		{
			name: "wrong signature",
			typ:  reflect.TypeOf(enumBadSignature(0)),
			err:  "must have signature",
		},
		{
			name: "nil map",
			typ:  reflect.TypeOf(enumNilMap(0)),
			err:  "returned nil",
		},
		{
			name: "unsupported value kind",
			typ:  reflect.TypeOf(enumUnsupportedValue(0)),
			err:  "must return map[string]<integer>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := extractEnumMembersViaMethod(tt.typ)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.err) {
				t.Fatalf("expected error to contain %q, got %q", tt.err, err.Error())
			}
		})
	}
}

func TestConvertEnumValueToInt64Overflow(t *testing.T) {
	_, err := convertEnumValueToInt64(reflect.ValueOf(uint64(math.MaxInt64) + 1))
	if err == nil {
		t.Fatal("expected overflow error")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetermineEnumUnderlyingType(t *testing.T) {
	tests := []struct {
		name    string
		typ     reflect.Type
		want    string
		wantErr bool
	}{
		{
			name: "int8",
			typ:  reflect.TypeOf(enumInt8(0)),
			want: "Edm.SByte",
		},
		{
			name: "uint8",
			typ:  reflect.TypeOf(enumUint8(0)),
			want: "Edm.Byte",
		},
		{
			name: "int16",
			typ:  reflect.TypeOf(enumInt16(0)),
			want: "Edm.Int16",
		},
		{
			name: "uint16",
			typ:  reflect.TypeOf(enumUint16(0)),
			want: "Edm.Int32",
		},
		{
			name: "int32",
			typ:  reflect.TypeOf(enumInt32(0)),
			want: "Edm.Int32",
		},
		{
			name: "uint32",
			typ:  reflect.TypeOf(enumUint32(0)),
			want: "Edm.Int64",
		},
		{
			name: "int64",
			typ:  reflect.TypeOf(enumInt64(0)),
			want: "Edm.Int64",
		},
		{
			name:    "uint64",
			typ:     reflect.TypeOf(enumUint64(0)),
			wantErr: true,
		},
		{
			name: "int",
			typ:  reflect.TypeOf(enumInt(0)),
			want: func() string {
				if strconv.IntSize == 32 {
					return "Edm.Int32"
				}
				return "Edm.Int64"
			}(),
		},
		{
			name: "uint",
			typ:  reflect.TypeOf(enumUint(0)),
			want: "Edm.Int64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetermineEnumUnderlyingType(tt.typ)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("DetermineEnumUnderlyingType() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("DetermineEnumUnderlyingType() = %s, want %s", got, tt.want)
			}
		})
	}
}
