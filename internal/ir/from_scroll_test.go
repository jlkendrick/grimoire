package ir_test

import (
	"reflect"
	"testing"

	ir "github.com/jlkendrick/grimoire/internal/ir"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

func TestFromRitual_LowersAllStepKinds(t *testing.T) {
	r := &scroll.Ritual{
		Command: "report",
		Steps: []scroll.Step{
			{Id: "check", Spell: "fetch", Params: map[string]any{"n": 3}},
			{Let: "big", Value: "check.n > 1"},
			{If: "big",
				Then: []scroll.Step{{Print: "check.n"}},
				Else: []scroll.Step{{Spell: "fallback"}}},
			{Print: "check"},
		},
	}

	got := ir.FromRitual(r)
	want := &ir.Pipeline{
		Command: "report",
		Steps: []ir.Step{
			{Id: "check", Spell: "fetch", Params: map[string]any{"n": 3}},
			{Let: "big", Value: "check.n > 1"},
			{If: "big",
				Then: []ir.Step{{Print: "check.n"}},
				Else: []ir.Step{{Spell: "fallback"}}},
			{Print: "check"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FromRitual =\n%+v\nwant\n%+v", got, want)
	}
}

// The user model and the IR must classify every lowered step identically,
// or validation would judge a different step kind than the user wrote.
func TestFromRitual_KindsSurviveLowering(t *testing.T) {
	steps := []scroll.Step{
		{Spell: "a"},
		{Let: "x", Value: "1 > 0"},
		{If: "c", Then: []scroll.Step{{Spell: "b"}}},
		{Print: "x"},
	}
	p := ir.FromRitual(&scroll.Ritual{Command: "r", Steps: steps})
	for i := range steps {
		if got, want := p.Steps[i].Kind(), steps[i].Kind(); got != want {
			t.Errorf("step %d: ir kind %q != scroll kind %q", i, got, want)
		}
	}
}
