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
		Params:   []scroll.Param{{Name: "name", Type: "str", Default: "world"}},
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
		Params: []scroll.Param{{Name: "name", Type: "str", Default: "world"}},
	}
	b := a
	b.Params = []scroll.Param{{Name: "name", Type: "str", Default: "moon"}}

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

// TestParamUnmarshalYAML_StringifiesIntDefault verifies the canonicalization
// that lets the descriptor IR carry a single string form for defaults.
func TestParamUnmarshalYAML_StringifiesIntDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scroll.yaml")
	ts.WriteFile(t, path, `spells:
  - command: c
    path: c.py
    function: c
    params:
      - name: n
        type: int
        default: 5
`)
	s, err := scroll.ParseScroll(path)
	if err != nil {
		t.Fatalf("ParseScroll: %v", err)
	}
	if len(s.Spells) != 1 || len(s.Spells[0].Params) != 1 {
		t.Fatalf("unexpected parse shape: %+v", s)
	}
	got := s.Spells[0].Params[0].Default
	if got != "5" {
		t.Errorf("Default = %#v, want %q (string-coerced)", got, "5")
	}
}

func TestParamUnmarshalYAML_PreservesStringDefault(t *testing.T) {
	var p scroll.Param
	if err := yaml.Unmarshal([]byte("name: x\ntype: str\ndefault: hello\n"), &p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if p.Default != "hello" {
		t.Errorf("Default = %#v, want %q", p.Default, "hello")
	}
}

func TestParamUnmarshalYAML_NilDefaultStaysNil(t *testing.T) {
	var p scroll.Param
	if err := yaml.Unmarshal([]byte("name: x\ntype: str\n"), &p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if p.Default != nil {
		t.Errorf("Default = %#v, want nil", p.Default)
	}
}
