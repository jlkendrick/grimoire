package graph

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// opFunc adapts a bare function to Op for tests.
type opFunc func(ctx context.Context, in any, ex *Exec) (any, error)

func (f opFunc) Run(ctx context.Context, in any, ex *Exec) (any, error) {
	return f(ctx, in, ex)
}

// tag returns an op that appends its id to the incoming value, making data
// flow visible in the final output ("seed>a>b>c").
func tag(id string, order *[]string, mu *sync.Mutex) Op {
	return opFunc(func(_ context.Context, in any, _ *Exec) (any, error) {
		if order != nil {
			mu.Lock()
			*order = append(*order, id)
			mu.Unlock()
		}
		return fmt.Sprintf("%v>%s", in, id), nil
	})
}

func TestChain_SerializesAndPipes(t *testing.T) {
	var mu sync.Mutex
	var order []string
	g := &Graph{
		Nodes: []*Node{
			{ID: "a", Input: InputSpec{Kind: InputSeed}, Op: tag("a", &order, &mu)},
			{ID: "b", Deps: []string{"a"}, Input: InputSpec{Kind: InputChain}, Op: tag("b", &order, &mu)},
			{ID: "c", Deps: []string{"b"}, Input: InputSpec{Kind: InputChain}, Op: tag("c", &order, &mu)},
		},
		Out: "c",
	}

	out, err := Run(context.Background(), g, "seed")
	if err != nil {
		t.Fatal(err)
	}
	if out != "seed>a>b>c" {
		t.Errorf("out = %v, want seed>a>b>c", out)
	}
	if want := []string{"a", "b", "c"}; !slices.Equal(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestDiamond_IndependentNodesOverlapAndJoin(t *testing.T) {
	bStarted := make(chan struct{})
	cStarted := make(chan struct{})
	// meet proves overlap: each sibling refuses to finish until the other
	// has started, so a sequential scheduler would time out here.
	meet := func(mine chan<- struct{}, other <-chan struct{}) error {
		close(mine)
		select {
		case <-other:
			return nil
		case <-time.After(5 * time.Second):
			return errors.New("sibling never started: independent nodes did not overlap")
		}
	}

	passthrough := opFunc(func(_ context.Context, in any, _ *Exec) (any, error) { return in, nil })
	g := &Graph{
		Nodes: []*Node{
			{ID: "a", Input: InputSpec{Kind: InputSeed}, Op: passthrough},
			{ID: "b", Deps: []string{"a"}, Op: opFunc(func(_ context.Context, _ any, ex *Exec) (any, error) {
				if err := meet(bStarted, cStarted); err != nil {
					return nil, err
				}
				ex.Bind("b", 1.0)
				return 1.0, nil
			})},
			{ID: "c", Deps: []string{"a"}, Op: opFunc(func(_ context.Context, _ any, ex *Exec) (any, error) {
				if err := meet(cStarted, bStarted); err != nil {
					return nil, err
				}
				ex.Bind("c", 2.0)
				return 2.0, nil
			})},
			{
				ID: "d", Deps: []string{"b", "c"},
				Input: InputSpec{Kind: InputParams, Params: map[string]any{"left": "b", "right": "c", "lit": 42}},
				Op:    passthrough,
			},
		},
		Out: "d",
	}

	out, err := Run(context.Background(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("out = %T (%v), want map", out, out)
	}
	if got["left"] != 1.0 || got["right"] != 2.0 || got["lit"] != 42 {
		t.Errorf("join payload = %v, want left:1 right:2 lit:42", got)
	}
}

func TestError_StopsDownstreamAndReports(t *testing.T) {
	ran := false
	g := &Graph{
		Nodes: []*Node{
			{ID: "boom", Op: opFunc(func(_ context.Context, _ any, _ *Exec) (any, error) {
				return nil, errors.New("kaput")
			})},
			{ID: "after", Deps: []string{"boom"}, Op: opFunc(func(_ context.Context, _ any, _ *Exec) (any, error) {
				ran = true
				return nil, nil
			})},
		},
		Out: "after",
	}

	_, err := Run(context.Background(), g, nil)
	if err == nil || !strings.Contains(err.Error(), "node boom") || !strings.Contains(err.Error(), "kaput") {
		t.Fatalf("err = %v, want node boom: kaput", err)
	}
	if ran {
		t.Error("downstream node ran after upstream error")
	}
}

func TestCycle_ErrorsInsteadOfHanging(t *testing.T) {
	nop := opFunc(func(_ context.Context, _ any, _ *Exec) (any, error) { return nil, nil })
	g := &Graph{
		Nodes: []*Node{
			{ID: "a", Deps: []string{"b"}, Op: nop},
			{ID: "b", Deps: []string{"a"}, Op: nop},
		},
	}

	_, err := Run(context.Background(), g, nil)
	if err == nil || !strings.Contains(err.Error(), "never became ready") {
		t.Fatalf("err = %v, want cycle error", err)
	}
}

func TestGraphShape_Errors(t *testing.T) {
	nop := opFunc(func(_ context.Context, _ any, _ *Exec) (any, error) { return nil, nil })

	for _, tc := range []struct {
		name string
		g    *Graph
		want string
	}{
		{"duplicate id", &Graph{Nodes: []*Node{{ID: "a", Op: nop}, {ID: "a", Op: nop}}}, "duplicate node id"},
		{"unknown dep", &Graph{Nodes: []*Node{{ID: "a", Deps: []string{"ghost"}, Op: nop}}}, "unknown node"},
		{"unknown out", &Graph{Nodes: []*Node{{ID: "a", Op: nop}}, Out: "ghost"}, "does not exist"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Run(context.Background(), tc.g, nil); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestNoOut_ReturnsNil(t *testing.T) {
	nop := opFunc(func(_ context.Context, _ any, _ *Exec) (any, error) { return "ignored", nil })
	g := &Graph{Nodes: []*Node{{ID: "a", Op: nop}}}

	out, err := Run(context.Background(), g, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != nil {
		t.Errorf("out = %v, want nil for graph without Out", out)
	}
}
