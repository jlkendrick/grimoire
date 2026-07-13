// Package graph is the unified execution core: every Grimoire invocation —
// a single spell or a multi-step ritual — is compiled ahead of time into a
// Graph of Nodes and run by one scheduler.
//
// The scheduler is mode-blind. pipe/graph mode is a construction-time
// concern: a pipe scope compiles to a chain (each node depending on its
// predecessor) and a graph scope compiles to whatever DAG the declared
// references imply. At most one node of a chain is ever ready, so pipe
// scopes serialize without a sequential code path existing anywhere.
//
// Execution assumes a valid graph: acyclic, references in scope, first
// step a spell, sink rules respected. That validation belongs to the
// builder, which the reconciler runs at scroll load.
package graph

import "context"

// Op is a node's behavior. Implementations own their step kind's semantics
// (running a subprocess, evaluating an expression, presenting a result);
// the scheduler owns everything else — readiness, input resolution,
// concurrency, error propagation.
//
// Run returns the node's output: the value a Chain successor receives and
// the value recorded as the graph's result if this node is the Out node.
// Ops that produce no result of their own (let; if with no branch taken)
// must return in unchanged — that passthrough is what preserves "the
// previous result survives an if/let" from the linear engine.
type Op interface {
	Run(ctx context.Context, in any, ex *Exec) (any, error)
}

// InputKind says how a node's input is materialized before Op.Run.
type InputKind int

const (
	// InputNone: the node reads no input (if/let nodes in a graph scope,
	// where nothing flows implicitly).
	InputNone InputKind = iota
	// InputSeed: the graph's seed value — ritual inputs / CLI flags for
	// the root graph, the owning if-node's input for a branch subgraph.
	InputSeed
	// InputChain: the sole dependency's output. This is the implicit
	// stdout→stdin pipe; the builder only emits it inside pipe scopes.
	InputChain
	// InputParams: the Params map with reference strings resolved against
	// the shared bindings at the moment the node becomes ready.
	InputParams
)

// InputSpec is fixed at construction time; only the values it resolves to
// vary at runtime.
type InputSpec struct {
	Kind   InputKind
	Params map[string]any // literals and reference strings; InputParams only
}

// Node is one schedulable unit. Deps carry ordering only — data flows
// through recorded outputs (InputChain) and the shared bindings
// (InputParams), never along the edges themselves.
type Node struct {
	ID    string // unique within its Graph; the builder synthesizes ids for unnamed steps
	Deps  []string
	Input InputSpec
	Op    Op
}

// Graph is one scope's nodes. Out names the node whose output is the
// graph's result; it is empty when the scope has no single result (a
// multi-sink graph scope), in which case execution returns nil and any
// consumer must read named bindings instead.
type Graph struct {
	Nodes []*Node
	Out   string
}
