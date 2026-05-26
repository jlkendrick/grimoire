package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	runtime "github.com/jlkendrick/grimoire/internal/runtime"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func buildFunctionCommand(function_descriptor descriptor.FunctionDescriptor) (*cobra.Command, error) {
	command := &cobra.Command{
		Use: function_descriptor.CommandName,
		Run: func(cmd *cobra.Command, args []string) {
			// Pre-run: check if the function descriptor is stale relative to the source code.
			// Scroll hash is checked in the root command before generating commands.
			resolved_descriptor, err := resolve.ReconcileFunctionDescriptor(&function_descriptor)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reconciling function descriptor: %v\n", err)
				os.Exit(1)
			}

			payload, err := buildPayload(resolved_descriptor, cmd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error building arguments: %v\n", err)
				os.Exit(1)
			}

			start := time.Now()
			runResult, err := runtime.Run(&resolved_descriptor, payload, nil)
			elapsed := time.Since(start)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error executing function: %v\n", err)
				os.Exit(1)
			}

			fmt.Println(string(runResult.Output))

			footerParts := []string{fmt.Sprintf("%.2fs", elapsed.Seconds())}
			if runResult.CacheStatus != "" {
				footerParts = append(footerParts, runResult.CacheStatus)
			}
			if runResult.Runtime != "" {
				footerParts = append(footerParts, runResult.Runtime)
			}
			fmt.Fprintf(os.Stderr, "\n%s %s\n", accent_style("◈"), dim_style(strings.Join(footerParts, " · ")))
		},
	}

	for _, param := range function_descriptor.Params {
		if err := registerParamFlag(command, param); err != nil {
			return nil, err
		}
	}

	return command, nil
}
