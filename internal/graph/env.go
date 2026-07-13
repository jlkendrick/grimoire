package graph

import (
	"context"

	ir "github.com/jlkendrick/grimoire/internal/ir"
)

// Env supplies the outside world to a graph: how to resolve a spell name
// to a function, how to run one, and where printed values go. The graph
// core stays pure orchestration — the CLI wires runtime.Run and stdout
// here, a REST handler wires a response collector, tests wire fakes.
type Env struct {
	// ResolveSpell maps a step's spell name to a runnable Function,
	// including any cross-scroll lookup and staleness reconciliation.
	ResolveSpell func(name string) (*ir.Function, error)

	// RunSpell executes a function with a fully built payload and returns
	// its decoded (JSON) output.
	RunSpell func(ctx context.Context, fn *ir.Function, payload map[string]any) (any, error)

	// Present receives every printed value, in declaration order.
	Present func(v any) error
}
