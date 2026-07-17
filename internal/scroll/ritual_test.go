package scroll_test

import (
	"encoding/json"
	"testing"

	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

func TestRitualHash_StableAndOrderSensitive(t *testing.T) {
	r1 := scroll.Ritual{
		Command: "pipe",
		Steps:   []scroll.Step{{Spell: "a"}, {Spell: "b"}},
	}
	r2 := scroll.Ritual{
		Command: "pipe",
		Steps:   []scroll.Step{{Spell: "b"}, {Spell: "a"}},
	}

	h1a, err := r1.Hash()
	if err != nil {
		t.Fatalf("Hash r1: %v", err)
	}
	h1b, err := r1.Hash()
	if err != nil {
		t.Fatalf("Hash r1 again: %v", err)
	}
	if h1a != h1b {
		t.Errorf("Ritual.Hash unstable: %q vs %q", h1a, h1b)
	}

	h2, err := r2.Hash()
	if err != nil {
		t.Fatalf("Hash r2: %v", err)
	}
	if h1a == h2 {
		t.Errorf("expected reordered Steps to produce a different hash; both = %q", h1a)
	}
}

// TestRitualHash_SpellOnlyRitualUnaffectedByNewFields confirms a ritual
// with only spell-steps serializes exactly as it did before the newer
// step kinds (if/let/print) existed. omitempty on the new JSON tags is
// what keeps this invariant — if it regresses, every existing cached
// ritual would reconcile on first run after upgrade.
func TestRitualHash_SpellOnlyRitualUnaffectedByNewFields(t *testing.T) {
	r := scroll.Ritual{
		Command: "pipe",
		Steps:   []scroll.Step{{Id: "a", Spell: "foo"}, {Spell: "bar"}},
	}
	js, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// The canonical JSON the hash is computed over. Newer step fields
	// must not appear when unset; the always-present fields (Id, Spell,
	// Params) must keep their shape.
	want := `{"Command":"pipe","Steps":[{"Id":"a","Spell":"foo","Params":null},{"Id":"","Spell":"bar","Params":null}]}`
	if string(js) != want {
		t.Errorf("canonical JSON changed:\n got  %s\n want %s", js, want)
	}
}

func TestRitualHash_IfStepChangesHash(t *testing.T) {
	linear := scroll.Ritual{
		Command: "pipe",
		Steps: []scroll.Step{
			{Id: "check", Spell: "get_status"},
			{Spell: "handle"},
		},
	}
	branched := scroll.Ritual{
		Command: "pipe",
		Steps: []scroll.Step{
			{Id: "check", Spell: "get_status"},
			{If: "check.ok", Then: []scroll.Step{{Spell: "handle"}}},
		},
	}
	hLinear, err := linear.Hash()
	if err != nil {
		t.Fatalf("Hash linear: %v", err)
	}
	hBranched, err := branched.Hash()
	if err != nil {
		t.Fatalf("Hash branched: %v", err)
	}
	if hLinear == hBranched {
		t.Errorf("expected wrapping a step in an if-step to change the hash")
	}
}

func TestRitualHash_ElseBranchChangesHash(t *testing.T) {
	withoutElse := scroll.Ritual{
		Command: "pipe",
		Steps: []scroll.Step{
			{Id: "check", Spell: "get_status"},
			{If: "check.ok", Then: []scroll.Step{{Spell: "ok"}}},
		},
	}
	withElse := scroll.Ritual{
		Command: "pipe",
		Steps: []scroll.Step{
			{Id: "check", Spell: "get_status"},
			{If: "check.ok",
				Then: []scroll.Step{{Spell: "ok"}},
				Else: []scroll.Step{{Spell: "fallback"}},
			},
		},
	}
	h1, err := withoutElse.Hash()
	if err != nil {
		t.Fatalf("Hash withoutElse: %v", err)
	}
	h2, err := withElse.Hash()
	if err != nil {
		t.Fatalf("Hash withElse: %v", err)
	}
	if h1 == h2 {
		t.Errorf("expected adding an else branch to change the hash")
	}
}

func TestRitualHash_PrintStepChangesHash(t *testing.T) {
	without := scroll.Ritual{
		Command: "pipe",
		Steps:   []scroll.Step{{Id: "a", Spell: "foo"}},
	}
	with := scroll.Ritual{
		Command: "pipe",
		Steps:   []scroll.Step{{Id: "a", Spell: "foo"}, {Print: "a.result"}},
	}
	h1, err := without.Hash()
	if err != nil {
		t.Fatalf("Hash without: %v", err)
	}
	h2, err := with.Hash()
	if err != nil {
		t.Fatalf("Hash with: %v", err)
	}
	if h1 == h2 {
		t.Errorf("expected adding a print step to change the hash")
	}
}

func TestRitualHash_ModeChangesHash(t *testing.T) {
	pipe := scroll.Ritual{
		Command: "r",
		Steps:   []scroll.Step{{Id: "a", Spell: "foo"}},
	}
	graphMode := scroll.Ritual{
		Command: "r",
		Mode:    "graph",
		Steps:   []scroll.Step{{Id: "a", Spell: "foo"}},
	}
	h1, err := pipe.Hash()
	if err != nil {
		t.Fatalf("Hash pipe: %v", err)
	}
	h2, err := graphMode.Hash()
	if err != nil {
		t.Fatalf("Hash graph: %v", err)
	}
	if h1 == h2 {
		t.Errorf("expected mode: to change the ritual hash")
	}
}

func TestStep_Kind(t *testing.T) {
	tests := []struct {
		name string
		step scroll.Step
		want string
	}{
		{"spell step", scroll.Step{Spell: "foo"}, "spell"},
		{"id'd spell step", scroll.Step{Id: "x", Spell: "foo"}, "spell"},
		{"if step", scroll.Step{If: "cond", Then: []scroll.Step{{Spell: "a"}}}, "if"},
		{"let step", scroll.Step{Let: "x", Value: "1 > 0"}, "let"},
		{"print step", scroll.Step{Print: "a.result"}, "print"},
		{"empty step defaults to spell", scroll.Step{}, "spell"},
	}
	for _, tt := range tests {
		if got := tt.step.Kind(); got != tt.want {
			t.Errorf("%s: Kind() = %q, want %q", tt.name, got, tt.want)
		}
	}
}
