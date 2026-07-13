package graph

import (
	"context"
	"strings"
	"testing"

	ir "github.com/jlkendrick/grimoire/internal/ir"
)

func fnWith(params ...string) *ir.Function {
	fn := &ir.Function{CommandName: "f"}
	for _, p := range params {
		fn.Params = append(fn.Params, ir.Param{Name: p})
	}
	return fn
}

func TestAdaptChained(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   any
		fn   *ir.Function
		want map[string]any
	}{
		{"list unpacks positionally", []any{1.0, 2.0}, fnWith("x", "y"), map[string]any{"x": 1.0, "y": 2.0}},
		{"list to single param stays whole", []any{1.0, 2.0}, fnWith("items"), map[string]any{"items": []any{1.0, 2.0}}},
		{"map covering all params maps by name", map[string]any{"x": 1.0, "y": 2.0, "extra": 3.0}, fnWith("x", "y"), map[string]any{"x": 1.0, "y": 2.0}},
		{"partial map binds whole to first param", map[string]any{"x": 1.0}, fnWith("x", "y"), map[string]any{"x": map[string]any{"x": 1.0}}},
		{"scalar binds to first param", "hello", fnWith("msg"), map[string]any{"msg": "hello"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := adaptChained(tc.in, tc.fn)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("payload = %v, want %v", got, tc.want)
			}
			for k := range tc.want {
				// list-valued entries can't compare with !=; spot-check by key presence
				if _, ok := got[k]; !ok {
					t.Errorf("payload missing %q: %v", k, got)
				}
			}
		})
	}

	if _, err := adaptChained([]any{1.0}, fnWith("x", "y")); err == nil || !strings.Contains(err.Error(), "expects 2 params") {
		t.Errorf("arity mismatch: err = %v", err)
	}
}

func TestSpellOp_DirectFillsDefaultsAndBinds(t *testing.T) {
	fn := &ir.Function{CommandName: "f", Params: []ir.Param{
		{Name: "x"},
		{Name: "retries", Default: 3.0},
	}}

	var gotPayload map[string]any
	op := &SpellOp{Fn: fn, Id: "res", Runner: func(_ context.Context, _ *ir.Function, payload map[string]any) (any, error) {
		gotPayload = payload
		return "output", nil
	}}
	ex := &Exec{bindings: map[string]any{}}

	out, err := op.Run(context.Background(), map[string]any{"x": 1.0}, ex)
	if err != nil {
		t.Fatal(err)
	}
	if gotPayload["x"] != 1.0 || gotPayload["retries"] != 3.0 {
		t.Errorf("payload = %v, want explicit x and defaulted retries", gotPayload)
	}
	if out != "output" || ex.bindings["res"] != "output" {
		t.Errorf("out = %v, bound = %v, want output bound under the step id", out, ex.bindings["res"])
	}
}

func TestSpellOp_DirectRejectsNonMapInput(t *testing.T) {
	op := &SpellOp{Fn: fnWith("x"), Runner: func(_ context.Context, _ *ir.Function, _ map[string]any) (any, error) {
		return nil, nil
	}}

	if _, err := op.Run(context.Background(), "not a map", &Exec{bindings: map[string]any{}}); err == nil {
		t.Error("non-map direct input must error")
	}
}
