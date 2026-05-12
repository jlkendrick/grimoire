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

func buildPipelineCommand(pipeline_descriptor descriptor.PipelineDescriptor, descriptor_cache *cache.DescriptorCache) (*cobra.Command, error) {
	if len(pipeline_descriptor.Steps) == 0 {
		return nil, fmt.Errorf("pipeline %s has no steps", pipeline_descriptor.CommandName)
	}

	first_step_descriptor, ok := descriptor_cache.Functions[pipeline_descriptor.Steps[0].SpellName]
	if !ok {
		return nil, fmt.Errorf("pipeline %s: spell %s not found in descriptor cache", pipeline_descriptor.CommandName, pipeline_descriptor.Steps[0].SpellName)
	}

	command := &cobra.Command{
		Use: pipeline_descriptor.CommandName,
		Run: func(cmd *cobra.Command, args []string) {
			// Map of step ID to its outputs
			bindings := map[string]any{}
			var prev_result *runtime.RunResult

			start := time.Now()
			var runtimes []string
			seen := map[string]bool{}
			total := len(pipeline_descriptor.Steps)

			for i, step := range pipeline_descriptor.Steps {

				function_descriptor, ok := descriptor_cache.Functions[step.SpellName]
				if !ok {
					fmt.Fprintf(os.Stderr, "Spell %s not found in descriptor cache\n", step.SpellName)
					os.Exit(1)
				}

				var payload map[string]interface{}
				var err error
				if prev_result == nil {
					payload = buildPayload(function_descriptor, cmd)

					// No user-provided input overrides; do automatic binding
				} else if len(step.Params) == 0 {
					payload, err = buildPayloadFromResult(prev_result.Output, function_descriptor)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
						os.Exit(1)
					}

					// Use user-provided input overrides and references to previous step outputs
				} else {
					payload, err = buildPayloadFromBindings(step.Params, bindings, function_descriptor)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
						os.Exit(1)
					}
				}

				isTerminal := i == total-1

				var runResult *runtime.RunResult
				if isTerminal {
					// Terminal step: header first, then stream stderr and print
					// captured stdout exactly like a normal single-spell run.
					label := fmt.Sprintf("step %d/%d · %s", i+1, total, step.SpellName)
					fmt.Fprintf(os.Stderr, "%s %s\n", accent_style("◈"), label)

					runResult, err = runtime.Run(&function_descriptor, payload, &runtime.RunOptions{
						SuppressFraming: true,
					})
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error executing step: %v\n", err)
						os.Exit(1)
					}

					fmt.Println(string(runResult.Output))
				} else {
					view := newStepView(i+1, total, step.SpellName)
					view.start()
					runResult, err = runtime.Run(&function_descriptor, payload, &runtime.RunOptions{
						SuppressFraming: true,
						OnStderrLine:    view.updatePreview,
					})
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error executing step: %v\n", err)
						os.Exit(1)
					}
					view.finish(runResult.Output)
				}

				if runResult.Runtime != "" && !seen[runResult.Runtime] {
					seen[runResult.Runtime] = true
					runtimes = append(runtimes, runResult.Runtime)
				}

				prev_result = runResult

				if step.Id != "" {
					decoded, err := decodeStepOutput(runResult.Output)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%v\n", err)
						os.Exit(1)
					}
					bindings[step.Id] = decoded
				}
			}

			elapsed := time.Since(start)
			footerParts := []string{fmt.Sprintf("%.2fs", elapsed.Seconds())}
			if len(runtimes) > 0 {
				footerParts = append(footerParts, strings.Join(runtimes, ", "))
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
