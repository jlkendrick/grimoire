package ir_test

import (
	"path/filepath"
	"testing"

	ir "github.com/jlkendrick/grimoire/internal/ir"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	ts "github.com/jlkendrick/grimoire/internal/testsupport"
)

func TestFromSpell_ExtractsAndMergesOverrides(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	ts.WriteFile(t, filepath.Join(dir, "greet.py"), `def greet(name: str = "world", times: int = 1):
    return name * times
`)

	s := scroll.Spell{
		Command:    "greet",
		Path:       "greet.py",
		Function:   "greet",
		Params:     map[string]any{"name": "james"}, // scroll override
		ScrollPath: filepath.Join(dir, "scroll.yaml"),
	}

	fn, err := ir.FromSpell(s)
	if err != nil {
		t.Fatalf("FromSpell: %v", err)
	}

	if fn.CommandName != "greet" || fn.FunctionName != "greet" {
		t.Errorf("names = %q/%q, want greet/greet", fn.CommandName, fn.FunctionName)
	}
	if fn.AbsPathToSourceFile != filepath.Join(dir, "greet.py") {
		t.Errorf("abs path = %q", fn.AbsPathToSourceFile)
	}
	if fn.SpellHash == "" {
		t.Error("SpellHash not set — sync would re-extract every run")
	}
	if len(fn.Params) != 2 || fn.Params[0].Name != "name" || fn.Params[1].Name != "times" {
		t.Fatalf("params = %+v, want [name times] from source extraction", fn.Params)
	}
	if fn.Params[0].Default != "james" {
		t.Errorf("name default = %v, want scroll override to win over source default", fn.Params[0].Default)
	}
}

func TestFromSpell_MissingFunctionErrors(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	ts.WriteFile(t, filepath.Join(dir, "greet.py"), "def greet():\n    return 1\n")

	s := scroll.Spell{
		Command:    "ghost",
		Path:       "greet.py",
		Function:   "ghost",
		ScrollPath: filepath.Join(dir, "scroll.yaml"),
	}
	if _, err := ir.FromSpell(s); err == nil {
		t.Error("expected extraction error for a function that does not exist in source")
	}
}
