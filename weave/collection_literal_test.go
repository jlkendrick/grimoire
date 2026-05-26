package weave

import (
	"reflect"
	"testing"

	parser "github.com/jlkendrick/grimoire/weave/parser"
)

func transpile(t *testing.T, src string) (steps []scrollStep, err error) {
	t.Helper()
	ast_ritual, perr := parser.Parse(src)
	if perr != nil {
		t.Fatalf("parse: %v", perr)
	}
	tr := Transpiler{}
	ritual, terr := tr.TranspileRitual(ast_ritual)
	if terr != nil {
		return nil, terr
	}
	out := make([]scrollStep, len(ritual.Steps))
	for i, s := range ritual.Steps {
		out[i] = scrollStep{Params: s.Params}
	}
	return out, nil
}

type scrollStep struct {
	Params map[string]any
}

func TestCollectionLiterals_AsCallArgs(t *testing.T) {
	src := `ritual r {
		summarize(tags = ["vip", "new"], scores = [1, 2, 3], person = { Name = "james", Age = 20 })
	}`
	steps, err := transpile(t, src)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(steps))
	}
	params := steps[0].Params

	if got := params["tags"]; !reflect.DeepEqual(got, []any{"vip", "new"}) {
		t.Errorf("tags = %#v, want [vip new]", got)
	}
	if got := params["scores"]; !reflect.DeepEqual(got, []any{1, 2, 3}) {
		t.Errorf("scores = %#v, want [1 2 3]", got)
	}
	wantPerson := map[string]any{"Name": "james", "Age": 20}
	if got := params["person"]; !reflect.DeepEqual(got, wantPerson) {
		t.Errorf("person = %#v, want %#v", got, wantPerson)
	}
}

func TestCollectionLiterals_NestedData(t *testing.T) {
	src := `ritual r {
		f(rows = [{ k = 1 }, { k = 2 }])
	}`
	steps, err := transpile(t, src)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	want := []any{map[string]any{"k": 1}, map[string]any{"k": 2}}
	if got := steps[0].Params["rows"]; !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %#v, want %#v", got, want)
	}
}

func TestCollectionLiterals_RejectedContexts(t *testing.T) {
	cases := map[string]string{
		"let value":            `ritual r { let x = [1, 2] }`,
		"operator expr":        `ritual r { let x = [1] == [1] }`,
		"if condition":         `ritual r { if [1] { f() } }`,
		"ref inside list":      `ritual r { f(xs = [a.b]) }`,
		"call inside list":     `ritual r { f(xs = [g()]) }`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := transpile(t, src); err == nil {
				t.Errorf("%s: expected transpile error, got nil", name)
			}
		})
	}
}
