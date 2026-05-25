package weave

import (
	"testing"

	parser "github.com/jlkendrick/grimoire/weave/parser"
)

func TestFlattenCallParamsPreserveLiteralTypes(t *testing.T) {
	src := `ritual r {
		classify(celsius = 35, enabled = true, label = "hot", probe = ref.field)
	}`
	ast_ritual, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tr := Transpiler{}
	ritual, err := tr.transpileRitual(ast_ritual)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	if len(ritual.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(ritual.Steps))
	}
	params := ritual.Steps[0].Params

	if v, ok := params["celsius"].(int); !ok || v != 35 {
		t.Errorf("celsius: got %#v, want int 35", params["celsius"])
	}
	if v, ok := params["enabled"].(bool); !ok || !v {
		t.Errorf("enabled: got %#v, want bool true", params["enabled"])
	}
	if v, ok := params["label"].(string); !ok || v != "hot" {
		t.Errorf("label: got %#v, want string %q", params["label"], "hot")
	}
	// Refs stay as strings so the engine can resolve them via bindings.
	if v, ok := params["probe"].(string); !ok || v != "ref.field" {
		t.Errorf("probe: got %#v, want string %q", params["probe"], "ref.field")
	}
}

func TestFlattenExprComposedQuotesStringLiterals(t *testing.T) {
	src := `ritual r {
		let extreme = temp.freezing || temp.category == "hot"
	}`
	ast_ritual, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tr := Transpiler{}
	ritual, err := tr.transpileRitual(ast_ritual)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	if len(ritual.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(ritual.Steps))
	}

	step := ritual.Steps[0]
	if step.Let != "extreme" {
		t.Errorf("let: got %q, want %q", step.Let, "extreme")
	}
	want := `temp.freezing || temp.category == "hot"`
	if step.Value != want {
		t.Errorf("value: got %q, want %q", step.Value, want)
	}
}
