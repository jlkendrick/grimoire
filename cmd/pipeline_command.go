package cmd

import (
	"os"
	"fmt"
	"bytes"
	"time"
	"strings"
	"encoding/json"

	"github.com/spf13/cobra"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	runtime "github.com/jlkendrick/grimoire/internal/runtime"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

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

			for _, step := range pipeline_descriptor.Steps {

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

				runResult, err := runtime.Run(&function_descriptor, payload)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error executing step: %v\n", err)
					os.Exit(1)
				}
				fmt.Println(string(runResult.Output))

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
