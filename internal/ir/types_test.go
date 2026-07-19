package ir_test

import (
	"strings"
	"testing"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	expr "github.com/jlkendrick/grimoire/internal/expr"
	ir "github.com/jlkendrick/grimoire/internal/ir"
)

func prim(name string) *ir.TypeInfo {
	return &ir.TypeInfo{Kind: descriptor.TypeKindPrimitive, Name: name}
}
func list(el *ir.TypeInfo) *ir.TypeInfo {
	return &ir.TypeInfo{Kind: descriptor.TypeKindList, Element: el}
}
func dict(el *ir.TypeInfo) *ir.TypeInfo {
	return &ir.TypeInfo{Kind: descriptor.TypeKindMap, Element: el}
}
func opt(el *ir.TypeInfo) *ir.TypeInfo {
	return &ir.TypeInfo{Kind: descriptor.TypeKindOptional, Element: el}
}
func structT(name string) *ir.TypeInfo {
	return &ir.TypeInfo{Kind: descriptor.TypeKindStruct, Name: name}
}
func noneT() *ir.TypeInfo {
	return &ir.TypeInfo{Kind: descriptor.TypeKindNone}
}

func TestWalkRef(t *testing.T) {
	for _, tc := range []struct {
		name      string
		t         *ir.TypeInfo
		accessors string
		want      *ir.TypeInfo // nil = Unknown
		wantErr   string
	}{
		{"no accessors returns the type", prim("int"), "", prim("int"), ""},
		{"map key yields element", dict(prim("int")), ".key", prim("int"), ""},
		{"list index yields element", list(prim("str")), "[0]", prim("str"), ""},
		{"nested composition", list(dict(prim("float"))), "[2].rate", prim("float"), ""},
		{"bare dict yields unknown", dict(nil), ".key", nil, ""},
		{"unknown absorbs accessors", nil, ".a[0].b", nil, ""},
		{"struct is opaque", structT("User"), ".name", nil, ""},
		{"optional walks through", opt(dict(prim("int"))), ".n", prim("int"), ""},
		{"field on primitive errors", prim("int"), ".x", nil, `cannot access field "x" on int`},
		{"index into map errors", dict(prim("int")), "[0]", nil, "cannot index [0] into dict"},
		{"field on none errors", noneT(), ".x", nil, `cannot access field "x" on none`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ir.WalkRef(tc.t, tc.accessors)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if (got == nil) != (tc.want == nil) {
				t.Fatalf("walked to %+v, want %+v", got, tc.want)
			}
			if got != nil && (got.Kind != tc.want.Kind || got.Name != tc.want.Name) {
				t.Errorf("walked to %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestCompatible(t *testing.T) {
	for _, tc := range []struct {
		name     string
		expected *ir.TypeInfo
		actual   *ir.TypeInfo
		ok       bool
	}{
		{"unknown expected passes", nil, prim("int"), true},
		{"unknown actual passes", prim("int"), nil, true},
		{"struct is opaque", prim("int"), structT("User"), true},
		{"same primitive", prim("int"), prim("int"), true},
		{"cross-language primitive", prim("str"), prim("string"), true},
		{"float widens from int", prim("float"), prim("int"), true},
		{"int does not narrow from float", prim("int"), prim("float"), false},
		{"str vs int", prim("str"), prim("int"), false},
		{"list recurses ok", list(prim("int")), list(prim("int")), true},
		{"list recurses mismatch", list(prim("int")), list(prim("str")), false},
		{"deep composition ok", list(dict(prim("int"))), list(dict(prim("int"))), true},
		{"bare container element passes", list(nil), list(prim("str")), true},
		{"map vs list", dict(prim("int")), list(prim("int")), false},
		{"primitive vs container", prim("int"), list(prim("int")), false},
		{"optional accepts value", opt(prim("int")), prim("int"), true},
		{"optional accepts none", opt(prim("int")), noneT(), true},
		{"optional into required is lenient", prim("int"), opt(prim("int")), true},
		{"optional into required still type-checks", prim("int"), opt(prim("str")), false},
		{"none into required errors", prim("int"), noneT(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ir.Compatible(tc.expected, tc.actual)
			if tc.ok && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Error("expected a wiring error")
			}
		})
	}
}

func TestInferExpr(t *testing.T) {
	env := map[string]*ir.TypeInfo{
		"res":   dict(prim("int")), // -> dict[str, int]
		"name":  prim("str"),
		"ok":    prim("bool"),
		"myst":  nil, // unannotated
		"items": list(prim("float")),
	}
	infer := func(t *testing.T, src string) (*ir.TypeInfo, error) {
		t.Helper()
		parsed, err := expr.ParseCondition(src)
		if err != nil {
			t.Fatalf("parse %q: %v", src, err)
		}
		return ir.InferExpr(parsed, env)
	}

	for _, tc := range []struct {
		name    string
		src     string
		want    string // canonical describe-ish: kind or primitive name; "unknown" for nil
		wantErr string
	}{
		{"the plan's example: res[key] is int", `res.key`, "int", ""},
		{"literal number", `42`, "float", ""},
		{"literal string", `"hi"`, "str", ""},
		{"comparison yields bool", `res.count > 1`, "bool", ""},
		{"logic on bools", `ok && res.flag > 0`, "bool", ""},
		{"unknown ref stays unknown", `myst.deep[3].thing`, "unknown", ""},
		{"list index", `items[0]`, "float", ""},
		{"ordering on str errors", `name > 3`, "", "requires numeric"},
		{"logic on str errors", `name && ok`, "", "requires bool"},
		{"not on non-bool errors", `!name`, "", "requires bool"},
		{"accessor on primitive errors", `name.length`, "", "cannot access field"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := infer(t, tc.src)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case tc.want == "unknown":
				if got != nil {
					t.Errorf("inferred %+v, want unknown", got)
				}
			case got == nil || got.Name != tc.want:
				t.Errorf("inferred %+v, want %s", got, tc.want)
			}
		})
	}
}
