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
	"os"
	"sync/atomic"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	graph "github.com/jlkendrick/grimoire/internal/graph"
	ir "github.com/jlkendrick/grimoire/internal/ir"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	runtime "github.com/jlkendrick/grimoire/internal/runtime"
)

// Observer receives spell lifecycle events from a running graph. It is
// presentation's window into execution: the engine never sees it —
// everything observable already crosses the Env seam, so observation is
// a property of how this environment runs spells.
//
// id is unique per spell invocation within one Env, assigned in start
// order. In graph scopes spells overlap, so callbacks MUST be safe under
// concurrent calls; events for one id are always ordered (start, then
// stderr lines, then finish). A nil Observer disables observation and
// routes spell stderr straight to os.Stderr.
type Observer struct {
	OnSpellStart  func(id int, spell string)
	OnSpellStderr func(id int, line string)
	OnSpellFinish func(id int, spell string, res FinishInfo)
}

// FinishInfo is everything a finished spell reports: the decoded return
// value (nil on error), the adapter's runtime-version string, and the
// environment cache status — both may be empty.
type FinishInfo struct {
	Out            any
	RuntimeVersion string
	CacheStatus    string
	Err            error
}

// Config tunes a real Env's presentation-facing behavior. Present
// receives every printed value in declaration order; Observer may be
// nil (spell stderr then passes straight through to os.Stderr).
// Framing lets runtime.Run print its own provisioning/casting lines —
// the bare-spell UX; ritual runs leave it false in favor of the
// renderer.
type Config struct {
	Present  func(v any) error
	Observer *Observer
	Framing  bool
}

// New builds a graph.Env backed by the real runtime. dc is the invoking
// scroll's descriptor cache; spells supplies cross-scroll resolution for
// dot-form spell names and may be nil when none are expected.
func New(dc *cache.DescriptorCache, spells *resolve.SpellIndex, cfg Config) graph.Env {
	obs := cfg.Observer
	var spellCounter atomic.Int64
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
		RunSpell: func(_ context.Context, fn *ir.Function, payload map[string]any) (any, error) {
			id := int(spellCounter.Add(1))
			if obs != nil && obs.OnSpellStart != nil {
				obs.OnSpellStart(id, fn.CommandName)
			}

			// Spell prints (rerouted to the subprocess's stderr by the
			// wrappers) go to the observer when one is watching, else
			// straight to Grimoire's stderr: stdout carries only
			// declared prints.
			onStderr := func(line string) { fmt.Fprintln(os.Stderr, line) }
			if obs != nil && obs.OnSpellStderr != nil {
				onStderr = func(line string) { obs.OnSpellStderr(id, line) }
			}

			res, err := runtime.Run(fn, payload, &runtime.RunOptions{
				SuppressFraming: !cfg.Framing,
				OnStderrLine:    onStderr,
			})
			if err != nil {
				if obs != nil && obs.OnSpellFinish != nil {
					obs.OnSpellFinish(id, fn.CommandName, FinishInfo{Err: err})
				}
				return nil, err
			}

			out, err := decodeReturnValue(res.Output)
			if obs != nil && obs.OnSpellFinish != nil {
				obs.OnSpellFinish(id, fn.CommandName, FinishInfo{
					Out:            out,
					RuntimeVersion: res.Runtime,
					CacheStatus:    res.CacheStatus,
					Err:            err,
				})
			}
			return out, err
		},

		Present: cfg.Present,
	}
}

// SpellChecker returns the cache-backed existence check that
// graph.ValidatePipeline expects: it confirms a step's spell name
// resolves against the scroll's descriptor cache (and the cross-scroll
// index for dotted names) without touching source files or preparing
// anything to run.
func SpellChecker(dc *cache.DescriptorCache, spells *resolve.SpellIndex) func(name string) error {
	return func(name string) error {
		_, err := resolve.ResolveSpellRef(name, dc, spells)
		return err
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
