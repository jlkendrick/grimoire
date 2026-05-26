package runtime

import (
	"testing"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func TestQualifyGoType(t *testing.T) {
	prim := func(n string) *descriptor.TypeInfo {
		return &descriptor.TypeInfo{Kind: descriptor.TypeKindPrimitive, Name: n}
	}
	tests := []struct {
		info *descriptor.TypeInfo
		want string
	}{
		{prim("int"), "int"},
		{prim("string"), "string"},
		{&descriptor.TypeInfo{Kind: descriptor.TypeKindStruct, Name: "Person"}, "userpkg.Person"},
		{&descriptor.TypeInfo{Kind: descriptor.TypeKindList, Element: prim("string"), Name: "[]string"}, "[]string"},
		{&descriptor.TypeInfo{Kind: descriptor.TypeKindList, Element: &descriptor.TypeInfo{Kind: descriptor.TypeKindStruct, Name: "Person"}, Name: "[]Person"}, "[]userpkg.Person"},
		{&descriptor.TypeInfo{Kind: descriptor.TypeKindMap, Element: prim("int"), Name: "map[string]int"}, "map[string]int"},
		{&descriptor.TypeInfo{Kind: descriptor.TypeKindMap, Element: &descriptor.TypeInfo{Kind: descriptor.TypeKindStruct, Name: "Person"}, Name: "map[string]Person"}, "map[string]userpkg.Person"},
		{&descriptor.TypeInfo{Kind: descriptor.TypeKindOptional, Element: &descriptor.TypeInfo{Kind: descriptor.TypeKindStruct, Name: "Person"}, Name: "*Person"}, "*userpkg.Person"},
		{nil, "interface{}"},
	}
	for _, tc := range tests {
		if got := qualifyGoType(tc.info); got != tc.want {
			t.Errorf("qualifyGoType(%+v) = %q, want %q", tc.info, got, tc.want)
		}
	}
}
