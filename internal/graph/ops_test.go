package graph

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// appendTag mirrors exec_test's tag op without order tracking.
func appendTag(id string) Op {
	return opFunc(func(_ context.Context, in any, _ *Exec) (any, error) {
		return fmt.Sprintf("%v>%s", in, id), nil
	})
}

func branchGraph(id string) *Graph {
	return &Graph{
		Nodes: []*Node{{ID: "b0", Input: InputSpec{Kind: InputSeed}, Op: appendTag(id)}},
		Out:   "b0",
	}
}

// The if-node's input seeds the chosen branch, and the branch's result
// becomes the if-node's output — so a chained successor sees the branch
// tail, exactly like the linear engine's prev_result rule.
func TestIfOp_ChosenBranchChains(t *testing.T) {
	for _, tc := range []struct {
		name string
		ok   bool
		want string
	}{
		{"then branch", true, "seed>then>final"},
		{"else branch", false, "seed>else>final"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ok := tc.ok
			g := &Graph{
				Nodes: []*Node{
					{ID: "check", Input: InputSpec{Kind: InputSeed}, Op: opFunc(func(_ context.Context, in any, ex *Exec) (any, error) {
						ex.Bind("check", map[string]any{"ok": ok})
						return in, nil
					})},
					{ID: "_if", Deps: []string{"check"}, Input: InputSpec{Kind: InputChain}, Op: &IfOp{
						Cond: "check.ok",
						Then: branchGraph("then"),
						Else: branchGraph("else"),
					}},
					{ID: "final", Deps: []string{"_if"}, Input: InputSpec{Kind: InputChain}, Op: appendTag("final")},
				},
				Out: "final",
			}

			out, err := Run(context.Background(), g, "seed")
			if err != nil {
				t.Fatal(err)
			}
			if out != tc.want {
				t.Errorf("out = %v, want %s", out, tc.want)
			}
		})
	}
}

func TestIfOp_NoBranchPassesThrough(t *testing.T) {
	op := &IfOp{Cond: "false", Then: branchGraph("then")} // no else
	ex := &Exec{bindings: map[string]any{}}

	out, err := op.Run(context.Background(), "payload", ex)
	if err != nil {
		t.Fatal(err)
	}
	if out != "payload" {
		t.Errorf("out = %v, want the input passed through", out)
	}
}

func TestIfOp_ConditionErrors(t *testing.T) {
	ex := &Exec{bindings: map[string]any{}}

	if _, err := (&IfOp{Cond: "a &&", Then: branchGraph("t")}).Run(context.Background(), nil, ex); err == nil {
		t.Error("malformed condition: want parse error")
	}
	if _, err := (&IfOp{Cond: "missing.ok", Then: branchGraph("t")}).Run(context.Background(), nil, ex); err == nil {
		t.Error("unresolvable reference: want evaluate error")
	}
}

func TestLetOp_BindsAndPassesThrough(t *testing.T) {
	var readerSaw, chainSaw any
	capture := func(dst *any) Op {
		return opFunc(func(_ context.Context, in any, _ *Exec) (any, error) {
			*dst = in
			return in, nil
		})
	}

	g := &Graph{
		Nodes: []*Node{
			{ID: "src", Input: InputSpec{Kind: InputSeed}, Op: appendTag("src")},
			{ID: "_let", Deps: []string{"src"}, Input: InputSpec{Kind: InputChain}, Op: &LetOp{Name: "greeting", Expr: `"hi"`}},
			{
				ID: "reader", Deps: []string{"_let"},
				Input: InputSpec{Kind: InputParams, Params: map[string]any{"g": "greeting"}},
				Op:    capture(&readerSaw),
			},
			{ID: "chained", Deps: []string{"_let"}, Input: InputSpec{Kind: InputChain}, Op: capture(&chainSaw)},
		},
		Out: "chained",
	}

	if _, err := Run(context.Background(), g, "seed"); err != nil {
		t.Fatal(err)
	}
	if m, ok := readerSaw.(map[string]any); !ok || m["g"] != "hi" {
		t.Errorf("params reader saw %v, want map with g:hi", readerSaw)
	}
	// The let passes its input through, so a node chaining off it still
	// sees the src output — the pipe survives a let step.
	if chainSaw != "seed>src" {
		t.Errorf("chained node saw %v, want seed>src", chainSaw)
	}
}

func TestPrintOp_NoExprPresentsInputAndPassesThrough(t *testing.T) {
	var got any
	op := &PrintOp{Present: func(v any) error { got = v; return nil }}

	out, err := op.Run(context.Background(), "result", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "result" || out != "result" {
		t.Errorf("got = %v, out = %v, want both to be result", got, out)
	}

	boom := &PrintOp{Present: func(any) error { return errors.New("tty gone") }}
	if _, err := boom.Run(context.Background(), "x", nil); err == nil {
		t.Error("Present error must propagate")
	}
}

func TestPrintOp_ExprEvaluatesAgainstBindings(t *testing.T) {
	ex := &Exec{bindings: map[string]any{"check": map[string]any{"n": 7.0}}}
	var got any
	op := &PrintOp{Expr: "check.n", Present: func(v any) error { got = v; return nil }}

	out, err := op.Run(context.Background(), "chain-value", ex)
	if err != nil {
		t.Fatal(err)
	}
	if got != 7.0 {
		t.Errorf("presented %v, want 7", got)
	}
	if out != "chain-value" {
		t.Errorf("out = %v, want passthrough of the input, not the printed value", out)
	}

	bad := &PrintOp{Expr: "missing.field", Present: func(any) error { return nil }}
	if _, err := bad.Run(context.Background(), nil, ex); err == nil {
		t.Error("unresolvable print expression must error")
	}
}
