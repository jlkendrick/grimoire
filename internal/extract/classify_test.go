package extract

import (
	"testing"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func TestClassifyPythonType(t *testing.T) {
	tests := []struct {
		raw      string
		kind     descriptor.TypeKind
		elemName string // expected Element.Name, "" if no element expected
	}{
		{"int", descriptor.TypeKindPrimitive, ""},
		{"str", descriptor.TypeKindPrimitive, ""},
		{"list[int]", descriptor.TypeKindList, "int"},
		{"List[str]", descriptor.TypeKindList, "str"},
		{"dict[str, int]", descriptor.TypeKindMap, "int"},
		{"Dict[str, Person]", descriptor.TypeKindMap, "Person"},
		{"Optional[str]", descriptor.TypeKindOptional, "str"},
		{"str | None", descriptor.TypeKindOptional, "str"},
		{"None | int", descriptor.TypeKindOptional, "int"},
		{"Person", descriptor.TypeKindStruct, ""},
		{"list[Person]", descriptor.TypeKindList, "Person"},
	}
	for _, tc := range tests {
		got := classifyPythonType(tc.raw)
		if got == nil {
			t.Errorf("%q: got nil", tc.raw)
			continue
		}
		if got.Kind != tc.kind {
			t.Errorf("%q: Kind = %q, want %q", tc.raw, got.Kind, tc.kind)
		}
		if tc.elemName != "" {
			if got.Element == nil || got.Element.Name != tc.elemName {
				t.Errorf("%q: Element = %+v, want Name %q", tc.raw, got.Element, tc.elemName)
			}
		}
	}
	if classifyPythonType("") != nil {
		t.Errorf("empty input should classify to nil")
	}
}

func TestClassifyGoType(t *testing.T) {
	tests := []struct {
		raw      string
		kind     descriptor.TypeKind
		elemName string
	}{
		{"int", descriptor.TypeKindPrimitive, ""},
		{"string", descriptor.TypeKindPrimitive, ""},
		{"float64", descriptor.TypeKindPrimitive, ""},
		{"[]string", descriptor.TypeKindList, "string"},
		{"[]Person", descriptor.TypeKindList, "Person"},
		{"map[string]int", descriptor.TypeKindMap, "int"},
		{"map[string]Person", descriptor.TypeKindMap, "Person"},
		{"*int", descriptor.TypeKindOptional, "int"},
		{"...int", descriptor.TypeKindList, "int"},
		{"Person", descriptor.TypeKindStruct, ""},
	}
	for _, tc := range tests {
		got := classifyGoType(tc.raw)
		if got == nil {
			t.Errorf("%q: got nil", tc.raw)
			continue
		}
		if got.Kind != tc.kind {
			t.Errorf("%q: Kind = %q, want %q", tc.raw, got.Kind, tc.kind)
		}
		if tc.elemName != "" {
			if got.Element == nil || got.Element.Name != tc.elemName {
				t.Errorf("%q: Element = %+v, want Name %q", tc.raw, got.Element, tc.elemName)
			}
		}
	}
}
