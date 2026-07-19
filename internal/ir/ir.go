// Package ir holds Grimoire's internal representation: the resolved,
// conventional-name mirror of the fantasy-named user model. A Spell
// resolves to a Function, a Ritual to a Pipeline — the name split itself
// marks the boundary between what users declare and what the engine runs.
package ir

import (
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

// Function is the resolved form of a spell. It aliases the legacy
// descriptor type while the extraction/cache layers are still being
// ported; new code says ir.Function and never FunctionDescriptor.
type Function = descriptor.FunctionDescriptor

// Param is one parameter of a Function.
type Param = descriptor.ParamDescriptor

// TypeInfo is the recursive type model shared by params and returns
// (alias of the legacy descriptor type until that package is absorbed).
type TypeInfo = descriptor.TypeInfo

// Mode is the data-flow contract of one scope (a step list). Pipe scopes
// chain implicitly in declaration order; graph scopes derive a dependency
// DAG from declared references and run independent steps concurrently.
// The zero value means "unset" and resolves to the enclosing scope's mode
// (pipe at the root).
type Mode string

const (
	ModePipe  Mode = "pipe"
	ModeGraph Mode = "graph"
)

// Pipeline is the resolved form of a ritual.
type Pipeline struct {
	Command string
	Mode    Mode // mode of the top-level scope; empty = pipe
	Steps   []Step

	// RitualHash is sync bookkeeping: the hash of the scroll ritual this
	// pipeline was lowered from, letting the reconciler skip unchanged
	// rituals. Function carries SourceHash/SpellHash for the same reason.
	RitualHash string `json:",omitempty"`
}

// Step is one of four kinds, discriminated by which fields are set —
// mirror of the scroll-level step, minus everything already resolved.
type Step struct {
	// spell-step: invoke a function. Id, when set, binds the decoded
	// output for later references.
	Id     string
	Spell  string
	Params map[string]any

	// if-step: branch on a condition. Mode, when set, overrides the
	// branch scopes' mode (valid on if-steps only).
	If   string
	Then []Step
	Else []Step
	Mode Mode

	// let-step: bind Value's result to Let.
	Let   string
	Value string

	// print-step: declare a ritual output. The expression's value goes to
	// the frontend's presenter (stdout for the CLI).
	Print string
}

// Kind reports the step kind: "spell", "if", "let", or "print".
func (s Step) Kind() string {
	switch {
	case s.If != "":
		return "if"
	case s.Let != "":
		return "let"
	case s.Print != "":
		return "print"
	default:
		return "spell"
	}
}
