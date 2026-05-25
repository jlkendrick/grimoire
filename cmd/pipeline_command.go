package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	engine "github.com/jlkendrick/grimoire/internal/engine"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

const previewMaxRunes = 80

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
			inputs := buildPayload(first_step_descriptor, cmd)

			// stepView lifecycle is keyed by the engine's spell-step
			// counter. The terminal step never enters this map — its
			// header is printed directly in OnStepStart and its stderr
			// streams via the fallback branch of OnStepStderr.
			views := map[int]*stepView{}
			hooks := engine.Hooks{
				OnStepStart: func(idx, total int, spellName string, isTerminal bool) {
					if isTerminal {
						label := fmt.Sprintf("step %d/%d · %s", idx, total, spellName)
						fmt.Fprintf(os.Stderr, "%s %s\n", accent_style("◈"), label)
						return
					}
					v := newStepView(idx, total, spellName)
					v.start()
					views[idx] = v
				},
				OnStepStderr: func(idx int, line string) {
					if v, ok := views[idx]; ok {
						v.updatePreview(line)
						return
					}
					// Terminal step: matches runtime.Execute's fallback
					// fmt.Println so the spell's stderr surfaces live.
					fmt.Println(line)
				},
				OnStepFinish: func(idx int, output []byte) {
					if v, ok := views[idx]; ok {
						v.finish(output)
						delete(views, idx)
					}
				},
			}

			start := time.Now()
			result, err := engine.RunPipeline(pipeline_descriptor, descriptor_cache, inputs, hooks)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

			if result.FinalOutput != nil {
				fmt.Println(string(result.FinalOutput))
			}

			elapsed := time.Since(start)
			footerParts := []string{fmt.Sprintf("%.2fs", elapsed.Seconds())}
			if len(result.Runtimes) > 0 {
				footerParts = append(footerParts, strings.Join(result.Runtimes, ", "))
			}
			fmt.Fprintf(os.Stderr, "\n%s %s\n", accent_style("◈"), dim_style(strings.Join(footerParts, " · ")))
		},
	}

	// Forward the entry step's params as flags on the pipeline command, so
	// `grimoire <pipeline> --x 4` reaches the first step the same way
	// `grimoire <spell> --x 4` reaches a directly-cast spell. A ritual-level
	// override in the first step's `params:` wins over the spell's own default.
	first_step_overrides := pipeline_descriptor.Steps[0].Params
	for _, param := range first_step_descriptor.Params {
		if override, ok := first_step_overrides[param.Name]; ok {
			param.Default = override
		}
		if err := registerParamFlag(command, param); err != nil {
			return nil, err
		}
	}

	return command, nil
}
