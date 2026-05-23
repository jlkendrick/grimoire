package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	runtime "github.com/jlkendrick/grimoire/internal/runtime"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

const previewMaxRunes = 80

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

// previewLine returns the first non-empty line of output, trimmed and
// truncated to previewMaxRunes runes (with an ellipsis if it overflowed).
func previewLine(output []byte) string {
	for _, raw := range bytes.Split(output, []byte("\n")) {
		line := strings.TrimSpace(string(raw))
		if line == "" {
			continue
		}
		if utf8.RuneCountInString(line) <= previewMaxRunes {
			return line
		}
		runes := []rune(line)
		return string(runes[:previewMaxRunes]) + "…"
	}
	return ""
}

// stepView owns the per-step output block for non-terminal steps.
// On a TTY it writes a header and animates a live preview sub-line beneath
// it; on non-TTY it stays quiet during the run and prints a single static
// block at finish — chatty spells would otherwise flood piped logs with
// every stderr line.
//
// The terminal step does NOT use stepView: its stderr (the user-function's
// visible output, redirected from stdout) needs to stream live to the user
// rather than being swallowed by a preview spinner.
type stepView struct {
	header string
	isTTY  bool
	sp     *utils.Spinner
}

func newStepView(idx, total int, spellName string) *stepView {
	return &stepView{
		header: fmt.Sprintf("step %d/%d · %s", idx, total, spellName),
		isTTY:  utils.StderrIsTTY(),
	}
}

func (v *stepView) start() {
	fmt.Fprintf(os.Stderr, "%s %s\n", accent_style("◈"), v.header)
	if !v.isTTY {
		return
	}
	v.sp = utils.NewSpinner("")
	v.sp.Start("")
}

func (v *stepView) updatePreview(line string) {
	if v.sp == nil {
		return
	}
	v.sp.UpdateHint(line)
}

func (v *stepView) finish(output []byte) {
	if v.sp != nil {
		v.sp.Stop()
	}

	if preview := previewLine(output); preview != "" {
		fmt.Fprintf(os.Stderr, "  %s %s\n\n", accent_style("→"), dim_style(preview))
	} else {
		fmt.Fprintln(os.Stderr)
	}
}

// pipelineExecState carries the shared state threaded through the recursive
// step walker. bindings and prev_result mutate across branches; k/total
// drive the step header counter. terminalPrinted catches the case where the
// chosen branch ran no spell-step but we still need to surface a final
// stdout to the user.
type pipelineExecState struct {
	bindings         map[string]any
	prev_result      *runtime.RunResult
	seen             map[string]bool
	runtimes         []string
	k                int
	total            int
	cmd              *cobra.Command
	descriptor_cache *cache.DescriptorCache
	terminalPrinted  bool
}

// countSpellSteps walks the descriptor tree and counts spell-steps across
// all branches. The result is used as the "N" in "step k/N" headers. For
// branched rituals where some branches are skipped, k will not always
// reach N — that's acceptable; the alternative (recomputing N at runtime
// without evaluating conditions) is impossible.
func countSpellSteps(steps []descriptor.StepDescriptor) int {
	n := 0
	for _, s := range steps {
		if s.Kind() == "if" {
			n += countSpellSteps(s.Then)
			n += countSpellSteps(s.Else)
			continue
		}
		n++
	}
	return n
}

// executeSteps runs a step list, recursing into the chosen branch on an
// if-step. terminalScope propagates "this scope ends the ritual" so the
// final spell-step can stream stderr live and print stdout via fmt.Println
// instead of going through the spinner-preview UI. State mutates in place.
func executeSteps(steps []descriptor.StepDescriptor, state *pipelineExecState, terminalScope bool) error {
	for i, step := range steps {
		isLastInList := i == len(steps)-1
		thisTerminalScope := terminalScope && isLastInList

		if step.Kind() == "if" {
			expr, err := resolve.ParseCondition(step.Condition)
			if err != nil {
				return fmt.Errorf("parse condition %q: %v", step.Condition, err)
			}
			cond, err := resolve.EvaluateCondition(expr, state.bindings)
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

		function_descriptor, ok := state.descriptor_cache.Functions[step.SpellName]
		if !ok {
			return fmt.Errorf("spell %s not found in descriptor cache", step.SpellName)
		}
		resolved_descriptor, err := resolve.ReconcileFunctionDescriptor(&function_descriptor)
		if err != nil {
			return fmt.Errorf("reconcile function descriptor: %v", err)
		}

		state.k++

		var payload map[string]interface{}
		if state.prev_result == nil {
			payload = buildPayload(resolved_descriptor, state.cmd)
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

		var runResult *runtime.RunResult
		if thisTerminalScope {
			label := fmt.Sprintf("step %d/%d · %s", state.k, state.total, step.SpellName)
			fmt.Fprintf(os.Stderr, "%s %s\n", accent_style("◈"), label)
			runResult, err = runtime.Run(&resolved_descriptor, payload, &runtime.RunOptions{
				SuppressFraming: true,
			})
			if err != nil {
				return fmt.Errorf("execute step: %v", err)
			}
			fmt.Println(string(runResult.Output))
			state.terminalPrinted = true
		} else {
			view := newStepView(state.k, state.total, step.SpellName)
			view.start()
			runResult, err = runtime.Run(&resolved_descriptor, payload, &runtime.RunOptions{
				SuppressFraming: true,
				OnStderrLine:    view.updatePreview,
			})
			if err != nil {
				return fmt.Errorf("execute step: %v", err)
			}
			view.finish(runResult.Output)
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

func buildPipelineCommand(pipeline_descriptor descriptor.PipelineDescriptor, descriptor_cache *cache.DescriptorCache) (*cobra.Command, error) {
	if len(pipeline_descriptor.Steps) == 0 {
		return nil, fmt.Errorf("pipeline %s has no steps", pipeline_descriptor.CommandName)
	}
	if pipeline_descriptor.Steps[0].Kind() != "spell" {
		return nil, fmt.Errorf("pipeline %s: first step must be a spell (reconciler invariant)", pipeline_descriptor.CommandName)
	}

	first_step_descriptor, ok := descriptor_cache.Functions[pipeline_descriptor.Steps[0].SpellName]
	if !ok {
		return nil, fmt.Errorf("pipeline %s: spell %s not found in descriptor cache", pipeline_descriptor.CommandName, pipeline_descriptor.Steps[0].SpellName)
	}

	command := &cobra.Command{
		Use: pipeline_descriptor.CommandName,
		Run: func(cmd *cobra.Command, args []string) {
			state := &pipelineExecState{
				bindings:         map[string]any{},
				seen:             map[string]bool{},
				total:            countSpellSteps(pipeline_descriptor.Steps),
				cmd:              cmd,
				descriptor_cache: descriptor_cache,
			}

			start := time.Now()
			if err := executeSteps(pipeline_descriptor.Steps, state, true); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

			// Branched rituals where the chosen branch contained no
			// spell-step at the terminal scope (cond false, no else) leave
			// terminalPrinted=false. Surface the last captured stdout so
			// the user still sees a final result.
			if !state.terminalPrinted && state.prev_result != nil {
				fmt.Println(string(state.prev_result.Output))
			}

			elapsed := time.Since(start)
			footerParts := []string{fmt.Sprintf("%.2fs", elapsed.Seconds())}
			if len(state.runtimes) > 0 {
				footerParts = append(footerParts, strings.Join(state.runtimes, ", "))
			}
			fmt.Fprintf(os.Stderr, "\n%s %s\n", accent_style("◈"), dim_style(strings.Join(footerParts, " · ")))
		},
	}

	// Forward the entry step's params as flags on the pipeline command, so
	// `grimoire <pipeline> --x 4` reaches the first step the same way
	// `grimoire <spell> --x 4` reaches a directly-cast spell.
	for _, param := range first_step_descriptor.Params {
		if err := registerParamFlag(command, param); err != nil {
			return nil, err
		}
	}

	return command, nil
}
