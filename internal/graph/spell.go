package graph

import (
	"context"
	"fmt"
	"maps"

	ir "github.com/jlkendrick/grimoire/internal/ir"
)

// SpellOp invokes one function through the environment's runner. Chained
// says how to adapt the input into a payload: a chained input is an
// upstream step's arbitrary output that must be mapped onto this
// function's params, while a direct input (seed or resolved params) is
// already a payload map and only needs defaults filled in.
type SpellOp struct {
	Fn      *ir.Function
	Id      string // binding name; empty = anonymous
	Chained bool
	Runner  func(ctx context.Context, fn *ir.Function, payload map[string]any) (any, error)
}

func (op *SpellOp) Run(ctx context.Context, in any, ex *Exec) (any, error) {
	payload, err := op.payload(in)
	if err != nil {
		return nil, err
	}
	out, err := op.Runner(ctx, op.Fn, payload)
	if err != nil {
		return nil, fmt.Errorf("spell %s: %w", op.Fn.CommandName, err)
	}
	if op.Id != "" {
		ex.Bind(op.Id, out)
	}
	return out, nil
}

func (op *SpellOp) payload(in any) (map[string]any, error) {
	if op.Chained {
		return adaptChained(in, op.Fn)
	}
	payload := make(map[string]any)
	switch v := in.(type) {
	case nil:
	case map[string]any:
		maps.Copy(payload, v)
	default:
		return nil, fmt.Errorf("spell %s: direct input must be a map, got %T", op.Fn.CommandName, in)
	}
	fillDefaults(payload, op.Fn)
	return payload, nil
}

// adaptChained maps an upstream step's decoded output onto fn's params.
// A list whose length matches a multi-param function unpacks positionally
// (mirroring Python's `return a, b`); a map that covers every param maps
// by name; anything else binds whole to the first param, so a function
// returning a list-as-data reaches a single-param step intact. Ported
// verbatim from the linear engine's buildPayloadFromResult — including its
// no-defaults behavior — so chained rituals behave identically.
func adaptChained(in any, fn *ir.Function) (map[string]any, error) {
	payload := make(map[string]any)

	if list, ok := in.([]any); ok && len(fn.Params) > 1 {
		if len(list) != len(fn.Params) {
			return nil, fmt.Errorf("spell %s: previous output has %d values but expects %d params", fn.CommandName, len(list), len(fn.Params))
		}
		for i, param := range fn.Params {
			payload[param.Name] = list[i]
		}
		return payload, nil
	}

	if m, ok := in.(map[string]any); ok {
		matched := 0
		for _, param := range fn.Params {
			if value, ok := m[param.Name]; ok {
				payload[param.Name] = value
				matched++
			}
		}
		if matched == len(fn.Params) {
			return payload, nil
		}
		payload = make(map[string]any)
	}

	if len(fn.Params) >= 1 {
		payload[fn.Params[0].Name] = in
	}
	return payload, nil
}

func fillDefaults(payload map[string]any, fn *ir.Function) {
	for _, param := range fn.Params {
		if _, ok := payload[param.Name]; !ok && param.Default != nil {
			payload[param.Name] = param.Default
		}
	}
}
