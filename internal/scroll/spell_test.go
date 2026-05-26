package scroll_test

import (
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"

	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	ts "github.com/jlkendrick/grimoire/internal/testsupport"
)

func TestSpellHash_StableAcrossCalls(t *testing.T) {
	s := scroll.Spell{
		Command:  "greet",
		Path:     "greet.py",
		Function: "greet",
		Params:   map[string]any{"name": "world"},
	}
	h1, err := s.Hash()
	if err != nil {
		t.Fatalf("Hash 1: %v", err)
	}
	h2, err := s.Hash()
	if err != nil {
		t.Fatalf("Hash 2: %v", err)
	}
	if h1 != h2 {
		t.Errorf("Spell.Hash unstable: %q vs %q", h1, h2)
	}
}

func TestSpellHash_ChangesWithParamDefault(t *testing.T) {
	a := scroll.Spell{
		Command: "greet", Path: "greet.py", Function: "greet",
		Params: map[string]any{"name": "world"},
	}
	b := a
	b.Params = map[string]any{"name": "moon"}

	ha, err := a.Hash()
	if err != nil {
		t.Fatalf("Hash a: %v", err)
	}
	hb, err := b.Hash()
	if err != nil {
		t.Fatalf("Hash b: %v", err)
	}
	if ha == hb {
		t.Errorf("expected different hashes for different defaults; both = %q", ha)
	}
}

// TestSpellHash_IgnoresRuntimeFields confirms ScrollPath and AbsPath (json:"-")
// don't perturb the hash. Reconciler relies on this so re-parsing the same
// scroll doesn't trigger spurious re-extraction.
func TestSpellHash_IgnoresRuntimeFields(t *testing.T) {
	a := scroll.Spell{Command: "x", Path: "x.py", Function: "x"}
	b := a
	b.ScrollPath = "/tmp/scroll.yaml"
	b.AbsPath = "/tmp/x.py"

	ha, err := a.Hash()
	if err != nil {
		t.Fatalf("Hash a: %v", err)
	}
	hb, err := b.Hash()
	if err != nil {
		t.Fatalf("Hash b: %v", err)
	}
	if ha != hb {
		t.Errorf("runtime fields shouldn't affect hash: %q vs %q", ha, hb)
	}
}

// TestParamUnmarshalYAML_PreservesIntDefault verifies that int value overrides
// arrive in the IR as a numeric Go type, not stringified — that's the signal
// flag registration uses to construct typed cobra defaults without re-parsing.
func TestParamUnmarshalYAML_PreservesIntDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scroll.yaml")
	ts.WriteFile(t, path, `spells:
  - command: c
    path: c.py
    function: c
    params:
      n: 5
`)
	s, err := scroll.ParseScroll(path)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	if len(s.Spells) != 1 || len(s.Spells[0].Params) != 1 {
		t.Fatalf("unexpected parse shape: %+v", s)
	}
	got := s.Spells[0].Params["n"]
	switch v := got.(type) {
	case int:
		if v != 5 { t.Errorf("Default = %d, want 5", v) }
	case int64:
		if v != 5 { t.Errorf("Default = %d, want 5", v) }
	case uint64:
		if v != 5 { t.Errorf("Default = %d, want 5", v) }
	case float64:
		if v != 5 { t.Errorf("Default = %v, want 5", v) }
	default:
		t.Errorf("Default = %#v (%T), want a numeric Go type (not stringified)", got, got)
	}
}

func TestParamUnmarshalYAML_PreservesStringDefault(t *testing.T) {
	var s scroll.Scroll
	if err := yaml.Unmarshal([]byte("spells:\n  - command: c\n    path: c.py\n    function: c\n    params:\n      name: hello\n"), &s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := s.Spells[0].Params["name"]; got != "hello" {
		t.Errorf("Default = %#v, want %q", got, "hello")
	}
}

func TestParamUnmarshalYAML_NoParamsStaysEmpty(t *testing.T) {
	var s scroll.Scroll
	if err := yaml.Unmarshal([]byte("spells:\n  - command: c\n    path: c.py\n    function: c\n"), &s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(s.Spells[0].Params) != 0 {
		t.Errorf("Params = %#v, want empty", s.Spells[0].Params)
	}
}
