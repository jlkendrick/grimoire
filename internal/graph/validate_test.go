package graph

import (
	"fmt"
	"strings"
	"testing"

	ir "github.com/jlkendrick/grimoire/internal/ir"
)

func TestValidatePipeline_ValidRitual(t *testing.T) {
	p := &ir.Pipeline{Command: "report", Steps: []ir.Step{
		{Id: "check", Spell: "fetch"},
		{Let: "big", Value: "check.n > 1"},
		{If: "big",
			Then: []ir.Step{
				{Spell: "handle", Params: map[string]any{"v": "check.n"}}, // ancestor ref
				{Print: "check.n"},
			},
			Else: []ir.Step{{Print: `"small"`}}},
	}}

	known := func(name string) error {
		if name == "fetch" || name == "handle" {
			return nil
		}
		return fmt.Errorf("spell %q not found", name)
	}
	if err := ValidatePipeline(p, known); err != nil {
		t.Errorf("valid pipeline rejected: %v", err)
	}
}

func TestValidatePipeline_ScopeRules(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    *ir.Pipeline
		want string // "" = valid
	}{
		{
			"branch-private id invisible to later siblings",
			&ir.Pipeline{Command: "r", Steps: []ir.Step{
				{Id: "check", Spell: "a"},
				{If: "check.ok", Then: []ir.Step{{Id: "inner", Spell: "b"}}},
				{Spell: "c", Params: map[string]any{"v": "inner.val"}},
			}},
			"not in scope",
		},
		{
			"forward reference errors in pipe scope",
			&ir.Pipeline{Command: "r", Steps: []ir.Step{
				{Spell: "a"},
				{Spell: "b", Params: map[string]any{"v": "later.x"}},
				{Id: "later", Spell: "c"},
			}},
			"not in scope",
		},
		{
			"forward reference allowed in graph scope",
			&ir.Pipeline{Command: "r", Mode: ir.ModeGraph, Steps: []ir.Step{
				{Spell: "a", Id: "seed"},
				{Spell: "b", Params: map[string]any{"v": "later.x"}, Id: "user"},
				{Id: "later", Spell: "c"},
				{Print: "user"},
			}},
			"",
		},
		{
			"bare string that names nothing is a literal, not an error",
			&ir.Pipeline{Command: "r", Steps: []ir.Step{
				{Spell: "a"},
				{Spell: "b", Params: map[string]any{"v": "nonexistent"}},
			}},
			"",
		},
		{
			"pipe branch inside graph ritual keeps declaration order",
			&ir.Pipeline{Command: "r", Mode: ir.ModeGraph, Steps: []ir.Step{
				{Id: "gate", Spell: "a"},
				{If: "gate.ok", Mode: ir.ModePipe, Then: []ir.Step{
					{Spell: "b", Params: map[string]any{"v": "inner.x"}}, // forward ref in pipe branch
					{Id: "inner", Spell: "c"},
				}},
			}},
			"not in scope",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePipeline(tc.p, nil)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("err = %v, want valid", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidatePipeline_ShapeRules(t *testing.T) {
	lead := ir.Step{Id: "a", Spell: "lead"} // valid entry step
	for _, tc := range []struct {
		name string
		step ir.Step
		want string
	}{
		{"let mixing spell", ir.Step{Let: "x", Value: "1 > 0", Spell: "s"}, "cannot mix"},
		{"let missing value", ir.Step{Let: "x"}, "requires a 'value:'"},
		{"let with redundant id", ir.Step{Let: "x", Value: "1 > 0", Id: "x"}, "redundant 'id:'"},
		{"if with id", ir.Step{Id: "z", If: "true", Then: []ir.Step{{Spell: "s"}}}, "cannot have an id"},
		{"if mixing spell", ir.Step{If: "true", Spell: "s", Then: []ir.Step{{Spell: "s"}}}, "cannot mix"},
		{"print mixing id", ir.Step{Print: "a", Id: "p"}, "cannot mix"},
		{"print mixing params", ir.Step{Print: "a", Params: map[string]any{"x": 1}}, "cannot mix"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &ir.Pipeline{Command: "r", Steps: []ir.Step{lead, tc.step}}
			err := ValidatePipeline(p, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidatePipeline_ExpressionErrors(t *testing.T) {
	lead := ir.Step{Id: "check", Spell: "lead"}
	for _, tc := range []struct {
		name string
		step ir.Step
		want string
	}{
		{"malformed let value", ir.Step{Let: "x", Value: "check.ok &&"}, "invalid let"},
		{"malformed print expr", ir.Step{Print: "check.ok &&"}, "invalid print"},
		{"print ref out of scope", ir.Step{Print: "ghost.x"}, "not in scope"},
		{"bare print ref out of scope", ir.Step{Print: "ghost"}, "not in scope"},
		{"condition ref out of scope", ir.Step{If: "ghost.ok", Then: []ir.Step{{Spell: "s"}}}, "not in scope"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &ir.Pipeline{Command: "r", Steps: []ir.Step{lead, tc.step}}
			err := ValidatePipeline(p, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestValidatePipeline_BuilderRulesSurface(t *testing.T) {
	unknown := func(name string) error { return fmt.Errorf("spell %q not found", name) }

	// Unknown spell via spellExists.
	p := &ir.Pipeline{Command: "r", Steps: []ir.Step{{Spell: "ghost"}}}
	if err := ValidatePipeline(p, unknown); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("unknown spell: err = %v", err)
	}

	// First step must be a spell (builder invariant).
	p = &ir.Pipeline{Command: "r", Steps: []ir.Step{{Let: "x", Value: "1 > 0"}}}
	if err := ValidatePipeline(p, nil); err == nil || !strings.Contains(err.Error(), "first step must be a spell") {
		t.Errorf("non-spell entry: err = %v", err)
	}

	// Graph cycle (builder structural rule).
	p = &ir.Pipeline{Command: "r", Mode: ir.ModeGraph, Steps: []ir.Step{
		{Spell: "s", Id: "a", Params: map[string]any{"v": "b.x"}},
		{Spell: "s", Id: "b", Params: map[string]any{"v": "a.x"}},
	}}
	if err := ValidatePipeline(p, nil); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Errorf("cycle: err = %v", err)
	}
}
