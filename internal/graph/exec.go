package graph

import (
	"context"
	"fmt"
	"maps"
	"sync"

	expr "github.com/jlkendrick/grimoire/internal/expr"
)

// Exec is the state shared by every scope of a single invocation: the
// named bindings that spell ids and lets publish and that params and
// conditions read. One Exec spans the root graph and all of its branch
// subgraphs — branch privacy is a static property enforced by validation,
// not a runtime one.
type Exec struct {
	mu       sync.Mutex
	bindings map[string]any
}

// Run executes a prebuilt Graph with fresh shared state and returns the
// output of its Out node (nil if the graph declares none). It is the
// single entrypoint for every frontend; it never exits the process and
// reports all failures as errors.
func Run(ctx context.Context, g *Graph, seed any) (any, error) {
	ex := &Exec{bindings: make(map[string]any)}
	return ex.Run(ctx, g, seed)
}

// Bind publishes a named value for later params and condition references.
func (ex *Exec) Bind(name string, val any) {
	ex.mu.Lock()
	defer ex.mu.Unlock()
	ex.bindings[name] = val
}

// Bindings returns a snapshot of the current bindings. Callers always get
// a copy: expr.ResolveReference memoizes intermediate accessor lookups
// by writing synthetic keys back into the map it is handed, so the live
// map must never escape this package.
func (ex *Exec) Bindings() map[string]any {
	ex.mu.Lock()
	defer ex.mu.Unlock()
	return maps.Clone(ex.bindings)
}

// Run executes one graph — one scope. It is Kahn's algorithm run live:
// every node with no unfinished dependencies launches, and each completion
// may unlock more. The first error cancels ctx and stops new launches;
// in-flight nodes run to completion before Run returns. A graph whose
// nodes cannot all become ready (a cycle that escaped validation) returns
// an error instead of deadlocking.
func (ex *Exec) Run(ctx context.Context, g *Graph, seed any) (any, error) {
	r, err := newRun(ex, g, seed)
	if err != nil {
		return nil, err
	}
	return r.execute(ctx)
}

// run is the transient scheduling state for one graph execution. Bindings
// live on Exec and outlive it; everything here dies with the scope.
type run struct {
	ex   *Exec
	g    *Graph
	seed any
	byID map[string]*Node

	mu         sync.Mutex
	wg         sync.WaitGroup
	cancel     context.CancelFunc
	outputs    map[string]any
	indegree   map[string]int
	dependents map[string][]string
	completed  int
	firstErr   error
}

func newRun(ex *Exec, g *Graph, seed any) (*run, error) {
	r := &run{
		ex:         ex,
		g:          g,
		seed:       seed,
		byID:       make(map[string]*Node, len(g.Nodes)),
		outputs:    make(map[string]any, len(g.Nodes)),
		indegree:   make(map[string]int, len(g.Nodes)),
		dependents: make(map[string][]string),
	}
	for _, n := range g.Nodes {
		if _, dup := r.byID[n.ID]; dup {
			return nil, fmt.Errorf("graph: duplicate node id %q", n.ID)
		}
		r.byID[n.ID] = n
	}
	for _, n := range g.Nodes {
		r.indegree[n.ID] = len(n.Deps)
		for _, dep := range n.Deps {
			if _, ok := r.byID[dep]; !ok {
				return nil, fmt.Errorf("graph: node %q depends on unknown node %q", n.ID, dep)
			}
			r.dependents[dep] = append(r.dependents[dep], n.ID)
		}
	}
	if g.Out != "" {
		if _, ok := r.byID[g.Out]; !ok {
			return nil, fmt.Errorf("graph: output node %q does not exist", g.Out)
		}
	}
	return r, nil
}

func (r *run) execute(parent context.Context) (any, error) {
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	defer cancel()

	r.mu.Lock()
	for _, n := range r.g.Nodes {
		if r.indegree[n.ID] == 0 {
			r.launch(ctx, n)
		}
	}
	r.mu.Unlock()

	r.wg.Wait()

	if r.firstErr != nil {
		return nil, r.firstErr
	}
	if r.completed != len(r.g.Nodes) {
		return nil, fmt.Errorf("graph: %d node(s) never became ready — cycle escaped validation", len(r.g.Nodes)-r.completed)
	}
	if r.g.Out == "" {
		return nil, nil
	}
	return r.outputs[r.g.Out], nil
}

// launch starts one node's goroutine. Must be called with r.mu held; the
// wg.Add happens before the unlocking caller's own wg.Done can run, so
// wg.Wait cannot observe a false zero between a completion and the
// dependents it unlocks.
func (r *run) launch(ctx context.Context, n *Node) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		out, err := r.runNode(ctx, n)

		r.mu.Lock()
		defer r.mu.Unlock()
		if err != nil {
			if r.firstErr == nil {
				r.firstErr = fmt.Errorf("node %s: %w", n.ID, err)
				r.cancel()
			}
			return
		}
		r.outputs[n.ID] = out
		r.completed++
		if r.firstErr != nil {
			return
		}
		for _, id := range r.dependents[n.ID] {
			r.indegree[id]--
			if r.indegree[id] == 0 {
				r.launch(ctx, r.byID[id])
			}
		}
	}()
}

func (r *run) runNode(ctx context.Context, n *Node) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	in, err := r.resolveInput(n)
	if err != nil {
		return nil, err
	}
	return n.Op.Run(ctx, in, r.ex)
}

// resolveInput materializes a node's input per its InputSpec: Chain reads
// the sole dependency's recorded output, Params resolves references
// against a bindings snapshot. Both are safe here because a node only
// resolves after every dependency has completed and published.
func (r *run) resolveInput(n *Node) (any, error) {
	switch n.Input.Kind {
	case InputNone:
		return nil, nil
	case InputSeed:
		return r.seed, nil
	case InputChain:
		if len(n.Deps) != 1 {
			return nil, fmt.Errorf("chain input requires exactly one dependency, have %d", len(n.Deps))
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.outputs[n.Deps[0]], nil
	case InputParams:
		bindings := r.ex.Bindings()
		resolved := make(map[string]any, len(n.Input.Params))
		for name, raw := range n.Input.Params {
			val, isRef, err := expr.ResolveReference(raw, bindings)
			if err != nil {
				return nil, fmt.Errorf("param %s: %w", name, err)
			}
			if isRef {
				resolved[name] = val
			} else {
				resolved[name] = raw
			}
		}
		return resolved, nil
	default:
		return nil, fmt.Errorf("unknown input kind %d", n.Input.Kind)
	}
}
