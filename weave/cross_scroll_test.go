package weave

import (
	"testing"

	parser "github.com/jlkendrick/grimoire/weave/parser"
)

func TestFlattenCall_BareSpellName(t *testing.T) {
	src := `ritual r {
		deploy()
	}`
	ast_ritual, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tr := Transpiler{}
	ritual, err := tr.TranspileRitual(ast_ritual)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	if len(ritual.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(ritual.Steps))
	}
	if ritual.Steps[0].Spell != "deploy" {
		t.Errorf("Spell = %q, want %q", ritual.Steps[0].Spell, "deploy")
	}
}

func TestFlattenCall_ModuleQualifiedSpellName(t *testing.T) {
	src := `ritual r {
		other.deploy()
	}`
	ast_ritual, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tr := Transpiler{}
	ritual, err := tr.TranspileRitual(ast_ritual)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	if len(ritual.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(ritual.Steps))
	}
	if ritual.Steps[0].Spell != "other.deploy" {
		t.Errorf("Spell = %q, want %q", ritual.Steps[0].Spell, "other.deploy")
	}
}

func TestFlattenCall_MixedBareAndQualified(t *testing.T) {
	src := `ritual r {
		seed()
		other.transform()
		print()
	}`
	ast_ritual, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tr := Transpiler{}
	ritual, err := tr.TranspileRitual(ast_ritual)
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	if len(ritual.Steps) != 3 {
		t.Fatalf("steps: got %d, want 3", len(ritual.Steps))
	}
	want := []string{"seed", "other.transform", "print"}
	for i, w := range want {
		if ritual.Steps[i].Spell != w {
			t.Errorf("Steps[%d].Spell = %q, want %q", i, ritual.Steps[i].Spell, w)
		}
	}
}
