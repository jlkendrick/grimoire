package engine

import (
	"bytes"
	"encoding/json"
	"fmt"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	expr "github.com/jlkendrick/grimoire/internal/expr"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	runtime "github.com/jlkendrick/grimoire/internal/runtime"
)

// Hooks lets the caller observe per-step lifecycle for presentation. CLI
// drives a TTY spinner/preview off these events; a REST handler would map
// them to JSON frames; a test harness can leave them nil. idx is the
// 1-based spell-step counter (matches the "step k/N" header).
type Hooks struct {
	OnStepStart  func(idx, total int, spellName string, isTerminal bool)
	OnStepStderr func(idx int, line string)
	OnStepFinish func(idx int, output []byte)
}

// Result captures what a pipeline run produced. FinalOutput is the tail
// of the executed branch — nil iff no spell-step ran at all (possible
// when a top-level if-step has no else and the condition is false).
// Runtimes is the distinct set of runtime-version strings reported by
// runtime.Run, preserved in first-seen order.
type Result struct {
	FinalOutput []byte
	Runtimes    []string
}

type pipelineExecState struct {
	bindings         map[string]any
	prev_result      *runtime.RunResult
	seen             map[string]bool
	runtimes         []string
	k                int
	total            int
	inputs           map[string]any
	descriptor_cache *cache.DescriptorCache
	spell_index      *resolve.SpellIndex
	hooks            Hooks
	terminalOutput   []byte
	terminalRan      bool
}

// RunPipeline executes a pipeline descriptor against the given cache,
// seeding the entry spell with inputs. It is the single programmatic
// entrypoint shared by every frontend (CLI today, REST later). spellIndex
// supplies cross-scroll spell resolution for steps whose SpellName uses
// dot-notation (e.g. `proj.deploy`); pass nil when no cross-scroll
// references are expected.
func RunPipeline(pd descriptor.PipelineDescriptor, dc *cache.DescriptorCache, spellIndex *resolve.SpellIndex, inputs map[string]any, hooks Hooks) (*Result, error) {
	if len(pd.Steps) == 0 {
		return nil, fmt.Errorf("pipeline %s has no steps", pd.CommandName)
	}
	if pd.Steps[0].Kind() != "spell" {
		return nil, fmt.Errorf("pipeline %s: first step must be a spell (reconciler invariant)", pd.CommandName)
	}

	state := &pipelineExecState{
		bindings:         map[string]any{},
		seen:             map[string]bool{},
		total:            countSpellSteps(pd.Steps),
		inputs:           inputs,
		descriptor_cache: dc,
		spell_index:      spellIndex,
		hooks:            hooks,
	}
	if err := executeSteps(pd.Steps, state, true); err != nil {
		return nil, err
	}

	result := &Result{Runtimes: state.runtimes}
	if state.terminalRan {
		result.FinalOutput = state.terminalOutput
	} else if state.prev_result != nil {
		// Branched ritual where the chosen branch ran no terminal-scope
		// spell-step (cond false, no else, or only let-steps after the
		// last spell). Surface the last captured output so callers still
		// see a final result.
		result.FinalOutput = state.prev_result.Output
	}
	return result, nil
}

// countSpellSteps walks the descriptor tree and counts spell-steps across
// all branches. For branched rituals where some branches are skipped, the
// runtime counter k will not always reach N — that's acceptable; the
// alternative (recomputing N without evaluating conditions) is impossible.
func countSpellSteps(steps []descriptor.StepDescriptor) int {
	n := 0
	for _, s := range steps {
		switch s.Kind() {
		case "if":
			n += countSpellSteps(s.Then)
			n += countSpellSteps(s.Else)
		case "let":
			// let-steps don't surface in the header counter
		default:
			n++
		}
	}
	return n
}

// executeSteps runs a step list, recursing into the chosen branch on an
// if-step. terminalScope propagates "this scope ends the ritual" so the
// final spell-step's output is recorded into state.terminalOutput. State
// mutates in place.
func executeSteps(steps []descriptor.StepDescriptor, state *pipelineExecState, terminalScope bool) error {
	for i, step := range steps {
		isLastInList := i == len(steps)-1
		thisTerminalScope := terminalScope && isLastInList

		if step.Kind() == "if" {
			parsed, err := expr.ParseCondition(step.Condition)
			if err != nil {
				return fmt.Errorf("parse condition %q: %v", step.Condition, err)
			}
			cond, err := expr.EvaluateCondition(parsed, state.bindings)
			if err != nil {
				return fmt.Errorf("evaluate condition %q: %v", step.Condition, err)
			}
			if cond {
				if err := executeSteps(step.Then, state, thisTerminalScope); err != nil {
					return err
				}
			} else if len(step.Else) > 0 {
				if err := executeSteps(step.Else, state, thisTerminalScope); err != nil {
					return err
				}
			}
			continue
		}

		if step.Kind() == "let" {
			parsed, err := expr.ParseCondition(step.Value)
			if err != nil {
				return fmt.Errorf("parse let %q value %q: %v", step.Let, step.Value, err)
			}
			val, err := expr.EvaluateExpression(parsed, state.bindings)
			if err != nil {
				return fmt.Errorf("evaluate let %q: %v", step.Let, err)
			}
			state.bindings[step.Let] = val
			continue
		}

		function_descriptor_ptr, err := resolve.ResolveSpellRef(step.SpellName, state.descriptor_cache, state.spell_index)
		if err != nil {
			return err
		}
		function_descriptor := *function_descriptor_ptr
		resolved_descriptor, err := resolve.ReconcileFunctionDescriptor(&function_descriptor)
		if err != nil {
			return fmt.Errorf("reconcile function descriptor: %v", err)
		}

		state.k++
		idx := state.k

		var payload map[string]interface{}
		if state.prev_result == nil {
			payload = state.inputs
		} else if len(step.Params) == 0 {
			payload, err = buildPayloadFromResult(state.prev_result.Output, resolved_descriptor)
			if err != nil {
				return err
			}
		} else {
			payload, err = buildPayloadFromBindings(step.Params, state.bindings, resolved_descriptor)
			if err != nil {
				return err
			}
		}

		if state.hooks.OnStepStart != nil {
			state.hooks.OnStepStart(idx, state.total, step.SpellName, thisTerminalScope)
		}

		runOpts := &runtime.RunOptions{SuppressFraming: true}
		if state.hooks.OnStepStderr != nil {
			runOpts.OnStderrLine = func(line string) {
				state.hooks.OnStepStderr(idx, line)
			}
		}

		runResult, err := runtime.Run(&resolved_descriptor, payload, runOpts)
		if err != nil {
			return fmt.Errorf("execute step: %v", err)
		}

		if state.hooks.OnStepFinish != nil {
			state.hooks.OnStepFinish(idx, runResult.Output)
		}

		if thisTerminalScope {
			state.terminalOutput = runResult.Output
			state.terminalRan = true
		}

		if runResult.Runtime != "" && !state.seen[runResult.Runtime] {
			state.seen[runResult.Runtime] = true
			state.runtimes = append(state.runtimes, runResult.Runtime)
		}

		state.prev_result = runResult

		if step.Id != "" {
			decoded, err := decodeStepOutput(runResult.Output)
			if err != nil {
				return err
			}
			state.bindings[step.Id] = decoded
		}
	}
	return nil
}

func decodeStepOutput(output []byte) (any, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, nil
	}

	var decoded any
	if err := json.Unmarshal(trimmed, &decoded); err != nil {
		return nil, fmt.Errorf("step output is not valid JSON: %v", err)
	}

	return decoded, nil
}

// buildPayloadFromResult turns the previous step's stdout (a single JSON
// value) into a payload for the next step. If the previous output is a JSON
// list whose length matches the next step's param count and there is more
// than one param, it unpacks positionally — mirroring Python's
// `return val1, val2`. If the previous output is a JSON map, it assigns
// values based on matching keys and param names. If all params are mapped,
// use that payload. Otherwise the whole decoded value is bound to the first
// param. Single-param steps never destructure, so a function that returns a
// list-as-data reaches the next step intact.
func buildPayloadFromResult(prev_output []byte, function_descriptor descriptor.FunctionDescriptor) (map[string]interface{}, error) {
	var decoded interface{}
	if len(bytes.TrimSpace(prev_output)) > 0 {
		if err := json.Unmarshal(prev_output, &decoded); err != nil {
			return nil, fmt.Errorf("step %s: previous output is not valid JSON: %v", function_descriptor.CommandName, err)
		}
	}

	payload := make(map[string]interface{})
	if list, ok := decoded.([]interface{}); ok && len(function_descriptor.Params) > 1 {
		if len(list) != len(function_descriptor.Params) {
			return nil, fmt.Errorf("step %s: previous output has %d values but step expects %d params", function_descriptor.CommandName, len(list), len(function_descriptor.Params))
		}
		for i, param := range function_descriptor.Params {
			payload[param.Name] = list[i]
		}
		return payload, nil
	} else if _map, ok := decoded.(map[string]interface{}); ok {
		mapped_params := 0
		for _, param := range function_descriptor.Params {
			if value, ok := _map[param.Name]; ok {
				payload[param.Name] = value
				mapped_params++
			}
		}
		if mapped_params == len(function_descriptor.Params) {
			return payload, nil
		}
	}

	payload = make(map[string]interface{})

	if len(function_descriptor.Params) >= 1 {
		payload[function_descriptor.Params[0].Name] = decoded
	}
	return payload, nil
}

func buildPayloadFromBindings(params map[string]any, bindings map[string]any, function_descriptor descriptor.FunctionDescriptor) (map[string]interface{}, error) {
	payload := make(map[string]interface{})

	for name, value := range params {
		ref_value, is_reference, err := expr.ResolveReference(value, bindings)
		if err != nil {
			return nil, err
		}
		if is_reference {
			payload[name] = ref_value
		} else {
			payload[name] = value
		}
	}

	for _, param := range function_descriptor.Params {
		if _, ok := payload[param.Name]; !ok {
			if param.Default != nil {
				payload[param.Name] = param.Default
			}
		}
	}

	return payload, nil
}
