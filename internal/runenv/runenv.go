// Package runenv wires the real Grimoire runtime into the graph core's Env
// seam: spell names resolve through the descriptor cache (with cross-scroll
// lookup and staleness re-extraction), and spells run as real subprocesses
// through the language adapters. Presentation stays caller-supplied —
// output belongs to the frontend, not to this package.
package runenv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	graph "github.com/jlkendrick/grimoire/internal/graph"
	ir "github.com/jlkendrick/grimoire/internal/ir"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	runtime "github.com/jlkendrick/grimoire/internal/runtime"
)

// New builds a graph.Env backed by the real runtime. dc is the invoking
// scroll's descriptor cache; spells supplies cross-scroll resolution for
// dot-form spell names and may be nil when none are expected. present
// receives every printed value in declaration order.
func New(dc *cache.DescriptorCache, spells *resolve.SpellIndex, present func(v any) error) graph.Env {
	return graph.Env{
		ResolveSpell: func(name string) (*ir.Function, error) {
			fd, err := resolve.ResolveSpellRef(name, dc, spells)
			if err != nil {
				return nil, err
			}
			// Copy before reconciling: the cache owns the original, and
			// reconciliation rewrites the descriptor when the source file
			// changed since it was cached.
			fn := *fd
			resolved, err := resolve.ReconcileFunctionDescriptor(&fn)
			if err != nil {
				return nil, fmt.Errorf("reconcile spell %s: %w", name, err)
			}
			return &resolved, nil
		},

		// The legacy runtime is not context-aware: on cancellation the
		// scheduler stops launching new nodes, but an in-flight subprocess
		// runs to completion — same behavior as the old engine. Plumbing
		// ctx into process kill is a runtime-layer rewrite concern.
		// RunResult.Runtime (the version string the CLI shows) is dropped
		// here for now; it returns with the observer.
		RunSpell: func(_ context.Context, fn *ir.Function, payload map[string]any) (any, error) {
			res, err := runtime.Run(fn, payload, &runtime.RunOptions{SuppressFraming: true})
			if err != nil {
				return nil, err
			}
			return decodeReturnValue(res.Output)
		},

		Present: present,
	}
}

// decodeReturnValue parses the function's return value from the spell
// subprocess's stdout; empty output (a function that returns nothing)
// decodes to nil.
//
// The subprocess's stdout IS the return value: the language wrappers
// reserve stdout for the JSON-encoded return and reroute anything the
// function itself prints onto stderr. So everything that flows between
// steps — bindings, chained inputs, printed values — is a returned value,
// never text the function printed. It is decoded exactly once, here.
// (Same contract as the old engine's decodeStepOutput.)
func decodeReturnValue(out []byte) (any, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var decoded any
	if err := json.Unmarshal(trimmed, &decoded); err != nil {
		return nil, fmt.Errorf("spell output is not valid JSON: %v", err)
	}
	return decoded, nil
}
